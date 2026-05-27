package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

type LocalConfig struct {
	APIURL string `json:"api_url,omitempty"`
}

func LocalConfigPath() (string, error) {
	dir, err := PachimDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(dir, "pachim.json"), nil
}

func LoadLocalConfig() (*LocalConfig, error) {
	path, err := LocalConfigPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &LocalConfig{}, nil
		}

		return nil, err
	}

	var cfg LocalConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func SaveLocalAPIURL(apiURL string) error {
	path, err := LocalConfigPath()
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	cfg := LocalConfig{APIURL: apiURL}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0600)
}

func ClearLocalAPIURL() error {
	path, err := LocalConfigPath()
	if err != nil {
		return err
	}

	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}

	return nil
}
