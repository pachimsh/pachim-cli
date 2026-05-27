package cmd

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/pachimsh/cli/internal/config"
	"github.com/spf13/cobra"
)

var setAPIURLReset bool

var setAPIURLCmd = &cobra.Command{
	Use:    "__set-api-url [url]",
	Hidden: true,
	Args:   cobra.MaximumNArgs(1),
	RunE:   runSetAPIURL,
}

func init() {
	setAPIURLCmd.Flags().BoolVar(&setAPIURLReset, "reset", false, "")
	setAPIURLCmd.Flags().MarkHidden("reset")
	rootCmd.AddCommand(setAPIURLCmd)
}

func runSetAPIURL(cmd *cobra.Command, args []string) error {
	if setAPIURLReset {
		if err := config.ClearGlobalAPIURL(); err != nil {
			return fmt.Errorf("failed to reset API URL: %w", err)
		}
		color.Green("API URL reset to default.")
		return nil
	}

	if len(args) == 0 {
		return fmt.Errorf("url is required (or use --reset)")
	}

	if err := config.SaveGlobalAPIURL(args[0]); err != nil {
		return fmt.Errorf("failed to save API URL: %w", err)
	}

	color.Green("API URL saved.")
	return nil
}
