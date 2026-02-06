#!/bin/bash
# Claude credentials extraction script
# Runs on HOST before container start, outputs KEY=value to stdout
# Handles token extraction and refresh from various sources

# Skip if ANTHROPIC_API_KEY already set
if [ -n "$ANTHROPIC_API_KEY" ]; then
    exit 0
fi

CLAUDE_OAUTH_CREDENTIALS=""

case "$(uname -s)" in
    Darwin)
        # macOS: Try keychain for Claude OAuth tokens
        SERVICE_NAME="Claude Code-credentials"

        # Try to get access token from keychain
        CLAUDE_OAUTH_CREDENTIALS_JSON=$(security find-generic-password -s "$SERVICE_NAME" -w 2>/dev/null || true)

        # Fallback: check for session key in ~/.claude credentials
        if [ -z "$CLAUDE_OAUTH_CREDENTIALS_JSON" ] && [ -f "$HOME/.claude/credentials.json" ]; then
            CLAUDE_OAUTH_CREDENTIALS_JSON=$(python3 -c "import json; print(json.load(open('$HOME/.claude/credentials.json')).get('sessionKey', ''))" 2>/dev/null || true)
        fi


        ;;

    Linux)
        # Linux: Try secret-tool (GNOME Keyring / KDE Wallet)
        #if command -v secret-tool &>/dev/null; then
        #    ACCESS_TOKEN=$(secret-tool lookup service claude.ai type access_token 2>/dev/null || true)
       # fi

        # Fallback: check credentials file
        if [ -z "$CLAUDE_OAUTH_CREDENTIALS_JSON" ] && [ -f "$HOME/.claude/credentials.json" ]; then
            CLAUDE_OAUTH_CREDENTIALS_JSON=$(python3 -c "import json; print(json.load(open('$HOME/.claude/credentials.json')).get('sessionKey', ''))" 2>/dev/null || true)
        fi
        ;;
esac

# If we have the credentials, base64 encode them and output them
if [ -n "$CLAUDE_OAUTH_CREDENTIALS_JSON" ]; then
    # base64 decode the access token json
    CLAUDE_OAUTH_CREDENTIALS_JSON=$(echo "$CLAUDE_OAUTH_CREDENTIALS_JSON" | base64)
    CLAUDE_OAUTH_CREDENTIALS=$(echo "$CLAUDE_OAUTH_CREDENTIALS_JSON")
    echo "CLAUDE_OAUTH_CREDENTIALS=$CLAUDE_OAUTH_CREDENTIALS"
fi