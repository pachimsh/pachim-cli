package cmd

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/pachimsh/cli/internal/update"
	"github.com/spf13/cobra"
)

var selfUpdateCmd = &cobra.Command{
	Use:   "self-update",
	Short: "Update the pachim CLI to the latest release",
	Long: `Checks mirrors.pachim.app for the latest release and updates this binary.

Example:
  pachim self-update`,
	RunE: runSelfUpdate,
}

func runSelfUpdate(cmd *cobra.Command, args []string) error {
	if err := update.SelfUpdate(CurrentVersion()); err != nil {
		color.Red("%s", err)
		return fmt.Errorf("%w", err)
	}

	return nil
}
