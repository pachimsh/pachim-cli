package config

import "strings"

func SetProfileWorkspace(profile string, workspaceID, workspaceName, workspaceSlug string) error {
	creds, err := LoadCredentials(profile)
	if err != nil {
		return err
	}

	creds.WorkspaceID = strings.TrimSpace(workspaceID)
	creds.WorkspaceName = strings.TrimSpace(workspaceName)
	creds.WorkspaceSlug = strings.TrimSpace(workspaceSlug)

	return SaveCredentials(profile, creds)
}

func ClearProfileWorkspace(profile string) error {
	creds, err := LoadCredentials(profile)
	if err != nil {
		return err
	}

	creds.WorkspaceID = ""
	creds.WorkspaceName = ""
	creds.WorkspaceSlug = ""

	return SaveCredentials(profile, creds)
}
