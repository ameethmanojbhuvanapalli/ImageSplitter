package config

import (
	"fmt"
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
		rootFolder := filepath.Join(dir, "images")
		if err := os.Mkdir(rootFolder, 0755); err != nil {
			t.Fatalf("Mkdir() error = %v", err)
		}
		configPath := filepath.Join(dir, "config.json")
		payload := fmt.Sprintf(`{"rootFolder":%q,"targetBaseNames":["front"]}`, rootFolder)
		if err := os.WriteFile(configPath, []byte(payload), 0644); err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}

		cfg, err := LoadRequired(dir)
		if err != nil {
			t.Fatalf("LoadRequired() error = %v", err)
		}
		if cfg.RootFolder != rootFolder {
			t.Fatalf("unexpected RootFolder: %q", cfg.RootFolder)
		}
	})
}
