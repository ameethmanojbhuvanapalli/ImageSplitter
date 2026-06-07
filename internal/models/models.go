package models

import "time"

// Status is the outcome of processing a single image file.
type Status string

const (
	StatusProcessed          Status = "Processed"
	StatusAlreadyProcessed   Status = "Already Processed"
	StatusTargetImageMissing Status = "Target Image Missing"
	StatusError              Status = "Error"
)

// Operation identifies which process produced this result.
type Operation string

const (
	OperationSplitting Operation = "Splitting"
	OperationPadding   Operation = "Padding"
)

// ImageResult records the outcome for one image file.
type ImageResult struct {
	FileName  string
	Operation Operation
	Status    Status
	Message   string
}

// FolderResult records the outcome for a single folder across all operations.
type FolderResult struct {
	FolderName   string
	FolderPath   string
	ImageResults []*ImageResult
	StartTime    time.Time
	EndTime      time.Time
}

// OverallStatus returns the worst status across all image results.
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
			if worst != StatusError {
				worst = StatusTargetImageMissing
			}
		case StatusAlreadyProcessed:
			if worst == StatusProcessed {
				worst = StatusAlreadyProcessed
			}
		}
	}
	return worst
}

// RunResult is the authoritative data structure for a complete execution.
// All reports and logs are generated from this — never from parsing each other.
type RunResult struct {
	RunNumber     int
	StartTime     time.Time
	EndTime       time.Time
	FolderResults []*FolderResult
}

func (r *RunResult) Duration() time.Duration { return r.EndTime.Sub(r.StartTime) }

// Counts returns summary counts broken down by status across all image results.
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

// CountsByOperation returns processed counts split by operation for the summary.
func (r *RunResult) CountsByOperation() (splitProcessed, splitErrors, padProcessed, padErrors int) {
	for _, fr := range r.FolderResults {
		for _, ir := range fr.ImageResults {
			switch ir.Operation {
			case OperationSplitting:
				if ir.Status == StatusProcessed {
					splitProcessed++
				} else if ir.Status == StatusError {
					splitErrors++
				}
			case OperationPadding:
				if ir.Status == StatusProcessed {
					padProcessed++
				} else if ir.Status == StatusError {
					padErrors++
				}
			}
		}
	}
	return
}

func (r *RunResult) TotalFolders() int { return len(r.FolderResults) }
