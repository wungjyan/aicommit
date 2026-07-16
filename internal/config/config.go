package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const configDir = ".aicommit"
const configFile = "config.json"

// Config holds the AI provider configuration.
//
// Backend selects which AI backend generates messages. It may be empty in
// configs written before multi-backend support; Resolve interprets an empty
// Backend as the OpenAI backend so existing API configs keep working.
type Config struct {
	Backend  string `json:"backend,omitempty"`
	APIKey   string `json:"api_key"`
	BaseURL  string `json:"base_url,omitempty"`
	Model    string `json:"model,omitempty"`
	Language string `json:"language,omitempty"`
}

// configPath returns ~/.aicommit/config.json.
func configPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot find home directory: %w", err)
	}
	return filepath.Join(home, configDir, configFile), nil
}

// ConfigPath returns the absolute path of the config file. It is the stable
// public accessor used by `aicommit config path`.
func ConfigPath() (string, error) {
	return configPath()
}

// LoadConfig reads the config file. Returns a zero Config (not an error) if the file does not exist.
func LoadConfig() (Config, error) {
	path, err := configPath()
	if err != nil {
		return Config{}, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Config{}, nil
		}
		return Config{}, fmt.Errorf("failed to read config: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("failed to parse config: %w", err)
	}
	return cfg, nil
}

// SaveConfig writes the config file atomically, creating the directory if
// needed. It writes to a temp file in the same directory and renames it into
// place, so a failed or interrupted write never corrupts the existing config.
func SaveConfig(cfg Config) error {
	path, err := configPath()
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	tmp, err := os.CreateTemp(dir, ".config-*.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temp config: %w", err)
	}
	tmpName := tmp.Name()
	// Best-effort cleanup if we fail before the rename succeeds.
	defer os.Remove(tmpName)

	if err := tmp.Chmod(0600); err != nil {
		tmp.Close()
		return fmt.Errorf("failed to set config permissions: %w", err)
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("failed to write config: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}
	return nil
}
