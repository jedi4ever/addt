#!/bin/bash
# GitHub Copilot CLI installation
# https://github.com/github/copilot-cli

set -e

echo "Installing GitHub Copilot CLI..."

# Install Copilot CLI globally via npm
sudo npm install -g @github/copilot

# Verify installation
if command -v copilot &> /dev/null; then
    echo "Copilot CLI installed successfully: $(copilot --version 2>/dev/null || echo 'version unknown')"
else
    echo "Warning: copilot command not found after installation"
fi
