package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"github.com/pachimsh/cli/internal/config"
	"github.com/pachimsh/cli/internal/git"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize this project for deployment to a Pachim site",
	Long: `Links the current project directory to one or more Pachim sites.
Creates a .pachim.json config file in the project root.`,
	RunE: runInit,
}

func runInit(cmd *cobra.Command, args []string) error {
	profile := resolveProfile()

	creds, err := config.LoadCredentials(profile)
	if err != nil {
		color.Red("You are not logged in. Run: pachim login")
		return nil
	}

	client := newCatalogAPIClient(creds)

	color.Yellow("Fetching your sites across all workspaces...")

	catalog, err := client.ListCatalog()
	if err != nil {
		color.Red("Failed to fetch sites: %s", err)
		return nil
	}

	entries := flattenCatalog(catalog)
	if len(entries) == 0 {
		color.Yellow("No sites found. Create a site on Pachim first.")
		return nil
	}

	printCatalogPicker("Select a site to link this project:", catalog, entries)

	reader := bufio.NewReader(os.Stdin)

	var existingCfg *config.ProjectConfig
	if existing, err := config.LoadProjectConfig(); err == nil {
		existingCfg = existing
	}

	if existingCfg == nil {
		existingCfg = &config.ProjectConfig{
			Sites: make(map[string]config.SiteConfig),
		}
	}

	fmt.Print("Enter number (comma-separated for multiple): ")
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	selections := strings.Split(input, ",")
	var selectedEntries []catalogEntry

	for _, sel := range selections {
		sel = strings.TrimSpace(sel)
		idx, err := strconv.Atoi(sel)
		if err != nil || idx < 1 || idx > len(entries) {
			color.Red("Invalid selection: %s", sel)
			return nil
		}
		entry, ok := findCatalogEntry(entries, idx)
		if !ok {
			color.Red("Invalid selection: %s", sel)
			return nil
		}
		selectedEntries = append(selectedEntries, entry)
	}

	for _, entry := range selectedEntries {
		site := entry.Site
		alias := site.Domain
		if len(selectedEntries) > 1 {
			fmt.Printf("Alias for %s (press Enter for '%s'): ", site.Domain, site.Domain)
			customAlias, _ := reader.ReadString('\n')
			customAlias = strings.TrimSpace(customAlias)
			if customAlias != "" {
				alias = customAlias
			}
		}

		deployBranch := resolveInitDeployBranch(reader, site)

		label := ""
		if len(selectedEntries) > 1 {
			fmt.Printf("Label for %s (optional, e.g. Production): ", site.Domain)
			labelInput, _ := reader.ReadString('\n')
			label = strings.TrimSpace(labelInput)
		}

		existingCfg.Sites[alias] = config.SiteConfig{
			ID:            site.ID,
			Domain:        site.Domain,
			DeployBranch:  deployBranch,
			Label:         label,
			WorkspaceID:   workspaceIDForEntry(entry),
			WorkspaceName: workspaceNameForEntry(entry),
		}
	}

	if len(selectedEntries) == 1 {
		alias := selectedEntries[0].Site.Domain
		for k, v := range existingCfg.Sites {
			if v.ID == selectedEntries[0].Site.ID {
				alias = k
				break
			}
		}
		existingCfg.Default = alias
	} else if len(selectedEntries) > 1 && existingCfg.Default == "" {
		fmt.Print("Select default site for 'pachim push' (enter alias): ")
		defaultAlias, _ := reader.ReadString('\n')
		defaultAlias = strings.TrimSpace(defaultAlias)
		if _, ok := existingCfg.Sites[defaultAlias]; ok {
			existingCfg.Default = defaultAlias
		} else {
			for k := range existingCfg.Sites {
				existingCfg.Default = k
				break
			}
		}
	}

	setProjectWorkspaceFromEntries(existingCfg, selectedEntries)

	if err := config.SaveProjectConfig(existingCfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	color.Green("✓ Project initialized! Config saved to .pachim.json")
	fmt.Printf("  Default site: %s\n", existingCfg.Default)

	if len(existingCfg.Sites) > 1 {
		fmt.Println("  Branch mappings:")
		for _, alias := range config.SortedSiteAliases(existingCfg) {
			site := existingCfg.Sites[alias]
			name := site.DisplayName(alias)
			branch := site.DeployBranch
			if branch == "" {
				branch = "(not set)"
			}
			fmt.Printf("    • %s → %s [branch: %s]\n", name, site.Domain, branch)
		}
		fmt.Println("  pachim push auto-selects the site from your current git branch.")
		fmt.Println("  Use --site <alias> or 'pachim use <alias>' to override the default.")
	} else if site, ok := existingCfg.Sites[existingCfg.Default]; ok && site.DeployBranch != "" {
		fmt.Printf("  Deploy branch: %s\n", site.DeployBranch)
	}

	addToGitignore()

	fmt.Println()
	color.Cyan("Next step:")
	fmt.Println("  • Run: pachim push")

	return nil
}

func promptDeployBranch(reader *bufio.Reader, domain string) string {
	cwd, err := os.Getwd()
	defaultBranch := ""
	if err == nil {
		if head, ok := git.CurrentHead(cwd); ok {
			defaultBranch = head.Branch
		}
	}

	if defaultBranch != "" {
		fmt.Printf("Git branch for %s (Enter for '%s'): ", domain, defaultBranch)
	} else {
		fmt.Printf("Git branch for %s (optional, e.g. develop or main): ", domain)
	}

	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input != "" {
		return input
	}

	return defaultBranch
}

func addToGitignore() {
	cmd := exec.Command("git", "rev-parse", "--is-inside-work-tree")
	if cmd.Run() != nil {
		return
	}

	gitignorePath := ".gitignore"
	entry := ".pachim.json"

	content, err := os.ReadFile(gitignorePath)
	if err == nil {
		for _, line := range strings.Split(string(content), "\n") {
			if strings.TrimSpace(line) == entry {
				return
			}
		}
	}

	f, err := os.OpenFile(gitignorePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()

	if len(content) > 0 && !strings.HasSuffix(string(content), "\n") {
		f.WriteString("\n")
	}

	f.WriteString(entry + "\n")
	color.Green("✓ Added .pachim.json to .gitignore")
}
