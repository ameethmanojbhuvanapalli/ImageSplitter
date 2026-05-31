package gui

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"imagesplitter/internal/config"
	"imagesplitter/internal/filesystem"
	"imagesplitter/internal/icon"
	"imagesplitter/internal/logging"
	"imagesplitter/internal/models"
	"imagesplitter/internal/processor"
	"imagesplitter/internal/report"
)

// Run launches the Fyne application and blocks until the window is closed.
func Run(appDir string, cfg *config.Config) {
	a := app.NewWithID("com.imagesplitter.app")
	a.Settings().SetTheme(theme.LightTheme())
	a.SetIcon(fyne.NewStaticResource("icon.png", icon.AppIconPNG))

	w := a.NewWindow("Image Splitter")
	w.SetIcon(fyne.NewStaticResource("icon.png", icon.AppIconPNG))
	w.Resize(fyne.NewSize(680, 680))
	w.SetFixedSize(false)
	w.CenterOnScreen()

	buildUI(a, w, appDir, cfg)
	w.ShowAndRun()
}

func buildUI(a fyne.App, w fyne.Window, appDir string, cfg *config.Config) {

	// ── Folder picker ────────────────────────────────────────────────────────
	folderEntry := widget.NewEntry()
	folderEntry.SetText(cfg.RootFolder)
	folderEntry.SetPlaceHolder("Select the folder containing your images…")

	browseBtn := widget.NewButton("Browse…", func() {
		dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
			if err != nil || uri == nil {
				return
			}
			folderEntry.SetText(uri.Path())
		}, w)
	})
	folderRow := container.NewBorder(nil, nil, nil, browseBtn, folderEntry)

	// ── Target base names (multi-entry list) ─────────────────────────────────
	namesContainer := container.NewVBox()
	var nameEntries []*widget.Entry

	var refreshNamesList func()

	addNameEntry := func(value string) {
		entry := widget.NewEntry()
		entry.SetText(value)
		entry.SetPlaceHolder("e.g. front")

		idx := len(nameEntries)
		nameEntries = append(nameEntries, entry)

		removeBtn := widget.NewButton("✕", nil)
		removeBtn.Importance = widget.LowImportance
		removeBtn.OnTapped = func() {
			nameEntries = append(nameEntries[:idx], nameEntries[idx+1:]...)
			refreshNamesList()
		}

		row := container.NewBorder(nil, nil, nil, removeBtn, entry)
		namesContainer.Add(row)
	}

	refreshNamesList = func() {
		namesContainer.Objects = nil
		entries := make([]*widget.Entry, len(nameEntries))
		copy(entries, nameEntries)
		nameEntries = nil
		for _, e := range entries {
			addNameEntry(e.Text)
		}
		namesContainer.Refresh()
	}

	for _, name := range cfg.TargetBaseNames {
		addNameEntry(name)
	}

	addNameBtn := widget.NewButton("+ Add another filename", func() {
		addNameEntry("")
		namesContainer.Refresh()
	})
	addNameBtn.Importance = widget.LowImportance

	namesSection := container.NewVBox(namesContainer, addNameBtn)

	// ── Scan depth (number + +/- buttons + live label) ───────────────────────
	depthVal := cfg.ScanDepth
	depthEntry := widget.NewEntry()
	depthLabel := widget.NewLabel("")

	updateDepthDisplay := func() {
		if depthVal < 0 {
			depthEntry.SetText("Unlimited")
			depthLabel.SetText("Searching through all subfolders, no matter how deep")
		} else if depthVal == 0 {
			depthEntry.SetText("0")
			depthLabel.SetText("Searching this folder only, no subfolders")
		} else if depthVal == 1 {
			depthEntry.SetText("1")
			depthLabel.SetText("Searching 1 level of subfolders")
		} else {
			depthEntry.SetText(strconv.Itoa(depthVal))
			depthLabel.SetText(fmt.Sprintf("Searching %d levels of subfolders deep", depthVal))
		}
	}

	minusBtn := widget.NewButton("−", func() {
		if depthVal > -1 {
			depthVal--
		}
		updateDepthDisplay()
	})

	plusBtn := widget.NewButton("+", func() {
		depthVal++
		updateDepthDisplay()
	})

	depthEntry.OnChanged = func(s string) {
		s = strings.TrimSpace(s)
		if strings.ToLower(s) == "unlimited" || s == "-1" {
			depthVal = -1
			depthLabel.SetText("Searching through all subfolders, no matter how deep")
			return
		}
		if n, err := strconv.Atoi(s); err == nil && n >= 0 {
			depthVal = n
			if n == 0 {
				depthLabel.SetText("Searching this folder only, no subfolders")
			} else if n == 1 {
				depthLabel.SetText("Searching 1 level of subfolders")
			} else {
				depthLabel.SetText(fmt.Sprintf("Searching %d levels of subfolders deep", n))
			}
		}
	}

	updateDepthDisplay()

	depthControls := container.NewBorder(nil, nil,
		minusBtn,
		plusBtn,
		depthEntry,
	)
	depthSection := container.NewVBox(depthControls, depthLabel)

	// ── Suffixes + live preview ───────────────────────────────────────────────
	leftEntry := widget.NewEntry()
	leftEntry.SetText(cfg.LeftSuffix)
	leftEntry.SetPlaceHolder("_left")

	rightEntry := widget.NewEntry()
	rightEntry.SetText(cfg.RightSuffix)
	rightEntry.SetPlaceHolder("_right")

	previewLabel := widget.NewLabel("")
	previewLabel.TextStyle = fyne.TextStyle{Monospace: true}

	updatePreview := func() {
		var firstName string
		if len(nameEntries) > 0 {
			firstName = strings.TrimSpace(nameEntries[0].Text)
		}
		if firstName == "" {
			firstName = "image"
		}
		l := strings.TrimSpace(leftEntry.Text)
		r := strings.TrimSpace(rightEntry.Text)
		if l == "" {
			l = "_left"
		}
		if r == "" {
			r = "_right"
		}
		previewLabel.SetText(fmt.Sprintf(
			"Example:   %s%s.jpg   and   %s%s.jpg",
			firstName, l, firstName, r,
		))
	}

	leftEntry.OnChanged = func(_ string) { updatePreview() }
	rightEntry.OnChanged = func(_ string) { updatePreview() }
	updatePreview()

	suffixRow := container.NewGridWithColumns(2,
		labeledWidget("Left half suffix", leftEntry),
		labeledWidget("Right half suffix", rightEntry),
	)

	// ── Checkboxes ───────────────────────────────────────────────────────────
	deleteCheck := widget.NewCheck("Delete original image after splitting", nil)
	deleteCheck.SetChecked(cfg.DeleteOriginal)

	overwriteCheck := widget.NewCheck("Re-process images that have already been split", nil)
	overwriteCheck.SetChecked(cfg.OverwriteExisting)

	debugCheck := widget.NewCheck("Debug mode — write detailed logs for each step", nil)
	debugCheck.SetChecked(cfg.DebugMode)

	// ── Run button + About ───────────────────────────────────────────────────
	runBtn := widget.NewButton("  Run  ", nil)
	runBtn.Importance = widget.HighImportance

	aboutBtn := widget.NewButton("About", func() { showAboutDialog(w) })

	runBtn.OnTapped = func() {
		// Collect base names from entries (skip empty).
		var baseNames []string
		for _, e := range nameEntries {
			if n := strings.TrimSpace(e.Text); n != "" {
				baseNames = append(baseNames, n)
			}
		}

		current := &config.Config{
			RootFolder:        strings.TrimSpace(folderEntry.Text),
			ScanDepth:         depthVal,
			TargetBaseNames:   baseNames,
			LeftSuffix:        strings.TrimSpace(leftEntry.Text),
			RightSuffix:       strings.TrimSpace(rightEntry.Text),
			DeleteOriginal:    deleteCheck.Checked,
			OverwriteExisting: overwriteCheck.Checked,
			DebugMode:         debugCheck.Checked,
		}

		if err := validateConfig(current); err != nil {
			dialog.ShowError(err, w)
			return
		}

		_ = config.Save(appDir, current)

		allWidgets := []interface{}{
			runBtn, browseBtn, folderEntry,
			leftEntry, rightEntry,
			deleteCheck, overwriteCheck, debugCheck,
			addNameBtn, minusBtn, plusBtn, depthEntry,
		}
		for _, e := range nameEntries {
			allWidgets = append(allWidgets, e)
		}
		setUIEnabled(false, allWidgets...)

		// Spinner dialog.
		spinLabel := widget.NewLabel("Processing… please wait")
		spinLabel.Alignment = fyne.TextAlignCenter
		spinner := widget.NewProgressBarInfinite()
		spinnerDlg := dialog.NewCustomWithoutButtons("Running",
			container.NewVBox(spinLabel, spinner), w)
		spinnerDlg.Resize(fyne.NewSize(300, 110))
		spinnerDlg.Show()

		go func() {
			result, latestReport, latestLog, runErr := executeRun(appDir, current)

			// Marshal back onto the Fyne main thread.
			done := make(chan struct{})
			go func() {
				spinnerDlg.Hide()
				setUIEnabled(true, allWidgets...)
				if runErr != nil {
					dialog.ShowError(fmt.Errorf("Run failed:\n%v", runErr), w)
				} else {
					showSuccessDialog(w, result, latestReport, latestLog)
				}
				close(done)
			}()
			<-done
		}()
	}

	// ── Layout ───────────────────────────────────────────────────────────────
	form := container.NewVBox(
		sectionLabel("Folder to process"),
		folderRow,

		widget.NewSeparator(),
		sectionLabel("Image filenames to split (without extension)"),
		hintLabel("The app will find these files in any supported format: JPG, PNG, BMP, TIFF, WebP"),
		namesSection,

		widget.NewSeparator(),
		sectionLabel("How deep to search subfolders"),
		depthSection,

		widget.NewSeparator(),
		sectionLabel("Output file naming"),
		suffixRow,
		previewLabel,

		widget.NewSeparator(),
		sectionLabel("Options"),
		deleteCheck,
		overwriteCheck,
		debugCheck,
	)

	scroll := container.NewVScroll(container.NewPadded(form))
	bottom := container.NewPadded(container.NewBorder(nil, nil, aboutBtn, runBtn))
	content := container.NewBorder(nil, bottom, nil, nil, scroll)
	w.SetContent(content)
}

// ── Helpers ──────────────────────────────────────────────────────────────────

func sectionLabel(text string) *widget.Label {
	l := widget.NewLabel(text)
	l.TextStyle = fyne.TextStyle{Bold: true}
	return l
}

func hintLabel(text string) *widget.Label {
	l := widget.NewLabel(text)
	l.TextStyle = fyne.TextStyle{Italic: true}
	return l
}

func labeledWidget(label string, w fyne.CanvasObject) *fyne.Container {
	return container.NewVBox(widget.NewLabel(label), w)
}

func validateConfig(cfg *config.Config) error {
	if cfg.RootFolder == "" {
		return fmt.Errorf("Please select a folder to process.")
	}
	if _, err := os.Stat(cfg.RootFolder); os.IsNotExist(err) {
		return fmt.Errorf("The selected folder does not exist:\n%s", cfg.RootFolder)
	}
	if len(cfg.TargetBaseNames) == 0 {
		return fmt.Errorf("Please enter at least one image filename to split.")
	}
	if cfg.LeftSuffix == "" || cfg.RightSuffix == "" {
		return fmt.Errorf("Left and right suffixes must not be empty.")
	}
	if cfg.LeftSuffix == cfg.RightSuffix {
		return fmt.Errorf("Left and right suffixes must be different from each other.")
	}
	return nil
}

func setUIEnabled(enabled bool, widgets ...interface{}) {
	type disabler interface{ Disable() }
	type enabler interface{ Enable() }
	for _, w := range widgets {
		if enabled {
			if e, ok := w.(enabler); ok {
				e.Enable()
			}
		} else {
			if d, ok := w.(disabler); ok {
				d.Disable()
			}
		}
	}
}

// executeRun performs the full processing run.
func executeRun(appDir string, cfg *config.Config) (
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
	logger.Debug(fmt.Sprintf("Config: root=%q depth=%d targets=%v leftSuffix=%q rightSuffix=%q",
		cfg.RootFolder, cfg.ScanDepth, cfg.TargetBaseNames, cfg.LeftSuffix, cfg.RightSuffix))

	folders, walkErrs := filesystem.DiscoverFolders(cfg.RootFolder, cfg.ScanDepth)
	for _, we := range walkErrs {
		logger.Warn(fmt.Sprintf("Skipped unreadable directory %q: %v", we.Path, we.Err))
	}
	logger.Info(fmt.Sprintf("Discovered %d folder(s)", len(folders)))

	for _, dir := range folders {
		fr := processor.ProcessFolder(dir, cfg, logger)
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

func showSuccessDialog(parent fyne.Window, result *models.RunResult, reportPath, logPath string) {
	processed, alreadyProcessed, missing, errors := result.Counts()

	summary := fmt.Sprintf(
		"Run %03d completed in %s\n\n"+
			"  \u2713  Images split:       %d\n"+
			"  \u21b7  Already processed:  %d\n"+
			"  \u26a0  Image missing:      %d\n"+
			"  \u2715  Errors:             %d",
		result.RunNumber,
		result.Duration().Round(time.Millisecond),
		processed, alreadyProcessed, missing, errors,
	)

	summaryLabel := widget.NewLabel(summary)
	summaryLabel.TextStyle = fyne.TextStyle{Monospace: true}

	reportBtn := widget.NewButton("View Report", func() { openFile(reportPath) })
	reportBtn.Importance = widget.HighImportance

	logBtn := widget.NewButton("View Log", func() { openFile(logPath) })

	content := container.NewVBox(
		summaryLabel,
		widget.NewSeparator(),
		container.NewHBox(layout.NewSpacer(), logBtn, reportBtn),
	)

	d := dialog.NewCustom("Run Complete", "Close", content, parent)
	d.Resize(fyne.NewSize(440, 270))
	d.Show()

	openFile(reportPath)
}

func showAboutDialog(parent fyne.Window) {
	title := widget.NewLabel("Image Splitter")
	title.TextStyle = fyne.TextStyle{Bold: true}
	title.Alignment = fyne.TextAlignCenter

	ver := widget.NewLabel("Version " + config.Version)
	ver.Alignment = fyne.TextAlignCenter

	desc := widget.NewLabel("Splits images in a folder hierarchy into left and right halves.\nSupports JPG, PNG, BMP, TIFF and WebP.")
	desc.Alignment = fyne.TextAlignCenter
	desc.Wrapping = fyne.TextWrapWord

	content := container.NewVBox(title, ver, widget.NewSeparator(), desc)
	d := dialog.NewCustom("About", "Close", content, parent)
	d.Resize(fyne.NewSize(340, 210))
	d.Show()
}

// openFile opens a file in the system's default application.
func openFile(path string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", path)
	case "darwin":
		cmd = exec.Command("open", path)
	default:
		cmd = exec.Command("xdg-open", path)
	}
	_ = cmd.Start()
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}
