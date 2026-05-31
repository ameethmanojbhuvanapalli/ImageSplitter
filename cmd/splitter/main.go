package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"imagesplitter/internal/cli"
	"imagesplitter/internal/config"
	"imagesplitter/internal/filesystem"
	"imagesplitter/internal/gui"
	"imagesplitter/internal/runner"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	appDir, err := filesystem.AppDir()
	if err != nil {
		writeCrashFile(".", err)
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	opts, err := cli.ParseArgs(args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	if opts.Run {
		return runHeadless(appDir, opts.OpenReport)
	}

	cfg := config.Load(appDir)
	gui.Run(appDir, cfg)
	return 0
}

func runHeadless(appDir string, openReport bool) int {
	cfg, err := config.LoadRequired(appDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if err := runner.ValidateConfig(cfg); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	result, latestReport, _, err := runner.Execute(appDir, cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	exitCode := 0
	if runner.HasFolderErrors(result) {
		exitCode = 2
	}

	if openReport {
		if _, err := os.Stat(latestReport); err == nil {
			if err := runner.OpenFile(latestReport); err != nil {
				fmt.Fprintf(os.Stderr, "could not open report: %v\n", err)
			}
		}
	}

	return exitCode
}

func writeCrashFile(appDir string, err error) {
	msg := fmt.Sprintf(
		"Image Splitter encountered a fatal error at %s:\n\n%v\n\nPlease share this file with your developer.\n",
		time.Now().Format("2006-01-02 15:04:05"), err,
	)
	_ = os.WriteFile(filepath.Join(appDir, "crash-error.txt"), []byte(msg), 0644)
}
