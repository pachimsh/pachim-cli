package cmd

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/pachimsh/cli/internal/api"
	"github.com/pachimsh/cli/internal/config"
	"github.com/pachimsh/cli/internal/git"
	"github.com/spf13/cobra"
)

var errLinkSyncRequired = fmt.Errorf("branch mappings incomplete — run: pachim link sync")

var linkCmd = &cobra.Command{
	Use:   "link",
	Short: "Manage .pachim.json project link settings",
	Long:  `Sync and inspect the link between this project and your Pachim sites.`,
}

var linkSyncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync .pachim.json with Pachim (sites + branch mappings)",
	RunE:  runLinkSync,
}

var linkShowCmd = &cobra.Command{
	Use:     "show",
	Aliases: []string{"list"},
	Short:   "Show configured site and branch mappings",
	RunE:    runLinkShow,
}

func init() {
	linkCmd.AddCommand(linkSyncCmd, linkShowCmd)
}

func runLinkSync(cmd *cobra.Command, args []string) error {
	ctx, err := loadProjectContext(true)
	if err != nil {
		return err
	}

	if err := syncProjectConfigWithServer(ctx.projCfg, ctx.client); err != nil {
		if err == errNoConfiguredSites {
			color.Red("No sites configured. Run: pachim init")
			return nil
		}
		return err
	}

	if err := runLinkBranchSetup(ctx.projCfg, ctx.client, ctx.cwd); err != nil {
		if err == errPushAborted {
			color.Cyan("Cancelled.")
			return nil
		}
		return err
	}

	if !quietFlag {
		fmt.Println()
		runLinkShow(cmd, args)
	}

	return nil
}

func runLinkShow(cmd *cobra.Command, args []string) error {
	projCfg, err := config.LoadProjectConfig()
	if err != nil {
		color.Red("Project not initialized. Run: pachim init")
		return nil
	}

	cwd, _ := os.Getwd()
	head, _ := git.CurrentHead(cwd)

	fmt.Println()
	fmt.Println("Project link (.pachim.json):")
	fmt.Printf("  Default: %s\n", valueOr(projCfg.Default, "(not set)"))
	fmt.Println()

	if len(projCfg.Sites) == 0 {
		color.Yellow("  No sites configured.")
		return nil
	}

	for _, alias := range config.SortedSiteAliases(projCfg) {
		site := projCfg.Sites[alias]
		name := site.DisplayName(alias)
		branch := valueOr(site.DeployBranch, "(not mapped)")
		marker := "  "
		if alias == projCfg.Default {
			marker = "* "
		}

		fmt.Printf("%s%s — %s\n", marker, name, site.Domain)
		fmt.Printf("     branch: %s\n", branch)
		if name != alias {
			fmt.Printf("     alias:  %s\n", alias)
		}
	}

	if head != nil && head.Branch != "" {
		fmt.Println()
		matches := config.SitesForBranch(projCfg, head.Branch)
		switch len(matches) {
		case 1:
			s := projCfg.Sites[matches[0]]
			color.Cyan("  Current git branch '%s' → %s (%s)", head.Branch, s.DisplayName(matches[0]), s.Domain)
		case 0:
			color.Yellow("  Current git branch '%s' has no site mapping.", head.Branch)
		default:
			color.Yellow("  Current git branch '%s' matches multiple sites: %v", head.Branch, matches)
		}
	}

	missing := config.SitesMissingDeployBranch(projCfg)
	if len(missing) > 0 {
		fmt.Println()
		color.Yellow("  %d site(s) missing deploy_branch. Run: pachim link sync", len(missing))
	}

	fmt.Println()
	return nil
}

func valueOr(v, fallback string) string {
	if v == "" {
		return fallback
	}
	return v
}

type projectContext struct {
	profile string
	creds   *config.Credentials
	client  *api.Client
	projCfg *config.ProjectConfig
	cwd     string
}

func loadProjectContext(requireLogin bool) (*projectContext, error) {
	profile := resolveProfile()

	var creds *config.Credentials
	var err error
	if requireLogin {
		creds, err = config.LoadCredentials(profile)
		if err != nil {
			color.Red("You are not logged in. Run: pachim login")
			return nil, errPushAborted
		}
	}

	projCfg, err := config.LoadProjectConfig()
	if err != nil {
		color.Red("Project not initialized. Run: pachim init")
		return nil, errPushAborted
	}

	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get working directory: %w", err)
	}

	ctx := &projectContext{
		profile: profile,
		creds:   creds,
		projCfg: projCfg,
		cwd:     cwd,
	}

	if creds != nil {
		client, err := newAPIClient(creds)
		if err != nil {
			color.Red("%s", err)
			return nil, errPushAborted
		}

		ctx.client = client
	}

	return ctx, nil
}
