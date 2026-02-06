#!/bin/bash
set -e
echo "Setup [claude]: Initializing Claude Code environment"

CLAUDE_JSON="$HOME/.claude.json"
CLAUDE_DIR="$HOME/.claude"
CLAUDE_INTERNAL_JSON="$CLAUDE_DIR/claude.json"
CLAUDE_CREDENTIALS_FILE="$CLAUDE_DIR/.credentials.json"

# Check if user already has authentication configured (from mounted config via automount)
if [ -d "$CLAUDE_DIR" ] "$CLAUDE_JSON" 2>/dev/null; then
    echo "Setup [claude]: Found existing Claude config (likely from automount), not modifying"
    exit 0
fi

# Continuining with setup as no $ClAUDE dir exists
echo "Setup [claude]: No $CLAUDE_DIR found, creating it"
mkdir -p "$CLAUDE_DIR"
# if no config file, create it
echo "Setup [claude]: Creating $CLAUDE_JSON"
cat > "$CLAUDE_JSON" << EOF
{
  "hasCompletedOnboarding": true,
  "hasTrustDialogAccepted": true,
  "projects": {
    "/workspace": {
      "allowedTools": [],
      "hasTrustDialogAccepted": true,
      "hasCompletedProjectOnboarding": true
    }
  }
}
EOF

# Create internal config (~/.claude/claude.json) - hooks trust dialog
echo "Setup [claude]: Creating $CLAUDE_INTERNAL_JSON (trusting hooks)"
cat > "$CLAUDE_INTERNAL_JSON" << 'EOF'
{
  "hasTrustDialogHooksAccepted": true,
  "hasCompletedOnboarding": true
}
EOF

# If ANTHROPIC_API_KEY is set, configure Claude Code for headless operation
if [ -n "$ANTHROPIC_API_KEY" ]; then
    # Extract last 20 characters of API key for trust configuration
    API_KEY_LAST_20="${ANTHROPIC_API_KEY: -20}"

    # Create user config (~/.claude.json) - onboarding, API key trust, and project trust
    echo "Setup [claude]: Found ANTHROPIC_API_KEY, configuring for API key authentication"
    # Insert customApiKeyResponses into existing $CLAUDE_JSON (don't overwrite, merge)
    if command -v jq >/dev/null 2>&1; then
        # Use jq to merge customApiKeyResponses
        tmpfile="$(mktemp)"
        jq --arg ak "$API_KEY_LAST_20" '
            .customApiKeyResponses = {
                approved: [$ak],
                rejected: []
            }
        ' "$CLAUDE_JSON" > "$tmpfile" && mv "$tmpfile" "$CLAUDE_JSON"
    else
        echo "Warning: jq not found, cannot insert API key into $CLAUDE_JSON"
        exit 1
    fi
EOF

    echo "Setup [claude]: Configured for API key authentication"
fi

# Next setup Claude credentials.json if it exists
if [ -n "$CLAUDE_OAUTH_CREDENTIALS" ]; then
    echo "Setup [claude]: Found $CLAUDE_CREDENTIALS_FILE, configuring for OAuth authentication"
    # base64 decode the Claude credentials.json
    echo "$CLAUDE_OAUTH_CREDENTIALS" | base64 -d > "$CLAUDE_CREDENTIALS_FILE"
    echo "Setup [claude]: Decoded $CLAUDE_CREDENTIALS_FILE"
    echo "Setup [claude]: Completed OAuth authentication setup"
fi

echo "Setup [claude]: Completed Claude Code environment setup"