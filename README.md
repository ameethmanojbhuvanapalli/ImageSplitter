# Image Splitter

A cross-platform desktop app that vertically splits images across a folder
hierarchy. Built with Go + Fyne. Distributed as a single executable — no
installation required.

---

## For End Users

1. Double-click `splitter.exe` (or `splitter` on Mac/Linux)
2. Fill in the settings form
3. Click **Run**
4. The report opens automatically in your browser

That's it. No config files to edit, no terminal, no setup.

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

Settings are remembered automatically between runs.

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
    main.go               ← entry point (boots GUI)
internal/
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
    gui.go                ← Fyne settings window, dialogs, run flow
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
