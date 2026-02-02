#!/bin/bash
# Gemini CLI installation (Google)
# https://github.com/google-gemini/gemini-cli

set -e

echo "Installing Gemini CLI..."

# Install Gemini CLI globally via npm
sudo npm install -g @google/gemini-cli

# Verify installation
if command -v gemini &> /dev/null; then
    echo "Gemini CLI installed successfully: $(gemini --version 2>/dev/null || echo 'version unknown')"
else
    echo "Warning: gemini command not found after installation"
fi
