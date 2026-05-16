# bbkit installer for Windows
# Usage: irm https://raw.githubusercontent.com/jinkp/bbkit/main/install.ps1 | iex

$ErrorActionPreference = "Stop"

$releasesUrl = "https://github.com/jinkp/bbkit/releases"

function Fail-Install {
    param([string]$Message)

    Write-Host ""
    Write-Host "bbkit install failed: $Message" -ForegroundColor Red
    Write-Host "Download a release manually from: $releasesUrl" -ForegroundColor Yellow
    exit 1
}

try {
    if ($env:OS -ne "Windows_NT") {
        Fail-Install "This installer only supports Windows."
    }

    $arch = $null
    if ([System.Environment]::Is64BitOperatingSystem) {
        $cpuArch = $env:PROCESSOR_ARCHITECTURE
        if ($cpuArch -eq "ARM64") {
            $arch = "arm64"
        } else {
            $arch = "amd64"
        }
    }
    if (-not $arch) {
        Fail-Install "Unsupported architecture: $env:PROCESSOR_ARCHITECTURE"
    }

    $assetName = "bbk-windows-$arch.exe"
    $downloadUrl = "https://github.com/jinkp/bbkit/releases/latest/download/$assetName"
    $installDir = Join-Path $env:LOCALAPPDATA "bbkit"
    $target = Join-Path $installDir "bbk.exe"

    Write-Host ""
    Write-Host "bbkit installer" -ForegroundColor Cyan
    Write-Host "Downloading $assetName..." -ForegroundColor Cyan

    New-Item -ItemType Directory -Force -Path $installDir | Out-Null
    Invoke-WebRequest -Uri $downloadUrl -OutFile $target

    $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
    $pathEntries = @($userPath -split ";" | Where-Object { $_ })
    if ($pathEntries -notcontains $installDir) {
        $newUserPath = if ([string]::IsNullOrWhiteSpace($userPath)) { $installDir } else { "$userPath;$installDir" }
        [Environment]::SetEnvironmentVariable("Path", $newUserPath, "User")
        Write-Host "Added $installDir to your user PATH." -ForegroundColor Green
    }

    if (-not (($env:Path -split ";") -contains $installDir)) {
        $env:Path = "$installDir;$env:Path"
    }

    $versionOutput = & bbk --version 2>&1
    if ($LASTEXITCODE -ne 0) {
        Fail-Install "Installed binary did not pass 'bbk --version'."
    }

    Write-Host ""
    Write-Host "bbkit installed successfully to $target" -ForegroundColor Green
    Write-Host $versionOutput
    Write-Host ""
    Write-Host "Get started:"
    Write-Host "  bbk setup"
    Write-Host "  bbk auth login"
    Write-Host "  bbk repo list"
}
catch {
    Fail-Install $_.Exception.Message
}
