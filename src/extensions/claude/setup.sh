#!/bin/bash
echo "Setup [claude]: Initializing Claude Code environment"

CLAUDE_JSON="$HOME/.claude.json"
CLAUDE_DIR="$HOME/.claude"
CLAUDE_INTERNAL_JSON="$CLAUDE_DIR/claude.json"

# Check if user already has authentication configured (from mounted config via automount)
# If hasCompletedOnboarding is set, they likely have OAuth credentials - don't overwrite
if [ -f "$CLAUDE_JSON" ] && grep -q '"hasCompletedOnboarding"' "$CLAUDE_JSON" 2>/dev/null; then
    echo "Setup [claude]: Found existing Claude config (likely from automount), preserving authentication"

    # Just ensure /workspace is trusted in the existing config
    if command -v jq &> /dev/null && ! grep -q '"/workspace"' "$CLAUDE_JSON" 2>/dev/null; then
        echo "Setup [claude]: Adding /workspace trust to existing config"
        tmp=$(mktemp)
        jq '.projects["/workspace"] = {"allowedTools": [], "hasTrustDialogAccepted": true, "hasCompletedProjectOnboarding": true}' "$CLAUDE_JSON" > "$tmp" && mv "$tmp" "$CLAUDE_JSON"
    fi
    exit 0
fi

# If ANTHROPIC_API_KEY is set, configure Claude Code for headless operation
if [ -n "$ANTHROPIC_API_KEY" ]; then
    # Extract last 20 characters of API key for trust configuration
    API_KEY_LAST_20="${ANTHROPIC_API_KEY: -20}"

    # Create user config (~/.claude.json) - onboarding, API key trust, and project trust
    echo "Setup [claude]: Creating $CLAUDE_JSON (skipping onboarding, trusting API key and /workspace)"
    cat > "$CLAUDE_JSON" << EOF
{
  "hasCompletedOnboarding": true,
  "hasTrustDialogAccepted": true,
  "customApiKeyResponses": {
    "approved": ["$API_KEY_LAST_20"],
    "rejected": []
  },
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
    mkdir -p "$CLAUDE_DIR"
    echo "Setup [claude]: Creating $CLAUDE_INTERNAL_JSON (trusting hooks)"
    cat > "$CLAUDE_INTERNAL_JSON" << 'EOF'
{
  "hasTrustDialogHooksAccepted": true,
  "hasCompletedOnboarding": true
}
EOF

    echo "Setup [claude]: Configured for API key authentication"
fi
