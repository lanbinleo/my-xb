# MyXB Installation Script for Windows
# Usage: irm https://raw.githubusercontent.com/lanbinleo/my-xb/main/install.ps1 | iex

$ErrorActionPreference = "Stop"

$REPO_OWNER = "lanbinleo"
$REPO_NAME = "my-xb"
$INSTALL_DIR = Join-Path $env:USERPROFILE ".myxb\bin"
$EXECUTABLE_NAME = "myxb.exe"

function Write-ColorOutput($message, $color = "White") {
    Write-Host $message -ForegroundColor $color
}

function Get-LatestRelease {
    try {
        Write-ColorOutput "Fetching latest release information..." "Cyan"
        $apiUrl = "https://api.github.com/repos/$REPO_OWNER/$REPO_NAME/releases/latest"
        $release = Invoke-RestMethod -Uri $apiUrl -Headers @{ "User-Agent" = "MyXB-Installer" }
        return $release
    }
    catch {
        Write-ColorOutput "Failed to fetch release information: $_" "Red"
        exit 1
    }
}

function Get-AssetUrl($release) {
    # Detect architecture
    $arch = if ([Environment]::Is64BitOperatingSystem) { "amd64" } else { "386" }
    $assetName = "myxb_windows_$arch.exe"

    $asset = $release.assets | Where-Object { $_.name -eq $assetName }

    if (-not $asset) {
        Write-ColorOutput "Asset '$assetName' not found in release" "Red"
        Write-ColorOutput "Available assets:" "Yellow"
        $release.assets | ForEach-Object { Write-ColorOutput "  - $($_.name)" "Gray" }
        exit 1
    }

    return $asset.browser_download_url
}

function Download-Binary($url, $destination) {
    try {
        Write-ColorOutput "Downloading from: $url" "Cyan"

        # Show progress bar during download
        $ProgressPreference = 'SilentlyContinue'
        Invoke-WebRequest -Uri $url -OutFile $destination -UseBasicParsing
        $ProgressPreference = 'Continue'

        Write-ColorOutput "Download completed!" "Green"
    }
    catch {
        Write-ColorOutput "Download failed: $_" "Red"
        exit 1
    }
}

function Install-Binary($source, $destination) {
    try {
        # Create installation directory if it doesn't exist
        if (-not (Test-Path $INSTALL_DIR)) {
            New-Item -ItemType Directory -Path $INSTALL_DIR -Force | Out-Null
            Write-ColorOutput "Created installation directory: $INSTALL_DIR" "Green"
        }

        # Move binary to installation directory
        Move-Item -Path $source -Destination $destination -Force
        Write-ColorOutput "Installed to: $destination" "Green"
    }
    catch {
        Write-ColorOutput "Installation failed: $_" "Red"
        exit 1
    }
}

function Add-ToPath($directory) {
    $userPath = [Environment]::GetEnvironmentVariable("Path", "User")

    if ($userPath -notlike "*$directory*") {
        Write-ColorOutput "Adding to PATH: $directory" "Cyan"

        $newPath = if ($userPath) { "$userPath;$directory" } else { $directory }
        [Environment]::SetEnvironmentVariable("Path", $newPath, "User")

        Write-ColorOutput "Added to PATH successfully!" "Green"
        Write-ColorOutput "Note: You may need to restart your terminal for PATH changes to take effect." "Yellow"
    }
    else {
        Write-ColorOutput "Installation directory already in PATH" "Gray"
    }
}

# Main installation flow
Write-ColorOutput "`n=== MyXB Installation ===" "Cyan"
Write-ColorOutput ""

# Get latest release
$release = Get-LatestRelease
$version = $release.tag_name
Write-ColorOutput "Latest version: $version" "Green"
Write-ColorOutput ""

# Get download URL
$downloadUrl = Get-AssetUrl $release

# Download to temporary location
$tempFile = Join-Path $env:TEMP $EXECUTABLE_NAME
Write-ColorOutput "Downloading..." "Cyan"
Download-Binary $downloadUrl $tempFile
Write-ColorOutput ""

# Install binary
$destinationPath = Join-Path $INSTALL_DIR $EXECUTABLE_NAME
Write-ColorOutput "Installing..." "Cyan"
Install-Binary $tempFile $destinationPath
Write-ColorOutput ""

# Add to PATH
Add-ToPath $INSTALL_DIR
Write-ColorOutput ""

# Success message
Write-ColorOutput "=== Installation Complete! ===" "Green"
Write-ColorOutput ""
Write-ColorOutput "MyXB $version has been installed successfully!" "Green"
Write-ColorOutput ""
Write-ColorOutput "Quick start:" "Cyan"
Write-ColorOutput "  myxb login      # Login and save credentials" "Gray"
Write-ColorOutput "  myxb            # Calculate GPA" "Gray"
Write-ColorOutput "  myxb help       # Show help message" "Gray"
Write-ColorOutput ""
Write-ColorOutput "Note: If 'myxb' command is not found, please restart your terminal." "Yellow"
