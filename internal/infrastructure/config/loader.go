package config

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/grpmsoft/gosh/internal/domain/config"
)

// Loader - configuration loader
type Loader struct {
	configPath string
}

// NewLoader creates a new configuration loader
func NewLoader() *Loader {
	home, _ := os.UserHomeDir()
	configPath := filepath.Join(home, ".goshrc")

	return &Loader{
		configPath: configPath,
	}
}

// Load loads configuration from file, or returns default
func (l *Loader) Load() (*config.Config, error) {
	// Check if file exists
	if _, err := os.Stat(l.configPath); os.IsNotExist(err) {
		// File doesn't exist - return default configuration
		return config.DefaultConfig(), nil
	}

	// Read file
	data, err := os.ReadFile(l.configPath)
	if err != nil {
		// Read error - return default with error for logging
		return config.DefaultConfig(), err
	}

	// Parse JSON
	cfg := config.DefaultConfig()
	if err := json.Unmarshal(data, cfg); err != nil {
		// Parse error - return default with error for logging
		return config.DefaultConfig(), err
	}

	return cfg, nil
}

// Save saves configuration to file
func (l *Loader) Save(cfg *config.Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(l.configPath, data, 0644)
}

// CreateDefault creates a file with default configuration
func (l *Loader) CreateDefault() error {
	cfg := config.DefaultConfig()
	return l.Save(cfg)
}
