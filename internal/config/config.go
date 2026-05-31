package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const Version = "1.1.0"

// Config holds all user-configurable settings, persisted to config.json.
type Config struct {
	RootFolder        string   `json:"rootFolder"`
	ScanDepth         int      `json:"scanDepth"`
	TargetBaseNames   []string `json:"targetBaseNames"` // e.g. ["front","back"] — no extension
	LeftSuffix        string   `json:"leftSuffix"`
	RightSuffix       string   `json:"rightSuffix"`
	DeleteOriginal    bool     `json:"deleteOriginal"`
	OverwriteExisting bool     `json:"overwriteExisting"`
	DebugMode         bool     `json:"debugMode"`
}

// SupportedExtensions lists every image format the processor can handle.
var SupportedExtensions = []string{".jpg", ".jpeg", ".png", ".bmp", ".tiff", ".tif", ".webp"}

// Defaults returns a Config pre-filled with sensible defaults for first launch.
func Defaults() *Config {
	return &Config{
		RootFolder:        "",
		ScanDepth:         -1,
		TargetBaseNames:   []string{"front"},
		LeftSuffix:        "_left",
		RightSuffix:       "_right",
		DeleteOriginal:    false,
		OverwriteExisting: false,
		DebugMode:         false,
	}
}

// Load reads config.json from appDir.
// If the file does not exist or is unreadable, returns defaults silently.
func Load(appDir string) *Config {
	cfg := Defaults()
	data, err := os.ReadFile(filepath.Join(appDir, "config.json"))
	if err != nil {
		return cfg
	}
	if err := json.Unmarshal(data, cfg); err != nil {
		return Defaults()
	}
	// Ensure we never return an empty slice — always at least one entry.
	if len(cfg.TargetBaseNames) == 0 {
		cfg.TargetBaseNames = []string{"front"}
	}
	return cfg
}

// LoadRequired reads config.json from appDir and returns an error if the file is
// missing or invalid.
func LoadRequired(appDir string) (*Config, error) {
	path := filepath.Join(appDir, "config.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read config.json: %w", err)
	}
	cfg := Defaults()
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("invalid config.json: %w", err)
	}
	return cfg, nil
}

// Save writes the current config to config.json in appDir.
func Save(appDir string, cfg *Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(appDir, "config.json"), data, 0644)
}
