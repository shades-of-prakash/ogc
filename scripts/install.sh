#!/bin/sh
# ogc - POSIX-compliant installer script
# This script detects the operating system and architecture, downloads the 
# precompiled ogc binary from GitHub Releases, and places it in the bin path.
# If no releases exist or the download fails, it compiles from source if Go is installed.
#
# Usage:
#   curl -sSfL https://raw.githubusercontent.com/shades-of-prakash/ogc/master/scripts/install.sh | sh
#
# Environment variables:
#   BINDIR: Target directory for installation (defaults to /usr/local/bin or ~/.local/bin)
#   VERSION: Target release version to download (defaults to latest)

set -e

# Use colors only if outputting to a terminal
if [ -t 1 ]; then
    RED='\033[0;31m'
    GREEN='\033[0;32m'
    YELLOW='\033[0;33m'
    BLUE='\033[0;34m'
    CYAN='\033[0;36m'
    BOLD='\033[1m'
    NC='\033[0m'
else
    RED=''
    GREEN=''
    YELLOW=''
    BLUE=''
    CYAN=''
    BOLD=''
    NC=''
fi

has_cmd() {
    command -v "$1" >/dev/null 2>&1
}

check_path() {
    _dir="$1"
    case ":$PATH:" in
        *:"$_dir":*)
            ;;
        *)
            printf "\n"
            printf "${YELLOW}⚠️  Warning: %s is not in your PATH.${NC}\n" "$_dir"
            printf "To run 'ogc' directly, you may need to add it to your PATH.\n"
            printf "You can do this by adding the following line to your shell profile (e.g., ~/.bashrc, ~/.zshrc, or ~/.profile):\n"
            printf "\n"
            printf "    ${BOLD}export PATH=\"\$PATH:%s\"${NC}\n" "$_dir"
            printf "\n"
            ;;
    esac
}

download_file() {
	_url="$1"
	_dest="$2"
	if has_cmd curl; then
		curl -sLS -o "$_dest" "$_url"
	elif has_cmd wget; then
		wget -qO "$_dest" "$_url"
	else
		return 1
	fi
}

calculate_sha256() {
	_file="$1"
	if has_cmd sha256sum; then
		sha256sum "$_file" | cut -d' ' -f1
	elif has_cmd shasum; then
		shasum -a 256 "$_file" | cut -d' ' -f1
	elif has_cmd openssl; then
		openssl dgst -sha256 "$_file" | cut -d' ' -f2
	else
		return 1
	fi
}

# Detect OS
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$OS" in
    linux*)   OS="linux" ;;
    darwin*)  OS="darwin" ;;
    freebsd*) OS="freebsd" ;;
    *)
        printf "${RED}Error: Unsupported operating system: %s${NC}\n" "$(uname -s)" >&2
        exit 1
        ;;
esac

# Detect Architecture
ARCH=$(uname -m | tr '[:upper:]' '[:lower:]')
case "$ARCH" in
    x86_64|amd64)  ARCH="amd64" ;;
    arm64|aarch64) ARCH="arm64" ;;
    i386|i686|386) ARCH="386" ;;
    armv7l|armv6l|arm) ARCH="arm" ;;
    *)
        printf "${RED}Error: Unsupported architecture: %s${NC}\n" "$(uname -m)" >&2
        exit 1
        ;;
esac

# Determine installation directory
if [ -n "$BINDIR" ]; then
    INSTALL_DIR="$BINDIR"
else
    if [ -w "/usr/local/bin" ]; then
        INSTALL_DIR="/usr/local/bin"
    elif [ "$(id -u)" -eq 0 ]; then
        INSTALL_DIR="/usr/local/bin"
    else
        INSTALL_DIR="$HOME/.local/bin"
    fi
fi

# Ensure target directory exists and is writable
if [ ! -d "$INSTALL_DIR" ]; then
    printf "${YELLOW}Creating installation directory: %s${NC}\n" "$INSTALL_DIR"
    mkdir -p "$INSTALL_DIR" || {
        printf "${RED}Error: Failed to create directory %s. Try running with sudo or setting BINDIR.${NC}\n" "$INSTALL_DIR" >&2
        exit 1
    }
fi

if [ ! -w "$INSTALL_DIR" ]; then
    printf "${RED}Error: Directory %s is not writable. Try running with sudo or setting BINDIR.${NC}\n" "$INSTALL_DIR" >&2
    exit 1
fi

REPO="shades-of-prakash/ogc"
TAG="${VERSION:-}"

# Check github for latest release if not specified
if [ -z "$TAG" ]; then
    printf "${BLUE}Checking GitHub for the latest release of ogc...${NC}\n"
    
    LATEST_RELEASE=""
    if has_cmd curl; then
        LATEST_RELEASE=$(curl -sLS "https://api.github.com/repos/${REPO}/releases/latest" 2>/dev/null || true)
    elif has_cmd wget; then
        LATEST_RELEASE=$(wget -qO- "https://api.github.com/repos/${REPO}/releases/latest" 2>/dev/null || true)
    fi

    if [ -n "$LATEST_RELEASE" ]; then
        # Parse "tag_name": "vX.Y.Z"
        TAG=$(printf "%s" "$LATEST_RELEASE" | grep '"tag_name":' | cut -d'"' -f4 || true)
    fi
fi

# Fallback: install from source if no releases found or API limit hit
if [ -z "$TAG" ]; then
    printf "${YELLOW}No precompiled releases found on GitHub (or GitHub API rate limited).${NC}\n"
    if has_cmd go; then
        printf "${BLUE}Go is installed! Attempting to install ogc from source...${NC}\n"
        if GOBIN="$INSTALL_DIR" go install "github.com/${REPO}@latest"; then
            printf "${GREEN}Successfully installed ogc from source to %s/ogc!${NC}\n" "$INSTALL_DIR"
            check_path "$INSTALL_DIR"
            exit 0
        else
            printf "${RED}Error: Failed to build and install from source using go install.${NC}\n" >&2
            exit 1
        fi
    else
        printf "${RED}Error: No precompiled releases found and Go is not installed to compile from source.${NC}\n" >&2
        printf "${RED}Please install Go (https://go.dev) or wait for a release tag on GitHub.${NC}\n" >&2
        exit 1
    fi
fi

BINARY_NAME="ogc"
DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${TAG}/ogc-${OS}-${ARCH}"
CHECKSUM_URL="https://github.com/${REPO}/releases/download/${TAG}/checksums.txt"

printf "${BLUE}Downloading ogc %s for %s-%s...${NC}\n" "$TAG" "$OS" "$ARCH"

# Create a secure temporary directory
TMP_DIR=$(mktemp -d 2>/dev/null || mktemp -d -t 'ogc' 2>/dev/null || (mkdir -p "./.ogc-tmp" && echo "./.ogc-tmp"))
TMP_BIN="${TMP_DIR}/${BINARY_NAME}"
TMP_SUMS="${TMP_DIR}/checksums.txt"

if download_file "$DOWNLOAD_URL" "$TMP_BIN" && download_file "$CHECKSUM_URL" "$TMP_SUMS"; then
	if [ -s "$TMP_BIN" ] && [ -s "$TMP_SUMS" ]; then
		# Perform checksum verification
		EXPECTED_SUM=$(grep "ogc-${OS}-${ARCH}$" "$TMP_SUMS" | cut -d' ' -f1)
		if [ -z "$EXPECTED_SUM" ]; then
			printf "${RED}Error: Checksum for ogc-${OS}-${ARCH} not found in release checksums.${NC}\n" >&2
			rm -rf "$TMP_DIR"
			exit 1
		fi

		ACTUAL_SUM=$(calculate_sha256 "$TMP_BIN" || true)
		if [ -z "$ACTUAL_SUM" ]; then
			printf "${YELLOW}Warning: Could not calculate checksum (sha256sum, shasum, or openssl not found).${NC}\n"
			printf "${YELLOW}Proceeding with caution...${NC}\n"
		elif [ "$EXPECTED_SUM" != "$ACTUAL_SUM" ]; then
			printf "${RED}Error: Checksum verification failed!${NC}\n" >&2
			printf "Expected: %s\n" "$EXPECTED_SUM" >&2
			printf "Actual:   %s\n" "$ACTUAL_SUM" >&2
			printf "${RED}The downloaded binary might be corrupted or tampered with.${NC}\n" >&2
			rm -rf "$TMP_DIR"
			exit 1
		else
			printf "${GREEN}✓ Checksum verified successfully!${NC}\n"
		fi

		mv "$TMP_BIN" "${INSTALL_DIR}/${BINARY_NAME}"
		chmod +x "${INSTALL_DIR}/${BINARY_NAME}"
		rm -rf "$TMP_DIR"
		printf "${GREEN}Successfully installed ogc %s to %s/${BINARY_NAME}!${NC}\n" "$TAG" "$INSTALL_DIR"
		check_path "$INSTALL_DIR"
		exit 0
	fi
fi

# Clean up temp dir if download/checksum failed
rm -rf "$TMP_DIR"

printf "${YELLOW}Binary download failed (URL: %s). Checking if Go is installed to compile from source...${NC}\n" "$DOWNLOAD_URL"
if has_cmd go; then
    printf "${BLUE}Go is installed! Installing from source...${NC}\n"
    if GOBIN="$INSTALL_DIR" go install "github.com/${REPO}@latest"; then
        printf "${GREEN}Successfully installed ogc from source to %s/${BINARY_NAME}!${NC}\n" "$INSTALL_DIR"
        check_path "$INSTALL_DIR"
        exit 0
    else
        printf "${RED}Error: Failed to install from source using go install.${NC}\n" >&2
        exit 1
    fi
else
    printf "${RED}Error: Failed to download binary and Go is not installed to compile from source.${NC}\n" >&2
    exit 1
fi
