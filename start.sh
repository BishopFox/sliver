#!/bin/bash
# Start Sliver server with auto-listener setup.
# Reuses existing daemon if running. Starts listeners, drops into client.
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

# ─── Create operator config if needed (before anything else) ───
CFG_DIR="$HOME/.sliver-client/configs"
mkdir -p "$CFG_DIR"

# ─── Check for existing server ───
SERVER_RUNNING=0
if pgrep -f "sliver-server" >/dev/null 2>&1; then
    SERVER_RUNNING=1
fi

if [ "$SERVER_RUNNING" = "1" ] && [ "$FRESH" = "0" ]; then
    echo "[+] Sliver server already running (PID $(pgrep -f 'sliver-server daemon' | head -1))"
    echo "    Reusing existing daemon — beacons and sessions preserved."
    echo "    Use --fresh to restart clean."
    SERVER_PID=$(pgrep -f "sliver-server daemon" | head -1)
else
    # Kill existing if --fresh or starting new
    if [ "$SERVER_RUNNING" = "1" ]; then
        echo "[*] --fresh: Killing existing sliver-server..."
        echo "    Active beacons will reconnect after restart."
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

    # ─── Wait for server daemon to initialize ───
    echo "[*] Waiting for server daemon..."
    sleep 5
fi

# ─── Create operator config if missing ───
if [ -z "$(ls -A "$CFG_DIR" 2>/dev/null)" ]; then
    echo "[*] Creating operator config..."
    "$SCRIPT_DIR/sliver-server" operator --name local --lhost localhost --permissions all --save "$CFG_DIR/local.cfg"
    if [ -z "$(ls -A "$CFG_DIR" 2>/dev/null)" ]; then
        echo "[-] Operator config creation failed. Check sliver-server logs."
        exit 1
    fi
    echo "[+] Operator config created"
fi

# ─── Verify client can connect ───
echo "[*] Verifying client connection..."
READY=0
for i in $(seq 1 20); do
    if "$SCRIPT_DIR/sliver-client" version &>/dev/null; then
        READY=1; break
    fi
    sleep 2
done
if [ "$READY" = "0" ]; then
    echo "[-] Client cannot connect to server."
    echo "    Config dir: $CFG_DIR"
    ls -la "$CFG_DIR" 2>/dev/null
    echo "    Try: $SCRIPT_DIR/start.sh --fresh"
    exit 1
fi
echo "[+] Server ready (PID ${SERVER_PID:-$(pgrep -f 'sliver-server daemon' | head -1)})"

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

# ─── Start listeners (with retry) ───
echo "[*] Starting mTLS listener on 0.0.0.0:$MTLS_PORT..."
LISTENER_CMDS="mtls --lhost 0.0.0.0 --lport $MTLS_PORT"

if [ -n "$DOMAIN" ]; then
    echo "[*] Starting HTTPS listener on 0.0.0.0:$HTTPS_PORT ($DOMAIN)..."
    LISTENER_CMDS="$LISTENER_CMDS
https --lhost 0.0.0.0 --lport $HTTPS_PORT -d $DOMAIN"
fi

# Retry listener start — server gRPC can be slow to accept after daemon start
LISTENER_OK=0
for attempt in 1 2 3; do
    if "$SCRIPT_DIR/sliver-client" << LISTENERS 2>/dev/null
$LISTENER_CMDS
LISTENERS
    then
        LISTENER_OK=1
        break
    fi
    echo "[*] Listener start attempt $attempt failed, retrying in 5s..."
    sleep 5
done

if [ "$LISTENER_OK" = "0" ]; then
    echo "[!] Listener auto-start failed. Start manually in console:"
    echo "    $LISTENER_CMDS"
fi

echo ""
echo "════════════════════════════════════════════════"
echo "  Sliver ready"
echo "  mTLS listener: 0.0.0.0:$MTLS_PORT"
[ -n "$DOMAIN" ] && echo "  HTTPS listener: 0.0.0.0:$HTTPS_PORT ($DOMAIN)"
echo "  Server PID: ${SERVER_PID:-$(pgrep -f 'sliver-server daemon' | head -1)}"
echo "════════════════════════════════════════════════"
echo ""
echo "[*] Dropping into interactive console..."
exec "$SCRIPT_DIR/sliver-client"
