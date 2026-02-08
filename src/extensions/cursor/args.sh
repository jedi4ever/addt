#!/bin/bash
# Cursor CLI argument transformer
# Transforms generic addt args to Cursor-specific args

ARGS=()

while [[ $# -gt 0 ]]; do
    case "$1" in
        --yolo)
            # Transform generic --yolo to Cursor's --force flag
            ARGS+=(--force)
            shift
            ;;
        *)
            ARGS+=("$1")
            shift
            ;;
    esac
done

# If ADDT_EXTENSION_CURSOR_YOLO is set via config/env and --force
# wasn't already added by a --yolo CLI flag, inject it now
if [ "${ADDT_EXTENSION_CURSOR_YOLO}" = "true" ]; then
    already_set=false
    for arg in "${ARGS[@]}"; do
        if [ "$arg" = "--force" ]; then
            already_set=true
            break
        fi
    done
    if [ "$already_set" = "false" ]; then
        ARGS+=(--force)
    fi
fi

# Output transformed args (null-delimited to preserve multi-line values)
printf '%s\0' "${ARGS[@]}"
