package config

import (
	"sort"
	"strings"
)

// SitesForBranch returns site aliases whose deploy_branch matches the given branch.
func SitesForBranch(cfg *ProjectConfig, branch string) []string {
	branch = strings.TrimSpace(branch)
	if cfg == nil || branch == "" {
		return nil
	}

	var aliases []string
	for alias, site := range cfg.Sites {
		if strings.EqualFold(strings.TrimSpace(site.DeployBranch), branch) {
			aliases = append(aliases, alias)
		}
	}

	sort.Strings(aliases)
	return aliases
}

// OnlySiteAlias returns the alias when exactly one site is configured.
func OnlySiteAlias(cfg *ProjectConfig) (string, bool) {
	if cfg == nil || len(cfg.Sites) != 1 {
		return "", false
	}

	for alias := range cfg.Sites {
		return alias, true
	}

	return "", false
}

// SortedSiteAliases returns configured site aliases in stable order.
func SortedSiteAliases(cfg *ProjectConfig) []string {
	if cfg == nil {
		return nil
	}

	aliases := make([]string, 0, len(cfg.Sites))
	for alias := range cfg.Sites {
		aliases = append(aliases, alias)
	}

	sort.Strings(aliases)
	return aliases
}

// SitesMissingDeployBranch returns aliases that have no deploy_branch configured.
func SitesMissingDeployBranch(cfg *ProjectConfig) []string {
	if cfg == nil {
		return nil
	}

	var missing []string
	for _, alias := range SortedSiteAliases(cfg) {
		if strings.TrimSpace(cfg.Sites[alias].DeployBranch) == "" {
			missing = append(missing, alias)
		}
	}

	return missing
}

// SiteAliasForBranch returns the first site alias mapped to the given branch.
func SiteAliasForBranch(cfg *ProjectConfig, branch string) string {
	branch = strings.TrimSpace(branch)
	if cfg == nil || branch == "" {
		return ""
	}

	for _, alias := range SortedSiteAliases(cfg) {
		if strings.EqualFold(strings.TrimSpace(cfg.Sites[alias].DeployBranch), branch) {
			return alias
		}
	}

	return ""
}

// PruneStaleSites removes configured sites whose IDs are not present on Pachim.
// Returns removed aliases and whether the config was changed (including default fixes).
func PruneStaleSites(cfg *ProjectConfig, validSiteIDs map[string]struct{}) (removed []string, changed bool) {
	if cfg == nil || len(cfg.Sites) == 0 {
		return nil, false
	}

	for alias, site := range cfg.Sites {
		if _, ok := validSiteIDs[site.ID]; ok {
			continue
		}

		delete(cfg.Sites, alias)
		removed = append(removed, alias)
		changed = true
	}

	if len(removed) == 0 {
		return nil, false
	}

	sort.Strings(removed)

	if cfg.Default != "" {
		if _, ok := cfg.Sites[cfg.Default]; !ok {
			aliases := SortedSiteAliases(cfg)
			if len(aliases) > 0 {
				cfg.Default = aliases[0]
			} else {
				cfg.Default = ""
			}
			changed = true
		}
	}

	return removed, changed
}

// IsProductionBranch reports common production branch names.
func IsProductionBranch(branch string) bool {
	switch strings.ToLower(strings.TrimSpace(branch)) {
	case "main", "master", "production", "prod":
		return true
	default:
		return false
	}
}
