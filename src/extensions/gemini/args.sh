#!/bin/bash
# Gemini CLI argument transformer
# Transforms generic addt args to Gemini-specific args

ARGS=()

while [[ $# -gt 0 ]]; do
    case "$1" in
        --yolo)
            # Gemini natively supports --yolo, pass through
            ARGS+=(--yolo)
            shift
            ;;
        *)
            ARGS+=("$1")
            shift
            ;;
    esac
done

# If ADDT_EXTENSION_GEMINI_YOLO is set via config/env and --yolo
# wasn't already added by a CLI flag, inject it now
if [ "${ADDT_EXTENSION_GEMINI_YOLO}" = "true" ]; then
    already_set=false
    for arg in "${ARGS[@]}"; do
        if [ "$arg" = "--yolo" ]; then
            already_set=true
            break
        fi
    done
    if [ "$already_set" = "false" ]; then
        ARGS+=(--yolo)
    fi
fi

# Output transformed args (null-delimited to preserve multi-line values)
printf '%s\0' "${ARGS[@]}"
