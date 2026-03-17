#!/bin/bash
# Start Sliver server, create config, start listeners, drop into console.
#
# Usage:
#   ./start.sh                         # mTLS on 8888
#   ./start.sh --domain cdn.example.com # + HTTPS on 443
#   ./start.sh --fresh                  # restart clean

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
export PATH="/usr/local/go/bin:$HOME/go/bin:$PATH"

MTLS_PORT="${1:-8888}"
DOMAIN=""
FRESH=0

for arg in "$@"; do
    case "$arg" in
        --fresh) FRESH=1 ;;
        --domain) shift; DOMAIN="$1" ;;
    esac
done

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
    # Free ports
    for P in 31337 $MTLS_PORT; do
        lsof -ti :"$P" 2>/dev/null | xargs -r kill -9 2>/dev/null || true
    done
    
    echo "[*] Starting sliver-server daemon..."
    "$SCRIPT_DIR/sliver-server" daemon > /tmp/sliver-daemon.log 2>&1 &
    DAEMON_PID=$!
    disown $DAEMON_PID 2>/dev/null || true
    
    echo "[*] Waiting for daemon to initialize..."
    for i in $(seq 1 20); do
        if ss -tlnp 2>/dev/null | grep -q ":31337"; then
            echo "[+] Daemon ready (PID $DAEMON_PID, gRPC on 31337)"
            break
        fi
        sleep 1
        if [ "$i" = "20" ]; then
            echo "[-] Daemon failed to start. Check /tmp/sliver-daemon.log"
            tail -20 /tmp/sliver-daemon.log 2>/dev/null
            exit 1
        fi
    done
else
    DAEMON_PID=$EXISTING_PID
fi

# ─── Create operator config if missing ───
CFG_DIR="$HOME/.sliver-client/configs"
mkdir -p "$CFG_DIR"
if [ -z "$(ls -A "$CFG_DIR" 2>/dev/null)" ]; then
    echo "[*] Creating operator config..."
    "$SCRIPT_DIR/sliver-server" operator --name local --lhost localhost --permissions all --save "$CFG_DIR/local.cfg" 2>/dev/null
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
    echo "[*] Installing armory extensions (first run — takes a few minutes)..."
    echo -e "armory install all\nexit" | \
        timeout 600 "$SCRIPT_DIR/sliver-client" 2>/dev/null || true
    mkdir -p "$HOME/.sliver" && touch "$ARMORY_MARKER"
    echo "[+] Armory done"
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
