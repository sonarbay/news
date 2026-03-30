$ErrorActionPreference = "Stop"

$Repo = "sonarbay/news"
$Binary = "sonarbay.exe"
$InstallDir = "$env:LOCALAPPDATA\SonarBay\bin"
$BaseUrl = "https://github.com/$Repo/releases/latest/download"

$Arch = if ($env:PROCESSOR_ARCHITECTURE -eq "ARM64") { "arm64" } else { "x64" }
$Url = "$BaseUrl/sonarbay-win-$Arch.exe"

Write-Host ""
Write-Host "  SonarBay CLI Installer"
Write-Host "  ---------------------" -ForegroundColor Blue
Write-Host ""
Write-Host "  Platform:  windows-$Arch"
Write-Host "  Binary:    $InstallDir\$Binary"
Write-Host ""

if (-not (Test-Path $InstallDir)) {
    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
}

Write-Host "  Downloading..."
Invoke-WebRequest -Uri $Url -OutFile "$InstallDir\$Binary" -UseBasicParsing

$CurrentPath = [Environment]::GetEnvironmentVariable("PATH", "User")
if ($CurrentPath -notlike "*$InstallDir*") {
    [Environment]::SetEnvironmentVariable("PATH", "$CurrentPath;$InstallDir", "User")
    $env:PATH = "$InstallDir;$env:PATH"
    Write-Host "  Added $InstallDir to PATH"
}

Write-Host ""
Write-Host "  Installed sonarbay to $InstallDir\$Binary" -ForegroundColor Green
Write-Host ""
Write-Host "  Restart your terminal, then run: sonarbay --help"
Write-Host ""
