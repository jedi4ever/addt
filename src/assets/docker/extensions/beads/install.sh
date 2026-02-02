#!/bin/bash
# Beads - Git-backed issue tracker for AI agents
# https://github.com/steveyegge/beads

set -e

echo "Extension [beads]: Installing Beads (bd)..."

# Install via Go
/usr/local/go/bin/go install github.com/steveyegge/beads/cmd/bd@latest

echo "Extension [beads]: Done. Installed bd at ~/go/bin/bd"
