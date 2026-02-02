#!/bin/bash
# Cursor CLI Agent installation
# https://cursor.com/docs/cli/installation

set -e

echo "Installing Cursor CLI..."

# Install Cursor CLI using official installer
curl https://cursor.com/install -fsSL | bash

# Create 'cursor' symlink for convenience (official installer creates 'agent')
if [ -f "$HOME/.local/bin/agent" ] && [ ! -f "$HOME/.local/bin/cursor" ]; then
    ln -s agent "$HOME/.local/bin/cursor"
fi

# Verify installation
if command -v agent &> /dev/null; then
    echo "Cursor CLI installed successfully: $(agent --version 2>/dev/null || echo 'version unknown')"
else
    echo "Warning: agent command not found after installation"
    echo "You may need to add ~/.local/bin to your PATH or install manually from https://cursor.com/cli"
fi
