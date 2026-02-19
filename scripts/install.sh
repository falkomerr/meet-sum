#!/usr/bin/env bash
# meet-sum installer
# Usage: curl -fsSL https://raw.githubusercontent.com/falkomerr/meet-sum/main/scripts/install.sh | bash
# Override version:     VERSION=v1.2.3 bash install.sh
# Override install dir: INSTALL_DIR=/custom/path bash install.sh
set -euo pipefail

# ---------------------------------------------------------------------------
# Color setup (only when stdout is a TTY)
# ---------------------------------------------------------------------------
setup_colors() {
    if [[ -t 1 ]]; then
        RED=$(printf '\033[31m')
        GREEN=$(printf '\033[32m')
        YELLOW=$(printf '\033[33m')
        BLUE=$(printf '\033[34m')
        CYAN=$(printf '\033[36m')
        BOLD=$(printf '\033[1m')
        DIM=$(printf '\033[2m')
        RESET=$(printf '\033[0m')
    else
        RED="" GREEN="" YELLOW="" BLUE="" CYAN="" BOLD="" DIM="" RESET=""
    fi
}

# ---------------------------------------------------------------------------
# Output helpers
# ---------------------------------------------------------------------------
info()    { printf "${BLUE}  ●${RESET} %s\n" "$*"; }
success() { printf "${GREEN}  ✓${RESET} %s\n" "$*"; }
warn()    { printf "${YELLOW}  ⚠${RESET} %s\n" "$*" >&2; }
error()   { printf "${RED}  ✗${RESET} %s\n" "$*" >&2; }
step()    { printf "\n${BOLD}${CYAN}▶ %s${RESET}\n" "$*"; }

# ---------------------------------------------------------------------------
# Banner
# ---------------------------------------------------------------------------
print_banner() {
    printf "\n"
    printf "${CYAN}  ███╗   ███╗███████╗███████╗████████╗      ███████╗██╗   ██╗███╗   ███╗${RESET}\n"
    printf "${CYAN}  ████╗ ████║██╔════╝██╔════╝╚══██╔══╝      ██╔════╝██║   ██║████╗ ████║${RESET}\n"
    printf "${CYAN}  ██╔████╔██║█████╗  █████╗     ██║   █████╗███████╗██║   ██║██╔████╔██║${RESET}\n"
    printf "${CYAN}  ██║╚██╔╝██║██╔══╝  ██╔══╝     ██║   ╚════╝╚════██║██║   ██║██║╚██╔╝██║${RESET}\n"
    printf "${CYAN}  ██║ ╚═╝ ██║███████╗███████╗   ██║         ███████║╚██████╔╝██║ ╚═╝ ██║${RESET}\n"
    printf "${CYAN}  ╚═╝     ╚═╝╚══════╝╚══════╝   ╚═╝         ╚══════╝ ╚═════╝ ╚═╝     ╚═╝${RESET}\n"
    printf "\n"
    printf "  ${DIM}Meeting Recorder & Summarizer — Local AI, No Cloud${RESET}\n"
    printf "\n"
}

# ---------------------------------------------------------------------------
# Variables
# ---------------------------------------------------------------------------
REPO="falkomerr/meet-sum"
BINARY_NAME="meet-sum"
VERSION="${VERSION:-}"
INSTALL_DIR="${INSTALL_DIR:-}"
OS=""
ARCH=""
PLATFORM=""

# ---------------------------------------------------------------------------
# Detect platform
# ---------------------------------------------------------------------------
detect_platform() {
    step "Detecting platform"

    OS=$(uname -s)
    ARCH=$(uname -m)

    case "$OS" in
        Darwin)
            case "$ARCH" in
                x86_64) PLATFORM="darwin_amd64" ;;
                arm64)  PLATFORM="darwin_arm64" ;;
                *)      error "Unsupported macOS arch: $ARCH"; exit 1 ;;
            esac
            ;;
        Linux)
            case "$ARCH" in
                x86_64)  PLATFORM="linux_amd64" ;;
                aarch64) PLATFORM="linux_arm64" ;;
                *)       error "Unsupported Linux arch: $ARCH"; exit 1 ;;
            esac
            ;;
        *)
            error "Unsupported OS: $OS"
            error "Windows users: download manually from https://github.com/${REPO}/releases"
            exit 1
            ;;
    esac

    success "Detected: $OS $ARCH → $PLATFORM"
}

# ---------------------------------------------------------------------------
# Fetch latest version from GitHub API
# ---------------------------------------------------------------------------
get_latest_version() {
    if [[ -n "$VERSION" ]]; then
        success "Using specified version: $VERSION"
        return
    fi

    info "Fetching latest release info..."

    if ! VERSION=$(curl -sf "https://api.github.com/repos/${REPO}/releases/latest" \
                    | grep '"tag_name"' \
                    | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/'); then
        error "Failed to fetch latest version. Check your internet connection."
        exit 1
    fi

    if [[ -z "$VERSION" ]]; then
        error "Could not parse version from GitHub API response."
        exit 1
    fi

    success "Latest version: $VERSION"
}

# ---------------------------------------------------------------------------
# Detect install directory
# ---------------------------------------------------------------------------
detect_install_dir() {
    step "Selecting install directory"

    if [[ -n "$INSTALL_DIR" ]]; then
        mkdir -p "$INSTALL_DIR"
        success "Using specified directory: $INSTALL_DIR"
        return
    fi

    if [[ -w "/usr/local/bin" ]]; then
        INSTALL_DIR="/usr/local/bin"
        success "Using /usr/local/bin (writable, no sudo needed)"
    else
        INSTALL_DIR="$HOME/.local/bin"
        mkdir -p "$INSTALL_DIR"
        success "Using ~/.local/bin"
    fi
}

# ---------------------------------------------------------------------------
# Check runtime dependencies with install hints
# ---------------------------------------------------------------------------
check_dependencies() {
    step "Checking dependencies"
    local missing=0

    # ffmpeg
    if command -v ffmpeg >/dev/null 2>&1; then
        success "ffmpeg found ($(ffmpeg -version 2>&1 | head -1 | cut -d' ' -f3))"
    else
        warn "ffmpeg not found"
        if [[ "$OS" == "Darwin" ]]; then
            printf "      ${DIM}→ Install: brew install ffmpeg${RESET}\n"
        else
            printf "      ${DIM}→ Install: sudo apt install ffmpeg${RESET}\n"
        fi
        missing=1
    fi

    # python3
    if command -v python3 >/dev/null 2>&1; then
        success "python3 found ($(python3 --version 2>&1))"
    else
        warn "python3 not found"
        if [[ "$OS" == "Darwin" ]]; then
            printf "      ${DIM}→ Install: brew install python${RESET}\n"
        else
            printf "      ${DIM}→ Install: sudo apt install python3${RESET}\n"
        fi
        missing=1
    fi

    # openai-whisper
    if python3 -c "import whisper" 2>/dev/null; then
        success "openai-whisper found"
    else
        warn "openai-whisper not found"
        printf "      ${DIM}→ Install: pip install openai-whisper${RESET}\n"
        missing=1
    fi

    # ollama
    if command -v ollama >/dev/null 2>&1; then
        success "ollama found"
    else
        warn "ollama not found"
        printf "      ${DIM}→ Install: https://ollama.ai${RESET}\n"
        missing=1
    fi

    if [[ $missing -eq 1 ]]; then
        printf "\n${YELLOW}  Some dependencies are missing. Install them and re-run: meet-sum init${RESET}\n"
    fi
}

# ---------------------------------------------------------------------------
# Download, verify checksum, extract, and install binary
# ---------------------------------------------------------------------------
download_binary() {
    step "Downloading meet-sum $VERSION"

    local archive="${BINARY_NAME}_${VERSION}_${PLATFORM}.tar.gz"
    local url="https://github.com/${REPO}/releases/download/${VERSION}/${archive}"
    local checksum_url="${url}.sha256"
    local tmpdir
    tmpdir=$(mktemp -d)

    # Cleanup temp dir on exit (success or failure)
    # shellcheck disable=SC2064
    trap "rm -rf '${tmpdir}'" EXIT

    info "URL: $url"

    # Download binary archive
    if ! curl --fail --location --progress-bar \
              --output "${tmpdir}/${archive}" "$url"; then
        error "Download failed. Check that version $VERSION exists:"
        error "  https://github.com/${REPO}/releases"
        exit 1
    fi

    success "Download complete"

    # Download and verify SHA256 checksum
    if curl -sf --output "${tmpdir}/${archive}.sha256" "$checksum_url"; then
        info "Verifying checksum..."
        local ok=0
        pushd "$tmpdir" >/dev/null
        if sha256sum -c "${archive}.sha256" >/dev/null 2>&1; then
            ok=1
        elif shasum -a 256 -c "${archive}.sha256" >/dev/null 2>&1; then
            ok=1
        fi
        popd >/dev/null

        if [[ $ok -eq 1 ]]; then
            success "Checksum verified"
        else
            error "Checksum verification failed — download may be corrupted."
            exit 1
        fi
    else
        warn "Checksum file not available — skipping verification"
    fi

    # Extract archive
    info "Extracting..."
    tar -xzf "${tmpdir}/${archive}" -C "$tmpdir"

    # Install binary
    cp "${tmpdir}/${BINARY_NAME}" "${INSTALL_DIR}/${BINARY_NAME}"
    chmod +x "${INSTALL_DIR}/${BINARY_NAME}"

    success "Installed to ${INSTALL_DIR}/${BINARY_NAME}"
}

# ---------------------------------------------------------------------------
# Ensure install dir is in PATH
# ---------------------------------------------------------------------------
ensure_in_path() {
    step "Checking PATH"

    if echo "$PATH" | tr ':' '\n' | grep -qx "$INSTALL_DIR"; then
        success "$INSTALL_DIR is already in PATH"
        return
    fi

    warn "$INSTALL_DIR is not in PATH"

    local shell_rc=""
    if [[ "$SHELL" == */zsh ]]; then
        shell_rc="$HOME/.zshrc"
    elif [[ "$SHELL" == */bash ]]; then
        shell_rc="$HOME/.bashrc"
    fi

    if [[ -n "$shell_rc" ]]; then
        {
            echo ""
            echo "# meet-sum — added by installer"
            echo "export PATH=\"\$PATH:${INSTALL_DIR}\""
        } >> "$shell_rc"
        success "Added to $shell_rc"
        printf "  ${DIM}  Run: source %s${RESET}\n" "$shell_rc"
    else
        printf "\n"
        printf "  ${YELLOW}Add this to your shell config manually:${RESET}\n"
        printf "  ${CYAN}export PATH=\"\$PATH:%s\"${RESET}\n" "$INSTALL_DIR"
    fi
}

# ---------------------------------------------------------------------------
# Verify the installed binary runs
# ---------------------------------------------------------------------------
verify_installation() {
    step "Verifying installation"

    if "${INSTALL_DIR}/${BINARY_NAME}" --version >/dev/null 2>&1 || \
       "${INSTALL_DIR}/${BINARY_NAME}" --help >/dev/null 2>&1; then
        success "Binary works correctly"
    else
        warn "Binary installed but could not verify — it may need dependencies first"
    fi
}

# ---------------------------------------------------------------------------
# Final success message
# ---------------------------------------------------------------------------
print_success() {
    printf "\n"
    printf "${GREEN}${BOLD}  ╔══════════════════════════════════════════╗${RESET}\n"
    printf "${GREEN}${BOLD}  ║   meet-sum installed successfully! 🎉   ║${RESET}\n"
    printf "${GREEN}${BOLD}  ╚══════════════════════════════════════════╝${RESET}\n"
    printf "\n"
    printf "  ${BOLD}Get started:${RESET}\n"
    printf "\n"
    printf "    ${CYAN}meet-sum init${RESET}        # Configure Whisper model & directories\n"
    printf "    ${CYAN}meet-sum record${RESET}      # Start recording a meeting\n"
    printf "    ${CYAN}meet-sum summarize${RESET}   # Transcribe & summarize\n"
    printf "\n"
    printf "  ${DIM}Documentation: https://github.com/%s${RESET}\n" "$REPO"
    printf "\n"
}

# ---------------------------------------------------------------------------
# Entry point
# ---------------------------------------------------------------------------
main() {
    setup_colors
    print_banner

    step "Installing meet-sum"

    detect_platform
    get_latest_version
    detect_install_dir
    check_dependencies
    download_binary
    ensure_in_path
    verify_installation
    print_success
}

main "$@"
