#!/bin/bash
# Claude credentials extraction script
# Runs on HOST before container start, outputs KEY=value to stdout
# Handles token extraction and refresh from various sources

# Skip if ANTHROPIC_API_KEY already set
if [ -n "$ANTHROPIC_API_KEY" ]; then
    exit 0
fi

ACCESS_TOKEN=""

case "$(uname -s)" in
    Darwin)
        # macOS: Try keychain for Claude OAuth tokens
        SERVICE_NAME="Claude Code-credentials"

        # Try to get access token
        ACCESS_TOKEN_JSON=$(security find-generic-password -s "$SERVICE_NAME" -w 2>/dev/null || true)

        ACCESS_TOKEN=$(echo "$ACCESS_TOKEN_JSON" | python3 -c "import sys, json; d=json.load(sys.stdin); print(d.get('claudeAiOauth', {}).get('accessToken', ''))" 2>/dev/null || true)

        # if [ -n "$ACCESS_TOKEN" ]; then
        #     # Check if token is expired and needs refresh
        #     EXPIRES_AT=$(python3 -c "import json; print(json.load(open('$ACCESS_TOKEN_JSON')).get('expiresAt', ''))" 2>/dev/null || true)
        #     REFRESH_TOKEN=$(python3 -c "import json; print(json.load(open('$ACCESS_TOKEN_JSON')).get('refreshToken', ''))" 2>/dev/null || true)

        #     NOW=$(date +%s)
        #     # Refresh if expired or expiring within 5 minutes
        #     if [ -n "$EXPIRES_AT" ] && [ -n "$REFRESH_TOKEN" ] && [ "$NOW" -gt "$((EXPIRES_AT - 300))" ]; then
        #         # Attempt to refresh token
        #         RESPONSE=$(curl -s -X POST "https://api.anthropic.com/v1/oauth/token" \
        #             -H "Content-Type: application/x-www-form-urlencoded" \
        #             -d "grant_type=refresh_token&refresh_token=$REFRESH_TOKEN" 2>/dev/null || true)

        #         if echo "$RESPONSE" | grep -q '"access_token"'; then
        #             NEW_TOKEN=$(echo "$RESPONSE" | sed -n 's/.*"access_token"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p')
        #             NEW_EXPIRES=$(echo "$RESPONSE" | sed -n 's/.*"expires_in"[[:space:]]*:[[:space:]]*\([0-9]*\).*/\1/p')

        #             if [ -n "$NEW_TOKEN" ]; then
        #                 ACCESS_TOKEN="$NEW_TOKEN"

        #                 # Update keychain with new token
        #                 security add-generic-password -U -s "$SERVICE_NAME" -a "access_token" -w "$ACCESS_TOKEN" 2>/dev/null || true

        #                 if [ -n "$NEW_EXPIRES" ]; then
        #                     NEW_EXPIRES_AT=$((NOW + NEW_EXPIRES))
        #                     security add-generic-password -U -s "$SERVICE_NAME" -a "expires_at" -w "$NEW_EXPIRES_AT" 2>/dev/null || true
        #                 fi

        #                 echo "# Token refreshed" >&2
        #             fi
        #         fi
        #     fi
        # fi

        # Fallback: check for session key in ~/.claude credentials
        if [ -z "$ACCESS_TOKEN" ] && [ -f "$HOME/.claude/credentials.json" ]; then
            ACCESS_TOKEN=$(python3 -c "import json; print(json.load(open('$HOME/.claude/credentials.json')).get('sessionKey', ''))" 2>/dev/null || true)
        fi
        ;;

    Linux)
        # Linux: Try secret-tool (GNOME Keyring / KDE Wallet)
        #if command -v secret-tool &>/dev/null; then
        #    ACCESS_TOKEN=$(secret-tool lookup service claude.ai type access_token 2>/dev/null || true)
       # fi

        # Fallback: check credentials file
        if [ -z "$ACCESS_TOKEN" ] && [ -f "$HOME/.claude/credentials.json" ]; then
            ACCESS_TOKEN=$(python3 -c "import json; print(json.load(open('$HOME/.claude/credentials.json')).get('sessionKey', ''))" 2>/dev/null || true)
        fi
        ;;
esac

# Output the token if found
if [ -n "$ACCESS_TOKEN" ]; then
    echo "ANTHROPIC_API_KEY=$ACCESS_TOKEN"
fi
