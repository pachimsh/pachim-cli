package cmd

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/pachimsh/cli/internal/api"
	"github.com/pachimsh/cli/internal/config"
	"github.com/spf13/cobra"
)

var (
	deploymentsSiteFlag  string
	deploymentsLimitFlag int
)

var deploymentsCmd = &cobra.Command{
	Use:   "deployments",
	Short: "List deployments for a site",
	Long:  `Show recent deployments with their status and details.`,
	RunE:  runDeployments,
}

func init() {
	deploymentsCmd.Flags().StringVar(&deploymentsSiteFlag, "site", "", "Target site alias (from .pachim.json)")
	deploymentsCmd.Flags().IntVar(&deploymentsLimitFlag, "limit", 16, "Maximum number of deployments to show")
}

func runDeployments(cmd *cobra.Command, args []string) error {
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

	targetAlias := deploymentsSiteFlag
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

	baseURL := resolveBaseURL()
	client := api.NewClient(baseURL, creds.Token)

	result, err := client.ListDeployments(site.ID, deploymentsLimitFlag)
	if err != nil {
		color.Red("Failed to fetch deployments: %s", err)
		return nil
	}

	if len(result.Deployments) == 0 {
		color.Yellow("No deployments found for %s.", result.Domain)
		return nil
	}

	fmt.Println()
	color.Cyan("Deployments for: %s", result.Domain)
	fmt.Println(strings.Repeat("─", 120))
	fmt.Printf("  %-28s %-12s %-10s %-16s %-20s %-20s\n",
		"ID", "STATUS", "TYPE", "INITIATED BY", "STARTED", "FINISHED")
	fmt.Println(strings.Repeat("─", 120))

	for _, deployment := range result.Deployments {
		fmt.Printf("  %-28s %-12s %-10s %-16s %-20s %-20s\n",
			truncateDeploymentID(deployment.ID),
			colorizeDeploymentStatus(deployment.Status),
			deployment.Type,
			truncateText(deployment.InitiatedBy, 16),
			formatDeploymentTime(deployment.StartedAt),
			formatDeploymentTime(deployment.FinishedAt),
		)
	}

	fmt.Println(strings.Repeat("─", 120))
	fmt.Printf("  Showing %d deployment(s)\n", len(result.Deployments))
	fmt.Println()

	return nil
}

func truncateDeploymentID(id string) string {
	if len(id) <= 26 {
		return id
	}

	return id[:23] + "..."
}

func truncateText(value string, maxLen int) string {
	if value == "" {
		return "-"
	}

	if len(value) <= maxLen {
		return value
	}

	return value[:maxLen-3] + "..."
}

func formatDeploymentTime(value string) string {
	if value == "" {
		return "-"
	}

	if len(value) >= 19 {
		return value[:19]
	}

	return value
}

func colorizeDeploymentStatus(status string) string {
	switch status {
	case "finished":
		return color.GreenString(status)
	case "failed":
		return color.RedString(status)
	case "deploying":
		return color.YellowString(status)
	case "queued":
		return color.CyanString(status)
	default:
		return status
	}
}
