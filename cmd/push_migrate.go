package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/pachimsh/cli/internal/api"
	"github.com/pachimsh/cli/internal/config"
	"github.com/pachimsh/cli/internal/git"
)

func runLinkBranchSetup(projCfg *config.ProjectConfig, creds *config.Credentials, cwd string) error {
	missing := config.SitesMissingDeployBranch(projCfg)
	if len(missing) == 0 {
		return nil
	}

	fmt.Println()
	color.Cyan("Checking branch mappings from Pachim...")

	synced, stillMissing := syncDeployBranchesFromServer(projCfg, creds, missing)
	if synced {
		if err := config.SaveProjectConfig(projCfg); err != nil {
			return fmt.Errorf("failed to save .pachim.json: %w", err)
		}
		color.Green("✓ Branch mappings synced from Pachim to .pachim.json")
		fmt.Println()
	}

	missing = stillMissing
	if len(missing) == 0 {
		return nil
	}

	if pushYesFlag {
		return fmt.Errorf("deploy_branch is missing in .pachim.json and on Pachim; run once without --yes to configure branch mappings")
	}

	fmt.Println()
	color.Cyan("Some sites still need a git branch mapping in .pachim.json.")
	fmt.Println("  Map each site to a git branch so pachim push picks the right target automatically.")
	fmt.Println("  Example: develop → staging, main → production.")
	fmt.Println()

	head, _ := git.CurrentHead(cwd)
	currentBranch := ""
	if head != nil {
		currentBranch = head.Branch
	}

	reader := bufio.NewReader(os.Stdin)
	updated := false
	singleMissing := len(missing) == 1

	for _, alias := range missing {
		site := projCfg.Sites[alias]
		defaultBranch := ""
		if singleMissing {
			defaultBranch = currentBranch
		}

		branch := promptDeployBranchForSite(reader, site.Domain, defaultBranch)
		if branch == "" {
			color.Yellow("  Skipped %s — no branch saved.", alias)
			continue
		}

		for {
			if conflict := config.SiteAliasForBranch(projCfg, branch); conflict != "" && conflict != alias {
				color.Yellow("  Branch '%s' is already mapped to '%s'.", branch, conflict)
				branch = promptDeployBranchForSite(reader, site.Domain, "")
				if branch == "" {
					break
				}
				continue
			}
			break
		}

		if branch == "" {
			continue
		}

		site.DeployBranch = branch
		projCfg.Sites[alias] = site
		updated = true
	}

	if !updated {
		return nil
	}

	if err := config.SaveProjectConfig(projCfg); err != nil {
		return fmt.Errorf("failed to save .pachim.json: %w", err)
	}

	color.Green("✓ Branch mappings saved to .pachim.json")
	fmt.Println()
	return nil
}

func syncDeployBranchesFromServer(projCfg *config.ProjectConfig, creds *config.Credentials, aliases []string) (synced bool, stillMissing []string) {
	for _, alias := range aliases {
		site := projCfg.Sites[alias]

		client, err := newAPIClientForSite(creds, site)
		if err != nil {
			color.Yellow("  Could not resolve workspace for %s: %s", site.Domain, err)
			stillMissing = append(stillMissing, alias)
			continue
		}

		info, err := client.GetSiteInfo(site.ID)
		if err != nil {
			color.Yellow("  Could not fetch branch for %s: %s", site.Domain, err)
			stillMissing = append(stillMissing, alias)
			continue
		}

		serverBranch := strings.TrimSpace(info.DeployBranch)
		if serverBranch == "" {
			stillMissing = append(stillMissing, alias)
			continue
		}

		if conflict := config.SiteAliasForBranch(projCfg, serverBranch); conflict != "" && conflict != alias {
			color.Yellow("  %s: Pachim branch '%s' conflicts with site '%s' — needs manual mapping.", site.Domain, serverBranch, conflict)
			stillMissing = append(stillMissing, alias)
			continue
		}

		site.DeployBranch = serverBranch
		projCfg.Sites[alias] = site
		synced = true
		if !quietFlag {
			fmt.Printf("  %s → %s (from Pachim)\n", site.Domain, serverBranch)
		}
	}

	return synced, stillMissing
}

func promptDeployBranchForSite(reader *bufio.Reader, domain, defaultBranch string) string {
	if defaultBranch != "" {
		fmt.Printf("  Git branch for %s (Enter for '%s'): ", domain, defaultBranch)
	} else {
		fmt.Printf("  Git branch for %s (e.g. main, develop): ", domain)
	}

	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input != "" {
		return input
	}

	return defaultBranch
}

func offerSaveDeployBranchMapping(projCfg *config.ProjectConfig, creds *config.Credentials, alias, branch string) {
	branch = strings.TrimSpace(branch)
	site, ok := projCfg.Sites[alias]
	if !ok || strings.TrimSpace(site.DeployBranch) != "" {
		return
	}

	if branch == "" && creds != nil {
		if client, err := newAPIClientForSite(creds, site); err == nil {
			if info, err := client.GetSiteInfo(site.ID); err == nil {
				branch = strings.TrimSpace(info.DeployBranch)
			}
		}
	}

	if branch == "" {
		return
	}

	if conflict := config.SiteAliasForBranch(projCfg, branch); conflict != "" && conflict != alias {
		return
	}

	if !readYesNo(fmt.Sprintf("Save branch '%s' for site '%s' in .pachim.json? [Y/n]: ", branch, alias), true) {
		return
	}

	site.DeployBranch = branch
	projCfg.Sites[alias] = site
	if err := config.SaveProjectConfig(projCfg); err != nil {
		color.Red("Failed to save .pachim.json: %s", err)
		return
	}

	color.Green("✓ Saved deploy_branch for %s", alias)
	fmt.Println()
}

func resolveInitDeployBranch(reader *bufio.Reader, site api.Site) string {
	serverBranch := strings.TrimSpace(site.DeployBranch)
	if serverBranch != "" {
		fmt.Printf("Git branch for %s (from Pachim: '%s', Enter to keep): ", site.Domain, serverBranch)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		if input != "" {
			return input
		}
		return serverBranch
	}

	return promptDeployBranch(reader, site.Domain)
}

// syncBranchesFromServerOnly pulls deploy_branch from Pachim without interactive prompts.
func syncBranchesFromServerOnly(projCfg *config.ProjectConfig, creds *config.Credentials) error {
	missing := config.SitesMissingDeployBranch(projCfg)
	if len(missing) == 0 {
		return nil
	}

	logVerbose("Checking branch mappings from Pachim...")
	synced, stillMissing := syncDeployBranchesFromServer(projCfg, creds, missing)
	if synced {
		if err := config.SaveProjectConfig(projCfg); err != nil {
			return fmt.Errorf("failed to save .pachim.json: %w", err)
		}
		if verboseFlag {
			logSuccess("✓ Branch mappings synced from Pachim to .pachim.json")
		}
	}

	if len(stillMissing) > 0 {
		return errLinkSyncRequired
	}

	return nil
}
