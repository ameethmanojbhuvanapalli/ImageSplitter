package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"imagesplitter/internal/config"
	"imagesplitter/internal/filesystem"
	"imagesplitter/internal/gui"
)

func main() {
	appDir, err := filesystem.AppDir()
	if err != nil {
		writeCrashFile(".", err)
		os.Exit(1)
	}

	cfg := config.Load(appDir)
	gui.Run(appDir, cfg)
}

func writeCrashFile(appDir string, err error) {
	msg := fmt.Sprintf(
		"Image Splitter encountered a fatal error at %s:\n\n%v\n\nPlease share this file with your developer.\n",
		time.Now().Format("2006-01-02 15:04:05"), err,
	)
	_ = os.WriteFile(filepath.Join(appDir, "crash-error.txt"), []byte(msg), 0644)
}
