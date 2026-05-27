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
	"github.com/pachimsh/cli/internal/ui"
	"github.com/spf13/cobra"
)

var pushSiteFlag string

var pushCmd = &cobra.Command{
	Use:   "push",
	Short: "Deploy the current project to your Pachim site",
	Long: `Packages the project (respecting .gitignore), uploads it,
and triggers a deployment on the selected site.`,
	RunE: runPush,
}

func init() {
	pushCmd.Flags().StringVar(&pushSiteFlag, "site", "", "Target site alias (from .pachim.json)")
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
		color.Red("Failed to fetch site info: %s", err)
		return nil
	}

	if !confirmGitMergeBeforePush(client, site.ID, siteInfo) {
		return nil
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

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
		deployResp, deployErr = client.DeployWithProgress(site.ID, zipPath, setProgress)
		return deployErr
	}); err != nil {
		color.Red("Deployment failed: %s", err)
		return nil
	}

	color.Green("✓ Deployment started (ID: %s)", deployResp.DeploymentID)
	fmt.Println()

	color.Yellow("⟳ Waiting for deployment to complete...")
	wasFirstDeploy := siteInfo.SetupType == ""
	return pollDeployment(client, site.ID, deployResp.DeploymentID, wasFirstDeploy)
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
	maxAttempts := 60
	interval := 5 * time.Second

	for i := 0; i < maxAttempts; i++ {
		time.Sleep(interval)

		status, err := client.GetDeploymentStatus(siteID, deploymentID)
		if err != nil {
			color.Yellow("  Checking status... (retrying)")
			continue
		}

		switch status.Status {
		case "finished":
			fmt.Println()
			color.Green("✓ Deployment completed successfully!")
			if status.FinishedAt != "" {
				fmt.Printf("  Finished at: %s\n", status.FinishedAt)
			}
			offerGitMergeAfterFirstDeploy(client, siteID, wasFirstDeploy)
			offerGitPush()
			return nil
		case "failed":
			fmt.Println()
			color.Red("✗ Deployment failed.")
			if status.Output != "" {
				fmt.Println()
				color.Yellow("── Deployment Log (last lines) ──")
				fmt.Println(status.Output)
				color.Yellow("─────────────────────────────────")
			}
			return nil
		case "deploying":
			fmt.Printf("\r  Status: deploying...   ")
		case "queued":
			fmt.Printf("\r  Status: queued...      ")
		default:
			fmt.Printf("\r  Status: %s   ", status.Status)
		}
	}

	color.Yellow("Timed out waiting for deployment. Check the Pachim dashboard for status.")
	return nil
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
