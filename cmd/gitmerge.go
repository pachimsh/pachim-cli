package cmd

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/pachimsh/cli/internal/api"
	"github.com/pachimsh/cli/internal/config"
	"github.com/spf13/cobra"
)

var gitMergeEnableFlag bool
var gitMergeDisableFlag bool

var gitMergeCmd = &cobra.Command{
	Use:   "git-merge",
	Short: "Manage git merge mode for uploads",
	Long: `When git merge is enabled, uploaded code is merged with the existing
code on the server (like a git push). When disabled, the uploaded code
completely replaces the existing code.`,
	RunE: runGitMerge,
}

func init() {
	gitMergeCmd.Flags().BoolVar(&gitMergeEnableFlag, "enable", false, "Enable git merge for uploads")
	gitMergeCmd.Flags().BoolVar(&gitMergeDisableFlag, "disable", false, "Disable git merge for uploads")
	rootCmd.AddCommand(gitMergeCmd)
}

func runGitMerge(cmd *cobra.Command, args []string) error {
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

	site, ok := projCfg.Sites[projCfg.Default]
	if !ok {
		color.Red("Default site not found in .pachim.json")
		return nil
	}

	baseURL := resolveBaseURL()
	if creds.APIUrl != "" {
		baseURL = creds.APIUrl
	}

	client := api.NewClient(baseURL, creds.Token)

	if !gitMergeEnableFlag && !gitMergeDisableFlag {
		info, err := client.GetSiteInfo(site.ID)
		if err != nil {
			color.Red("Failed to get site info: %s", err)
			return nil
		}

		fmt.Printf("Site: %s\n", info.Domain)
		if info.GitMerge {
			color.Green("Git merge: enabled")
			fmt.Println("  Uploaded code is merged with existing code (like git push).")
		} else {
			color.Yellow("Git merge: disabled")
			fmt.Println("  Uploaded code completely replaces existing code.")
		}
		fmt.Println()
		fmt.Println("Use --enable or --disable to change.")
		return nil
	}

	if gitMergeEnableFlag && gitMergeDisableFlag {
		color.Red("Cannot use --enable and --disable together.")
		return nil
	}

	info, err := client.GetSiteInfo(site.ID)
	if err != nil {
		color.Red("Failed to get site info: %s", err)
		return nil
	}

	needsToggle := (gitMergeEnableFlag && !info.GitMerge) || (gitMergeDisableFlag && info.GitMerge)

	if !needsToggle {
		if gitMergeEnableFlag {
			color.Green("Git merge is already enabled.")
		} else {
			color.Yellow("Git merge is already disabled.")
		}
		return nil
	}

	newState, err := client.ToggleGitMerge(site.ID)
	if err != nil {
		color.Red("Failed to toggle git merge: %s", err)
		return nil
	}

	if newState {
		color.Green("✓ Git merge enabled for %s", site.Domain)
		fmt.Println("  Uploaded code will be merged with existing code.")
	} else {
		color.Yellow("✓ Git merge disabled for %s", site.Domain)
		fmt.Println("  Uploaded code will completely replace existing code.")
	}

	return nil
}
