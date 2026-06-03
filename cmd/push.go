package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/pachimsh/cli/internal/api"
	"github.com/pachimsh/cli/internal/archive"
	"github.com/pachimsh/cli/internal/config"
	"github.com/pachimsh/cli/internal/git"
	"github.com/pachimsh/cli/internal/ui"
	"github.com/spf13/cobra"
)

var pushSiteFlag string
var pushBranchFlag string
var pushSaveBranchFlag bool

var pushCmd = &cobra.Command{
	Use:   "push",
	Short: "Deploy the current project to your Pachim site",
	Long: `Packages the project (respecting .gitignore), uploads it,
and triggers a deployment on the selected site.`,
	RunE: runPush,
}

func init() {
	pushCmd.Flags().StringVar(&pushSiteFlag, "site", "", "Target site alias (from .pachim.json)")
	pushCmd.Flags().StringVar(&pushBranchFlag, "branch", "", "Git branch to deploy (defaults to current checkout)")
	pushCmd.Flags().BoolVar(&pushSaveBranchFlag, "save-branch", false, "Save deploy branch as site default on Pachim")
	rootCmd.AddCommand(pushCmd)
}

func runPush(cmd *cobra.Command, args []string) error {
	profile := resolveProfile()

	creds, err := config.LoadCredentials(profile)
	if err != nil {
		color.Red("You are not logged in. Run: pachim login")
		return nil
	}

	projCfg, err := config.LoadProjectConfig()
	if err != nil {
		color.Red("Project not initialized. Run: pachim init")
		return nil
	}

	targetAlias := pushSiteFlag
	if targetAlias == "" {
		targetAlias = projCfg.Default
	}

	site, ok := projCfg.Sites[targetAlias]
	if !ok {
		color.Red("Site '%s' not found in .pachim.json", targetAlias)
		fmt.Println("Available sites:")
		for alias := range projCfg.Sites {
			fmt.Printf("  • %s\n", alias)
		}
		return nil
	}

	color.Cyan("Deploying to: %s", site.Domain)
	fmt.Println()

	baseURL := resolveBaseURL()
	client := api.NewClient(baseURL, creds.Token)

	siteInfo, err := client.GetSiteInfo(site.ID)
	if err != nil {
		ui.PrintAPIError("Failed to fetch site info", err)
		return nil
	}

	wasFirstDeploy := siteInfo.SetupType == ""

	if decision := resolveActiveDeploymentBeforePush(siteInfo); decision == pushCancelled {
		color.Cyan("Cancelled.")
		return nil
	} else if decision == pushWatchExisting {
		fmt.Println()
		return pollDeployment(client, site.ID, siteInfo.ActiveDeployment.ID, wasFirstDeploy)
	}

	if !confirmGitMergeBeforePush(client, site.ID, siteInfo) {
		return nil
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	deployOpts := resolveDeployUploadOptions(cwd, site, siteInfo)

	tmpDir := os.TempDir()
	zipPath := filepath.Join(tmpDir, fmt.Sprintf("pachim-deploy-%d.zip", time.Now().UnixNano()))
	defer os.Remove(zipPath)

	if err := ui.RunWithProgress("Packaging project...", "Project packaged", func(setProgress func(int)) error {
		return archive.CreateProjectZipWithProgress(cwd, zipPath, archive.ProgressFunc(setProgress))
	}); err != nil {
		color.Red("Failed to package project: %s", err)
		return nil
	}

	info, _ := os.Stat(zipPath)
	fmt.Printf("  Package size: %.2f MB\n", float64(info.Size())/(1024*1024))
	fmt.Println()

	var deployResp *api.DeployResponse
	if err := ui.RunWithProgress("Uploading and starting deployment...", "Upload complete", func(setProgress func(int)) error {
		var deployErr error
		deployResp, deployErr = client.DeployWithProgress(site.ID, zipPath, deployOpts, setProgress)
		return deployErr
	}); err != nil {
		ui.PrintAPIError("Deployment failed", err)
		return nil
	}

	color.Green("✓ Deployment started (ID: %s)", deployResp.DeploymentID)
	fmt.Println()

	return pollDeployment(client, site.ID, deployResp.DeploymentID, wasFirstDeploy)
}

type pushDecision int

const (
	pushContinue pushDecision = iota
	pushWatchExisting
	pushCancelled
)

func resolveDeployUploadOptions(cwd string, site config.SiteConfig, siteInfo *api.SiteInfo) *api.DeployUploadOptions {
	opts := &api.DeployUploadOptions{}

	branch := strings.TrimSpace(pushBranchFlag)
	if branch == "" {
		branch = strings.TrimSpace(site.DeployBranch)
	}

	if head, ok := git.CurrentHead(cwd); ok {
		if branch == "" {
			branch = head.Branch
		}
		if head.Commit != "" {
			opts.CommitHash = head.Commit
		}
	}

	if branch == "" {
		branch = strings.TrimSpace(siteInfo.DeployBranch)
	}

	opts.Branch = branch
	opts.SaveBranchAsDefault = pushSaveBranchFlag

	if branch != "" || opts.CommitHash != "" {
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

func resolveActiveDeploymentBeforePush(siteInfo *api.SiteInfo) pushDecision {
	active := siteInfo.ActiveDeployment
	if active == nil {
		return pushContinue
	}

	fmt.Println()
	color.Yellow("⚠ This site already has an active deployment.")
	fmt.Printf("  Deployment ID: %s\n", active.ID)
	fmt.Printf("  Status: %s\n", active.Status)
	if active.InitiatedBy != "" {
		fmt.Printf("  Initiated by: %s\n", active.InitiatedBy)
	}
	if active.StartedAt != "" {
		fmt.Printf("  Started at: %s\n", active.StartedAt)
	}
	fmt.Println()
	fmt.Println("  [W] Watch current deployment")
	fmt.Println("  [Q] Queue new deployment anyway")
	fmt.Println("  [C] Cancel")
	fmt.Println()
	fmt.Print("Choose an option [W/q/c]: ")

	reader := bufio.NewReader(os.Stdin)
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(strings.ToLower(answer))

	switch answer {
	case "", "w", "watch":
		return pushWatchExisting
	case "q", "queue":
		fmt.Println()
		color.Cyan("Queueing a new deployment...")
		fmt.Println()
		return pushContinue
	default:
		return pushCancelled
	}
}

func confirmGitMergeBeforePush(client *api.Client, siteID string, siteInfo *api.SiteInfo) bool {
	if siteInfo.GitMerge {
		return true
	}

	// First deploy: site has no existing code yet, git merge is not relevant.
	if siteInfo.SetupType == "" {
		return true
	}

	fmt.Println()
	color.Yellow("⚠ Git merge is disabled for this site.")
	fmt.Println("  Your upload will completely replace the code on the server.")
	fmt.Println("  Enabling git merge lets future uploads merge changes (like git push).")
	fmt.Println()
	fmt.Print("Enable git merge and continue? [Y/n]: ")

	reader := bufio.NewReader(os.Stdin)
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(strings.ToLower(answer))

	if answer == "" || answer == "y" || answer == "yes" {
		if !enableGitMerge(client, siteID) {
			return false
		}
		fmt.Println()
		return true
	}

	fmt.Print("Continue with full replace instead? [y/N]: ")
	answer, _ = reader.ReadString('\n')
	answer = strings.TrimSpace(strings.ToLower(answer))
	if answer == "y" || answer == "yes" {
		fmt.Println()
		return true
	}

	color.Cyan("Cancelled.")
	return false
}

func enableGitMerge(client *api.Client, siteID string) bool {
	enabled, err := client.ToggleGitMerge(siteID)
	if err != nil {
		color.Red("Failed to enable git merge: %s", err)
		return false
	}
	if enabled {
		color.Green("✓ Git merge enabled.")
	}
	return true
}

func pollDeployment(client *api.Client, siteID, deploymentID string, wasFirstDeploy bool) error {
	const pollInterval = 2 * time.Second
	const maxPollDuration = 2 * time.Hour // deployments can take up to ~1 hour

	color.Yellow("⟳ Waiting for deployment to complete...")
	fmt.Println()

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
			color.Yellow("  Checking status... (retrying)")
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
			if status.FinishedAt != "" {
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
			fmt.Printf("\r  Status: queued...      ")
			onStatusLine = true
		default:
			fmt.Printf("\r  Status: %s   ", status.Status)
			onStatusLine = true
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
	if !wasFirstDeploy || !isGitRepo() {
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
	fmt.Print("Enable git merge for future pushes? [Y/n]: ")

	reader := bufio.NewReader(os.Stdin)
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(strings.ToLower(answer))

	if answer != "" && answer != "y" && answer != "yes" {
		return
	}

	enableGitMerge(client, siteID)
}

func offerGitPush() {
	if !isGitRepo() {
		return
	}

	if !hasUnpushedCommits() {
		return
	}

	fmt.Println()
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("You have unpushed commits. Push to remote? [y/N]: ")
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(strings.ToLower(answer))

	if answer != "y" && answer != "yes" {
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
