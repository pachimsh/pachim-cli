package cmd

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/pachimsh/cli/internal/config"
	"github.com/spf13/cobra"
)

var whoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "Show the currently authenticated user",
	RunE:  runWhoami,
}

func runWhoami(cmd *cobra.Command, args []string) error {
	profile := resolveProfile()

	creds, err := config.LoadCredentials(profile)
	if err != nil {
		color.Red("You are not logged in. Run: pachim login")
		return nil
	}

	fmt.Println()
	color.Cyan("Logged in as:")
	fmt.Printf("  Name:    %s\n", creds.Name)
	fmt.Printf("  Email:   %s\n", creds.Email)
	fmt.Printf("  Profile: %s\n", profile)
	fmt.Println()

	return nil
}
