package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const Version = "1.2.0"

// SupportedExtensions lists every image format the processors can handle.
// Format in = format out. Zero conversions.
var SupportedExtensions = []string{".jpg", ".jpeg", ".png", ".bmp", ".tiff", ".tif", ".webp"}

// SplittingConfig holds all settings for the vertical split operation.
type SplittingConfig struct {
	Enabled           bool     `json:"enabled"`
	TargetBaseNames   []string `json:"targetBaseNames"`
	LeftSuffix        string   `json:"leftSuffix"`
	RightSuffix       string   `json:"rightSuffix"`
	DeleteOriginal    bool     `json:"deleteOriginal"`
	OverwriteExisting bool     `json:"overwriteExisting"`
}

// PaddingConfig holds all settings for the white-space padding operation.
type PaddingConfig struct {
	Enabled           bool     `json:"enabled"`
	LeftPadNames      []string `json:"leftPadNames"`      // add white space on LEFT  (image sits on right)
	RightPadNames     []string `json:"rightPadNames"`     // add white space on RIGHT (image sits on left)
	CreateNewFile     bool     `json:"createNewFile"`     // false = overwrite original
	LeftSuffix        string   `json:"leftSuffix"`        // only used when CreateNewFile=true
	RightSuffix       string   `json:"rightSuffix"`       // only used when CreateNewFile=true
	OverwriteExisting bool     `json:"overwriteExisting"` // only used when CreateNewFile=true
	PadColor          string   `json:"padColor"`          // hex e.g. "#FFFFFF"
}

// Config is the root configuration structure persisted to config.json.
type Config struct {
	RootFolder string          `json:"rootFolder"`
	ScanDepth  int             `json:"scanDepth"`
	DebugMode  bool            `json:"debugMode"`
	Splitting  SplittingConfig `json:"splitting"`
	Padding    PaddingConfig   `json:"padding"`
}

// Defaults returns a Config pre-filled with sensible defaults for first launch.
func Defaults() *Config {
	return &Config{
		RootFolder: "",
		ScanDepth:  -1,
		DebugMode:  false,
		Splitting: SplittingConfig{
			Enabled:           true,
			TargetBaseNames:   []string{"front"},
			LeftSuffix:        "_left",
			RightSuffix:       "_right",
			DeleteOriginal:    false,
			OverwriteExisting: false,
		},
		Padding: PaddingConfig{
			Enabled:           false,
			LeftPadNames:      []string{},
			RightPadNames:     []string{},
			CreateNewFile:     false,
			LeftSuffix:        "_padded",
			RightSuffix:       "_padded",
			OverwriteExisting: false,
			PadColor:          "#FFFFFF",
		},
	}
}

// Load reads config.json from appDir.
// Returns defaults silently if the file is missing or unreadable.
func Load(appDir string) *Config {
	cfg := Defaults()
	data, err := os.ReadFile(filepath.Join(appDir, "config.json"))
	if err != nil {
		return cfg
	}
	if err := json.Unmarshal(data, cfg); err != nil {
		return Defaults()
	}
	// Safety: ensure slices are never nil.
	if len(cfg.Splitting.TargetBaseNames) == 0 {
		cfg.Splitting.TargetBaseNames = []string{"front"}
	}
	if cfg.Padding.LeftPadNames == nil {
		cfg.Padding.LeftPadNames = []string{}
	}
	if cfg.Padding.RightPadNames == nil {
		cfg.Padding.RightPadNames = []string{}
	}
	if cfg.Padding.PadColor == "" {
		cfg.Padding.PadColor = "#FFFFFF"
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

// Save writes config to config.json in appDir.
func Save(appDir string, cfg *Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(appDir, "config.json"), data, 0644)
}
