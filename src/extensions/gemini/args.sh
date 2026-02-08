#!/bin/bash
# Gemini CLI argument transformer
# Transforms generic addt args to Gemini-specific args

ARGS=()
YOLO=false

while [[ $# -gt 0 ]]; do
    case "$1" in
        --yolo)
            YOLO=true
            shift
            ;;
        *)
            ARGS+=("$1")
            shift
            ;;
    esac
done

# Enable yolo from any source: CLI flag, per-extension env, or global security.yolo
if [ "$YOLO" = "true" ] || [ "${ADDT_EXTENSION_GEMINI_YOLO}" = "true" ] || [ "${ADDT_SECURITY_YOLO}" = "true" ]; then
    ARGS+=(--yolo)
fi

# Output transformed args (null-delimited to preserve multi-line values)
printf '%s\0' "${ARGS[@]}"
