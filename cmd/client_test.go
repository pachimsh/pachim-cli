package cmd

import (
	"testing"

	"github.com/pachimsh/cli/internal/api"
	"github.com/pachimsh/cli/internal/config"
)

func TestResolveWorkspaceSelectorPriority(t *testing.T) {
	originalFlag := workspaceFlag
	t.Cleanup(func() {
		workspaceFlag = originalFlag
	})

	creds := &config.Credentials{
		Token:       "token",
		WorkspaceID: "profile-ws",
	}

	workspaceFlag = "flag-ws"
	t.Setenv("PACHIM_WORKSPACE_ID", "env-ws")
	t.Setenv("PACHIM_WORKSPACE", "env-slug")

	if got := resolveWorkspaceSelector(creds); got != "flag-ws" {
		t.Fatalf("expected flag workspace, got %q", got)
	}

	workspaceFlag = ""
	if got := resolveWorkspaceSelector(creds); got != "env-ws" {
		t.Fatalf("expected env workspace id, got %q", got)
	}

	t.Setenv("PACHIM_WORKSPACE_ID", "")
	if got := resolveWorkspaceSelector(creds); got != "env-slug" {
		t.Fatalf("expected env workspace slug, got %q", got)
	}

	t.Setenv("PACHIM_WORKSPACE", "")
	if got := resolveWorkspaceSelector(creds); got != "profile-ws" {
		t.Fatalf("expected profile workspace, got %q", got)
	}
}

func TestFindWorkspaceBySlug(t *testing.T) {
	workspaces := []api.Workspace{
		{ID: "01abc", Name: "Team Alpha", Slug: "team-alpha"},
		{ID: "01xyz", Name: "Personal", Slug: "personal", IsPersonal: true},
	}

	workspace, ok := findWorkspace(workspaces, "team-alpha")
	if !ok || workspace.ID != "01abc" {
		t.Fatalf("expected slug match, got %#v ok=%v", workspace, ok)
	}

	workspace, ok = findWorkspace(workspaces, "01xyz")
	if !ok || workspace.ID != "01xyz" {
		t.Fatalf("expected id match, got %#v ok=%v", workspace, ok)
	}
}

func TestIsPersonalWorkspaceTarget(t *testing.T) {
	for _, target := range []string{"personal", "default", "reset", "clear", " Personal "} {
		if !isPersonalWorkspaceTarget(target) {
			t.Fatalf("expected %q to reset workspace", target)
		}
	}

	if isPersonalWorkspaceTarget("team-alpha") {
		t.Fatal("expected team slug to remain explicit")
	}
}
