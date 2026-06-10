package cmd

import (
	"github.com/fatih/color"
	"github.com/pachimsh/cli/internal/config"
	"github.com/spf13/cobra"
)

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Log out of your Pachim account",
	RunE:  runLogout,
}

func runLogout(cmd *cobra.Command, args []string) error {
	profile := resolveProfile()

	creds, err := config.LoadCredentials(profile)
	if err != nil {
		color.Yellow("You are not logged in.")
		return nil
	}

	client, err := newAPIClient(creds)
	if err != nil {
		color.Red("%s", err)
		return nil
	}

	_ = client.Logout()

	if err := config.DeleteCredentials(profile); err != nil {
		color.Red("Failed to remove local credentials: %s", err)
		return nil
	}

	color.Green("✓ Logged out successfully.")
	return nil
}
