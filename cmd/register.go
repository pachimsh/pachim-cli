package cmd

import "github.com/spf13/cobra"

func init() {
	cobra.EnableCommandSorting = false

	registerCommands()
}

func registerCommands() {
	initOutputFlags()
	initClientFlags()

	rootCmd.AddCommand(
		loginCmd,
		logoutCmd,
		whoamiCmd,
		profilesCmd,
		workspaceCmd,
		initCmd,
		useCmd,
		linkCmd,
		statusCmd,
		pushCmd,
		selfUpdateCmd,
		deploymentsCmd,
		sitesCmd,
		gitMergeCmd,
	)
}
