#!/usr/bin/env bash
set -e

OUTPUT="dist"
mkdir -p "$OUTPUT"

echo "Building scan-projects for multiple platforms into $OUTPUT/"

# Linux
GOOS=linux GOARCH=amd64 go build -o "$OUTPUT/scan-projects-linux-amd64" ./cmd/scan-projects
GOOS=linux GOARCH=arm64 go build -o "$OUTPUT/scan-projects-linux-arm64" ./cmd/scan-projects

# macOS
GOOS=darwin GOARCH=amd64 go build -o "$OUTPUT/scan-projects-macos-amd64" ./cmd/scan-projects
GOOS=darwin GOARCH=arm64 go build -o "$OUTPUT/scan-projects-macos-arm64" ./cmd/scan-projects

# Windows
GOOS=windows GOARCH=amd64 go build -o "$OUTPUT/scan-projects-windows-amd64.exe" ./cmd/scan-projects
GOOS=windows GOARCH=arm64 go build -o "$OUTPUT/scan-projects-windows-arm64.exe" ./cmd/scan-projects

echo "✔ Done."
