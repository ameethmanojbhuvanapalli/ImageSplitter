#!/usr/bin/env bash
# ─────────────────────────────────────────────────────────────
#  Image Splitter — build script
#  Usage:
#    ./build.sh                 # build for current OS
#    ./build.sh windows         # cross-compile → splitter.exe
#    ./build.sh linux           # cross-compile → splitter (Linux)
#    ./build.sh darwin          # cross-compile → splitter (macOS)
#
#  Requirements: Go 1.21+, gcc (for Fyne's CGO dependency)
#  Cross-compiling to Windows also requires mingw-w64:
#    Ubuntu/Debian: sudo apt install gcc-mingw-w64-x86-64
#    macOS:         brew install mingw-w64
# ─────────────────────────────────────────────────────────────
set -euo pipefail

TARGET="${1:-$(go env GOOS)}"
ARCH="amd64"
mkdir -p dist

echo "[1/3] Downloading dependencies..."
go mod tidy

echo "[2/3] Building for $TARGET/$ARCH..."

case "$TARGET" in
  windows)
    OUT="dist/splitter.exe"
    # -H windowsgui suppresses the console window on Windows.
    GOOS=windows GOARCH=$ARCH CGO_ENABLED=1 \
      CC=x86_64-w64-mingw32-gcc \
      go build -ldflags="-s -w -H windowsgui" -o "$OUT" ./cmd/splitter
    ;;
  linux)
    OUT="dist/splitter"
    GOOS=linux GOARCH=$ARCH CGO_ENABLED=1 \
      go build -ldflags="-s -w" -o "$OUT" ./cmd/splitter
    ;;
  darwin)
    OUT="dist/splitter"
    GOOS=darwin GOARCH=$ARCH CGO_ENABLED=1 \
      go build -ldflags="-s -w" -o "$OUT" ./cmd/splitter
    ;;
  *)
    echo "Unknown target: $TARGET. Use windows, linux, or darwin."
    exit 1
    ;;
esac

echo "[3/3] Done."
echo ""
echo "  Output: $OUT"
echo "  Send only this file to your users."
echo "  Everything else is auto-created on first run."
