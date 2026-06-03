package cmd

import (
	"fmt"
	"os"

	"github.com/pachimsh/cli/internal/api"
	"github.com/pachimsh/cli/internal/config"
	"github.com/pachimsh/cli/internal/update"
	"github.com/spf13/cobra"
)

var version = "dev"

var apiURLFlag string
var profileFlag string

var rootCmd = &cobra.Command{
	Use:   "pachim",
	Short: "Pachim CLI - Deploy your projects with ease",
	Long: `Pachim CLI allows you to deploy your projects directly to your servers
managed by Pachim. Authenticate, select your site, and push changes seamlessly.`,
	Version: version,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	rootCmd.PersistentFlags().StringVar(&apiURLFlag, "api-url", "", "Override API base URL (default: https://api.pachim.sh)")
	_ = rootCmd.PersistentFlags().MarkHidden("api-url")
	rootCmd.PersistentFlags().StringVar(&profileFlag, "profile", "", "Use a specific profile (default: \"default\")")

	rootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		if shouldSkipUpdateCheck(cmd) {
			return
		}
		_ = update.MaybePromptAndUpdate(version, os.Args)
	}
}

func shouldSkipUpdateCheck(cmd *cobra.Command) bool {
	if cmd.Name() == "__set-api-url" || cmd.Name() == "self-update" {
		return true
	}

	for _, arg := range os.Args[1:] {
		switch arg {
		case "--version", "-v", "--help", "-h":
			return true
		}
	}

	return false
}

// CurrentVersion returns the CLI build version.
func CurrentVersion() string {
	return version
}

func resolveBaseURL() string {
	if apiURLFlag != "" {
		return apiURLFlag
	}

	if cfg, err := config.LoadGlobalConfig(); err == nil && cfg.APIURL != "" {
		return cfg.APIURL
	}

	return api.DefaultBaseURL
}

func resolveProfile() string {
	if profileFlag != "" {
		return profileFlag
	}

	if envProfile := os.Getenv("PACHIM_PROFILE"); envProfile != "" {
		return envProfile
	}

	return "default"
}
