package cmd

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/pachimsh/cli/internal/config"
	"github.com/spf13/cobra"
)

var profilesCmd = &cobra.Command{
	Use:   "profiles",
	Short: "List all authenticated profiles",
	RunE:  runProfiles,
}

func init() {
	rootCmd.AddCommand(profilesCmd)
}

func runProfiles(cmd *cobra.Command, args []string) error {
	profiles, err := config.ListProfiles()
	if err != nil || len(profiles) == 0 {
		color.Yellow("No profiles found. Run: pachim login")
		return nil
	}

	activeProfile := resolveProfile()

	fmt.Println()
	color.Cyan("Profiles:")
	fmt.Println(strings.Repeat("─", 50))
	for _, p := range profiles {
		creds, err := config.LoadCredentials(p)
		if err != nil {
			continue
		}

		marker := "  "
		if p == activeProfile {
			marker = "▸ "
		}

		fmt.Printf("%s%-15s %s (%s)\n", marker, p, creds.Name, creds.Email)
	}
	fmt.Println(strings.Repeat("─", 50))
	fmt.Println()
	fmt.Println("Switch profile: pachim --profile <name> <command>")
	fmt.Println("Or set:         PACHIM_PROFILE=<name>")
	fmt.Println()

	return nil
}
