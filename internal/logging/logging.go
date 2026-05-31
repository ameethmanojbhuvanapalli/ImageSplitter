package logging

import (
	"fmt"
	"io"
	"os"
	"time"

	"imagesplitter/internal/models"
)

// Logger writes structured log lines to one or more writers simultaneously.
type Logger struct {
	writers   []io.Writer
	debugMode bool
}

// NewFileLogger opens (or creates) a log file and returns a Logger writing to it.
func NewFileLogger(path string, debugMode bool) (*Logger, *os.File, error) {
	f, err := os.Create(path)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot create log file %s: %w", path, err)
	}
	return &Logger{writers: []io.Writer{f}, debugMode: debugMode}, f, nil
}

func (l *Logger) write(level, message string) {
	line := fmt.Sprintf("[%s] [%s] %s\n",
		time.Now().Format("2006-01-02 15:04:05"), level, message)
	for _, w := range l.writers {
		_, _ = io.WriteString(w, line)
	}
}

func (l *Logger) Info(msg string)  { l.write("INFO", msg) }
func (l *Logger) Warn(msg string)  { l.write("WARN", msg) }
func (l *Logger) Error(msg string) { l.write("ERROR", msg) }

// Debug writes only when debug mode is enabled.
func (l *Logger) Debug(msg string) {
	if l.debugMode {
		l.write("DEBUG", msg)
	}
}

// AddWriter adds an additional destination (e.g. a second log file).
func (l *Logger) AddWriter(w io.Writer) {
	l.writers = append(l.writers, w)
}

// LogRunResult writes a complete structured run summary derived from RunResult.
func (l *Logger) LogRunResult(result *models.RunResult) {
	l.Info(fmt.Sprintf("Run started  RunNumber=%d", result.RunNumber))

	for _, fr := range result.FolderResults {
		l.Debug(fmt.Sprintf("Folder=%q entered", fr.FolderPath))

		for _, ir := range fr.ImageResults {
			switch ir.Status {
			case models.StatusProcessed:
				l.Info(fmt.Sprintf("Folder=%q File=%q Status=Processed Message=%q",
					fr.FolderName, ir.FileName, ir.Message))
			case models.StatusAlreadyProcessed:
				l.Warn(fmt.Sprintf("Folder=%q File=%q Status=AlreadyProcessed Reason=%q",
					fr.FolderName, ir.FileName, ir.Message))
			case models.StatusTargetImageMissing:
				l.Warn(fmt.Sprintf("Folder=%q File=%q Status=TargetImageMissing Reason=%q",
					fr.FolderName, ir.FileName, ir.Message))
			case models.StatusError:
				l.Error(fmt.Sprintf("Folder=%q File=%q Status=Error Reason=%q",
					fr.FolderName, ir.FileName, ir.Message))
			}
		}

		l.Debug(fmt.Sprintf("Folder=%q done Duration=%s",
			fr.FolderName, fr.EndTime.Sub(fr.StartTime).Round(time.Millisecond)))
	}

	processed, alreadyProcessed, missing, errors := result.Counts()
	l.Info(fmt.Sprintf(
		"Run completed RunNumber=%d Duration=%s Processed=%d AlreadyProcessed=%d Missing=%d Errors=%d",
		result.RunNumber,
		result.Duration().Round(time.Millisecond),
		processed, alreadyProcessed, missing, errors,
	))
}
