package cmd

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/pachimsh/cli/internal/config"
	"github.com/pachimsh/cli/internal/ui"
	"github.com/spf13/cobra"
)

var statusSiteFlag string

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show deploy plan for the current project without deploying",
	Long: `Displays which site would receive a push, branch mapping, git state,
and server status — without uploading or changing anything.`,
	RunE: runStatus,
}

func init() {
	statusCmd.Flags().StringVar(&statusSiteFlag, "site", "", "Preview deploy target for a specific site alias")
}

func runStatus(cmd *cobra.Command, args []string) error {
	ctx, err := loadProjectContext(true)
	if err != nil {
		if err == errPushAborted {
			return nil
		}
		return err
	}

	savedSiteFlag := pushSiteFlag
	pushSiteFlag = statusSiteFlag
	defer func() { pushSiteFlag = savedSiteFlag }()

	target, err := resolvePushTarget(ctx.projCfg, ctx.creds, ctx.cwd)
	if err != nil {
		if err == errPushAborted {
			return nil
		}
		color.Red("%s", err)
		return nil
	}

	siteClient, err := newAPIClientForSite(ctx.creds, target.Site)
	if err != nil {
		color.Red("%s", err)
		return nil
	}

	siteInfo, err := siteClient.GetSiteInfo(target.Site.ID)
	if err != nil {
		ui.PrintAPIError("Failed to fetch site info", err)
		return nil
	}

	plan := buildDeployPlan(target, siteInfo, ctx.cwd)
	printDeployPlan(plan)

	if plan.canSilentDeploy(false, false) {
		fmt.Println()
		color.Green("  Ready to deploy — run: pachim push")
	} else {
		fmt.Println()
		color.Yellow("  Review warnings above — run: pachim push")
	}

	missing := config.SitesMissingDeployBranch(ctx.projCfg)
	if len(missing) > 0 {
		color.Yellow("  %d site(s) need branch mapping — run: pachim link sync", len(missing))
	}

	fmt.Println()
	return nil
}
