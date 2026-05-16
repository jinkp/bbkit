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
    if (-not [System.Runtime.InteropServices.RuntimeInformation]::IsOSPlatform([System.Runtime.InteropServices.OSPlatform]::Windows)) {
        Fail-Install "This installer only supports Windows."
    }

    $arch = switch ([System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture) {
        ([System.Runtime.InteropServices.Architecture]::X64) { "amd64" }
        ([System.Runtime.InteropServices.Architecture]::Arm64) { "arm64" }
        default { Fail-Install "Unsupported architecture: $([System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture)" }
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
