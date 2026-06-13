package cmd

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/pachimsh/cli/internal/config"
)

var errNoConfiguredSites = fmt.Errorf("no sites configured")

func syncProjectConfigWithServer(projCfg *config.ProjectConfig, creds *config.Credentials) error {
	if len(projCfg.Sites) == 0 {
		return errNoConfiguredSites
	}

	client := newCatalogAPIClient(creds)
	catalog, err := client.ListCatalog()
	if err != nil {
		return fmt.Errorf("failed to fetch sites from Pachim: %w", err)
	}

	validIDs := remoteSiteIDsFromCatalog(catalog)

	removed, changed := config.PruneStaleSites(projCfg, validIDs)
	if !changed {
		return nil
	}

	if !quietFlag {
		fmt.Println()
		color.Yellow("Removed stale sites from .pachim.json (no longer on Pachim):")
		for _, alias := range removed {
			fmt.Printf("  • %s\n", alias)
		}

		if projCfg.Default != "" {
			fmt.Printf("  Default site is now: %s\n", projCfg.Default)
		}
	} else {
		logVerbose("Pruned stale sites: %v", removed)
	}

	if err := config.SaveProjectConfig(projCfg); err != nil {
		return fmt.Errorf("failed to save .pachim.json: %w", err)
	}

	if !quietFlag {
		color.Green("✓ Updated .pachim.json")
		fmt.Println()
	}

	if len(projCfg.Sites) == 0 {
		return errNoConfiguredSites
	}

	return nil
}
