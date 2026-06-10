package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/pachimsh/cli/internal/api"
	"github.com/pachimsh/cli/internal/config"
)

func resolveWorkspaceSelector(creds *config.Credentials) string {
	if selector := strings.TrimSpace(workspaceFlag); selector != "" {
		return selector
	}

	if selector := strings.TrimSpace(os.Getenv("PACHIM_WORKSPACE_ID")); selector != "" {
		return selector
	}

	if selector := strings.TrimSpace(os.Getenv("PACHIM_WORKSPACE")); selector != "" {
		return selector
	}

	if projCfg, err := config.LoadProjectConfig(); err == nil {
		if selector := strings.TrimSpace(projCfg.WorkspaceID); selector != "" {
			return selector
		}
	}

	if creds != nil {
		if selector := strings.TrimSpace(creds.WorkspaceID); selector != "" {
			return selector
		}

		if selector := strings.TrimSpace(creds.WorkspaceSlug); selector != "" {
			return selector
		}
	}

	return ""
}

func resolveWorkspaceID(creds *config.Credentials, client *api.Client) (string, error) {
	selector := resolveWorkspaceSelector(creds)
	if selector == "" || isPersonalWorkspaceTarget(selector) {
		return "", nil
	}

	workspaces, err := client.ListWorkspaces()
	if err != nil {
		return "", fmt.Errorf("failed to resolve workspace %q: %w", selector, err)
	}

	workspace, ok := findWorkspace(workspaces, selector)
	if !ok {
		return "", fmt.Errorf("workspace not found: %s (run: pachim workspace list)", selector)
	}

	return workspace.ID, nil
}

func findWorkspace(workspaces []api.Workspace, target string) (api.Workspace, bool) {
	target = strings.TrimSpace(target)
	lowerTarget := strings.ToLower(target)

	for _, workspace := range workspaces {
		if workspace.ID == target {
			return workspace, true
		}
	}

	for _, workspace := range workspaces {
		if strings.EqualFold(workspace.Slug, target) {
			return workspace, true
		}
	}

	for _, workspace := range workspaces {
		if strings.Contains(strings.ToLower(workspace.Name), lowerTarget) {
			return workspace, true
		}
	}

	return api.Workspace{}, false
}
