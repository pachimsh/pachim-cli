package cmd

import (
	"github.com/fatih/color"
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

	client := newCatalogAPIClient(creds)

	catalog, err := client.ListCatalog()
	if err != nil {
		color.Red("Failed to fetch sites: %s", err)
		return nil
	}

	entries := flattenCatalog(catalog)
	if len(entries) == 0 {
		color.Yellow("No sites found.")
		return nil
	}

	printCatalogTable(catalog, entries)

	return nil
}
