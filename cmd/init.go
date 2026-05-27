package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"github.com/pachim/cli/internal/api"
	"github.com/pachim/cli/internal/config"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize this project for deployment to a Pachim site",
	Long: `Links the current project directory to one or more Pachim sites.
Creates a .pachim.json config file in the project root.`,
	RunE: runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	profile := resolveProfile()

	creds, err := config.LoadCredentials(profile)
	if err != nil {
		color.Red("You are not logged in. Run: pachim login")
		return nil
	}

	baseURL := resolveBaseURL()
	if creds.APIUrl != "" {
		baseURL = creds.APIUrl
	}

	client := api.NewClient(baseURL, creds.Token)

	color.Yellow("Fetching your sites...")

	sites, err := client.ListSites()
	if err != nil {
		color.Red("Failed to fetch sites: %s", err)
		return nil
	}

	if len(sites) == 0 {
		color.Yellow("No sites found. Create a site on Pachim first.")
		return nil
	}

	fmt.Println()
	fmt.Println("Your sites:")
	fmt.Println(strings.Repeat("-", 60))
	for i, site := range sites {
		status := ""
		if site.SetupType == "" {
			status = " (not setup - will auto-setup on first push)"
		}
		fmt.Printf("  %d) %s [%s] (%s)%s\n", i+1, site.Domain, site.AppType, site.Server.Name, status)
	}
	fmt.Println(strings.Repeat("-", 60))
	fmt.Println()

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

	fmt.Print("Select site number (or comma-separated for multiple): ")
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	selections := strings.Split(input, ",")
	var selectedSites []api.Site

	for _, sel := range selections {
		sel = strings.TrimSpace(sel)
		idx, err := strconv.Atoi(sel)
		if err != nil || idx < 1 || idx > len(sites) {
			color.Red("Invalid selection: %s", sel)
			return nil
		}
		selectedSites = append(selectedSites, sites[idx-1])
	}

	for _, site := range selectedSites {
		alias := site.Domain
		if len(selectedSites) > 1 {
			fmt.Printf("Alias for %s (press Enter for '%s'): ", site.Domain, site.Domain)
			customAlias, _ := reader.ReadString('\n')
			customAlias = strings.TrimSpace(customAlias)
			if customAlias != "" {
				alias = customAlias
			}
		}

		existingCfg.Sites[alias] = config.SiteConfig{
			ID:     site.ID,
			Domain: site.Domain,
		}
	}

	if len(selectedSites) == 1 {
		alias := selectedSites[0].Domain
		for k, v := range existingCfg.Sites {
			if v.ID == selectedSites[0].ID {
				alias = k
				break
			}
		}
		existingCfg.Default = alias
	} else if len(selectedSites) > 1 && existingCfg.Default == "" {
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

	if err := config.SaveProjectConfig(existingCfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	color.Green("✓ Project initialized! Config saved to .pachim.json")
	fmt.Printf("  Default site: %s\n", existingCfg.Default)

	if len(existingCfg.Sites) > 1 {
		fmt.Println("  Use --site <alias> or 'pachim use <alias>' to change default")
	}

	addToGitignore()

	fmt.Println()
	color.Cyan("Next step:")
	fmt.Println("  • Run: pachim push")

	return nil
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
