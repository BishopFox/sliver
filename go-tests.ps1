#!/usr/bin/env pwsh

# Sliver Implant Framework
# Copyright (C) 2019  Bishop Fox
#
# This program is free software: you can redistribute it and/or modify
# it under the terms of the GNU General Public License as published by
# the Free Software Foundation, either version 3 of the License, or
# (at your option) any later version.
#
# This program is distributed in the hope that it will be useful,
# but WITHOUT ANY WARRANTY; without even the implied warranty of
# MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
# GNU General Public License for more details.
#
# You should have received a copy of the GNU General Public License
# along with this program.  If not, see <https://www.gnu.org/licenses/>.

[CmdletBinding()]
param(
    [switch]$SkipGenerate,
    [switch]$UnitOnly,
    [Parameter(ValueFromRemainingArguments = $true)]
    [string[]]$RemainingArgs
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

foreach ($arg in $RemainingArgs) {
    switch ($arg) {
        "--skip-generate" {
            $SkipGenerate = $true
            continue
        }
        "--unit-only" {
            $UnitOnly = $true
            $SkipGenerate = $true
            continue
        }
        default {
            throw "Unknown argument: $arg"
        }
    }
}

if ($UnitOnly) {
    $SkipGenerate = $true
}

$script:TestTmpRoot = $null
$script:CreatedTestRoot = $false

function Invoke-ExternalCommand {
    param(
        [Parameter(Mandatory = $true)]
        [string]$FilePath,
        [string[]]$Arguments = @(),
        [hashtable]$Environment = @{}
    )

    $previous = @{}
    try {
        foreach ($name in $Environment.Keys) {
            $previous[$name] = [System.Environment]::GetEnvironmentVariable($name, "Process")
            [System.Environment]::SetEnvironmentVariable($name, $Environment[$name], "Process")
        }

        & $FilePath @Arguments
        if ($LASTEXITCODE -ne 0) {
            throw "Command exited with code ${LASTEXITCODE}: $FilePath $($Arguments -join ' ')"
        }
    } catch {
        Write-Host $_
        throw
    } finally {
        foreach ($name in $Environment.Keys) {
            [System.Environment]::SetEnvironmentVariable($name, $previous[$name], "Process")
        }
    }
}

function Write-FailureLogs {
    $logPath = Join-Path $env:SLIVER_ROOT_DIR "logs\sliver.log"
    if (Test-Path -LiteralPath $logPath) {
        Get-Content -LiteralPath $logPath -ErrorAction SilentlyContinue
    }
}

function Invoke-TestStep {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Name,
        [Parameter(Mandatory = $true)]
        [scriptblock]$Action
    )

    Write-Host ""
    Write-Host "==> $Name"
    try {
        & $Action
        return
    } catch {
        Write-FailureLogs
        throw "Test step failed: $Name"
    }
}

function Test-SkipPackage {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Package,
        [string]$Tags = ""
    )

    $goListArgs = @("list", "-e", "-f", "{{if .Error}}{{.Error}}{{end}}")
    if ($Tags) {
        $goListArgs += "-tags=$Tags"
    }
    $goListArgs += $Package

    $goListError = & go @goListArgs 2>$null
    $goListText = ($goListError | Out-String)
    if ($goListText -like "*build constraints exclude all Go files*") {
        Write-Host ""
        Write-Host "==> Skipping $Package (unsupported on current platform)"
        return $true
    }

    return $false
}

function Get-TestDirectories {
    Get-ChildItem -Path @("client", "implant", "server", "util") -Recurse -Filter "*_test.go" -File |
        ForEach-Object {
            [string](Resolve-Path -LiteralPath $_.DirectoryName -Relative).Replace("\", "/")
        } |
        Sort-Object -Unique
}

function Get-ServerBinaryPath {
    $candidates = @(
        (Join-Path $PWD.Path "sliver-server.exe"),
        (Join-Path $PWD.Path "sliver-server")
    )

    foreach ($candidate in $candidates) {
        if (Test-Path -LiteralPath $candidate) {
            return $candidate
        }
    }

    return $null
}

function Invoke-UnpackServerAssets {
    $serverBinary = Get-ServerBinaryPath
    if ($serverBinary) {
        try {
            Invoke-ExternalCommand -FilePath $serverBinary -Arguments @("unpack", "--force")
            return
        } catch {
            Write-Host "sliver-server unpack failed, falling back to go run ./server unpack --force"
        }
    }

    Invoke-ExternalCommand -FilePath "go" -Arguments @("run", "-tags=server,go_sqlite", "./server", "unpack", "--force")
}

function Stop-TestProcesses {
    if (-not $script:CreatedTestRoot -or -not $script:TestTmpRoot) {
        return
    }

    try {
        $escapedRoot = [Regex]::Escape($script:TestTmpRoot)
        $relatedProcesses = Get-CimInstance Win32_Process -ErrorAction Stop |
            Where-Object {
                $_.ParentProcessId -eq $PID -or
                ($_.CommandLine -and $_.CommandLine -match $escapedRoot)
            } |
            Sort-Object ProcessId -Unique

        foreach ($proc in $relatedProcesses) {
            Stop-Process -Id $proc.ProcessId -Force -ErrorAction SilentlyContinue
        }
    } catch {
        # Best-effort cleanup only.
    }
}

function Remove-TestRoot {
    if (-not $script:CreatedTestRoot -or -not $script:TestTmpRoot) {
        return
    }

    for ($attempt = 0; $attempt -lt 5; $attempt++) {
        try {
            Remove-Item -LiteralPath $script:TestTmpRoot -Recurse -Force -ErrorAction Stop
        } catch {
            Start-Sleep -Seconds 1
        }

        if (-not (Test-Path -LiteralPath $script:TestTmpRoot)) {
            return
        }
    }

    if (Test-Path -LiteralPath $script:TestTmpRoot) {
        Write-Warning "Failed to fully clean temp dir: $script:TestTmpRoot"
    }
}

try {
    Write-Host "----------------------------------------------------------------"
    Write-Host "WARNING: Running unit tests on slow systems can take a LONG time"
    Write-Host "         Recommended to only run on 16+ CPU cores and 32Gb+ RAM"
    Write-Host "----------------------------------------------------------------"

    $tags = "osusergo,netgo,go_sqlite"

    $script:TestTmpRoot = Join-Path ([System.IO.Path]::GetTempPath()) ("sliver-go-tests-" + [System.Guid]::NewGuid().ToString("N"))
    New-Item -ItemType Directory -Path $script:TestTmpRoot -Force | Out-Null
    $script:CreatedTestRoot = $true

    $env:SLIVER_ROOT_DIR = Join-Path $script:TestTmpRoot "sliver"
    $env:SLIVER_CLIENT_DIR = Join-Path $script:TestTmpRoot "sliver-client"
    $env:SLIVER_CLIENT_ROOT_DIR = $env:SLIVER_CLIENT_DIR
    $env:HOME = Join-Path $script:TestTmpRoot "home"
    $env:XDG_CONFIG_HOME = Join-Path $script:TestTmpRoot "xdg-config"
    $env:XDG_CACHE_HOME = Join-Path $script:TestTmpRoot "xdg-cache"
    $env:XDG_DATA_HOME = Join-Path $script:TestTmpRoot "xdg-data"
    $env:TMPDIR = Join-Path $script:TestTmpRoot "tmp"
    $env:TMP = $env:TMPDIR
    $env:TEMP = $env:TMPDIR
    $env:GOCACHE = Join-Path $script:TestTmpRoot "go-cache"
    $env:GOTMPDIR = Join-Path $script:TestTmpRoot "go-tmp"
    if ($env:GOFLAGS) {
        $env:GOFLAGS = ($env:GOFLAGS.Trim() + " -mod=vendor").Trim()
    } else {
        $env:GOFLAGS = "-mod=vendor"
    }

    @(
        $env:SLIVER_ROOT_DIR,
        $env:SLIVER_CLIENT_DIR,
        $env:HOME,
        $env:XDG_CONFIG_HOME,
        $env:XDG_CACHE_HOME,
        $env:XDG_DATA_HOME,
        $env:TMPDIR,
        $env:GOCACHE,
        $env:GOTMPDIR
    ) | ForEach-Object {
        New-Item -ItemType Directory -Path $_ -Force | Out-Null
    }

    $isWindowsPlatform = [System.Runtime.InteropServices.RuntimeInformation]::IsOSPlatform(
        [System.Runtime.InteropServices.OSPlatform]::Windows
    )
    $sliverGoBin = Join-Path $env:SLIVER_ROOT_DIR "go\bin"
    if (-not $isWindowsPlatform) {
        $env:PATH = [string]::Join([System.IO.Path]::PathSeparator, @($sliverGoBin, $env:PATH))
    }

    Write-Host "Using isolated temp directories:"
    Write-Host "  SLIVER_ROOT_DIR=$env:SLIVER_ROOT_DIR"
    Write-Host "  SLIVER_CLIENT_DIR=$env:SLIVER_CLIENT_DIR"

    Invoke-TestStep -Name "unpack server assets" -Action { Invoke-UnpackServerAssets }

    $testDirs = @(Get-TestDirectories)
    $clientTestPkgs = @()
    $implantTestPkgs = @()
    $serverUtilTestPkgs = @()

    foreach ($testDir in $testDirs) {
        $pkg = if ($testDir.StartsWith(".")) { $testDir } else { "./$testDir" }
        switch -Wildcard ($pkg) {
            "./server/c2" { continue }
            "./server/generate" { continue }
            "./client/*" {
                $clientTestPkgs += $pkg
                continue
            }
            "./implant/*" {
                $implantTestPkgs += $pkg
                continue
            }
            "./server/*" {
                $serverUtilTestPkgs += $pkg
                continue
            }
            "./util*" {
                $serverUtilTestPkgs += $pkg
                continue
            }
        }
    }

    foreach ($pkg in $clientTestPkgs) {
        if (Test-SkipPackage -Package $pkg -Tags "client,$tags") {
            continue
        }

        Invoke-TestStep -Name $pkg -Action {
            Invoke-ExternalCommand -FilePath "go" -Arguments @("test", "-tags=client,$tags", $pkg)
        }
    }

    foreach ($pkg in $implantTestPkgs) {
        if (Test-SkipPackage -Package $pkg) {
            continue
        }

        Invoke-TestStep -Name $pkg -Action {
            Invoke-ExternalCommand -FilePath "go" -Arguments @("test", $pkg)
        }
    }

    foreach ($pkg in $serverUtilTestPkgs) {
        if (Test-SkipPackage -Package $pkg -Tags "server,$tags") {
            continue
        }

        switch ($pkg) {
            "./server/assets/traffic-encoders" {
                Invoke-TestStep -Name $pkg -Action {
                    Invoke-ExternalCommand -FilePath "go" -Arguments @("test", "-timeout", "10m", "-tags=server,$tags", $pkg)
                }
                continue
            }
            "./server/encoders" {
                Invoke-TestStep -Name $pkg -Action {
                    Invoke-ExternalCommand -FilePath "go" -Arguments @("test", "-timeout", "10m", "-tags=server,$tags", $pkg)
                }
                continue
            }
            "./server/rpc" {
                Invoke-TestStep -Name $pkg -Action {
                    Invoke-ExternalCommand -FilePath "go" -Arguments @("test", "-vet=off", "-timeout", "30m", "-tags=server,$tags", $pkg)
                }
                continue
            }
            default {
                Invoke-TestStep -Name $pkg -Action {
                    Invoke-ExternalCommand -FilePath "go" -Arguments @("test", "-tags=server,$tags", $pkg)
                }
            }
        }
    }

    Invoke-TestStep -Name "./server/c2" -Action {
        Invoke-ExternalCommand -FilePath "go" -Arguments @("test", "-tags=server,$tags", "./server/c2")
    }

    if ($UnitOnly) {
        Write-Host ""
        Write-Host "Skipping ./server/c2 e2e tests (--unit-only)"
    } else {
        Invoke-TestStep -Name "./server/c2 (e2e yamux)" -Action {
            Invoke-ExternalCommand -FilePath "go" -Arguments @("test", "-tags=server,$tags,sliver_e2e", "./server/c2", "-run", "Test(MTLS|WG)Yamux_", "-count=1")
        }
        Invoke-TestStep -Name "./server/c2 (e2e dns)" -Action {
            Invoke-ExternalCommand -FilePath "go" -Arguments @("test", "-tags=server,$tags,sliver_e2e", "./server/c2", "-run", "TestDNS_", "-count=1")
        }
    }

    if (-not $SkipGenerate) {
        $generateGoMaxProcs = if ($env:SLIVER_GENERATE_GOMAXPROCS) { $env:SLIVER_GENERATE_GOMAXPROCS } else { "2" }
        $generateGoP = if ($env:SLIVER_GENERATE_GO_P) { $env:SLIVER_GENERATE_GO_P } else { "1" }
        $generateTestParallel = if ($env:SLIVER_GENERATE_TEST_PARALLEL) { $env:SLIVER_GENERATE_TEST_PARALLEL } else { "1" }

        Invoke-TestStep -Name "./server/generate" -Action {
            Invoke-ExternalCommand -FilePath "go" `
                -Arguments @("test", "-timeout", "6h", "-p", $generateGoP, "-parallel", $generateTestParallel, "-tags=server,$tags", "./server/generate") `
                -Environment @{
                    GOMAXPROCS = $generateGoMaxProcs
                    GOPROXY    = "off"
                }
        }
    } else {
        Write-Host ""
        if ($UnitOnly) {
            Write-Host "Skipping ./server/generate tests (--unit-only)"
        } else {
            Write-Host "Skipping ./server/generate tests (--skip-generate)"
        }
    }
} finally {
    Stop-TestProcesses
    Remove-TestRoot
}
