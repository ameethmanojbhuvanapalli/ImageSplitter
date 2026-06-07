package runner

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"imagesplitter/internal/config"
	"imagesplitter/internal/filesystem"
	"imagesplitter/internal/logging"
	"imagesplitter/internal/models"
	"imagesplitter/internal/padder"
	"imagesplitter/internal/processor"
	"imagesplitter/internal/report"
)

// Execute performs one processing run and writes all run artifacts.
func Execute(appDir string, cfg *config.Config) (
	result *models.RunResult,
	latestReport string,
	latestLog string,
	err error,
) {
	runNumber, err := filesystem.RunCounter(appDir)
	if err != nil {
		return nil, "", "", fmt.Errorf("run counter: %w", err)
	}

	runDir, err := filesystem.RunDir(appDir, runNumber)
	if err != nil {
		return nil, "", "", fmt.Errorf("run directory: %w", err)
	}

	runLogPath := filepath.Join(runDir, "execution.log")
	logger, runLogFile, err := logging.NewFileLogger(runLogPath, cfg.DebugMode)
	if err != nil {
		return nil, "", "", fmt.Errorf("log file: %w", err)
	}
	defer runLogFile.Close()

	result = &models.RunResult{
		RunNumber: runNumber,
		StartTime: time.Now(),
	}

	logger.Info(fmt.Sprintf("Run started RunNumber=%d DebugMode=%v", runNumber, cfg.DebugMode))
	logger.Debug(fmt.Sprintf(
		"Config: root=%q depth=%d targets=%v leftSuffix=%q rightSuffix=%q",
		cfg.RootFolder,
		cfg.ScanDepth,
		cfg.Splitting.TargetBaseNames,
		cfg.Splitting.LeftSuffix,
		cfg.Splitting.RightSuffix,
	))

	folders, walkErrs := filesystem.DiscoverFolders(cfg.RootFolder, cfg.ScanDepth)
	for _, we := range walkErrs {
		logger.Warn(fmt.Sprintf("Skipped unreadable directory %q: %v", we.Path, we.Err))
	}
	logger.Info(fmt.Sprintf("Discovered %d folder(s)", len(folders)))

	for _, dir := range folders {
		fr := processor.ProcessFolder(dir, cfg, logger)

		if cfg.Padding.Enabled {
			padder.ProcessFolder(fr, cfg, logger)
		}

		result.FolderResults = append(result.FolderResults, fr)
	}

	result.EndTime = time.Now()
	logger.LogRunResult(result)

	runReportPath := filepath.Join(runDir, "report.html")
	if werr := report.WriteReport(runReportPath, result); werr != nil {
		logger.Error(fmt.Sprintf("Could not write run report: %v", werr))
	}
	if werr := report.WriteMetadata(runDir, result); werr != nil {
		logger.Error(fmt.Sprintf("Could not write metadata.json: %v", werr))
	}

	latestReport = filepath.Join(appDir, "Latest Report.html")
	latestLog = filepath.Join(appDir, "Latest Log.log")

	if werr := copyFile(runReportPath, latestReport); werr != nil {
		logger.Error(fmt.Sprintf("Could not write Latest Report.html: %v", werr))
	}
	if werr := copyFile(runLogPath, latestLog); werr != nil {
		logger.Error(fmt.Sprintf("Could not write Latest Log.log: %v", werr))
	}

	return result, latestReport, latestLog, nil
}

func ValidateConfig(cfg *config.Config) error {
	if cfg.RootFolder == "" {
		return fmt.Errorf("root folder is required")
	}

	info, err := os.Stat(cfg.RootFolder)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("The selected folder does not exist:\n%s", cfg.RootFolder)
		}
		return fmt.Errorf("Cannot access the selected folder:\n%s\n\n%v", cfg.RootFolder, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("The selected path is not a folder:\n%s", cfg.RootFolder)
	}

	nonEmptyNames := 0
	for _, n := range cfg.Splitting.TargetBaseNames {
		if strings.TrimSpace(n) != "" {
			nonEmptyNames++
		}
	}
	if nonEmptyNames == 0 {
		return fmt.Errorf("at least one target image filename is required")
	}

	leftSuffix := strings.TrimSpace(cfg.Splitting.LeftSuffix)
	rightSuffix := strings.TrimSpace(cfg.Splitting.RightSuffix)

	if leftSuffix == "" || rightSuffix == "" {
		return fmt.Errorf("left and right suffixes must not be empty")
	}
	if leftSuffix == rightSuffix {
		return fmt.Errorf("left and right suffixes must be different from each other")
	}

	return nil
}

func HasFolderErrors(result *models.RunResult) bool {
	if result == nil {
		return false
	}
	for _, folder := range result.FolderResults {
		if folder.OverallStatus() == models.StatusError {
			return true
		}
	}
	return false
}

func OpenFile(path string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", path)
	case "darwin":
		cmd = exec.Command("open", path)
	default:
		cmd = exec.Command("xdg-open", path)
	}
	return cmd.Start()
}

func copyFile(src, dst string) (err error) {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := out.Close(); err == nil && cerr != nil {
			err = cerr
		}
	}()

	_, err = io.Copy(out, in)
	return err
}
