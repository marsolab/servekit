#!/bin/bash
set -e

# Setup golangci-lint for a Go project with production-ready configuration
# Usage: ./setup_golangci_lint.sh [project-root]

PROJECT_ROOT="${1:-.}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CONFIG_SOURCE="$SCRIPT_DIR/../assets/golangci.yml"

echo "🔧 Setting up golangci-lint for Go project..."
echo "   Project root: $PROJECT_ROOT"

# Detect operating system
detect_os() {
    local os_type="${OSTYPE:-$(uname -s)}"
    case "$os_type" in
        darwin*)
            echo "macos"
            ;;
        linux*)
            # Check if running in WSL
            if grep -qi microsoft /proc/version 2>/dev/null || [ -n "$WSL_DISTRO_NAME" ]; then
                echo "wsl"
            else
                echo "linux"
            fi
            ;;
        msys*|cygwin*|mingw*)
            echo "windows"
            ;;
        *)
            echo "unknown"
            ;;
    esac
}

# Detect Linux distribution
detect_linux_distro() {
    if [ -f /etc/os-release ]; then
        . /etc/os-release
        echo "$ID"
    elif [ -f /etc/redhat-release ]; then
        echo "rhel"
    elif [ -f /etc/arch-release ]; then
        echo "arch"
    else
        echo "unknown"
    fi
}

# Install using Homebrew (macOS)
install_with_brew() {
    if ! command -v brew &> /dev/null; then
        return 1
    fi
    
    echo "📦 Installing golangci-lint using Homebrew..."
    if brew install golangci-lint; then
        echo "✅ golangci-lint installed via Homebrew"
        return 0
    fi
    return 1
}

# Install using apt-get (Debian/Ubuntu)
install_with_apt() {
    if ! command -v apt-get &> /dev/null; then
        return 1
    fi
    
    echo "📦 Installing golangci-lint using apt-get..."
    echo "   Note: This may require sudo privileges"
    
    if command -v sudo &> /dev/null; then
        if sudo apt-get update -qq && sudo apt-get install -y golangci-lint; then
            echo "✅ golangci-lint installed via apt-get"
            return 0
        fi
    else
        if apt-get update -qq && apt-get install -y golangci-lint 2>/dev/null; then
            echo "✅ golangci-lint installed via apt-get"
            return 0
        fi
    fi
    return 1
}

# Install using dnf (Fedora/RHEL 8+)
install_with_dnf() {
    if ! command -v dnf &> /dev/null; then
        return 1
    fi
    
    echo "📦 Installing golangci-lint using dnf..."
    echo "   Note: This may require sudo privileges"
    
    if command -v sudo &> /dev/null; then
        if sudo dnf install -y golangci-lint; then
            echo "✅ golangci-lint installed via dnf"
            return 0
        fi
    else
        if dnf install -y golangci-lint 2>/dev/null; then
            echo "✅ golangci-lint installed via dnf"
            return 0
        fi
    fi
    return 1
}

# Install using yum (RHEL/CentOS 7)
install_with_yum() {
    if ! command -v yum &> /dev/null; then
        return 1
    fi
    
    echo "📦 Installing golangci-lint using yum..."
    echo "   Note: This may require sudo privileges"
    
    if command -v sudo &> /dev/null; then
        if sudo yum install -y golangci-lint; then
            echo "✅ golangci-lint installed via yum"
            return 0
        fi
    else
        if yum install -y golangci-lint 2>/dev/null; then
            echo "✅ golangci-lint installed via yum"
            return 0
        fi
    fi
    return 1
}

# Install using pacman (Arch Linux)
install_with_pacman() {
    if ! command -v pacman &> /dev/null; then
        return 1
    fi
    
    echo "📦 Installing golangci-lint using pacman..."
    echo "   Note: This may require sudo privileges"
    
    if command -v sudo &> /dev/null; then
        if sudo pacman -S --noconfirm golangci-lint; then
            echo "✅ golangci-lint installed via pacman"
            return 0
        fi
    else
        if pacman -S --noconfirm golangci-lint 2>/dev/null; then
            echo "✅ golangci-lint installed via pacman"
            return 0
        fi
    fi
    return 1
}

# Install using snap (Universal Linux)
install_with_snap() {
    if ! command -v snap &> /dev/null; then
        return 1
    fi
    
    echo "📦 Installing golangci-lint using snap..."
    
    if snap install golangci-lint; then
        echo "✅ golangci-lint installed via snap"
        return 0
    fi
    return 1
}

# Install using flatpak (Universal Linux)
install_with_flatpak() {
    if ! command -v flatpak &> /dev/null; then
        return 1
    fi
    
    echo "📦 Installing golangci-lint using flatpak..."
    
    if flatpak install -y flathub org.golangci.golangci-lint 2>/dev/null; then
        echo "✅ golangci-lint installed via flatpak"
        return 0
    fi
    return 1
}

# Install using curl (fallback method)
install_with_curl() {
    echo "📦 Installing golangci-lint using official installation script..."
    echo "   (Latest version from GitHub)"
    
    if curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | \
        sh -s -- -b $(go env GOPATH)/bin; then
        echo "✅ golangci-lint installed via official script"
        return 0
    fi
    return 1
}

# Main installation function
install_golangci_lint() {
    local os=$(detect_os)
    local installed=0
    
    case "$os" in
        macos)
            echo "🖥️  Detected macOS"
            install_with_brew || install_with_curl
            installed=$?
            ;;
        linux|wsl)
            echo "🖥️  Detected Linux"
            local distro=$(detect_linux_distro)
            
            case "$distro" in
                ubuntu|debian)
                    install_with_apt || install_with_snap || install_with_flatpak || install_with_curl
                    installed=$?
                    ;;
                fedora|rhel|centos)
                    install_with_dnf || install_with_yum || install_with_snap || install_with_flatpak || install_with_curl
                    installed=$?
                    ;;
                arch|manjaro)
                    install_with_pacman || install_with_snap || install_with_flatpak || install_with_curl
                    installed=$?
                    ;;
                *)
                    # Try all package managers in order
                    install_with_apt || \
                    install_with_dnf || \
                    install_with_yum || \
                    install_with_pacman || \
                    install_with_snap || \
                    install_with_flatpak || \
                    install_with_curl
                    installed=$?
                    ;;
            esac
            ;;
        windows)
            echo "🖥️  Detected Windows (via Git Bash/WSL)"
            echo "   Note: For native Windows, consider using Chocolatey, Scoop, or winget"
            install_with_curl
            installed=$?
            ;;
        *)
            echo "⚠️  Unknown operating system, using fallback installation method"
            install_with_curl
            installed=$?
            ;;
    esac
    
    return $installed
}

# Check if golangci-lint is installed
if ! command -v golangci-lint &> /dev/null; then
    echo "❌ golangci-lint not found. Installing..."
    
    if ! install_golangci_lint; then
        echo "❌ Failed to install golangci-lint"
        exit 1
    fi
else
    echo "✅ golangci-lint already installed ($(golangci-lint --version))"
fi

# Copy configuration file
if [ -f "$CONFIG_SOURCE" ]; then
    cp "$CONFIG_SOURCE" "$PROJECT_ROOT/.golangci.yml"
    echo "✅ Configuration copied to $PROJECT_ROOT/.golangci.yml"
else
    echo "❌ Configuration file not found at $CONFIG_SOURCE"
    exit 1
fi

# Create Makefile target for linting if Makefile exists
if [ -f "$PROJECT_ROOT/Makefile" ]; then
    if ! grep -q "^lint:" "$PROJECT_ROOT/Makefile"; then
        echo "" >> "$PROJECT_ROOT/Makefile"
        echo "# Linting" >> "$PROJECT_ROOT/Makefile"
        echo "lint:" >> "$PROJECT_ROOT/Makefile"
        echo "	golangci-lint run" >> "$PROJECT_ROOT/Makefile"
        echo "" >> "$PROJECT_ROOT/Makefile"
        echo "lint-fix:" >> "$PROJECT_ROOT/Makefile"
        echo "	golangci-lint run --fix" >> "$PROJECT_ROOT/Makefile"
        echo "✅ Added lint targets to Makefile"
    fi
fi

# Create pre-commit hook (optional)
read -p "Install pre-commit hook for automatic linting? (y/N) " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    mkdir -p "$PROJECT_ROOT/.git/hooks"
    cat > "$PROJECT_ROOT/.git/hooks/pre-commit" << 'EOF'
#!/bin/bash
# Run golangci-lint before commit
golangci-lint run --config=.golangci.yml
EOF
    chmod +x "$PROJECT_ROOT/.git/hooks/pre-commit"
    echo "✅ Pre-commit hook installed"
fi

echo ""
echo "🎉 Setup complete! Run linting with:"
echo "   golangci-lint run"
echo "   # or with auto-fix:"
echo "   golangci-lint run --fix"
