$ErrorActionPreference = "Stop"

$Repo = "pachimsh/cli"
$Binary = "pachim"
$MirrorBase = "https://pachim.sh/cli"

function Get-LatestVersion {
    try {
        $mirrorVersion = (Invoke-WebRequest -Uri "$MirrorBase/latest.txt" -UseBasicParsing).Content.Trim()
        if ($mirrorVersion) {
            return $mirrorVersion
        }
    } catch {
        Write-Host "Mirror version check failed, trying GitHub..." -ForegroundColor Yellow
    }

    $response = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/releases/latest"
    return $response.tag_name
}

function Get-Arch {
    if ([Environment]::Is64BitOperatingSystem) {
        if ($env:PROCESSOR_ARCHITECTURE -eq "ARM64") {
            return "arm64"
        }
        return "amd64"
    }
    Write-Error "Unsupported architecture"
    exit 1
}

function Download-Archive {
    param(
        [string]$Version,
        [string]$Filename,
        [string]$Destination
    )

    $mirrorUrl = "$MirrorBase/$Version/$Filename"
    try {
        Write-Host "Downloading $mirrorUrl..." -ForegroundColor Yellow
        Invoke-WebRequest -Uri $mirrorUrl -OutFile $Destination -UseBasicParsing
        Write-Host "Downloaded from pachim.sh mirror." -ForegroundColor Green
        return $true
    } catch {
        Write-Host "Mirror unavailable, trying GitHub..." -ForegroundColor Yellow
    }

    $githubUrl = "https://github.com/$Repo/releases/download/$Version/$Filename"
    Write-Host "Downloading $githubUrl..." -ForegroundColor Yellow
    Invoke-WebRequest -Uri $githubUrl -OutFile $Destination -UseBasicParsing
    Write-Host "Downloaded from GitHub." -ForegroundColor Green
    return $true
}

function Install-Pachim {
    $arch = Get-Arch
    $version = Get-LatestVersion

    Write-Host "Detected: windows/$arch" -ForegroundColor Cyan
    Write-Host "Latest version: $version" -ForegroundColor Cyan

    $filename = "${Binary}_windows_${arch}.zip"

    $installDir = "$env:LOCALAPPDATA\Programs\pachim"
    $tmpDir = Join-Path $env:TEMP "pachim-install-$(Get-Random)"

    New-Item -ItemType Directory -Path $tmpDir -Force | Out-Null
    New-Item -ItemType Directory -Path $installDir -Force | Out-Null

    Download-Archive -Version $version -Filename $filename -Destination "$tmpDir\$filename"

    Write-Host "Extracting..." -ForegroundColor Yellow
    Expand-Archive -Path "$tmpDir\$filename" -DestinationPath $tmpDir -Force

    Copy-Item "$tmpDir\$Binary.exe" "$installDir\$Binary.exe" -Force

    Remove-Item -Recurse -Force $tmpDir

    $currentPath = [Environment]::GetEnvironmentVariable("Path", "User")
    if ($currentPath -notlike "*$installDir*") {
        [Environment]::SetEnvironmentVariable("Path", "$currentPath;$installDir", "User")
        $env:Path = "$env:Path;$installDir"
        Write-Host "Added $installDir to PATH" -ForegroundColor Green
    }

    Write-Host ""
    Write-Host "pachim $version installed successfully!" -ForegroundColor Green
    Write-Host "  Restart your terminal, then run 'pachim --help' to get started." -ForegroundColor Cyan
}

Install-Pachim
