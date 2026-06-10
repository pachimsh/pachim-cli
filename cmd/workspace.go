package cmd

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/pachimsh/cli/internal/config"
	"github.com/spf13/cobra"
)

var workspaceCmd = &cobra.Command{
	Use:   "workspace",
	Short: "Manage the active Pachim workspace context",
	Long:  `Workspaces scope which sites and servers appear in pachim sites, init, and push.`,
}

var workspaceListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List workspaces available to your account",
	RunE:    runWorkspaceList,
}

var workspaceUseCmd = &cobra.Command{
	Use:   "use [id-or-slug]",
	Short: "Set the active workspace for this profile",
	Args:  cobra.ExactArgs(1),
	RunE:  runWorkspaceUse,
}

var workspaceCurrentCmd = &cobra.Command{
	Use:   "current",
	Short: "Show the active workspace context",
	RunE:  runWorkspaceCurrent,
}

func init() {
	workspaceCmd.AddCommand(workspaceListCmd, workspaceUseCmd, workspaceCurrentCmd)
}

func runWorkspaceList(cmd *cobra.Command, args []string) error {
	profile := resolveProfile()

	creds, err := config.LoadCredentials(profile)
	if err != nil {
		color.Red("You are not logged in. Run: pachim login")
		return nil
	}

	client := newBaseAPIClient(creds)
	workspaces, err := client.ListWorkspaces()
	if err != nil {
		color.Red("Failed to fetch workspaces: %s", err)
		return nil
	}

	if len(workspaces) == 0 {
		color.Yellow("No workspaces found.")
		return nil
	}

	activeID, err := resolveWorkspaceID(creds, client)
	if err != nil {
		color.Red("%s", err)
		return nil
	}

	fmt.Println()
	color.Cyan("Workspaces:")
	fmt.Println(strings.Repeat("─", 72))
	fmt.Printf("  %-26s %-12s %-10s %s\n", "NAME", "TYPE", "ROLE", "ID")
	fmt.Println(strings.Repeat("─", 72))

	for _, workspace := range workspaces {
		marker := " "
		if workspace.ID == activeID || (activeID == "" && workspace.IsPersonal) {
			marker = "*"
		}

		workspaceType := workspace.Type
		if workspaceType == "" && workspace.IsPersonal {
			workspaceType = "personal"
		}

		role := workspace.CurrentUserRole
		if role == "" {
			role = "—"
		}

		fmt.Printf("%s %-25s %-12s %-10s %s\n", marker, workspace.Name, workspaceType, role, workspace.ID)
	}

	fmt.Println(strings.Repeat("─", 72))
	fmt.Println("  * active context (explicit ID, profile default, or personal fallback)")
	fmt.Println()

	return nil
}

func runWorkspaceUse(cmd *cobra.Command, args []string) error {
	profile := resolveProfile()
	target := strings.TrimSpace(args[0])

	creds, err := config.LoadCredentials(profile)
	if err != nil {
		color.Red("You are not logged in. Run: pachim login")
		return nil
	}

	if isPersonalWorkspaceTarget(target) {
		if err := config.ClearProfileWorkspace(profile); err != nil {
			return fmt.Errorf("failed to reset workspace preference: %w", err)
		}

		color.Green("Active workspace reset to personal (default)")
		return nil
	}

	client := newBaseAPIClient(creds)
	workspaces, err := client.ListWorkspaces()
	if err != nil {
		color.Red("Failed to fetch workspaces: %s", err)
		return nil
	}

	workspace, ok := findWorkspace(workspaces, target)
	if !ok {
		color.Red("Workspace not found: %s", target)
		fmt.Println("  Run: pachim workspace list")
		return nil
	}

	if err := config.SetProfileWorkspace(profile, workspace.ID, workspace.Name, workspace.Slug); err != nil {
		return fmt.Errorf("failed to save workspace preference: %w", err)
	}

	color.Green("Active workspace set to: %s", workspace.Name)
	logVerbose("Workspace ID: %s", workspace.ID)

	return nil
}

func runWorkspaceCurrent(cmd *cobra.Command, args []string) error {
	profile := resolveProfile()

	creds, err := config.LoadCredentials(profile)
	if err != nil {
		color.Red("You are not logged in. Run: pachim login")
		return nil
	}

	client, err := newAPIClient(creds)
	if err != nil {
		color.Red("%s", err)
		return nil
	}

	current, err := client.GetCurrentWorkspace()
	if err != nil {
		color.Red("Failed to fetch current workspace: %s", err)
		return nil
	}

	fmt.Println()
	color.Cyan("Active workspace context:")
	if selector := resolveWorkspaceSelector(creds); selector != "" && !isPersonalWorkspaceTarget(selector) {
		fmt.Printf("  Selected: %s\n", selector)
		if client.WorkspaceID != "" {
			fmt.Printf("  ID:       %s\n", client.WorkspaceID)
		}
	} else {
		fmt.Println("  Selected: (not pinned — API uses personal workspace by default)")
	}

	if current != nil {
		fmt.Printf("  Resolved: %s (%s)\n", current.Name, current.ID)
		if current.CurrentUserRole != "" {
			fmt.Printf("  Role:     %s\n", current.CurrentUserRole)
		}
	}

	fmt.Printf("  Profile:  %s\n", profile)
	fmt.Println()

	return nil
}

func isPersonalWorkspaceTarget(target string) bool {
	switch strings.ToLower(strings.TrimSpace(target)) {
	case "personal", "default", "reset", "clear":
		return true
	default:
		return false
	}
}
