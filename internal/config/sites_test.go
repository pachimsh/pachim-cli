package config

import "testing"

func TestSitesForBranch(t *testing.T) {
	cfg := &ProjectConfig{
		Sites: map[string]SiteConfig{
			"staging":    {DeployBranch: "develop"},
			"production": {DeployBranch: "main"},
			"legacy":     {DeployBranch: "main"},
		},
	}

	matches := SitesForBranch(cfg, "develop")
	if len(matches) != 1 || matches[0] != "staging" {
		t.Fatalf("expected [staging], got %#v", matches)
	}

	matches = SitesForBranch(cfg, "main")
	if len(matches) != 2 {
		t.Fatalf("expected 2 matches for main, got %#v", matches)
	}

	if len(SitesForBranch(cfg, "feature")) != 0 {
		t.Fatal("expected no matches for unknown branch")
	}
}

func TestSitesMissingDeployBranch(t *testing.T) {
	cfg := &ProjectConfig{
		Sites: map[string]SiteConfig{
			"a": {DeployBranch: "main"},
			"b": {},
			"c": {DeployBranch: "develop"},
		},
	}

	missing := SitesMissingDeployBranch(cfg)
	if len(missing) != 1 || missing[0] != "b" {
		t.Fatalf("expected [b], got %#v", missing)
	}

	if SiteAliasForBranch(cfg, "main") != "a" {
		t.Fatal("expected site a for main branch")
	}
}

func TestPruneStaleSites(t *testing.T) {
	cfg := &ProjectConfig{
		Default: "gone",
		Sites: map[string]SiteConfig{
			"gone": {ID: "id-1", Domain: "gone.example.com"},
			"live": {ID: "id-2", Domain: "live.example.com"},
		},
	}

	valid := map[string]struct{}{
		"id-2": {},
	}

	removed, changed := PruneStaleSites(cfg, valid)
	if !changed || len(removed) != 1 || removed[0] != "gone" {
		t.Fatalf("unexpected prune result: changed=%v removed=%#v", changed, removed)
	}

	if cfg.Default != "live" {
		t.Fatalf("expected default live, got %q", cfg.Default)
	}

	if len(cfg.Sites) != 1 {
		t.Fatal("expected one site left")
	}
}

func TestOnlySiteAlias(t *testing.T) {
	cfg := &ProjectConfig{
		Sites: map[string]SiteConfig{
			"only": {Domain: "api.example.com"},
		},
	}

	alias, ok := OnlySiteAlias(cfg)
	if !ok || alias != "only" {
		t.Fatalf("expected only site alias, got %q ok=%v", alias, ok)
	}

	cfg.Sites["second"] = SiteConfig{}
	if _, ok := OnlySiteAlias(cfg); ok {
		t.Fatal("expected false when multiple sites configured")
	}
}
