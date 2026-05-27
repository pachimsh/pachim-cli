package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Credentials struct {
	Token  string `json:"token"`
	Email  string `json:"email"`
	Name   string `json:"name"`
	APIUrl string `json:"api_url,omitempty"`
}

type SiteConfig struct {
	ID     string `json:"site_id"`
	Domain string `json:"domain"`
}

type ProjectConfig struct {
	Default string                `json:"default"`
	Sites   map[string]SiteConfig `json:"sites"`
}

func PachimDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(home, ".pachim"), nil
}

func ProfilePath(profile string) (string, error) {
	dir, err := PachimDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(dir, "profiles", profile+".json"), nil
}

func ProjectConfigPath() string {
	return ".pachim.json"
}

func LoadCredentials(profile string) (*Credentials, error) {
	path, err := ProfilePath(profile)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var creds Credentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, err
	}

	return &creds, nil
}

func SaveCredentials(profile string, creds *Credentials) error {
	path, err := ProfilePath(profile)
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(creds, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0600)
}

func DeleteCredentials(profile string) error {
	path, err := ProfilePath(profile)
	if err != nil {
		return err
	}

	return os.Remove(path)
}

func ListProfiles() ([]string, error) {
	dir, err := PachimDir()
	if err != nil {
		return nil, err
	}

	profilesDir := filepath.Join(dir, "profiles")
	entries, err := os.ReadDir(profilesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var profiles []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if ext := filepath.Ext(name); ext == ".json" {
			profiles = append(profiles, name[:len(name)-len(ext)])
		}
	}

	return profiles, nil
}

func LoadProjectConfig() (*ProjectConfig, error) {
	data, err := os.ReadFile(ProjectConfigPath())
	if err != nil {
		return nil, err
	}

	var cfg ProjectConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func SaveProjectConfig(cfg *ProjectConfig) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(ProjectConfigPath(), data, 0644)
}
