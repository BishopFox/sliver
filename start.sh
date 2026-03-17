#!/bin/bash
# Start Sliver server with auto-listener setup.
# Reuses existing daemon if running and healthy. Starts listeners, drops into client.
#
# Usage:
#   ./start.sh                                    # mTLS on 8888
#   ./start.sh --domain cdn.yourdomain.com        # mTLS on 8888 + HTTPS on 443
#   ./start.sh --mtls-port 9999                   # custom mTLS port
#   ./start.sh --fresh                            # restart clean

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
export PATH="/usr/local/go/bin:$HOME/go/bin:$PATH"

MTLS_PORT=8888
HTTPS_PORT=443
GRPC_PORT=31337
DOMAIN=""
FRESH=0

while [[ $# -gt 0 ]]; do
    case $1 in
        --domain|-d) DOMAIN="$2"; shift 2 ;;
        --mtls-port) MTLS_PORT="$2"; shift 2 ;;
        --https-port) HTTPS_PORT="$2"; shift 2 ;;
        --fresh) FRESH=1; shift ;;
        -h|--help)
            echo "Usage: $0 [--domain DOMAIN] [--mtls-port PORT] [--fresh]"
            exit 0 ;;
        *) echo "Unknown: $1"; exit 1 ;;
    esac
done

CFG_DIR="$HOME/.sliver-client/configs"
mkdir -p "$CFG_DIR"

# ─── Helper: stop sliver processes ───
stop_server() {
    local PIDS
    PIDS=$(pgrep -f "sliver-server|sliver-client|garble" 2>/dev/null || true)
    [ -z "$PIDS" ] && return 0
    echo "[*] Sending SIGTERM..."
    echo "$PIDS" | xargs -r kill 2>/dev/null || true
    for i in $(seq 1 10); do
        pgrep -f "sliver-server" >/dev/null 2>&1 || return 0
        sleep 0.5
    done
    echo "[*] SIGKILL..."
    PIDS=$(pgrep -f "sliver-server|sliver-client|garble" 2>/dev/null || true)
    [ -n "$PIDS" ] && echo "$PIDS" | xargs -r kill -9 2>/dev/null || true
    sleep 2
}

# ─── Helper: free a port ───
free_port() {
    local PIDS
    PIDS=$(lsof -ti :"$1" 2>/dev/null || true)
    [ -n "$PIDS" ] && { echo "[*] Freeing port $1..."; echo "$PIDS" | xargs -r kill -9 2>/dev/null || true; sleep 1; }
}

# ─── Helper: run RC script ───
run_rc() {
    local RC_FILE="$1"
    local TIMEOUT="${2:-60}"
    timeout "$TIMEOUT" "$SCRIPT_DIR/sliver-client" console --rc "$RC_FILE" 2>&1 || true
}

# ─── Check for existing server ───
SERVER_RUNNING=0
pgrep -f "sliver-server" >/dev/null 2>&1 && SERVER_RUNNING=1

# Auto-restart if binary is newer than running daemon
if [ "$SERVER_RUNNING" = "1" ] && [ "$FRESH" = "0" ]; then
    RUNNING_PID=$(pgrep -f "sliver-server" | head -1)
    if [ -n "$RUNNING_PID" ] && [ -f "$SCRIPT_DIR/sliver-server" ]; then
        BINARY_MTIME=$(stat -c %Y "$SCRIPT_DIR/sliver-server" 2>/dev/null || echo 0)
        PROC_START=$(stat -c %Y "/proc/$RUNNING_PID" 2>/dev/null || echo 0)
        if [ "$BINARY_MTIME" -gt "$PROC_START" ] 2>/dev/null; then
            echo "[!] Binary newer than running daemon — auto-restarting..."
            FRESH=1
        fi
    fi
fi

if [ "$SERVER_RUNNING" = "1" ] && [ "$FRESH" = "0" ]; then
    echo "[*] Server running (PID $(pgrep -f 'sliver-server' | head -1)), checking health..."
    RC_TMP=$(mktemp /tmp/sliver-health-XXXXX.rc)
    echo "version" > "$RC_TMP"
    echo "exit" >> "$RC_TMP"
    if timeout 10 "$SCRIPT_DIR/sliver-client" console --rc "$RC_TMP" &>/dev/null; then
        echo "[+] Server healthy — reusing (beacons preserved)"
        SERVER_PID=$(pgrep -f "sliver-server" | head -1)
    else
        echo "[!] Server unresponsive. Restarting..."
        FRESH=1
    fi
    rm -f "$RC_TMP"
fi

if [ "$FRESH" = "1" ] || [ "$SERVER_RUNNING" = "0" ]; then
    [ "$SERVER_RUNNING" = "1" ] && { echo "[*] Stopping existing server..."; stop_server; }
    for PORT in $GRPC_PORT $MTLS_PORT $HTTPS_PORT; do
        free_port "$PORT"
    done
    echo "[*] Starting sliver-server daemon..."
    "$SCRIPT_DIR/sliver-server" daemon &
    SERVER_PID=$!
    disown $SERVER_PID
    echo "[*] Waiting for daemon..."
    sleep 8
fi

# ─── Create operator config ───
if [ -z "$(ls -A "$CFG_DIR" 2>/dev/null)" ]; then
    echo "[*] Creating operator config..."
    "$SCRIPT_DIR/sliver-server" operator --name local --lhost localhost --permissions all --save "$CFG_DIR/local.cfg"
    [ -z "$(ls -A "$CFG_DIR" 2>/dev/null)" ] && { echo "[-] Config creation failed"; exit 1; }
    echo "[+] Operator config created"
fi

# ─── Verify client connection ───
echo "[*] Verifying client connection..."
READY=0
RC_TMP=$(mktemp /tmp/sliver-verify-XXXXX.rc)
echo "version" > "$RC_TMP"
echo "exit" >> "$RC_TMP"
for i in $(seq 1 15); do
    if timeout 15 "$SCRIPT_DIR/sliver-client" console --rc "$RC_TMP" &>/dev/null; then
        READY=1; break
    fi
    sleep 2
done
rm -f "$RC_TMP"
[ "$READY" = "0" ] && { echo "[-] Client cannot connect. Try: $0 --fresh"; exit 1; }
echo "[+] Server ready (PID ${SERVER_PID:-$(pgrep -f 'sliver-server' | head -1)})"

# ─── Import C2 profile (first run only) ───
MARKER="$HOME/.sliver/.profile_imported"
PROFILE="$SCRIPT_DIR/opsec-profiles/microsoft365-c2.json"
if [ ! -f "$MARKER" ] && [ -f "$PROFILE" ]; then
    echo "[*] Importing microsoft365 C2 profile..."
    RC_TMP=$(mktemp /tmp/sliver-profile-XXXXX.rc)
    echo "c2profiles import -n microsoft365 -f $PROFILE" > "$RC_TMP"
    echo "exit" >> "$RC_TMP"
    run_rc "$RC_TMP" 30
    rm -f "$RC_TMP"
    mkdir -p "$HOME/.sliver" 2>/dev/null
    touch "$MARKER"
    echo "[+] Profile imported"
fi

# ─── Install armory (first run only) ───
ARMORY_MARKER="$HOME/.sliver/.armory_installed"
if [ ! -f "$ARMORY_MARKER" ]; then
    echo "[*] Installing ALL armory extensions (first run — takes several minutes)..."
    RC_TMP=$(mktemp /tmp/sliver-armory-XXXXX.rc)
    echo "armory install all" > "$RC_TMP"
    echo "exit" >> "$RC_TMP"
    run_rc "$RC_TMP" 600
    rm -f "$RC_TMP"
    mkdir -p "$HOME/.sliver" 2>/dev/null
    touch "$ARMORY_MARKER"
    echo "[+] Armory extensions installed"
fi

# ─── Start listeners ───
echo "[*] Starting mTLS listener on 0.0.0.0:$MTLS_PORT..."
RC_TMP=$(mktemp /tmp/sliver-listener-XXXXX.rc)
echo "mtls --lhost 0.0.0.0 --lport $MTLS_PORT" > "$RC_TMP"
if [ -n "$DOMAIN" ]; then
    echo "[*] Starting HTTPS listener on 0.0.0.0:$HTTPS_PORT ($DOMAIN)..."
    echo "https --lhost 0.0.0.0 --lport $HTTPS_PORT -d $DOMAIN" >> "$RC_TMP"
fi
echo "exit" >> "$RC_TMP"
run_rc "$RC_TMP" 20
rm -f "$RC_TMP"
echo "[+] Listeners started"

echo ""
echo "════════════════════════════════════════════════"
echo "  Sliver ready"
echo "  mTLS listener: 0.0.0.0:$MTLS_PORT"
[ -n "$DOMAIN" ] && echo "  HTTPS listener: 0.0.0.0:$HTTPS_PORT ($DOMAIN)"
echo "  Server PID: ${SERVER_PID:-$(pgrep -f 'sliver-server' | head -1)}"
echo "════════════════════════════════════════════════"
echo ""
echo "[*] Dropping into interactive console..."
exec "$SCRIPT_DIR/sliver-client"
