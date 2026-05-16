# bbkit-cli installer for Windows
# Usage: irm https://raw.githubusercontent.com/jinkp/bbkit/main/install.ps1 | iex

$ErrorActionPreference = "Stop"

Write-Host ""
Write-Host "  bbkit — Bitbucket Cloud CLI" -ForegroundColor Cyan
Write-Host ""

# Check Node.js
if (-not (Get-Command node -ErrorAction SilentlyContinue)) {
    Write-Host "  [ERROR] Node.js is required but not found." -ForegroundColor Red
    Write-Host "  Install it from https://nodejs.org (v20+)" -ForegroundColor Yellow
    Write-Host ""
    exit 1
}

$nodeVersion = (node --version) -replace "^v", ""
$major = [int]($nodeVersion -split "\.")[0]
if ($major -lt 20) {
    Write-Host "  [ERROR] Node.js v20+ is required. Found v$nodeVersion" -ForegroundColor Red
    Write-Host "  Update from https://nodejs.org" -ForegroundColor Yellow
    Write-Host ""
    exit 1
}

Write-Host "  Node.js v$nodeVersion detected" -ForegroundColor Green

# Install bbkit-cli globally via npm
Write-Host "  Installing bbkit-cli..." -ForegroundColor Cyan

try {
    npm install -g bbkit-cli 2>&1 | Out-Null
} catch {
    Write-Host "  [ERROR] Installation failed: $_" -ForegroundColor Red
    exit 1
}

# Verify installation
if (-not (Get-Command bbk -ErrorAction SilentlyContinue)) {
    Write-Host "  [ERROR] bbk command not found after install." -ForegroundColor Red
    Write-Host "  Try running: npm install -g bbkit-cli" -ForegroundColor Yellow
    exit 1
}

$bbkVersion = bbk --version
Write-Host ""
Write-Host "  bbkit v$bbkVersion installed successfully!" -ForegroundColor Green
Write-Host ""
Write-Host "  Get started:" -ForegroundColor Cyan
Write-Host "    bbk setup        Configure credentials and workspace"
Write-Host "    bbk auth login   Authenticate with Bitbucket"
Write-Host "    bbk repo list    List your repositories"
Write-Host "    bbk --help       See all commands"
Write-Host ""
