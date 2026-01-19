#!/bin/bash
set -e

# Setup golangci-lint for a Go project with production-ready configuration
# Usage: ./setup_golangci_lint.sh [project-root]

PROJECT_ROOT="${1:-.}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CONFIG_SOURCE="$SCRIPT_DIR/../assets/golangci.yml"

echo "🔧 Setting up golangci-lint for Go project..."
echo "   Project root: $PROJECT_ROOT"

# Check if golangci-lint is installed
if ! command -v golangci-lint &> /dev/null; then
    echo "❌ golangci-lint not found. Installing..."
    
    # Install golangci-lint (latest version)
    curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | \
        sh -s -- -b $(go env GOPATH)/bin
    
    echo "✅ golangci-lint installed"
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
