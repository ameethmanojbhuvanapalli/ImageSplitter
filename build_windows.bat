@echo off
REM ─────────────────────────────────────────────────────────
REM  Image Splitter — Windows build script
REM  Requirements: Go 1.21+, TDM-GCC 64-bit
REM  Run from the project root (where go.mod lives).
REM ─────────────────────────────────────────────────────────

echo [1/4] Downloading dependencies...
go mod tidy
if %ERRORLEVEL% NEQ 0 (echo FAILED: go mod tidy & exit /b 1)

echo [2/4] Embedding exe icon...
where rsrc >nul 2>&1
if %ERRORLEVEL% NEQ 0 (
    echo   rsrc not found — installing...
    go install github.com/akavel/rsrc@latest
    if %ERRORLEVEL% NEQ 0 (echo FAILED: could not install rsrc & exit /b 1)
)

if exist icon.ico (
    rsrc -ico icon.ico -o cmd\splitter\rsrc.syso
    if %ERRORLEVEL% NEQ 0 (echo FAILED: rsrc icon embed & exit /b 1)
    echo   icon.ico embedded successfully.
) else (
    echo   icon.ico not found — skipping exe icon.
    echo   To add an exe icon: place icon.ico in the project root and rebuild.
)

echo [3/4] Building...
set GOOS=windows
set GOARCH=amd64
set CGO_ENABLED=1

go build -ldflags="-s -w -H windowsgui" -o splitter.exe .\cmd\splitter
if %ERRORLEVEL% NEQ 0 (echo FAILED: build & exit /b 1)

echo [4/4] Done.
echo.
echo   Output:  splitter.exe
echo   This is the only file your users need.
echo   Everything else is created automatically on first run.
echo.
