#!/bin/bash
# Claude Code argument transformer
# Transforms generic dclaude args to Claude-specific args

ARGS=()

while [[ $# -gt 0 ]]; do
    case "$1" in
        --yolo)
            # Transform generic --yolo to Claude's flag
            ARGS+=(--dangerously-skip-permissions)
            shift
            ;;
        *)
            ARGS+=("$1")
            shift
            ;;
    esac
done

# Add system prompt if set (for port mappings, etc.)
if [ -n "$DCLAUDE_SYSTEM_PROMPT" ]; then
    ARGS+=(--append-system-prompt "$DCLAUDE_SYSTEM_PROMPT")
fi

# Output transformed args (one per line for proper handling)
printf '%s\n' "${ARGS[@]}"
