package cmd

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/pachimsh/cli/internal/api"
	"github.com/pachimsh/cli/internal/config"
)

type catalogEntry struct {
	Index     int
	Site      api.Site
	Workspace catalogWorkspaceRef
}

type catalogWorkspaceRef struct {
	ID              string
	Name            string
	Slug            string
	Type            string
	IsPersonal      bool
	CurrentUserRole string
}

func flattenCatalog(catalog *api.Catalog) []catalogEntry {
	if catalog == nil {
		return nil
	}

	var entries []catalogEntry
	index := 1

	for _, workspace := range catalog.Workspaces {
		ref := catalogWorkspaceRefFromAPI(workspace)

		for _, site := range workspace.Sites {
			entryRef := ref
			if site.WorkspaceID != "" {
				entryRef.ID = site.WorkspaceID
			}
			if site.WorkspaceName != "" {
				entryRef.Name = site.WorkspaceName
			}

			entries = append(entries, catalogEntry{
				Index:     index,
				Site:      site,
				Workspace: entryRef,
			})
			index++
		}
	}

	return entries
}

func catalogWorkspaceRefFromAPI(workspace api.CatalogWorkspace) catalogWorkspaceRef {
	return catalogWorkspaceRef{
		ID:              workspace.ID,
		Name:            workspace.Name,
		Slug:            workspace.Slug,
		Type:            workspace.Type,
		IsPersonal:      workspace.IsPersonal,
		CurrentUserRole: workspace.CurrentUserRole,
	}
}

func catalogSiteCount(catalog *api.Catalog) int {
	count := 0
	for _, workspace := range catalog.Workspaces {
		count += len(workspace.Sites)
	}
	return count
}

func printCatalogPicker(title string, catalog *api.Catalog, entries []catalogEntry) {
	fmt.Println()
	color.Cyan("%s", title)
	fmt.Println()

	workspaceCount := 0
	for _, workspace := range catalog.Workspaces {
		if len(workspace.Sites) == 0 {
			continue
		}
		workspaceCount++

		ref := catalogWorkspaceRefFromAPI(workspace)
		label := workspaceLabelRef(ref)
		role := workspaceRoleLabelRef(ref)
		if role != "" {
			color.New(color.FgHiCyan, color.Bold).Printf("  %s", label)
			fmt.Printf("  %s\n", color.New(color.FgHiBlack).Sprint(role))
		} else {
			color.New(color.FgHiCyan, color.Bold).Printf("  %s\n", label)
		}

		fmt.Println("  " + strings.Repeat("─", 68))

		workspaceKey := catalogWorkspaceKey(ref)
		for _, entry := range entries {
			if catalogWorkspaceKey(entry.Workspace) != workspaceKey {
				continue
			}

			status := ""
			if entry.Site.SetupType == "" {
				status = color.New(color.FgYellow).Sprint(" · not setup")
			}

			appType := entry.Site.AppType
			if appType == "" {
				appType = "—"
			}

			serverName := entry.Site.Server.Name
			if serverName == "" {
				serverName = "—"
			}

			fmt.Printf("   %2d  %-26s %-12s @ %s%s\n",
				entry.Index,
				entry.Site.Domain,
				appType,
				serverName,
				status,
			)
		}

		fmt.Println()
	}

	fmt.Println("  " + strings.Repeat("─", 68))
	fmt.Printf("  Total: %d sites across %d workspace(s)\n", len(entries), workspaceCount)
	fmt.Println()
}

func printCatalogTable(catalog *api.Catalog, entries []catalogEntry) {
	fmt.Println()
	color.Cyan("Your Sites (all workspaces):")
	fmt.Println(strings.Repeat("─", 88))
	fmt.Printf("  %-26s %-10s %-12s %-15s %s\n", "DOMAIN", "TYPE", "STATUS", "SERVER", "WORKSPACE")
	fmt.Println(strings.Repeat("─", 88))

	for _, entry := range entries {
		status := entry.Site.SetupType
		if status == "" {
			status = "not setup"
		}

		appType := entry.Site.AppType
		if appType == "" {
			appType = "—"
		}

		serverName := entry.Site.Server.Name
		if serverName == "" {
			serverName = "—"
		}

		fmt.Printf("  %-26s %-10s %-12s %-15s %s\n",
			entry.Site.Domain,
			appType,
			status,
			serverName,
			workspaceLabelRef(entry.Workspace),
		)
	}

	fmt.Println(strings.Repeat("─", 88))
	fmt.Printf("  Total: %d sites\n", len(entries))
	fmt.Println()
}

func catalogWorkspaceKey(ref catalogWorkspaceRef) string {
	if id := strings.TrimSpace(ref.ID); id != "" {
		return "id:" + id
	}

	return "name:" + strings.TrimSpace(ref.Name)
}

func workspaceLabelRef(ref catalogWorkspaceRef) string {
	name := strings.TrimSpace(ref.Name)
	if name == "" {
		name = "Account"
	}

	if ref.IsPersonal {
		return name + " (personal)"
	}

	return name
}

func workspaceRoleLabelRef(ref catalogWorkspaceRef) string {
	role := strings.TrimSpace(ref.CurrentUserRole)
	if role == "" {
		return ""
	}

	return "· " + role
}

func workspaceIDForEntry(entry catalogEntry) string {
	if entry.Workspace.IsPersonal {
		return ""
	}

	if id := strings.TrimSpace(entry.Site.WorkspaceID); id != "" {
		return id
	}

	return strings.TrimSpace(entry.Workspace.ID)
}

func workspaceNameForEntry(entry catalogEntry) string {
	if name := strings.TrimSpace(entry.Site.WorkspaceName); name != "" {
		return name
	}

	return strings.TrimSpace(entry.Workspace.Name)
}

func setProjectWorkspaceFromEntries(cfg *config.ProjectConfig, entries []catalogEntry) {
	if len(entries) == 0 {
		return
	}

	firstID := workspaceIDForEntry(entries[0])
	for _, entry := range entries[1:] {
		if workspaceIDForEntry(entry) != firstID {
			cfg.WorkspaceID = ""
			return
		}
	}

	cfg.WorkspaceID = firstID
}

func findCatalogEntry(entries []catalogEntry, index int) (catalogEntry, bool) {
	for _, entry := range entries {
		if entry.Index == index {
			return entry, true
		}
	}

	return catalogEntry{}, false
}
