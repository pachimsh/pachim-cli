package cmd

import "github.com/spf13/cobra"

func init() {
	cobra.EnableCommandSorting = false

	registerCommands()
}

func registerCommands() {
	rootCmd.AddCommand(
		loginCmd,
		logoutCmd,
		whoamiCmd,
		profilesCmd,
		initCmd,
		useCmd,
		pushCmd,
		selfUpdateCmd,
		deploymentsCmd,
		sitesCmd,
		gitMergeCmd,
	)
}
