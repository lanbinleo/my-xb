param(
    [string]$InstallDir = "",
    [string]$Version = "",
    [switch]$NoBackup
)

$ErrorActionPreference = "Stop"

$RepoRoot = Split-Path -Parent $MyInvocation.MyCommand.Path
$InstallDir = if ($InstallDir) { $InstallDir } else { Join-Path $env:USERPROFILE ".myxb\bin" }
$TargetPath = Join-Path $InstallDir "myxb.exe"
$BackupPath = "$TargetPath.bak"
$TempPath = Join-Path $env:TEMP ("myxb-local-" + [guid]::NewGuid().ToString() + ".exe")

function Write-Step($message, $color = "Cyan") {
    Write-Host $message -ForegroundColor $color
}

try {
    if (-not (Test-Path $InstallDir)) {
        New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
    }

    Push-Location $RepoRoot

    $buildArgs = @("build")
    if ($Version) {
        $buildArgs += @("-ldflags", "-X main.version=$Version")
    }
    $buildArgs += @("-o", $TempPath, "./cmd/myxb")

    Write-Step "Building local MyXB..."
    & go @buildArgs
    if ($LASTEXITCODE -ne 0) {
        throw "go build failed"
    }

    if ((Test-Path $TargetPath) -and -not $NoBackup) {
        Copy-Item -LiteralPath $TargetPath -Destination $BackupPath -Force
        Write-Step "Backed up existing install to: $BackupPath" "DarkGray"
    }

    Move-Item -LiteralPath $TempPath -Destination $TargetPath -Force

    Write-Step "Installed local build to: $TargetPath" "Green"
    if ($Version) {
        Write-Step "Embedded version: $Version" "DarkGray"
    } else {
        Write-Step "Embedded version: Dev" "DarkGray"
    }

    Write-Step "Quick check:" "Cyan"
    & $TargetPath help
}
finally {
    Pop-Location
    if (Test-Path $TempPath) {
        Remove-Item -LiteralPath $TempPath -Force
    }
}
