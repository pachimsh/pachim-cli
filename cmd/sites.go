package cmd

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/pachimsh/cli/internal/api"
	"github.com/pachimsh/cli/internal/config"
	"github.com/spf13/cobra"
)

var sitesCmd = &cobra.Command{
	Use:   "sites",
	Short: "List all your Pachim sites",
	RunE:  runSites,
}

func runSites(cmd *cobra.Command, args []string) error {
	profile := resolveProfile()

	creds, err := config.LoadCredentials(profile)
	if err != nil {
		color.Red("You are not logged in. Run: pachim login")
		return nil
	}

	baseURL := resolveBaseURL()
	client := api.NewClient(baseURL, creds.Token)

	sites, err := client.ListSites()
	if err != nil {
		color.Red("Failed to fetch sites: %s", err)
		return nil
	}

	if len(sites) == 0 {
		color.Yellow("No sites found.")
		return nil
	}

	fmt.Println()
	color.Cyan("Your Sites:")
	fmt.Println(strings.Repeat("─", 70))
	fmt.Printf("  %-28s %-10s %-12s %-15s\n", "DOMAIN", "TYPE", "STATUS", "SERVER")
	fmt.Println(strings.Repeat("─", 70))
	for _, site := range sites {
		status := site.SetupType
		if status == "" {
			status = "not setup"
		}
		fmt.Printf("  %-28s %-10s %-12s %-15s\n", site.Domain, site.AppType, status, site.Server.Name)
	}
	fmt.Println(strings.Repeat("─", 70))
	fmt.Printf("  Total: %d sites\n", len(sites))
	fmt.Println()

	return nil
}
