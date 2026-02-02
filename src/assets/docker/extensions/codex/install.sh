#!/bin/bash
# OpenAI Codex CLI installation
# https://github.com/openai/codex

set -e

echo "Installing OpenAI Codex CLI..."

# Install codex globally via npm
sudo npm install -g @openai/codex

# Verify installation
if command -v codex &> /dev/null; then
    echo "Codex CLI installed successfully: $(codex --version 2>/dev/null || echo 'version unknown')"
else
    echo "Warning: codex command not found after installation"
fi
