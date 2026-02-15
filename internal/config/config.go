// Package config loads and validates the tool's own configuration.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Config holds the tool's own configuration.
type Config struct {
	ConfigDir string `json:"configDir"`
}

// DefaultPath returns the default config file location, or "" if the home
// directory cannot be determined.
func DefaultPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".claude-config-merge.json")
}

// Load reads and validates the config at path.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config %s: %w", path, err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config %s: %w", path, err)
	}

	if cfg.ConfigDir == "" {
		return nil, fmt.Errorf("configDir is required in %s", path)
	}

	if _, err := os.Stat(cfg.ConfigDir); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("configDir %q does not exist (check %s)", cfg.ConfigDir, path)
		}
		return nil, fmt.Errorf("checking configDir %q: %w", cfg.ConfigDir, err)
	}

	return &cfg, nil
}
