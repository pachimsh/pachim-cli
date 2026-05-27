package cmd

import (
	"github.com/fatih/color"
	"github.com/pachim/cli/internal/api"
	"github.com/pachim/cli/internal/config"
	"github.com/spf13/cobra"
)

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Log out of your Pachim account",
	RunE:  runLogout,
}

func init() {
	rootCmd.AddCommand(logoutCmd)
}

func runLogout(cmd *cobra.Command, args []string) error {
	profile := resolveProfile()

	creds, err := config.LoadCredentials(profile)
	if err != nil {
		color.Yellow("You are not logged in.")
		return nil
	}

	baseURL := resolveBaseURL()
	if creds.APIUrl != "" {
		baseURL = creds.APIUrl
	}

	client := api.NewClient(baseURL, creds.Token)
	_ = client.Logout()

	if err := config.DeleteCredentials(profile); err != nil {
		color.Red("Failed to remove local credentials: %s", err)
		return nil
	}

	color.Green("✓ Logged out successfully.")
	return nil
}
