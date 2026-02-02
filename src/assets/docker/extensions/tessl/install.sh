#!/bin/bash
# Tessl - Agent enablement platform
# https://tessl.io/
# https://docs.tessl.io/

set -e

echo "Extension [tessl]: Installing Tessl CLI..."

# Install via npm (globally, requires root)
sudo npm install -g @tessl/cli

echo "Extension [tessl]: Done. Installed tessl CLI"
echo "  Run 'tessl init' to authenticate and configure"
echo "  Run 'tessl skill search' to find skills"
echo "  Run 'tessl mcp' to start MCP server"
