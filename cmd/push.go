package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/pachimsh/cli/internal/api"
	"github.com/pachimsh/cli/internal/archive"
	"github.com/pachimsh/cli/internal/git"
	"github.com/pachimsh/cli/internal/ui"
	"github.com/spf13/cobra"
)

var pushSiteFlag string
var pushBranchFlag string
var pushSaveBranchFlag bool
var pushYesFlag bool
var pushDryRunFlag bool
var pushForceFlag bool

var pushCmd = &cobra.Command{
	Use:   "push",
	Short: "Deploy the current project to your Pachim site",
	Long: `Packages the project (respecting .gitignore), uploads it,
and triggers a deployment on the selected site.

When multiple sites are configured, pachim push selects the target from your
current git branch using deploy_branch in .pachim.json.

Run 'pachim link sync' first to configure branch mappings.
Run 'pachim status' to preview the deploy plan without uploading.`,
	RunE: runPush,
}

func init() {
	pushCmd.Flags().StringVar(&pushSiteFlag, "site", "", "Target site alias (from .pachim.json)")
	pushCmd.Flags().StringVar(&pushBranchFlag, "branch", "", "Git branch to deploy (defaults to current checkout)")
	pushCmd.Flags().BoolVar(&pushSaveBranchFlag, "save-branch", false, "Save deploy branch as site default on Pachim")
	pushCmd.Flags().BoolVarP(&pushYesFlag, "yes", "y", false, "Skip deploy confirmation prompts")
	pushCmd.Flags().BoolVar(&pushDryRunFlag, "dry-run", false, "Show deploy plan without uploading")
	pushCmd.Flags().BoolVar(&pushForceFlag, "force", false, "Deploy even with production branch + uncommitted changes")
}

func runPush(cmd *cobra.Command, args []string) error {
	ctx, err := loadProjectContext(true)
	if err != nil {
		if err == errPushAborted {
			return nil
		}
		return err
	}

	if err := syncProjectConfigWithServer(ctx.projCfg, ctx.creds); err != nil {
		if err == errNoConfiguredSites {
			color.Red("No sites configured. Run: pachim init")
			return nil
		}
		ui.PrintAPIError("Failed to sync project config", err)
		return nil
	}

	if err := syncBranchesFromServerOnly(ctx.projCfg, ctx.creds); err != nil {
		if err == errLinkSyncRequired {
			color.Red("Branch mappings are incomplete.")
			fmt.Println("  Run: pachim link sync")
			return nil
		}
		return err
	}

	target, err := resolvePushTarget(ctx.projCfg, ctx.creds, ctx.cwd)
	if err != nil {
		if err == errPushAborted {
			color.Cyan("Cancelled.")
			return nil
		}
		color.Red("%s", err)
		return nil
	}

	siteClient, err := newAPIClientForSite(ctx.creds, target.Site)
	if err != nil {
		color.Red("%s", err)
		return nil
	}

	siteInfo, err := siteClient.GetSiteInfo(target.Site.ID)
	if err != nil {
		ui.PrintAPIError("Failed to fetch site info", err)
		return nil
	}

	plan := buildDeployPlan(target, siteInfo, ctx.cwd)
	decision, _, err := resolveDeployPlan(siteClient, plan, pushYesFlag, pushForceFlag, pushDryRunFlag)
	if err != nil {
		return err
	}

	switch decision {
	case planCancelled:
		if !pushDryRunFlag {
			color.Cyan("Cancelled.")
		}
		return nil
	case planWatch:
		fmt.Println()
		return pollDeployment(siteClient, target.Site.ID, siteInfo.ActiveDeployment.ID, plan.FirstDeploy)
	case planDeploy:
		// continue below
	}

	if pushDryRunFlag {
		return nil
	}

	deployOpts := resolveDeployUploadOptions(ctx.cwd, target, siteInfo)

	tmpDir := os.TempDir()
	zipPath := filepath.Join(tmpDir, fmt.Sprintf("pachim-deploy-%d.zip", time.Now().UnixNano()))
	defer os.Remove(zipPath)

	if err := ui.RunWithProgress("Packaging project...", "Project packaged", func(setProgress func(int)) error {
		return archive.CreateProjectZipWithProgress(ctx.cwd, zipPath, archive.ProgressFunc(setProgress))
	}); err != nil {
		color.Red("Failed to package project: %s", err)
		return nil
	}

	info, _ := os.Stat(zipPath)
	if !quietFlag {
		fmt.Printf("  Package size: %.2f MB\n", float64(info.Size())/(1024*1024))
		fmt.Println()
	}

	var deployResp *api.DeployResponse
	if err := ui.RunWithProgress("Uploading and starting deployment...", "Upload complete", func(setProgress func(int)) error {
		var deployErr error
		deployResp, deployErr = siteClient.DeployWithProgress(target.Site.ID, zipPath, deployOpts, setProgress)
		return deployErr
	}); err != nil {
		ui.PrintAPIError("Deployment failed", err)
		return nil
	}

	if !quietFlag {
		color.Green("✓ Deployment started (ID: %s)", deployResp.DeploymentID)
		fmt.Println()
	}

	return pollDeployment(siteClient, target.Site.ID, deployResp.DeploymentID, plan.FirstDeploy)
}

func resolveDeployUploadOptions(cwd string, target *pushTarget, siteInfo *api.SiteInfo) *api.DeployUploadOptions {
	opts := &api.DeployUploadOptions{}

	branch := resolveDeployBranch(target, siteInfo, cwd)

	if target.Head != nil && target.Head.Commit != "" {
		opts.CommitHash = target.Head.Commit
	} else if head, ok := git.CurrentHead(cwd); ok && head.Commit != "" {
		opts.CommitHash = head.Commit
	}

	opts.Branch = branch
	opts.SaveBranchAsDefault = pushSaveBranchFlag

	if verboseFlag && (branch != "" || opts.CommitHash != "") {
		label := branch
		if label == "" {
			label = "(detached)"
		}
		if opts.CommitHash != "" {
			short := opts.CommitHash
			if len(short) > 7 {
				short = short[:7]
			}
			fmt.Printf("  Deploy branch: %s @ %s\n", label, short)
		} else {
			fmt.Printf("  Deploy branch: %s\n", label)
		}
	}

	return opts
}

func pollDeployment(client *api.Client, siteID, deploymentID string, wasFirstDeploy bool) error {
	const pollInterval = 2 * time.Second
	const maxPollDuration = 2 * time.Hour

	if !quietFlag {
		color.Yellow("⟳ Waiting for deployment to complete...")
		fmt.Println()
	}

	streamer := newDeployOutputStreamer()
	onStatusLine := false
	deadline := time.Now().Add(maxPollDuration)

	for time.Now().Before(deadline) {
		time.Sleep(pollInterval)

		status, err := client.GetDeploymentStatus(siteID, deploymentID)
		if err != nil {
			if onStatusLine {
				fmt.Println()
				onStatusLine = false
			}
			if !quietFlag {
				color.Yellow("  Checking status... (retrying)")
			}
			continue
		}

		switch status.Status {
		case "deploying":
			if onStatusLine {
				fmt.Println()
				onStatusLine = false
			}
			streamer.Write(status.Output, false)
		case "finished":
			if onStatusLine {
				fmt.Println()
				onStatusLine = false
			}
			streamer.Write(status.Output, true)
			fmt.Println()
			color.Green("✓ Deployment completed successfully!")
			if status.FinishedAt != "" && !quietFlag {
				fmt.Printf("  Finished at: %s\n", status.FinishedAt)
			}
			offerGitMergeAfterFirstDeploy(client, siteID, wasFirstDeploy)
			offerGitPush()
			return nil
		case "failed":
			if onStatusLine {
				fmt.Println()
				onStatusLine = false
			}
			streamer.Write(status.Output, true)
			fmt.Println()
			color.Red("✗ Deployment failed.")
			return nil
		case "queued":
			if !quietFlag {
				fmt.Printf("\r  Status: queued...      ")
				onStatusLine = true
			}
		default:
			if !quietFlag {
				fmt.Printf("\r  Status: %s   ", status.Status)
				onStatusLine = true
			}
		}
	}

	if onStatusLine {
		fmt.Println()
	}
	streamer.Write("", true)
	color.Yellow("Timed out waiting for deployment. Check the Pachim dashboard for status.")
	return nil
}

type deployOutputStreamer struct {
	printedBytes int
	lineBuffer   strings.Builder
}

func newDeployOutputStreamer() *deployOutputStreamer {
	return &deployOutputStreamer{}
}

func (s *deployOutputStreamer) Write(output string, final bool) {
	if len(output) < s.printedBytes {
		s.printedBytes = 0
		s.lineBuffer.Reset()
	}

	if len(output) <= s.printedBytes && !final {
		return
	}

	chunk := output[s.printedBytes:]
	s.printedBytes = len(output)

	content := s.lineBuffer.String() + chunk
	s.lineBuffer.Reset()

	if content == "" {
		return
	}

	lines := strings.Split(content, "\n")

	if !final && !strings.HasSuffix(content, "\n") && len(lines) > 0 {
		s.lineBuffer.WriteString(lines[len(lines)-1])
		lines = lines[:len(lines)-1]
	}

	for _, line := range lines {
		line = strings.TrimRight(line, "\r")
		fmt.Println(line)
	}

	if final && s.lineBuffer.Len() > 0 {
		fmt.Println(s.lineBuffer.String())
		s.lineBuffer.Reset()
	}
}

func offerGitMergeAfterFirstDeploy(client *api.Client, siteID string, wasFirstDeploy bool) {
	if !wasFirstDeploy || !isGitRepo() || quietFlag {
		return
	}

	siteInfo, err := client.GetSiteInfo(siteID)
	if err != nil || siteInfo.GitMerge {
		return
	}

	fmt.Println()
	color.Cyan("First deploy completed successfully.")
	fmt.Println("  Enable git merge so future uploads merge with the server (like git push).")
	fmt.Println()

	if !readYesNo("Enable git merge for future pushes? [Y/n]: ", true) {
		return
	}

	enableGitMergeOnServer(client, siteID)
}

func offerGitPush() {
	if !isGitRepo() || quietFlag {
		return
	}

	if !hasUnpushedCommits() {
		return
	}

	fmt.Println()
	if !readYesNo("You have unpushed commits. Push to remote? [y/N]: ", false) {
		return
	}

	cmd := exec.Command("git", "push")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		color.Red("git push failed: %s", err)
		return
	}

	color.Green("✓ Pushed to remote.")
}

func isGitRepo() bool {
	cmd := exec.Command("git", "rev-parse", "--is-inside-work-tree")
	return cmd.Run() == nil
}

func hasUnpushedCommits() bool {
	cmd := exec.Command("git", "status", "--porcelain", "-b")
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	return strings.Contains(string(output), "ahead")
}
