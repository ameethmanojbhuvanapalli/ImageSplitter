package gui

import (
	"fmt"
	"image/color"
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
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"imagesplitter/internal/config"
	"imagesplitter/internal/filesystem"
	appicon "imagesplitter/internal/icon"
	"imagesplitter/internal/logging"
	"imagesplitter/internal/models"
	"imagesplitter/internal/padder"
	"imagesplitter/internal/processor"
	"imagesplitter/internal/report"
)

// ── Entry point ───────────────────────────────────────────────────────────────

func Run(appDir string, cfg *config.Config) {
	a := app.NewWithID("com.imagesplitter.app")
	a.Settings().SetTheme(theme.LightTheme())
	a.SetIcon(fyne.NewStaticResource("icon.png", appicon.AppIconPNG))

	w := a.NewWindow("Image Splitter")
	w.SetIcon(fyne.NewStaticResource("icon.png", appicon.AppIconPNG))
	w.Resize(fyne.NewSize(700, 700))
	w.CenterOnScreen()

	showWizard(a, w, appDir, cfg)
	w.ShowAndRun()
}

// ── Wizard state ──────────────────────────────────────────────────────────────

type wizardState struct {
	a      fyne.App
	w      fyne.Window
	appDir string
	cfg    *config.Config

	currentStep int // 0-3
	totalSteps  int // 4

	// Step indicator labels
	stepDots []*canvas.Circle

	// Content area
	contentArea *fyne.Container

	// Navigation buttons
	backBtn *widget.Button
	nextBtn *widget.Button

	// Step 1 — General
	folderEntry *widget.Entry
	depthVal    int
	depthEntry  *widget.Entry
	depthLabel  *widget.Label
	debugCheck  *widget.Check

	// Step 2 — Splitting
	splitEnabled     *widget.Check
	splitNames       []*widget.Entry
	splitNamesBox    *fyne.Container
	splitLeftSuffix  *widget.Entry
	splitRightSuffix *widget.Entry
	splitPreview     *widget.Label
	splitDelete      *widget.Check
	splitOverwrite   *widget.Check

	// Step 3 — Padding
	padEnabled       *widget.Check
	padLeftNames     []*widget.Entry
	padLeftBox       *fyne.Container
	padRightNames    []*widget.Entry
	padRightBox      *fyne.Container
	padCreateNew     *widget.Check
	padLeftSuffix    *widget.Entry
	padRightSuffix   *widget.Entry
	padOverwrite     *widget.Check
	padSuffixSection *fyne.Container
	padColor         *widget.Entry // hex string
}

func showWizard(a fyne.App, w fyne.Window, appDir string, cfg *config.Config) {
	ws := &wizardState{
		a:          a,
		w:          w,
		appDir:     appDir,
		cfg:        cfg,
		totalSteps: 4,
		depthVal:   cfg.ScanDepth,
	}

	ws.buildChrome()
	ws.goToStep(0)
}

// buildChrome builds the persistent wizard shell (step indicator + nav buttons).
func (ws *wizardState) buildChrome() {
	// ── Step indicator ────────────────────────────────────────────────────────
	stepLabels := []string{"General", "Splitting", "Padding", "Review"}
	indicatorRow := container.NewHBox(layout.NewSpacer())
	ws.stepDots = make([]*canvas.Circle, ws.totalSteps)

	for i, label := range stepLabels {
		idx := i
		dot := canvas.NewCircle(color.RGBA{R: 200, G: 200, B: 210, A: 255})
		dot.Resize(fyne.NewSize(12, 12))
		ws.stepDots[i] = dot

		lbl := widget.NewLabel(label)
		lbl.TextStyle = fyne.TextStyle{Bold: false}

		dotContainer := container.NewWithoutLayout(dot)
		dotContainer.Resize(fyne.NewSize(12, 12))
		dot.Resize(fyne.NewSize(12, 12))
		stepCol := container.NewVBox(
			container.NewCenter(dotContainer),
			container.NewCenter(lbl),
		)
		_ = idx
		indicatorRow.Add(stepCol)
		if i < ws.totalSteps-1 {
			sep := canvas.NewLine(color.RGBA{R: 210, G: 210, B: 220, A: 255})
			sep.StrokeWidth = 1
			indicatorRow.Add(container.NewCenter(sep))
		}
	}
	indicatorRow.Add(layout.NewSpacer())

	// ── Content area ──────────────────────────────────────────────────────────
	ws.contentArea = container.NewStack()

	// ── Navigation ───────────────────────────────────────────────────────────
	ws.backBtn = widget.NewButton("← Back", func() { ws.goToStep(ws.currentStep - 1) })
	ws.nextBtn = widget.NewButton("Next →", func() { ws.advance() })
	ws.nextBtn.Importance = widget.HighImportance
	ws.backBtn.Disable()

	navRow := container.NewBorder(nil, nil, ws.backBtn, ws.nextBtn)

	// ── About ─────────────────────────────────────────────────────────────────
	aboutBtn := widget.NewButton("About", func() { ws.showAbout() })

	bottomBar := container.NewBorder(nil, nil, aboutBtn, nil, navRow)

	root := container.NewBorder(
		container.NewPadded(indicatorRow),
		container.NewPadded(bottomBar),
		nil, nil,
		container.NewPadded(ws.contentArea),
	)
	ws.w.SetContent(root)
}

func (ws *wizardState) updateIndicator() {
	accent := color.RGBA{R: 79, G: 70, B: 229, A: 255}
	inactive := color.RGBA{R: 200, G: 200, B: 210, A: 255}
	for i, dot := range ws.stepDots {
		if i == ws.currentStep {
			dot.FillColor = accent
		} else if i < ws.currentStep {
			dot.FillColor = color.RGBA{R: 34, G: 197, B: 94, A: 255} // green = done
		} else {
			dot.FillColor = inactive
		}
		dot.Refresh()
	}
}

func (ws *wizardState) goToStep(step int) {
	ws.currentStep = step
	ws.updateIndicator()

	// Back button
	if step == 0 {
		ws.backBtn.Disable()
	} else {
		ws.backBtn.Enable()
	}

	// Next/Run button label
	if step == ws.totalSteps-1 {
		ws.nextBtn.SetText(ws.runLabel())
		ws.nextBtn.OnTapped = func() { ws.runClicked() }
	} else {
		ws.nextBtn.SetText("Next →")
		ws.nextBtn.OnTapped = func() { ws.advance() }
	}

	// Render step content
	var content fyne.CanvasObject
	switch step {
	case 0:
		content = ws.buildStep1()
	case 1:
		content = ws.buildStep2()
	case 2:
		content = ws.buildStep3()
	case 3:
		content = ws.buildStep4()
	}

	ws.contentArea.Objects = []fyne.CanvasObject{container.NewVScroll(content)}
	ws.contentArea.Refresh()
}

func (ws *wizardState) advance() {
	if err := ws.validateStep(ws.currentStep); err != nil {
		dialog.ShowError(err, ws.w)
		return
	}
	ws.collectStep(ws.currentStep)
	ws.goToStep(ws.currentStep + 1)
}

func (ws *wizardState) runLabel() string {
	splitOn := ws.cfg.Splitting.Enabled
	padOn := ws.cfg.Padding.Enabled
	switch {
	case splitOn && padOn:
		return "▶  Run (Splitting + Padding)"
	case splitOn:
		return "▶  Run (Splitting only)"
	case padOn:
		return "▶  Run (Padding only)"
	default:
		return "▶  Run"
	}
}

// ── Step 1: General ───────────────────────────────────────────────────────────

func (ws *wizardState) buildStep1() fyne.CanvasObject {
	ws.folderEntry = widget.NewEntry()
	ws.folderEntry.SetText(ws.cfg.RootFolder)
	ws.folderEntry.SetPlaceHolder("Select the folder containing your images…")

	browseBtn := widget.NewButton("Browse…", func() {
		dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
			if err != nil || uri == nil {
				return
			}
			ws.folderEntry.SetText(uri.Path())
		}, ws.w)
	})

	ws.depthEntry = widget.NewEntry()
	ws.depthLabel = widget.NewLabel("")
	ws.depthLabel.TextStyle = fyne.TextStyle{Italic: true}

	ws.updateDepthDisplay()

	ws.depthEntry.OnChanged = func(s string) {
		s = strings.TrimSpace(s)
		if strings.ToLower(s) == "unlimited" || s == "-1" {
			ws.depthVal = -1
		} else if n, err := strconv.Atoi(s); err == nil && n >= 0 {
			ws.depthVal = n
		}
		ws.updateDepthLabel()
	}

	minusBtn := widget.NewButton("−", func() {
		if ws.depthVal > -1 {
			ws.depthVal--
		}
		ws.updateDepthDisplay()
	})
	plusBtn := widget.NewButton("+", func() {
		ws.depthVal++
		ws.updateDepthDisplay()
	})

	depthRow := container.NewBorder(nil, nil, minusBtn, plusBtn, ws.depthEntry)

	ws.debugCheck = widget.NewCheck("Debug mode — write detailed step-by-step logs", nil)
	ws.debugCheck.SetChecked(ws.cfg.DebugMode)

	return container.NewVBox(
		stepTitle("Step 1 of 4 — General Settings"),
		widget.NewSeparator(),
		sectionLabel("Folder to process"),
		container.NewBorder(nil, nil, nil, browseBtn, ws.folderEntry),
		vspace(),
		sectionLabel("How deep to search subfolders"),
		hintLabel("Use − and + to adjust, or type a number. -1 or \"Unlimited\" means no limit."),
		depthRow,
		ws.depthLabel,
		vspace(),
		sectionLabel("Logging"),
		ws.debugCheck,
	)
}

func (ws *wizardState) updateDepthDisplay() {
	if ws.depthVal < 0 {
		ws.depthEntry.SetText("Unlimited")
	} else {
		ws.depthEntry.SetText(strconv.Itoa(ws.depthVal))
	}
	ws.updateDepthLabel()
}

func (ws *wizardState) updateDepthLabel() {
	switch {
	case ws.depthVal < 0:
		ws.depthLabel.SetText("Searching through all subfolders, no matter how deep")
	case ws.depthVal == 0:
		ws.depthLabel.SetText("Searching this folder only — no subfolders")
	case ws.depthVal == 1:
		ws.depthLabel.SetText("Searching 1 level of subfolders")
	default:
		ws.depthLabel.SetText(fmt.Sprintf("Searching %d levels of subfolders deep", ws.depthVal))
	}
}

// ── Step 2: Splitting ─────────────────────────────────────────────────────────

func (ws *wizardState) buildStep2() fyne.CanvasObject {
	ws.splitEnabled = widget.NewCheck("Enable splitting", nil)
	ws.splitEnabled.SetChecked(ws.cfg.Splitting.Enabled)

	ws.splitNamesBox = container.NewVBox()
	ws.splitNames = nil
	for _, n := range ws.cfg.Splitting.TargetBaseNames {
		ws.addSplitName(n)
	}
	if len(ws.splitNames) == 0 {
		ws.addSplitName("front")
	}

	addNameBtn := widget.NewButton("+ Add image name", func() {
		ws.addSplitName("")
	})
	addNameBtn.Importance = widget.LowImportance

	ws.splitLeftSuffix = widget.NewEntry()
	ws.splitLeftSuffix.SetText(ws.cfg.Splitting.LeftSuffix)
	ws.splitLeftSuffix.SetPlaceHolder("_left")

	ws.splitRightSuffix = widget.NewEntry()
	ws.splitRightSuffix.SetText(ws.cfg.Splitting.RightSuffix)
	ws.splitRightSuffix.SetPlaceHolder("_right")

	ws.splitPreview = widget.NewLabel("")
	ws.splitPreview.TextStyle = fyne.TextStyle{Monospace: true}
	ws.updateSplitPreview()

	ws.splitLeftSuffix.OnChanged = func(_ string) { ws.updateSplitPreview() }
	ws.splitRightSuffix.OnChanged = func(_ string) { ws.updateSplitPreview() }

	ws.splitDelete = widget.NewCheck("Delete original image after splitting", nil)
	ws.splitDelete.SetChecked(ws.cfg.Splitting.DeleteOriginal)

	ws.splitOverwrite = widget.NewCheck("Re-process images that have already been split", nil)
	ws.splitOverwrite.SetChecked(ws.cfg.Splitting.OverwriteExisting)

	suffixRow := container.NewGridWithColumns(2,
		labeledWidget("Left half named with suffix", ws.splitLeftSuffix),
		labeledWidget("Right half named with suffix", ws.splitRightSuffix),
	)

	settingsBox := container.NewVBox(
		vspace(),
		sectionLabel("Image filenames to split"),
		hintLabel("Base name only — no extension. App finds JPG, JPEG, PNG, BMP, TIFF, WebP automatically."),
		ws.splitNamesBox,
		addNameBtn,
		vspace(),
		sectionLabel("Output file naming"),
		suffixRow,
		ws.splitPreview,
		vspace(),
		sectionLabel("Options"),
		ws.splitDelete,
		ws.splitOverwrite,
	)

	ws.splitEnabled.OnChanged = func(on bool) {
		setContainerEnabled(on, settingsBox)
	}
	setContainerEnabled(ws.cfg.Splitting.Enabled, settingsBox)

	return container.NewVBox(
		stepTitle("Step 2 of 4 — Splitting"),
		hintLabel("Splits each image vertically into two equal halves."),
		widget.NewSeparator(),
		ws.splitEnabled,
		settingsBox,
	)
}

func (ws *wizardState) addSplitName(value string) {
	idx := len(ws.splitNames)
	entry := widget.NewEntry()
	entry.SetText(value)
	entry.SetPlaceHolder("e.g. front")
	entry.OnChanged = func(_ string) { ws.updateSplitPreview() }
	ws.splitNames = append(ws.splitNames, entry)

	removeBtn := widget.NewButton("✕", nil)
	removeBtn.Importance = widget.LowImportance
	removeBtn.OnTapped = func() {
		ws.splitNames = append(ws.splitNames[:idx], ws.splitNames[idx+1:]...)
		ws.rebuildSplitNames()
	}

	ws.splitNamesBox.Add(container.NewBorder(nil, nil, nil, removeBtn, entry))
	if ws.splitNamesBox != nil {
		ws.splitNamesBox.Refresh()
	}
}

func (ws *wizardState) rebuildSplitNames() {
	existing := make([]string, len(ws.splitNames))
	for i, e := range ws.splitNames {
		existing[i] = e.Text
	}
	ws.splitNamesBox.Objects = nil
	ws.splitNames = nil
	for _, v := range existing {
		ws.addSplitName(v)
	}
	ws.splitNamesBox.Refresh()
}

func (ws *wizardState) updateSplitPreview() {
	if ws.splitPreview == nil {
		return
	}
	name := "image"
	for _, e := range ws.splitNames {
		if t := strings.TrimSpace(e.Text); t != "" {
			name = t
			break
		}
	}
	l := ws.splitLeftSuffix.Text
	r := ws.splitRightSuffix.Text
	if l == "" {
		l = "_left"
	}
	if r == "" {
		r = "_right"
	}
	ws.splitPreview.SetText(fmt.Sprintf("Example: %s%s.jpg  +  %s%s.jpg", name, l, name, r))
}

// ── Step 3: Padding ───────────────────────────────────────────────────────────

func (ws *wizardState) buildStep3() fyne.CanvasObject {
	ws.padEnabled = widget.NewCheck("Enable padding", nil)
	ws.padEnabled.SetChecked(ws.cfg.Padding.Enabled)

	// Left pad names
	ws.padLeftBox = container.NewVBox()
	ws.padLeftNames = nil
	for _, n := range ws.cfg.Padding.LeftPadNames {
		ws.addPadName(n, true)
	}

	addLeftBtn := widget.NewButton("+ Add image name", func() { ws.addPadName("", true) })
	addLeftBtn.Importance = widget.LowImportance

	// Right pad names
	ws.padRightBox = container.NewVBox()
	ws.padRightNames = nil
	for _, n := range ws.cfg.Padding.RightPadNames {
		ws.addPadName(n, false)
	}

	addRightBtn := widget.NewButton("+ Add image name", func() { ws.addPadName("", false) })
	addRightBtn.Importance = widget.LowImportance

	// Output mode
	ws.padCreateNew = widget.NewCheck("Save as new file (instead of overwriting original)", nil)
	ws.padCreateNew.SetChecked(ws.cfg.Padding.CreateNewFile)

	ws.padLeftSuffix = widget.NewEntry()
	ws.padLeftSuffix.SetText(ws.cfg.Padding.LeftSuffix)
	ws.padLeftSuffix.SetPlaceHolder("e.g. _padded")

	ws.padRightSuffix = widget.NewEntry()
	ws.padRightSuffix.SetText(ws.cfg.Padding.RightSuffix)
	ws.padRightSuffix.SetPlaceHolder("e.g. _padded")

	ws.padOverwrite = widget.NewCheck("Skip if output file already exists", nil)
	ws.padOverwrite.SetChecked(!ws.cfg.Padding.OverwriteExisting)

	ws.padSuffixSection = container.NewVBox(
		container.NewGridWithColumns(2,
			labeledWidget("Left image output suffix", ws.padLeftSuffix),
			labeledWidget("Right image output suffix", ws.padRightSuffix),
		),
		ws.padOverwrite,
	)
	if !ws.cfg.Padding.CreateNewFile {
		setContainerEnabled(false, ws.padSuffixSection)
	}

	ws.padCreateNew.OnChanged = func(on bool) {
		setContainerEnabled(on, ws.padSuffixSection)
	}

	// Pad colour
	ws.padColor = widget.NewEntry()
	ws.padColor.SetText(ws.cfg.Padding.PadColor)
	ws.padColor.SetPlaceHolder("#FFFFFF")

	colorHint := hintLabel("Hex colour for the white space area. #FFFFFF = pure white.")

	settingsBox := container.NewVBox(
		vspace(),
		sectionLabel("Add white space to LEFT side  (image sits on the right)"),
		hintLabel("Base name only — no extension."),
		ws.padLeftBox,
		addLeftBtn,
		vspace(),
		sectionLabel("Add white space to RIGHT side  (image sits on the left)"),
		hintLabel("Base name only — no extension."),
		ws.padRightBox,
		addRightBtn,
		vspace(),
		sectionLabel("Output"),
		ws.padCreateNew,
		ws.padSuffixSection,
		vspace(),
		sectionLabel("White space colour"),
		ws.padColor,
		colorHint,
	)

	ws.padEnabled.OnChanged = func(on bool) {
		setContainerEnabled(on, settingsBox)
	}
	setContainerEnabled(ws.cfg.Padding.Enabled, settingsBox)

	return container.NewVBox(
		stepTitle("Step 3 of 4 — Padding"),
		hintLabel("Doubles image width by adding a solid colour canvas on one side."),
		widget.NewSeparator(),
		ws.padEnabled,
		settingsBox,
	)
}

func (ws *wizardState) addPadName(value string, isLeft bool) {
	box := ws.padLeftBox
	names := &ws.padLeftNames
	if !isLeft {
		box = ws.padRightBox
		names = &ws.padRightNames
	}

	idx := len(*names)
	entry := widget.NewEntry()
	entry.SetText(value)
	entry.SetPlaceHolder("e.g. 00000")
	*names = append(*names, entry)

	removeBtn := widget.NewButton("✕", nil)
	removeBtn.Importance = widget.LowImportance
	removeBtn.OnTapped = func() {
		*names = append((*names)[:idx], (*names)[idx+1:]...)
		ws.rebuildPadNames(isLeft)
	}

	box.Add(container.NewBorder(nil, nil, nil, removeBtn, entry))
	box.Refresh()
}

func (ws *wizardState) rebuildPadNames(isLeft bool) {
	box := ws.padLeftBox
	names := &ws.padLeftNames
	if !isLeft {
		box = ws.padRightBox
		names = &ws.padRightNames
	}
	existing := make([]string, len(*names))
	for i, e := range *names {
		existing[i] = e.Text
	}
	box.Objects = nil
	*names = nil
	for _, v := range existing {
		ws.addPadName(v, isLeft)
	}
	box.Refresh()
}

// ── Step 4: Review & Run ──────────────────────────────────────────────────────

func (ws *wizardState) buildStep4() fyne.CanvasObject {
	// collectStep(2) is intentionally NOT called here to avoid nil widget panics.
	// Config is already collected by advance() before goToStep(3) is called.

	split := ws.cfg.Splitting
	pad := ws.cfg.Padding

	splitStatus := "OFF"
	if split.Enabled {
		splitStatus = "ON"
	}
	padStatus := "OFF"
	if pad.Enabled {
		padStatus = "ON"
	}

	depthStr := "All subfolders (unlimited)"
	if ws.cfg.ScanDepth == 0 {
		depthStr = "This folder only"
	} else if ws.cfg.ScanDepth > 0 {
		depthStr = fmt.Sprintf("%d level(s) deep", ws.cfg.ScanDepth)
	}

	debugStr := "OFF"
	if ws.cfg.DebugMode {
		debugStr = "ON"
	}

	padOutputStr := "Overwrite original"
	if pad.CreateNewFile {
		padOutputStr = fmt.Sprintf("New file — left suffix: %q  right suffix: %q",
			pad.LeftSuffix, pad.RightSuffix)
	}

	summary := fmt.Sprintf(
		"Folder:         %s\n"+
			"Scan depth:     %s\n"+
			"Debug logs:     %s\n\n"+
			"─── Splitting ──────────────  %s\n"+
			"  Images:       %s\n"+
			"  Left suffix:  %s\n"+
			"  Right suffix: %s\n\n"+
			"─── Padding ────────────────  %s\n"+
			"  Pad left:     %s\n"+
			"  Pad right:    %s\n"+
			"  Output:       %s\n"+
			"  Pad colour:   %s",
		ws.cfg.RootFolder,
		depthStr,
		debugStr,
		splitStatus,
		strings.Join(split.TargetBaseNames, ", "),
		split.LeftSuffix,
		split.RightSuffix,
		padStatus,
		strings.Join(pad.LeftPadNames, ", "),
		strings.Join(pad.RightPadNames, ", "),
		padOutputStr,
		pad.PadColor,
	)

	summaryLabel := widget.NewLabel(summary)
	summaryLabel.TextStyle = fyne.TextStyle{Monospace: true}
	summaryLabel.Wrapping = fyne.TextWrapWord

	// Update run button label now that config is final.
	ws.nextBtn.SetText(ws.runLabel())

	bothOff := !split.Enabled && !pad.Enabled
	if bothOff {
		ws.nextBtn.Disable()
	} else {
		ws.nextBtn.Enable()
	}

	note := ""
	if bothOff {
		note = "⚠  Both operations are OFF. Enable at least one on the previous steps."
	}
	noteLabel := widget.NewLabel(note)
	noteLabel.TextStyle = fyne.TextStyle{Italic: true}

	return container.NewVBox(
		stepTitle("Step 4 of 4 — Review & Run"),
		hintLabel("Check everything looks right, then click Run."),
		widget.NewSeparator(),
		summaryLabel,
		noteLabel,
	)
}

// ── Validation & collection ───────────────────────────────────────────────────

func (ws *wizardState) validateStep(step int) error {
	switch step {
	case 0:
		if strings.TrimSpace(ws.folderEntry.Text) == "" {
			return fmt.Errorf("Please select a folder to process.")
		}
		if _, err := os.Stat(strings.TrimSpace(ws.folderEntry.Text)); os.IsNotExist(err) {
			return fmt.Errorf("The selected folder does not exist:\n%s", ws.folderEntry.Text)
		}
	case 1:
		if ws.splitEnabled.Checked {
			var names []string
			for _, e := range ws.splitNames {
				if t := strings.TrimSpace(e.Text); t != "" {
					names = append(names, t)
				}
			}
			if len(names) == 0 {
				return fmt.Errorf("Splitting is ON but no image names are entered.")
			}
			if strings.TrimSpace(ws.splitLeftSuffix.Text) == "" ||
				strings.TrimSpace(ws.splitRightSuffix.Text) == "" {
				return fmt.Errorf("Left and right suffixes must not be empty.")
			}
			if strings.TrimSpace(ws.splitLeftSuffix.Text) == strings.TrimSpace(ws.splitRightSuffix.Text) {
				return fmt.Errorf("Left and right suffixes must be different from each other.")
			}
		}
	case 2:
		if ws.padEnabled.Checked {
			// Collect all names from both sides.
			leftSet := map[string]bool{}
			rightSet := map[string]bool{}
			for _, e := range ws.padLeftNames {
				if t := strings.TrimSpace(e.Text); t != "" {
					leftSet[t] = true
				}
			}
			for _, e := range ws.padRightNames {
				if t := strings.TrimSpace(e.Text); t != "" {
					rightSet[t] = true
				}
			}
			if len(leftSet) == 0 && len(rightSet) == 0 {
				return fmt.Errorf("Padding is ON but no image names are entered for either side.")
			}
			// Check overlap.
			for name := range leftSet {
				if rightSet[name] {
					return fmt.Errorf("Image %q appears in both Left and Right padding lists.\nA name can only be in one list.", name)
				}
			}
			// Validate colour.
			col := strings.TrimSpace(ws.padColor.Text)
			if !strings.HasPrefix(col, "#") || len(col) != 7 {
				return fmt.Errorf("Pad colour must be a hex value like #FFFFFF.")
			}
			// Validate suffixes if creating new files.
			if ws.padCreateNew.Checked {
				if strings.TrimSpace(ws.padLeftSuffix.Text) == "" &&
					strings.TrimSpace(ws.padRightSuffix.Text) == "" {
					return fmt.Errorf("Please enter at least one output suffix for the padded files.")
				}
			}
		}
	}
	return nil
}

func (ws *wizardState) collectStep(step int) {
	switch step {
	case 0:
		ws.cfg.RootFolder = strings.TrimSpace(ws.folderEntry.Text)
		ws.cfg.ScanDepth = ws.depthVal
		ws.cfg.DebugMode = ws.debugCheck.Checked

	case 1:
		ws.cfg.Splitting.Enabled = ws.splitEnabled.Checked
		var names []string
		for _, e := range ws.splitNames {
			if t := strings.TrimSpace(e.Text); t != "" {
				names = append(names, t)
			}
		}
		ws.cfg.Splitting.TargetBaseNames = names
		ws.cfg.Splitting.LeftSuffix = strings.TrimSpace(ws.splitLeftSuffix.Text)
		ws.cfg.Splitting.RightSuffix = strings.TrimSpace(ws.splitRightSuffix.Text)
		ws.cfg.Splitting.DeleteOriginal = ws.splitDelete.Checked
		ws.cfg.Splitting.OverwriteExisting = ws.splitOverwrite.Checked

	case 2:
		if ws.padEnabled == nil {
			return // step 3 widgets not built yet
		}
		ws.cfg.Padding.Enabled = ws.padEnabled.Checked
		var leftNames, rightNames []string
		for _, e := range ws.padLeftNames {
			if t := strings.TrimSpace(e.Text); t != "" {
				leftNames = append(leftNames, t)
			}
		}
		for _, e := range ws.padRightNames {
			if t := strings.TrimSpace(e.Text); t != "" {
				rightNames = append(rightNames, t)
			}
		}
		ws.cfg.Padding.LeftPadNames = leftNames
		ws.cfg.Padding.RightPadNames = rightNames
		ws.cfg.Padding.CreateNewFile = ws.padCreateNew.Checked
		ws.cfg.Padding.LeftSuffix = strings.TrimSpace(ws.padLeftSuffix.Text)
		ws.cfg.Padding.RightSuffix = strings.TrimSpace(ws.padRightSuffix.Text)
		ws.cfg.Padding.OverwriteExisting = !ws.padOverwrite.Checked
		ws.cfg.Padding.PadColor = strings.TrimSpace(ws.padColor.Text)
	}
}

// ── Run ───────────────────────────────────────────────────────────────────────

func (ws *wizardState) runClicked() {
	_ = config.Save(ws.appDir, ws.cfg)

	// Disable nav during run.
	ws.backBtn.Disable()
	ws.nextBtn.Disable()

	// Spinner.
	spinLabel := widget.NewLabel("Processing… please wait")
	spinLabel.Alignment = fyne.TextAlignCenter
	spinner := widget.NewProgressBarInfinite()
	spinDlg := dialog.NewCustomWithoutButtons("Running",
		container.NewVBox(spinLabel, spinner), ws.w)
	spinDlg.Resize(fyne.NewSize(300, 110))
	spinDlg.Show()

	go func() {
		result, latestReport, latestLog, runErr := executeRun(ws.appDir, ws.cfg)

		done := make(chan struct{})
		go func() {
			spinDlg.Hide()
			ws.backBtn.Enable()
			// Leave nextBtn with run label but re-enable.
			ws.nextBtn.Enable()

			if runErr != nil {
				dialog.ShowError(fmt.Errorf("Run failed:\n%v", runErr), ws.w)
			} else {
				ws.showSuccess(result, latestReport, latestLog)
			}
			close(done)
		}()
		<-done
	}()
}

func (ws *wizardState) showSuccess(result *models.RunResult, reportPath, logPath string) {
	processed, alreadyProcessed, missing, errors := result.Counts()
	splitProcessed, splitErrors, padProcessed, padErrors := result.CountsByOperation()

	summary := fmt.Sprintf(
		"Run %03d completed in %s\n\n"+
			"  \u2713  Total processed:    %d\n"+
			"  \u21b7  Already done:       %d\n"+
			"  \u26a0  Missing:            %d\n"+
			"  \u2715  Errors:             %d\n\n"+
			"  Split: %d done, %d errors\n"+
			"  Pad:   %d done, %d errors",
		result.RunNumber,
		result.Duration().Round(time.Millisecond),
		processed, alreadyProcessed, missing, errors,
		splitProcessed, splitErrors,
		padProcessed, padErrors,
	)

	lbl := widget.NewLabel(summary)
	lbl.TextStyle = fyne.TextStyle{Monospace: true}

	reportBtn := widget.NewButton("View Report", func() { openFile(reportPath) })
	reportBtn.Importance = widget.HighImportance
	logBtn := widget.NewButton("View Log", func() { openFile(logPath) })

	content := container.NewVBox(
		lbl,
		widget.NewSeparator(),
		container.NewHBox(layout.NewSpacer(), logBtn, reportBtn),
	)

	d := dialog.NewCustom("Run Complete", "Close", content, ws.w)
	d.Resize(fyne.NewSize(460, 310))
	d.Show()
	openFile(reportPath)
}

func (ws *wizardState) showAbout() {
	title := widget.NewLabel("Image Splitter")
	title.TextStyle = fyne.TextStyle{Bold: true}
	title.Alignment = fyne.TextAlignCenter
	ver := widget.NewLabel("Version " + config.Version)
	ver.Alignment = fyne.TextAlignCenter
	desc := widget.NewLabel("Splits and pads images across folder hierarchies.\nSupports JPG, JPEG, PNG, BMP, TIFF, WebP.")
	desc.Alignment = fyne.TextAlignCenter
	desc.Wrapping = fyne.TextWrapWord
	d := dialog.NewCustom("About", "Close",
		container.NewVBox(title, ver, widget.NewSeparator(), desc), ws.w)
	d.Resize(fyne.NewSize(340, 200))
	d.Show()
}

// ── Execution engine ──────────────────────────────────────────────────────────

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

	logger.Info(fmt.Sprintf("Run started RunNumber=%d Splitting=%v Padding=%v DebugMode=%v",
		runNumber, cfg.Splitting.Enabled, cfg.Padding.Enabled, cfg.DebugMode))

	folders, walkErrs := filesystem.DiscoverFolders(cfg.RootFolder, cfg.ScanDepth)
	for _, we := range walkErrs {
		logger.Warn(fmt.Sprintf("Skipped unreadable directory %q: %v", we.Path, we.Err))
	}
	logger.Info(fmt.Sprintf("Discovered %d folder(s)", len(folders)))

	for _, dir := range folders {
		fr := &models.FolderResult{
			FolderName: filepath.Base(dir),
			FolderPath: dir,
			StartTime:  time.Now(),
		}

		if cfg.Splitting.Enabled {
			splitResult := processor.ProcessFolder(dir, cfg, logger)
			fr.ImageResults = append(fr.ImageResults, splitResult.ImageResults...)
		}

		if cfg.Padding.Enabled {
			padder.ProcessFolder(fr, cfg, logger)
		}

		fr.EndTime = time.Now()
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

// ── UI helpers ────────────────────────────────────────────────────────────────

func stepTitle(text string) *widget.Label {
	l := widget.NewLabel(text)
	l.TextStyle = fyne.TextStyle{Bold: true}
	return l
}

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

func vspace() fyne.CanvasObject {
	return widget.NewLabel(" ")
}

// setContainerEnabled recursively enables or disables all widgets in a container.
func setContainerEnabled(enabled bool, c fyne.CanvasObject) {
	type disabler interface{ Disable() }
	type enabler interface{ Enable() }

	if cont, ok := c.(*fyne.Container); ok {
		for _, obj := range cont.Objects {
			setContainerEnabled(enabled, obj)
		}
		return
	}
	if enabled {
		if e, ok := c.(enabler); ok {
			e.Enable()
		}
	} else {
		if d, ok := c.(disabler); ok {
			d.Disable()
		}
	}
}

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
