#!/bin/bash
# Amp CLI installation (Sourcegraph)
# https://ampcode.com/

set -e

echo "Installing Amp CLI..."

# Install Amp globally via npm
sudo npm install -g @sourcegraph/amp

# Verify installation
if command -v amp &> /dev/null; then
    echo "Amp CLI installed successfully: $(amp --version 2>/dev/null || echo 'version unknown')"
else
    echo "Warning: amp command not found after installation"
fi
