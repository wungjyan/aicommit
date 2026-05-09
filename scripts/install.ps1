$ErrorActionPreference = "Stop"

$REPO = "wungjyan/aicommit"
$BINARY = "aicommit"
$INSTALL_DIR = "$env:LOCALAPPDATA\aicommit\bin"

function Get-Architecture {
    if ($env:PROCESSOR_ARCHITECTURE -eq "ARM64" -or $env:PROCESSOR_ARCHITEW6432 -eq "ARM64") {
        return "arm64"
    }
    if ($env:PROCESSOR_ARCHITECTURE -eq "AMD64" -or $env:PROCESSOR_ARCHITEW6432 -eq "AMD64") {
        return "amd64"
    }
    Write-Error "Unsupported architecture: $env:PROCESSOR_ARCHITECTURE"
    exit 1
}

function Get-LatestVersion {
    $headers = @{ "User-Agent" = "$REPO-installer" }
    $response = Invoke-RestMethod -Uri "https://api.github.com/repos/$REPO/releases/latest" -Headers $headers
    $version = $response.tag_name -replace '^v', ''
    if (-not $version) {
        Write-Error "Failed to fetch latest version"
        exit 1
    }
    return $version
}

function Add-ToPath {
    $currentPath = [Environment]::GetEnvironmentVariable("Path", "User")
    $entries = $currentPath -split ";" | ForEach-Object { $_.TrimEnd("\") }
    $target = $INSTALL_DIR.TrimEnd("\")
    if ($entries -notcontains $target) {
        $newPath = if ($currentPath) { "$currentPath;$INSTALL_DIR" } else { $INSTALL_DIR }
        [Environment]::SetEnvironmentVariable("Path", $newPath, "User")
        $env:Path = if ($env:Path) { "$env:Path;$INSTALL_DIR" } else { $INSTALL_DIR }
        Write-Host "Added $INSTALL_DIR to user PATH."
    }
}

$arch = Get-Architecture
$version = Get-LatestVersion
$filename = "$BINARY-windows-$arch.exe"
$url = "https://github.com/$REPO/releases/download/v$version/$filename"

Write-Host "Installing aicommit v$version..."
Write-Host "  Architecture: $arch"
Write-Host "  Install to:   $INSTALL_DIR\$BINARY.exe"

if (-not (Test-Path $INSTALL_DIR)) {
    New-Item -ItemType Directory -Path $INSTALL_DIR -Force | Out-Null
}

$tmpFile = Join-Path $env:TEMP "$BINARY-$([System.IO.Path]::GetRandomFileName()).exe"
try {
    Invoke-WebRequest -Uri $url -OutFile $tmpFile
    $verify = & $tmpFile version 2>&1
    if ($LASTEXITCODE -ne 0) {
        Write-Error "Downloaded binary is invalid. Aborting."
        exit 1
    }
    Move-Item -Path $tmpFile -Destination "$INSTALL_DIR\$BINARY.exe" -Force
} catch {
    if (Test-Path $tmpFile) { Remove-Item $tmpFile -Force }
    throw
}

Add-ToPath

Write-Host ""
Write-Host "Installed: $(& "$INSTALL_DIR\$BINARY.exe" version)"
Write-Host ""
Write-Host "Restart your terminal for PATH changes to take effect."
