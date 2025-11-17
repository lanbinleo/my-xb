#!/bin/bash
# MyXB Installation Script for macOS and Linux
# Usage: curl -fsSL https://raw.githubusercontent.com/lanbinleo/my-xb/main/install.sh | bash

set -e

REPO_OWNER="lanbinleo"
REPO_NAME="my-xb"
EXECUTABLE_NAME="myxb"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
GRAY='\033[0;90m'
NC='\033[0m' # No Color

# Print colored output
print_color() {
    local color=$1
    shift
    echo -e "${color}$@${NC}"
}

print_error() {
    print_color "$RED" "Error: $@"
}

print_success() {
    print_color "$GREEN" "$@"
}

print_info() {
    print_color "$CYAN" "$@"
}

print_warning() {
    print_color "$YELLOW" "$@"
}

print_gray() {
    print_color "$GRAY" "$@"
}

# Detect operating system
detect_os() {
    case "$(uname -s)" in
        Darwin*)
            echo "darwin"
            ;;
        Linux*)
            echo "linux"
            ;;
        *)
            print_error "Unsupported operating system: $(uname -s)"
            exit 1
            ;;
    esac
}

# Detect architecture
detect_arch() {
    local arch=$(uname -m)
    case "$arch" in
        x86_64|amd64)
            echo "amd64"
            ;;
        aarch64|arm64)
            echo "arm64"
            ;;
        *)
            print_error "Unsupported architecture: $arch"
            exit 1
            ;;
    esac
}

# Get latest release information
get_latest_release() {
    print_info "Fetching latest release information..."

    local api_url="https://api.github.com/repos/$REPO_OWNER/$REPO_NAME/releases/latest"
    local response=$(curl -fsSL -H "User-Agent: MyXB-Installer" "$api_url")

    if [ $? -ne 0 ]; then
        print_error "Failed to fetch release information"
        exit 1
    fi

    echo "$response"
}

# Extract version from release JSON
get_version() {
    echo "$1" | grep -o '"tag_name": *"[^"]*"' | sed 's/"tag_name": *"\(.*\)"/\1/'
}

# Extract download URL from release JSON
get_download_url() {
    local release_json=$1
    local os=$2
    local arch=$3
    local asset_name="myxb_${os}_${arch}"

    # For macOS binaries, they don't have .exe extension
    local download_url=$(echo "$release_json" | grep -o "\"browser_download_url\": *\"[^\"]*${asset_name}\"" | sed 's/"browser_download_url": *"\(.*\)"/\1/')

    if [ -z "$download_url" ]; then
        print_error "Asset 'myxb_${os}_${arch}' not found in release"
        print_warning "Available assets:"
        echo "$release_json" | grep -o '"name": *"myxb_[^"]*"' | sed 's/"name": *"\(.*\)"/  - \1/'
        exit 1
    fi

    echo "$download_url"
}

# Download binary
download_binary() {
    local url=$1
    local destination=$2

    print_info "Downloading from: $url"

    if ! curl -fsSL --progress-bar -o "$destination" "$url"; then
        print_error "Download failed"
        exit 1
    fi

    print_success "Download completed!"
}

# Install binary
install_binary() {
    local source=$1
    local install_dir=""
    local needs_sudo=false

    # Try to install to /usr/local/bin first
    if [ -w "/usr/local/bin" ]; then
        install_dir="/usr/local/bin"
    elif [ -d "/usr/local/bin" ] && command -v sudo &> /dev/null; then
        install_dir="/usr/local/bin"
        needs_sudo=true
    else
        # Fall back to ~/.local/bin
        install_dir="$HOME/.local/bin"
        mkdir -p "$install_dir"
    fi

    local destination="$install_dir/$EXECUTABLE_NAME"

    print_info "Installing to: $destination"

    if [ "$needs_sudo" = true ]; then
        print_warning "Installation to $install_dir requires sudo privileges"
        if ! sudo mv "$source" "$destination"; then
            print_error "Installation failed"
            exit 1
        fi
        sudo chmod +x "$destination"
    else
        if ! mv "$source" "$destination"; then
            print_error "Installation failed"
            exit 1
        fi
        chmod +x "$destination"
    fi

    print_success "Installed successfully!"

    # Check if install_dir is in PATH
    if [[ ":$PATH:" != *":$install_dir:"* ]]; then
        print_warning "Installation directory is not in PATH: $install_dir"
        print_info "Add the following line to your shell profile (~/.bashrc, ~/.zshrc, etc.):"
        print_gray "  export PATH=\"\$PATH:$install_dir\""
    fi
}

# Handle macOS Gatekeeper
handle_macos_gatekeeper() {
    if [ "$(detect_os)" = "darwin" ]; then
        print_warning "Note: On macOS, you may need to allow the binary to run:"
        print_gray "  If you see a security warning, go to:"
        print_gray "  System Preferences → Security & Privacy → General"
        print_gray "  and click 'Allow Anyway' next to the myxb message"
        echo ""
    fi
}

# Main installation flow
main() {
    echo ""
    print_info "=== MyXB Installation ==="
    echo ""

    # Detect system
    local os=$(detect_os)
    local arch=$(detect_arch)
    print_gray "Detected system: $os/$arch"
    echo ""

    # Get latest release
    local release_json=$(get_latest_release)
    local version=$(get_version "$release_json")
    print_success "Latest version: $version"
    echo ""

    # Get download URL
    local download_url=$(get_download_url "$release_json" "$os" "$arch")

    # Download to temporary location
    local temp_file="/tmp/$EXECUTABLE_NAME.$$"
    print_info "Downloading..."
    download_binary "$download_url" "$temp_file"
    echo ""

    # Install binary
    print_info "Installing..."
    install_binary "$temp_file"
    echo ""

    # macOS specific instructions
    handle_macos_gatekeeper

    # Success message
    print_success "=== Installation Complete! ==="
    echo ""
    print_success "MyXB $version has been installed successfully!"
    echo ""
    print_info "Quick start:"
    print_gray "  myxb login      # Login and save credentials"
    print_gray "  myxb            # Calculate GPA"
    print_gray "  myxb help       # Show help message"
    echo ""
}

main
