#!/bin/sh
# Helper script to cross-compile ogc binaries locally for release

set -e

# Target directory for built binaries
BUILD_DIR="./build"
mkdir -p "$BUILD_DIR"

printf "Building ogc binaries...\n"

# Linux
printf "  - linux/amd64... "
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o "$BUILD_DIR/ogc-linux-amd64"
printf "Done\n"

printf "  - linux/arm64... "
GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o "$BUILD_DIR/ogc-linux-arm64"
printf "Done\n"

printf "  - linux/386...   "
GOOS=linux GOARCH=386 go build -ldflags="-s -w" -o "$BUILD_DIR/ogc-linux-386"
printf "Done\n"

printf "  - linux/arm...   "
GOOS=linux GOARCH=arm go build -ldflags="-s -w" -o "$BUILD_DIR/ogc-linux-arm"
printf "Done\n"

# macOS (Darwin)
printf "  - darwin/amd64... "
GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o "$BUILD_DIR/ogc-darwin-amd64"
printf "Done\n"

printf "  - darwin/arm64... "
GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o "$BUILD_DIR/ogc-darwin-arm64"
printf "Done\n"

# FreeBSD
printf "  - freebsd/amd64... "
GOOS=freebsd GOARCH=amd64 go build -ldflags="-s -w" -o "$BUILD_DIR/ogc-freebsd-amd64"
printf "Done\n"

printf "\nBinaries successfully built in %s/\n" "$BUILD_DIR"
ls -lh "$BUILD_DIR"
