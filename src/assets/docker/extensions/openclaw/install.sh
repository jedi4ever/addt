#!/bin/bash
# OpenClaw installation (formerly Clawdbot/Moltbot)
# https://github.com/openclaw/openclaw

set -e

echo "Installing OpenClaw..."

# Install OpenClaw globally via npm
sudo npm install -g openclaw@latest

# Verify installation
if command -v openclaw &> /dev/null; then
    echo "OpenClaw installed successfully: $(openclaw --version 2>/dev/null || echo 'version unknown')"
else
    echo "Warning: openclaw command not found after installation"
fi
