package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/fatih/color"
	"github.com/pachim/cli/internal/api"
	"github.com/pachim/cli/internal/config"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with your Pachim account",
	RunE:  runLogin,
}

func init() {
	rootCmd.AddCommand(loginCmd)
}

func runLogin(cmd *cobra.Command, args []string) error {
	profile := resolveProfile()
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Email: ")
	email, _ := reader.ReadString('\n')
	email = strings.TrimSpace(email)

	if email == "" {
		return fmt.Errorf("email is required")
	}

	fmt.Print("Password: ")
	passwordBytes, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return fmt.Errorf("failed to read password: %w", err)
	}
	fmt.Println()

	password := string(passwordBytes)
	if password == "" {
		return fmt.Errorf("password is required")
	}

	baseURL := resolveBaseURL()
	client := api.NewClient(baseURL, "")

	color.Yellow("Authenticating...")

	loginResp, err := client.Login(email, password)
	if err != nil {
		color.Red("Login failed: %s", err)
		return nil
	}

	creds := &config.Credentials{
		Token:  loginResp.Token,
		Email:  loginResp.User.Email,
		Name:   loginResp.User.Name,
		APIUrl: baseURL,
	}

	if err := config.SaveCredentials(profile, creds); err != nil {
		return fmt.Errorf("failed to save credentials: %w", err)
	}

	color.Green("✓ Logged in as %s (%s)", creds.Name, creds.Email)
	if profile != "default" {
		color.Cyan("  Profile: %s", profile)
	}
	return nil
}
