#!/bin/bash
# Claude Code - AI coding assistant by Anthropic
# https://github.com/anthropics/claude-code

set -e

echo "Extension [claude]: Installing Claude Code..."
echo "Extension [claude]: NPM_CONFIG_PREFIX: $NPM_CONFIG_PREFIX"

# Get version from environment or default to latest
CLAUDE_VERSION="${CLAUDE_VERSION:-latest}"

# native installer
# we need to install this first as it will DELETE the npm install
echo "Extension [claude]: Installing Claude Code Native Installer"
# this will install in $HOME/.local/bin/claude
# this has precedence over the npm install
# simple removing it selects the npm install
# https://code.claude.com/docs/en/setup#install-a-specific-version
# The native installer only accepts semver versions; "latest"/"stable" mean no version arg
if [ "$CLAUDE_VERSION" = "latest" ] || [ "$CLAUDE_VERSION" = "stable" ]; then
    curl -fsSL https://claude.ai/install.sh | bash
else
    curl -fsSL https://claude.ai/install.sh | bash -s $CLAUDE_VERSION
fi

#TODO - figure how to set the version

# Disable AUTO UPDATE
# DISABLE_AUTOUPDATER=1 ??

# Install via npm (globally, to user-owned NPM_CONFIG_PREFIX)
if [ "$CLAUDE_VERSION" = "latest" ] || [ "$CLAUDE_VERSION" = "stable" ]; then
    npm install -g @anthropic-ai/claude-code
else
    npm install -g @anthropic-ai/claude-code@$CLAUDE_VERSION 
fi

# Verify installation
INSTALLED_VERSION=$(claude --version 2>/dev/null | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -1 || echo "unknown")
echo "Extension [claude]: Done. Installed Claude Code v${INSTALLED_VERSION}"
