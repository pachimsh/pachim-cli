package cmd

import (
	"testing"

	"github.com/pachimsh/cli/internal/api"
	"github.com/pachimsh/cli/internal/config"
)

func TestFlattenCatalog(t *testing.T) {
	catalog := &api.Catalog{
		Workspaces: []api.CatalogWorkspace{
			{
				ID:   "ws-personal",
				Name: "Personal",
				Sites: []api.Site{
					{ID: "s1", Domain: "one.test", WorkspaceID: "ws-personal"},
					{ID: "s2", Domain: "two.test", WorkspaceID: "ws-personal"},
				},
			},
			{
				ID:   "ws-team",
				Name: "Team",
				Sites: []api.Site{
					{ID: "s3", Domain: "three.test", WorkspaceID: "ws-team"},
				},
			},
		},
	}

	entries := flattenCatalog(catalog)
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}

	if entries[0].Index != 1 || entries[0].Site.ID != "s1" || entries[0].Workspace.ID != "ws-personal" {
		t.Fatalf("unexpected first entry: %#v", entries[0])
	}

	if entries[2].Index != 3 || entries[2].Site.ID != "s3" {
		t.Fatalf("unexpected third entry: %#v", entries[2])
	}
}

func TestSetProjectWorkspaceFromEntries(t *testing.T) {
	entries := []catalogEntry{
		{Workspace: catalogWorkspaceRef{ID: "ws-team", IsPersonal: false}},
		{Workspace: catalogWorkspaceRef{ID: "ws-team", IsPersonal: false}},
	}

	cfg := &config.ProjectConfig{}
	setProjectWorkspaceFromEntries(cfg, entries)
	if cfg.WorkspaceID != "ws-team" {
		t.Fatalf("expected shared workspace id, got %q", cfg.WorkspaceID)
	}

	mixed := []catalogEntry{
		{Workspace: catalogWorkspaceRef{ID: "ws-a", IsPersonal: false}},
		{Workspace: catalogWorkspaceRef{ID: "ws-b", IsPersonal: false}},
	}

	cfg = &config.ProjectConfig{}
	setProjectWorkspaceFromEntries(cfg, mixed)
	if cfg.WorkspaceID != "" {
		t.Fatalf("expected empty workspace id for mixed workspaces, got %q", cfg.WorkspaceID)
	}
}

func TestCatalogWorkspaceKeyUsesID(t *testing.T) {
	a := catalogWorkspaceRef{ID: "ws-1", Name: "Alpha"}
	b := catalogWorkspaceRef{ID: "ws-2", Name: "Alpha"}

	if catalogWorkspaceKey(a) == catalogWorkspaceKey(b) {
		t.Fatal("expected different keys for different workspace ids")
	}
}
