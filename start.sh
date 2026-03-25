#!/bin/bash
# Start Sliver server, create config, start listeners, drop into console.
#
# Usage:
#   ./start.sh                              # mTLS on 8888
#   ./start.sh --mtls-port 9999            # mTLS on custom port
#   ./start.sh --domain cdn.example.com    # + HTTPS on 443
#   ./start.sh --fresh                     # restart clean
#   ./start.sh --reinstall-armory          # force armory reinstall

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
export PATH="/usr/local/go/bin:$HOME/go/bin:$PATH"

MTLS_PORT="8888"
DOMAIN=""
FRESH=0
REINSTALL_ARMORY=0

while [[ $# -gt 0 ]]; do
    case "$1" in
        --fresh)             FRESH=1 ; shift ;;
        --domain)            DOMAIN="$2" ; shift 2 ;;
        --reinstall-armory)  REINSTALL_ARMORY=1 ; shift ;;
        --mtls-port)         MTLS_PORT="$2" ; shift 2 ;;
        *)                   echo "[!] Unknown argument: $1 (ignored)" ; shift ;;
    esac
done

# Handle --reinstall-armory
if [ "$REINSTALL_ARMORY" = "1" ]; then
    rm -f "$HOME/.sliver/.armory_installed" 2>/dev/null || true
fi

echo ""
echo "════════════════════════════════════════════════"
echo "  mgstate/sliver — start"
echo "════════════════════════════════════════════════"

# ─── Stop existing if --fresh or binary is newer ───
EXISTING_PID=$(pgrep -f "sliver-server" 2>/dev/null | head -1 || true)
if [ -n "$EXISTING_PID" ]; then
    if [ "$FRESH" = "1" ]; then
        echo "[*] --fresh: stopping existing server (PID $EXISTING_PID)..."
        pgrep -f "sliver-server|sliver-client" 2>/dev/null | xargs -r kill 2>/dev/null || true
        sleep 3
        pgrep -f "sliver-server" 2>/dev/null | xargs -r kill -9 2>/dev/null || true
        sleep 1
        EXISTING_PID=""
    else
        # Check if binary is newer
        BIN_TIME=$(stat -c %Y "$SCRIPT_DIR/sliver-server" 2>/dev/null || echo 0)
        PROC_TIME=$(stat -c %Y "/proc/$EXISTING_PID" 2>/dev/null || echo 0)
        if [ "$BIN_TIME" -gt "$PROC_TIME" ] 2>/dev/null; then
            echo "[!] Binary is newer than running daemon — restarting..."
            pgrep -f "sliver-server|sliver-client" 2>/dev/null | xargs -r kill 2>/dev/null || true
            sleep 3
            EXISTING_PID=""
        else
            echo "[+] Server already running (PID $EXISTING_PID)"
        fi
    fi
fi

# ─── Start daemon if needed ───
if [ -z "$EXISTING_PID" ] || ! pgrep -f "sliver-server" >/dev/null 2>&1; then
    # Verify binary exists
    if [ ! -x "$SCRIPT_DIR/sliver-server" ]; then
        echo "[-] sliver-server binary not found at $SCRIPT_DIR/sliver-server"
        echo "    Run setup.sh first to build it."
        exit 1
    fi

    # Free ports
    for P in 443 31337 $MTLS_PORT; do
        PIDS=$(lsof -ti :"$P" 2>/dev/null || true)
        if [ -n "$PIDS" ]; then
            echo "[*] Freeing port $P..."
            echo "$PIDS" | xargs -r kill -9 2>/dev/null || true
        fi
    done

    echo "[*] Starting sliver-server daemon..."
    # --disable-wg: use plain mTLS TCP on port 31337 instead of the default
    # WireGuard-wrapped mode.  WireGuard wraps the listener inside a userspace
    # network stack, making port 31337 invisible to `ss` (breaking the readiness
    # check) and unreachable by sliver-client ("context deadline exceeded").
    "$SCRIPT_DIR/sliver-server" daemon --disable-wg > /tmp/sliver-daemon.log 2>&1 &
    DAEMON_PID=$!
    disown $DAEMON_PID 2>/dev/null || true

    # First run unpacks Go/Zig toolchains + generates certs/keys — can take
    # several minutes.  Detect first-run by checking for the asset-version file
    # that the server writes after unpacking; extend the wait accordingly.
    ASSET_VERSION_FILE="$HOME/.sliver/version"
    if [ ! -f "$ASSET_VERSION_FILE" ]; then
        WAIT_SECS=600
        echo "[*] First run detected — unpacking assets + generating certs (up to ${WAIT_SECS} s)..."
        echo "    This is a one-time step; subsequent starts will be much faster."
    else
        WAIT_SECS=90
        echo "[*] Waiting for daemon to initialize (up to ${WAIT_SECS} s)..."
    fi

    READY=0
    for i in $(seq 1 "$WAIT_SECS"); do
        # Fail fast: if the process already exited, no point waiting further.
        if ! kill -0 "$DAEMON_PID" 2>/dev/null; then
            echo ""
            echo "[-] Daemon process (PID $DAEMON_PID) exited before gRPC port 31337 opened."
            echo "    Last log output:"
            tail -30 /tmp/sliver-daemon.log 2>/dev/null
            exit 1
        fi
        if ss -tlnp 2>/dev/null | grep -q ":31337"; then
            echo ""
            echo "[+] Daemon ready (PID $DAEMON_PID, gRPC on 31337)"
            READY=1
            break
        fi
        # Print a progress dot every 10 seconds so the operator sees activity.
        if [ $(( i % 10 )) -eq 0 ]; then
            printf "."
        fi
        sleep 1
    done
    if [ "$READY" = "0" ]; then
        echo ""
        echo "[-] Daemon failed to start after ${WAIT_SECS} s. Check /tmp/sliver-daemon.log"
        tail -30 /tmp/sliver-daemon.log 2>/dev/null
        exit 1
    fi
else
    DAEMON_PID=$EXISTING_PID
fi

# ─── Create operator config if missing ───
CFG_DIR="$HOME/.sliver-client/configs"
mkdir -p "$CFG_DIR"
if [ -z "$(ls -A "$CFG_DIR" 2>/dev/null)" ]; then
    echo "[*] Creating operator config..."
    "$SCRIPT_DIR/sliver-server" operator --name local --lhost localhost --disable-wg --permissions all --save "$CFG_DIR/local.cfg" 2>/dev/null
    if [ -f "$CFG_DIR/local.cfg" ]; then
        echo "[+] Config created"
    else
        echo "[-] Config creation failed"
        exit 1
    fi
fi

# ─── Import C2 profile (first run) ───
PROFILE_MARKER="$HOME/.sliver/.profile_imported"
PROFILE="$SCRIPT_DIR/opsec-profiles/microsoft365-c2.json"
if [ ! -f "$PROFILE_MARKER" ] && [ -f "$PROFILE" ]; then
    echo "[*] Importing microsoft365 C2 profile..."
    echo -e "c2profiles import -n microsoft365 -f $PROFILE\nexit" | \
        timeout 30 "$SCRIPT_DIR/sliver-client" 2>/dev/null || true
    mkdir -p "$HOME/.sliver" && touch "$PROFILE_MARKER"
    echo "[+] Profile imported"
fi

# ─── Install armory (first run) ───
ARMORY_MARKER="$HOME/.sliver/.armory_installed"
if [ ! -f "$ARMORY_MARKER" ]; then
    echo "[*] Installing armory extensions (first run — targeted packages only)..."

    # Only what ATTACKPATH.md actually uses — skip the other 40+ packages
    ARMORY_PKGS=(
        # Core credential tools
        rubeus mimikatz nanodump sharpsecdump
        # AMSI/ETW bypass BOFs
        inject-amsi-bypass inject-etw-bypass unhook-bof
        # Situational awareness BOFs (sa-* family)
        sa-whoami sa-ipconfig sa-arp sa-netstat
        sa-ldapsearch sa-netgroup sa-netlocalgroup
        sa-driversigs sa-enum-filter-driver
        # Lateral movement
        sharpwmi
        # Credential access
        sharpdpapi sharpchrome credman
    )

    # Prefetch armory index (all parallel jobs share the cached index)
    echo "[*] Fetching armory index..."
    echo -e "armory\nexit" | timeout 60 "$SCRIPT_DIR/sliver-client" >/dev/null 2>&1 || true

    TOTAL=${#ARMORY_PKGS[@]}
    MAX_PARALLEL=4
    INSTALLED=0
    FAILED=()
    pids=()
    pkg_names=()
    BATCH=0

    for pkg in "${ARMORY_PKGS[@]}"; do
        echo -e "armory install $pkg\nexit" | \
            timeout 90 "$SCRIPT_DIR/sliver-client" >/dev/null 2>&1 &
        pids+=($!)
        pkg_names+=("$pkg")

        # Drain when we hit MAX_PARALLEL
        if [ ${#pids[@]} -ge $MAX_PARALLEL ]; then
            BATCH=$((BATCH + MAX_PARALLEL))
            echo "[*] Installing batch... ($BATCH/$TOTAL)"
            for i in "${!pids[@]}"; do
                wait "${pids[$i]}" 2>/dev/null && \
                    INSTALLED=$((INSTALLED + 1)) || \
                    FAILED+=("${pkg_names[$i]}")
            done
            pids=()
            pkg_names=()
        fi
    done
    # Drain remaining
    if [ ${#pids[@]} -gt 0 ]; then
        echo "[*] Installing final batch... ($TOTAL/$TOTAL)"
        for i in "${!pids[@]}"; do
            wait "${pids[$i]}" 2>/dev/null && \
                INSTALLED=$((INSTALLED + 1)) || \
                FAILED+=("${pkg_names[$i]}")
        done
    fi

    echo "[+] Armory: $INSTALLED/$TOTAL installed"
    if [ ${#FAILED[@]} -gt 0 ]; then
        echo "[!] Failed (install manually): ${FAILED[*]}"
    fi

    mkdir -p "$HOME/.sliver" && touch "$ARMORY_MARKER"
    echo "[+] Armory done (remove $ARMORY_MARKER to reinstall)"
fi

# ─── Start mTLS listener ───
echo "[*] Starting mTLS on 0.0.0.0:$MTLS_PORT..."
echo -e "mtls --lhost 0.0.0.0 --lport $MTLS_PORT\nexit" | \
    timeout 15 "$SCRIPT_DIR/sliver-client" 2>/dev/null || true

if [ -n "$DOMAIN" ]; then
    echo "[*] Starting HTTPS on 0.0.0.0:443 ($DOMAIN)..."
    echo -e "https --lhost 0.0.0.0 --lport 443 -d $DOMAIN\nexit" | \
        timeout 15 "$SCRIPT_DIR/sliver-client" 2>/dev/null || true
fi

echo ""
echo "════════════════════════════════════════════════"
echo "  Sliver ready!"
echo "  Daemon PID: $DAEMON_PID"
echo "  mTLS: 0.0.0.0:$MTLS_PORT"
[ -n "$DOMAIN" ] && echo "  HTTPS: 0.0.0.0:443 ($DOMAIN)"
echo ""
echo "  Dropping into console..."
echo "════════════════════════════════════════════════"
echo ""
exec "$SCRIPT_DIR/sliver-client"
