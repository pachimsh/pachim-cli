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

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	tmpDir := os.TempDir()
	zipPath := filepath.Join(tmpDir, fmt.Sprintf("pachim-deploy-%d.zip", time.Now().UnixNano()))
	defer os.Remove(zipPath)

	color.Yellow("⟳ Packaging project...")
	if err := archive.CreateProjectZip(cwd, zipPath); err != nil {
		color.Red("Failed to package project: %s", err)
		return nil
	}

	info, _ := os.Stat(zipPath)
	fmt.Printf("  Package size: %.2f MB\n", float64(info.Size())/(1024*1024))
	fmt.Println()

	baseURL := resolveBaseURL()
	client := api.NewClient(baseURL, creds.Token)

	siteInfo, err := client.GetSiteInfo(site.ID)
	if err == nil && !siteInfo.GitMerge {
		fmt.Println()
		color.Yellow("⚠ Git merge is disabled for this site.")
		fmt.Println("  This means the uploaded code will completely replace the existing code.")
		fmt.Println()
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Continue? [y/N]: ")
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(strings.ToLower(answer))
		if answer != "y" && answer != "yes" {
			color.Cyan("Cancelled. To enable git merge, run: pachim git-merge --enable")
			return nil
		}
		fmt.Println()
	}

	color.Yellow("⟳ Uploading and starting deployment...")
	deployResp, err := client.Deploy(site.ID, zipPath)
	if err != nil {
		color.Red("Deployment failed: %s", err)
		return nil
	}

	color.Green("✓ Deployment started (ID: %s)", deployResp.DeploymentID)
	fmt.Println()

	color.Yellow("⟳ Waiting for deployment to complete...")
	return pollDeployment(client, site.ID, deployResp.DeploymentID)
}

func pollDeployment(client *api.Client, siteID, deploymentID string) error {
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
