package cmd

import (
	"fmt"
	"strings"

	"github.com/pachimsh/cli/internal/api"
	"github.com/pachimsh/cli/internal/config"
)

func newCatalogAPIClient(creds *config.Credentials) *api.Client {
	return newBaseAPIClient(creds)
}

func newAPIClientForSite(creds *config.Credentials, site config.SiteConfig) (*api.Client, error) {
	client := newBaseAPIClient(creds)

	workspaceID, err := resolveSiteWorkspaceID(creds, site)
	if err != nil {
		return nil, err
	}

	client.WorkspaceID = workspaceID

	return client, nil
}

func resolveSiteWorkspaceID(creds *config.Credentials, site config.SiteConfig) (string, error) {
	selector := strings.TrimSpace(site.WorkspaceID)
	if selector == "" {
		if projCfg, err := config.LoadProjectConfig(); err == nil {
			selector = strings.TrimSpace(projCfg.WorkspaceID)
		}
	}

	if selector == "" && creds != nil {
		selector = resolveWorkspaceSelector(creds)
	}

	if selector == "" || isPersonalWorkspaceTarget(selector) {
		return "", nil
	}

	client := newBaseAPIClient(creds)
	workspaces, err := client.ListWorkspaces()
	if err != nil {
		return "", fmt.Errorf("failed to resolve workspace %q: %w", selector, err)
	}

	workspace, ok := findWorkspace(workspaces, selector)
	if !ok {
		return "", fmt.Errorf("workspace not found: %s (run: pachim workspace list)", selector)
	}

	if workspace.IsPersonal {
		return "", nil
	}

	return workspace.ID, nil
}

func remoteSiteIDsFromCatalog(catalog *api.Catalog) map[string]struct{} {
	validIDs := make(map[string]struct{})
	for _, entry := range flattenCatalog(catalog) {
		validIDs[entry.Site.ID] = struct{}{}
	}

	return validIDs
}
