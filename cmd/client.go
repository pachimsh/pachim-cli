package cmd

import (
	"strings"

	"github.com/pachimsh/cli/internal/api"
	"github.com/pachimsh/cli/internal/config"
)

var workspaceFlag string

func initClientFlags() {
	rootCmd.PersistentFlags().StringVar(
		&workspaceFlag,
		"workspace",
		"",
		"Active workspace ID or slug (overrides profile and project settings)",
	)
}

func newBaseAPIClient(creds *config.Credentials) *api.Client {
	return api.NewClient(resolveBaseURL(), creds.Token)
}

func newAPIClient(creds *config.Credentials) (*api.Client, error) {
	client := newBaseAPIClient(creds)

	workspaceID, err := resolveWorkspaceID(creds, client)
	if err != nil {
		return nil, err
	}

	client.WorkspaceID = workspaceID

	return client, nil
}

func activeWorkspaceLabel(creds *config.Credentials) string {
	if creds == nil {
		return ""
	}

	if name := strings.TrimSpace(creds.WorkspaceName); name != "" {
		return name
	}

	if slug := strings.TrimSpace(creds.WorkspaceSlug); slug != "" {
		return slug
	}

	if id := strings.TrimSpace(creds.WorkspaceID); id != "" {
		return id
	}

	return "personal (default)"
}
