package runner

import (
	"testing"

	"imagesplitter/internal/config"
	"imagesplitter/internal/models"
)

func TestValidateConfig(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		cfg := &config.Config{
			RootFolder:      t.TempDir(),
			TargetBaseNames: []string{"front"},
			LeftSuffix:      "_left",
			RightSuffix:     "_right",
		}
		if err := ValidateConfig(cfg); err != nil {
			t.Fatalf("ValidateConfig() error = %v", err)
		}
	})

	t.Run("missing folder", func(t *testing.T) {
		cfg := &config.Config{
			RootFolder:      "",
			TargetBaseNames: []string{"front"},
			LeftSuffix:      "_left",
			RightSuffix:     "_right",
		}
		if err := ValidateConfig(cfg); err == nil {
			t.Fatalf("expected validation error")
		}
	})

	t.Run("empty target names", func(t *testing.T) {
		cfg := &config.Config{
			RootFolder:      t.TempDir(),
			TargetBaseNames: []string{"   "},
			LeftSuffix:      "_left",
			RightSuffix:     "_right",
		}
		if err := ValidateConfig(cfg); err == nil {
			t.Fatalf("expected validation error")
		}
	})
}

func TestHasFolderErrors(t *testing.T) {
	result := &models.RunResult{
		FolderResults: []*models.FolderResult{
			{
				ImageResults: []*models.ImageResult{
					{Status: models.StatusProcessed},
				},
			},
			{
				ImageResults: []*models.ImageResult{
					{Status: models.StatusError},
				},
			},
		},
	}
	if !HasFolderErrors(result) {
		t.Fatalf("expected HasFolderErrors=true")
	}
}
