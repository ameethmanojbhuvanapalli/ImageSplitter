package models

import "time"

// Status is the outcome of processing a single image file within a folder.
type Status string

const (
	StatusProcessed          Status = "Processed"
	StatusAlreadyProcessed   Status = "Already Processed"
	StatusTargetImageMissing Status = "Target Image Missing"
	StatusError              Status = "Error"
)

// ImageResult records the outcome for one image file inside a folder.
type ImageResult struct {
	FileName string
	Status   Status
	Message  string
}

// FolderResult records the outcome for a single folder.
// A folder may produce multiple ImageResults (one per target base name × format found).
type FolderResult struct {
	FolderName   string
	FolderPath   string
	ImageResults []*ImageResult
	StartTime    time.Time
	EndTime      time.Time
}

// OverallStatus returns the worst status across all image results in the folder.
// Error > Missing > AlreadyProcessed > Processed.
func (f *FolderResult) OverallStatus() Status {
	if len(f.ImageResults) == 0 {
		return StatusTargetImageMissing
	}
	worst := StatusProcessed
	for _, ir := range f.ImageResults {
		switch ir.Status {
		case StatusError:
			return StatusError
		case StatusTargetImageMissing:
			worst = StatusTargetImageMissing
		case StatusAlreadyProcessed:
			if worst == StatusProcessed {
				worst = StatusAlreadyProcessed
			}
		}
	}
	return worst
}

// RunResult is the authoritative data structure for a complete execution.
type RunResult struct {
	RunNumber     int
	StartTime     time.Time
	EndTime       time.Time
	FolderResults []*FolderResult
}

func (r *RunResult) Duration() time.Duration { return r.EndTime.Sub(r.StartTime) }

// Counts returns summary counts at the image-result level.
func (r *RunResult) Counts() (processed, alreadyProcessed, missingTarget, errors int) {
	for _, fr := range r.FolderResults {
		for _, ir := range fr.ImageResults {
			switch ir.Status {
			case StatusProcessed:
				processed++
			case StatusAlreadyProcessed:
				alreadyProcessed++
			case StatusTargetImageMissing:
				missingTarget++
			case StatusError:
				errors++
			}
		}
	}
	return
}

func (r *RunResult) TotalFolders() int { return len(r.FolderResults) }
