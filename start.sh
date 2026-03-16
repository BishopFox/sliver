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

# ─── Helper: kill sliver-server and wait until it's actually dead ───
kill_server() {
    if ! pgrep -f "sliver-server" >/dev/null 2>&1; then
        return 0
    fi
    echo "[*] Sending SIGTERM to sliver-server..."
    pkill -f "sliver-server" 2>/dev/null || true
    # Wait up to 5s for graceful shutdown
    for i in $(seq 1 10); do
        if ! pgrep -f "sliver-server" >/dev/null 2>&1; then
            return 0
        fi
        sleep 0.5
    done
    # Still alive — SIGKILL
    echo "[*] Server still running, sending SIGKILL..."
    pkill -9 -f "sliver-server" 2>/dev/null || true
    sleep 1
    # Final check
    if pgrep -f "sliver-server" >/dev/null 2>&1; then
        echo "[-] Failed to kill sliver-server. Kill manually: pkill -9 -f sliver-server"
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
    # Verify the existing server actually works (not just PID exists)
    echo "[*] Sliver server running (PID $(pgrep -f 'sliver-server' | head -1)), checking health..."
    if echo "jobs" | "$SCRIPT_DIR/sliver-client" &>/dev/null; then
        echo "[+] Server healthy — reusing existing daemon (beacons preserved)"
        echo "    Use --fresh to restart clean."
        SERVER_PID=$(pgrep -f "sliver-server" | head -1)
    else
        echo "[!] Server PID exists but client can't connect. Restarting..."
        FRESH=1
    fi
fi

if [ "$FRESH" = "1" ] || [ "$SERVER_RUNNING" = "0" ]; then
    if [ "$SERVER_RUNNING" = "1" ]; then
        echo "[*] Killing existing sliver-server..."
        echo "    Beacons will reconnect after restart."
        kill_server
    fi

    # Free ALL relevant ports (gRPC + listeners)
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
    if echo "version" | "$SCRIPT_DIR/sliver-client" &>/dev/null; then
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
# Check if mTLS listener is already running (persistent listeners auto-start from DB)
NEED_LISTENER=1
if "$SCRIPT_DIR/sliver-client" << CHECK 2>/dev/null | grep -q ":${MTLS_PORT}"
jobs
CHECK
then
    echo "[+] mTLS listener already active on port $MTLS_PORT"
    NEED_LISTENER=0
fi

if [ "$NEED_LISTENER" = "1" ]; then
    echo "[*] Starting mTLS listener on 0.0.0.0:$MTLS_PORT..."
    LISTENER_CMDS="mtls --lhost 0.0.0.0 --lport $MTLS_PORT"

    if [ -n "$DOMAIN" ]; then
        echo "[*] Starting HTTPS listener on 0.0.0.0:$HTTPS_PORT ($DOMAIN)..."
        LISTENER_CMDS="$LISTENER_CMDS
https --lhost 0.0.0.0 --lport $HTTPS_PORT -d $DOMAIN"
    fi

    LISTENER_OK=0
    for attempt in 1 2 3; do
        if "$SCRIPT_DIR/sliver-client" << LISTENERS 2>/dev/null
$LISTENER_CMDS
LISTENERS
        then
            LISTENER_OK=1
            break
        fi
        echo "[*] Listener attempt $attempt/3 failed, retrying in 5s..."
        sleep 5
    done

    if [ "$LISTENER_OK" = "0" ]; then
        echo "[!] Auto-start failed. Start manually in console:"
        echo "    mtls --lhost 0.0.0.0 --lport $MTLS_PORT"
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
