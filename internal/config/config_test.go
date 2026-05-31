package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadRequired(t *testing.T) {
	t.Run("missing file returns error", func(t *testing.T) {
		_, err := LoadRequired(t.TempDir())
		if err == nil {
			t.Fatalf("expected error for missing config.json")
		}
	})

	t.Run("reads valid config", func(t *testing.T) {
		dir := t.TempDir()
		configPath := filepath.Join(dir, "config.json")
		if err := os.WriteFile(configPath, []byte(`{"rootFolder":"C:/images","targetBaseNames":["front"]}`), 0644); err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}

		cfg, err := LoadRequired(dir)
		if err != nil {
			t.Fatalf("LoadRequired() error = %v", err)
		}
		if cfg.RootFolder != "C:/images" {
			t.Fatalf("unexpected RootFolder: %q", cfg.RootFolder)
		}
	})
}
