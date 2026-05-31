package filesystem

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// WalkError records a non-fatal error encountered during folder discovery.
type WalkError struct {
	Path string
	Err  error
}

// AppDir returns the directory containing the running executable.
func AppDir() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("cannot determine executable path: %w", err)
	}
	return filepath.Dir(exe), nil
}

// RunCounter reads, increments, and persists the run counter stored in
// History/run-counter.txt. Returns the new (incremented) run number.
func RunCounter(appDir string) (int, error) {
	historyDir := filepath.Join(appDir, "History")
	if err := os.MkdirAll(historyDir, 0755); err != nil {
		return 0, fmt.Errorf("cannot create History directory: %w", err)
	}

	counterPath := filepath.Join(historyDir, "run-counter.txt")

	current := 0
	if data, err := os.ReadFile(counterPath); err == nil {
		if n, err := strconv.Atoi(strings.TrimSpace(string(data))); err == nil {
			current = n
		}
	}

	next := current + 1
	if err := os.WriteFile(counterPath, []byte(strconv.Itoa(next)), 0644); err != nil {
		return 0, fmt.Errorf("cannot write run counter: %w", err)
	}
	return next, nil
}

// RunDir creates and returns the path for a numbered run directory.
func RunDir(appDir string, runNumber int) (string, error) {
	dir := filepath.Join(appDir, "History", fmt.Sprintf("Run %03d", runNumber))
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("cannot create run directory %s: %w", dir, err)
	}
	return dir, nil
}

// DiscoverFolders traverses root up to maxDepth levels and returns all
// directory paths. maxDepth -1 means unlimited. The root itself is always
// included. Unreadable subdirectories are recorded in walkErrs and skipped —
// they never abort the traversal.
func DiscoverFolders(root string, maxDepth int) (folders []string, walkErrs []WalkError) {
	root = filepath.Clean(root)
	walk(root, 0, maxDepth, &folders, &walkErrs)
	return
}

func walk(current string, depth, maxDepth int, folders *[]string, walkErrs *[]WalkError) {
	*folders = append(*folders, current)

	if maxDepth >= 0 && depth >= maxDepth {
		return
	}

	entries, err := os.ReadDir(current)
	if err != nil {
		*walkErrs = append(*walkErrs, WalkError{Path: current, Err: err})
		return
	}

	for _, e := range entries {
		if e.IsDir() {
			walk(filepath.Join(current, e.Name()), depth+1, maxDepth, folders, walkErrs)
		}
	}
}

// OutputPaths returns the expected left and right output file paths.
func OutputPaths(sourceImagePath, leftSuffix, rightSuffix string) (left, right string) {
	ext := filepath.Ext(sourceImagePath)
	base := strings.TrimSuffix(sourceImagePath, ext)
	return base + leftSuffix + ext, base + rightSuffix + ext
}

// ExistsAny returns true if any of the provided paths exist on disk.
func ExistsAny(paths ...string) bool {
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return true
		}
	}
	return false
}
