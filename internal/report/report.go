package report

import (
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"time"

	"imagesplitter/internal/models"
)

type Metadata struct {
	RunNumber       int    `json:"runNumber"`
	StartedAt       string `json:"startedAt"`
	CompletedAt     string `json:"completedAt"`
	DurationSeconds int64  `json:"durationSeconds"`
}

type templateData struct {
	RunResult        *models.RunResult
	ProcessedCount   int
	AlreadyProcessed int
	MissingCount     int
	ErrorCount       int
	SplitProcessed   int
	SplitErrors      int
	PadProcessed     int
	PadErrors        int
}

func funcMap() template.FuncMap {
	return template.FuncMap{
		"formatTime": func(t time.Time) string {
			return t.Format("2006-01-02 15:04:05")
		},
		"formatDuration": func(d time.Duration) string {
			d = d.Round(time.Millisecond)
			if d < time.Second {
				return fmt.Sprintf("%dms", d.Milliseconds())
			}
			if d < time.Minute {
				return fmt.Sprintf("%.2fs", d.Seconds())
			}
			return fmt.Sprintf("%dm %ds", int(d.Minutes()), int(d.Seconds())%60)
		},
		"overallClass": func(fr *models.FolderResult) string {
			return statusClass(fr.OverallStatus())
		},
		"overallLabel": func(fr *models.FolderResult) string {
			return string(fr.OverallStatus())
		},
		"statusClass": func(s models.Status) string {
			return statusClass(s)
		},
		"statusLabel": func(s models.Status) string { return string(s) },
		"runPad":      func(n int) string { return fmt.Sprintf("%03d", n) },
		"add":         func(a, b int) int { return a + b },
		"opClass": func(op models.Operation) string {
			if op == models.OperationSplitting {
				return "op-split"
			}
			return "op-pad"
		},
		"opLabel": func(op models.Operation) string {
			if op == models.OperationSplitting {
				return "Split"
			}
			return "Pad"
		},
	}
}

func statusClass(s models.Status) string {
	switch s {
	case models.StatusProcessed:
		return "status-processed"
	case models.StatusAlreadyProcessed:
		return "status-already"
	case models.StatusTargetImageMissing:
		return "status-missing"
	case models.StatusError:
		return "status-error"
	}
	return ""
}

func WriteMetadata(runDir string, result *models.RunResult) error {
	m := Metadata{
		RunNumber:       result.RunNumber,
		StartedAt:       result.StartTime.Format(time.RFC3339),
		CompletedAt:     result.EndTime.Format(time.RFC3339),
		DurationSeconds: int64(result.Duration().Seconds()),
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(runDir+"/metadata.json", data, 0644)
}

func WriteReport(path string, result *models.RunResult) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("cannot create report: %w", err)
	}
	defer f.Close()

	tmpl, err := template.New("report").Funcs(funcMap()).Parse(htmlTemplate)
	if err != nil {
		return fmt.Errorf("report template error: %w", err)
	}

	processed, alreadyProcessed, missing, errors := result.Counts()
	splitProcessed, splitErrors, padProcessed, padErrors := result.CountsByOperation()

	return tmpl.Execute(f, templateData{
		RunResult:        result,
		ProcessedCount:   processed,
		AlreadyProcessed: alreadyProcessed,
		MissingCount:     missing,
		ErrorCount:       errors,
		SplitProcessed:   splitProcessed,
		SplitErrors:      splitErrors,
		PadProcessed:     padProcessed,
		PadErrors:        padErrors,
	})
}
