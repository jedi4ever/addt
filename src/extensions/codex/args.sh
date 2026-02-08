#!/bin/bash
# OpenAI Codex argument transformer
# Transforms generic addt args to Codex-specific args

ARGS=()

while [[ $# -gt 0 ]]; do
    case "$1" in
        --yolo)
            # Transform generic --yolo to Codex's full-auto flag
            ARGS+=(--full-auto)
            shift
            ;;
        *)
            ARGS+=("$1")
            shift
            ;;
    esac
done

# If ADDT_EXTENSION_CODEX_YOLO is set via config/env and --full-auto
# wasn't already added by a --yolo CLI flag, inject it now
if [ "${ADDT_EXTENSION_CODEX_YOLO}" = "true" ]; then
    already_set=false
    for arg in "${ARGS[@]}"; do
        if [ "$arg" = "--full-auto" ]; then
            already_set=true
            break
        fi
    done
    if [ "$already_set" = "false" ]; then
        ARGS+=(--full-auto)
    fi
fi

# Output transformed args (null-delimited to preserve multi-line values)
printf '%s\0' "${ARGS[@]}"
