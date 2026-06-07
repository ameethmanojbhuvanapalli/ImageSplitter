# Image Splitter

A cross-platform desktop app that vertically splits images across a folder
A cross-platform desktop app that vertically splits and pads images across a folder
hierarchy. Built with Go + Fyne. Distributed as a single executable — no
installation required.

---

## For End Users

### GUI mode (default)

```bash
splitter.exe
```

1. Fill in the settings form
2. Click **Run**
3. The report opens automatically in your browser

### Headless / unattended mode

```bash
splitter.exe --run
# or
splitter.exe /run
```

- No GUI is shown.
- `config.json` is loaded and processing starts immediately.
- The same logs/reports/history artifacts are generated as GUI runs.

Optional:

```bash
splitter.exe --run --open-report
```

This opens `Latest Report.html` after a headless run.

That's it. No config files to edit, no terminal, no setup.

### Exit codes (headless mode)

- `0` = Run completed successfully
- `1` = Fatal startup/configuration error
- `2` = Processing completed, but one or more folders ended with `Error` status

---

## Settings Explained

| Field | What it means |
|---|---|
| **Folder to process** | The top-level folder containing your images |
| **Image filename to split** | The exact filename to look for in each subfolder (e.g. `front.jpg`) |
| **How deep to search subfolders** | How many levels of subfolders to scan |
| **Left / Right output suffix** | What gets added to the filename for each half — preview updates live |
| **Delete original after splitting** | Removes the source image once both halves are saved |
| **Re-process already split images** | Runs again on folders that were already processed |
| **Splitting: Target names** | The exact filenames to split in each subfolder (e.g. `front`) |
| **Splitting: Suffixes** | What gets added to the filename for each half (e.g. `_left`, `_right`) |
| **Padding: Target names** | Filenames to receive white space on the left or right side |
| **Padding: Output mode** | Save results as new files or overwrite the originals |
| **Padding: Color** | Hex code for the added canvas area (default `#FFFFFF`) |

Settings are remembered automatically between runs.

## Example Configuration

The repository includes a `config.example.json` file:

```json
{
  "rootFolder": "C:/Path/To/Your/Images",
  "scanDepth": -1,
  "targetBaseNames": ["front", "back", "cover"],
  "leftSuffix": "_left",
  "rightSuffix": "_right",
  "deleteOriginal": false,
  "overwriteExisting": false,
  "debugMode": false
  "debugMode": false,
  "splitting": {
    "enabled": true,
    "targetBaseNames": ["front", "back"],
    "leftSuffix": "_left",
    "rightSuffix": "_right",
    "deleteOriginal": false,
    "overwriteExisting": false
  },
  "padding": {
    "enabled": false,
    "leftPadNames": ["00001"],
    "rightPadNames": [],
    "createNewFile": false,
    "leftSuffix": "_padded",
    "rightSuffix": "_padded",
    "overwriteExisting": false,
    "padColor": "#FFFFFF"
  }
}
```

### Configuration Reference

| Setting | Description |
|----------|-------------|
| `rootFolder` | Root directory containing folders to process |
| `scanDepth` | Folder traversal depth. `-1` = unlimited recursion, `0` = root folder only, `1` = immediate child folders, `2+` = additional levels |
| `targetBaseNames` | Image base names to process. Example: `front` matches `front.jpg`, `front.png`, etc. |
| `leftSuffix` | Suffix appended to the left output image |
| `rightSuffix` | Suffix appended to the right output image |
| `deleteOriginal` | Delete the source image after successful processing |
| `overwriteExisting` | Overwrite output files if they already exist |
| `debugMode` | Enable additional logging for troubleshooting |

---

## What Gets Created (Auto-generated)

```
(wherever splitter.exe lives)/
├── splitter.exe              ← the only file you distribute
│
│   (created automatically)
├── config.json               ← saved settings
├── Latest Report.html        ← open this to see results
├── Latest Log.log            ← for troubleshooting
└── History/
    ├── run-counter.txt
    ├── Run 001/
    │   ├── report.html
    │   ├── execution.log
    │   └── metadata.json
    ├── Run 002/
    │   └── ...
    └── ...
```

---

## Building from Source

**Requirements:**
- Go 1.21 or later
- GCC (required by Fyne for CGO)
  - Windows: [TDM-GCC](https://jmeubank.github.io/tdm-gcc/) or MSYS2
  - macOS: Xcode Command Line Tools (`xcode-select --install`)
  - Linux: `sudo apt install gcc`
- Cross-compiling to Windows from Linux/Mac also needs:
  `sudo apt install gcc-mingw-w64-x86-64`  (or `brew install mingw-w64`)

**Build:**

```bash
# Windows (run in project root)
build_windows.bat

# Linux / macOS
chmod +x build.sh
./build.sh              # current OS
./build.sh windows      # cross-compile → dist/splitter.exe
./build.sh linux        # cross-compile → dist/splitter
./build.sh darwin       # cross-compile → dist/splitter
```

`go mod tidy` runs automatically as part of the build — no manual dependency
management needed.

---

## Windows Task Scheduler (headless)

Use **Action → Start a program**:

- **Program/script**: `C:\Path\To\splitter.exe`
- **Add arguments**: `--run`
- **Start in**: `C:\Path\To`

To open the report automatically after each scheduled run, use:

- **Add arguments**: `--run --open-report`

---

## Status Values in the Report

| Status | Meaning |
|---|---|
| ✓ **Processed** | Image split successfully |
| ↷ **Already Processed** | Output files already existed (re-process option was off) |
| ⚠ **Target Image Missing** | The image filename wasn't found in this folder |
| ✕ **Error** | Something unexpected went wrong (see Details column or log) |

---

## Project Structure

```
cmd/
  splitter/
    main.go               ← entry point (dispatches GUI/headless modes)
internal/
  cli/
    options.go            ← CLI flag parsing (`--run`, `/run`, `--open-report`)
  runner/
    runner.go             ← shared processing pipeline + validation + open-file helper
  config/
    config.go             ← load/save config.json, defaults, version
  models/
    models.go             ← RunResult, FolderResult, Status
  filesystem/
    filesystem.go         ← folder discovery, run counter, path helpers
  processor/
    processor.go          ← image load / split / save (stdlib only)
  logging/
    logging.go            ← structured log writer
  report/
    report.go             ← HTML report + metadata.json
    template.go           ← self-contained HTML template
  gui/
    gui.go                ← Fyne settings window, dialogs, invokes shared runner
```

### Key design decisions

- **Single executable distribution.** Users receive only the `.exe`. Config,
  logs, reports, and history folders are all auto-created on first run.
- **RunResult is the single source of truth.** Reports and logs are both
  generated from `RunResult` — logs are never parsed to produce reports.
- **Fault tolerant at every level.** A panic in any folder's processing is
  caught by `defer/recover`. Unreadable directories during discovery are
  skipped and logged, never aborting the run.
- **No third-party dependencies except Fyne.** Image encoding via stdlib
  `image/jpeg` and `image/png`.
- **CGO_ENABLED=1 required.** Fyne uses CGO for its rendering backend.
  Pure-CGO-disabled builds are not possible with Fyne.
