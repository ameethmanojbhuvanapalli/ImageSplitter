@echo off
REM ─────────────────────────────────────────────────────────
REM  Image Splitter — Windows build script
REM  Run from the project root (where go.mod lives).
REM  Requirements: Go 1.21+, TDM-GCC 64-bit
REM ─────────────────────────────────────────────────────────

echo [1/3] Downloading dependencies...
go mod tidy
if %ERRORLEVEL% NEQ 0 (echo FAILED: go mod tidy & exit /b 1)

echo [2/3] Building...
set GOOS=windows
set GOARCH=amd64
set CGO_ENABLED=1

REM -H windowsgui hides the console window — users never see a terminal.
go build -ldflags="-s -w -H windowsgui" -o splitter.exe .\cmd\splitter
if %ERRORLEVEL% NEQ 0 (echo FAILED: build & exit /b 1)

echo [3/3] Done.
echo.
echo   Output:  splitter.exe
echo   This is the only file your users need.
echo   Everything else is created automatically on first run.
echo.
