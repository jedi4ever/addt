#!/bin/bash
set -e

# Load secrets from files if secrets_to_files is enabled
# These are loaded here so they're available for auth but not inherited by subprocesses
if [ -n "$ADDT_SECRETS_DIR" ] && [ -d "$ADDT_SECRETS_DIR" ]; then
    for secret_file in "$ADDT_SECRETS_DIR"/*; do
        if [ -f "$secret_file" ]; then
            var_name=$(basename "$secret_file")
            export "$var_name"="$(cat "$secret_file")"
        fi
    done
fi

# Start nested Podman if in nested mode (Podman-in-Podman)
if [ "$ADDT_DIND" = "true" ]; then
    echo "Nested Podman mode enabled..."
    # Podman doesn't need a daemon - it's daemonless
    # Just verify podman is available
    if command -v podman >/dev/null 2>&1; then
        echo "Podman available for nested containers"
    else
        echo "Warning: Podman not available in container"
    fi
fi

# Initialize firewall if enabled
if [ "${ADDT_FIREWALL_ENABLED}" = "true" ] && [ -f /usr/local/bin/init-firewall.sh ]; then
    sudo /usr/local/bin/init-firewall.sh
fi

# Run extension setup scripts (if not already run in this session)
EXTENSIONS_DIR="/usr/local/share/addt/extensions"
EXTENSIONS_JSON="$HOME/.addt/extensions.json"
SETUP_MARKER="$HOME/.addt/.setup-done"

if [ -f "$EXTENSIONS_JSON" ] && [ ! -f "$SETUP_MARKER" ]; then
    # Extract extension names from JSON
    extensions=$(grep -oE '"[a-z]+":' "$EXTENSIONS_JSON" | tr -d '":' | sort -u)

    for ext in $extensions; do
        setup_script="$EXTENSIONS_DIR/$ext/setup.sh"
        if [ -f "$setup_script" ]; then
            echo "Running setup for extension: $ext"
            bash "$setup_script" || echo "Warning: setup.sh for $ext failed"
        fi
    done

    # Mark setup as done for this session
    touch "$SETUP_MARKER"
fi

# Build system prompt for port mappings (exported for args.sh to use)
export ADDT_SYSTEM_PROMPT=""

if [ -n "$ADDT_PORT_MAP" ]; then
    # Parse port mappings (format: "3000:30000,8080:30001")
    ADDT_SYSTEM_PROMPT="# Port Mapping Information

When you start a service inside this container on certain ports, tell the user the correct HOST port to access it from their browser.

Port mappings (container→host):
"
    IFS=',' read -ra MAPPINGS <<< "$ADDT_PORT_MAP"
    for mapping in "${MAPPINGS[@]}"; do
        IFS=':' read -ra PORTS <<< "$mapping"
        CONTAINER_PORT="${PORTS[0]}"
        HOST_PORT="${PORTS[1]}"
        ADDT_SYSTEM_PROMPT+="- Container port $CONTAINER_PORT → Host port $HOST_PORT (user accesses: http://localhost:$HOST_PORT)
"
    done

    ADDT_SYSTEM_PROMPT+="
IMPORTANT:
- When testing/starting services inside the container, use the container ports (e.g., http://localhost:3000)
- When telling the USER where to access services in their browser, use the HOST ports (e.g., http://localhost:30000)
- Always remind the user to use the host port in their browser"
fi

# Ensure ~/.local/bin and ~/go/bin are in PATH (for extensions installed there)
export PATH="$HOME/.local/bin:$HOME/go/bin:$PATH"

# Determine which command to run (entrypoint can be array: ["bash", "-i"])
ADDT_CMD=""
ADDT_CMD_ARGS=()

if [ -n "$ADDT_COMMAND" ]; then
    ADDT_CMD="$ADDT_COMMAND"
elif [ -f "$EXTENSIONS_JSON" ]; then
    # Auto-detect from first installed extension
    # Entrypoint is now a JSON array, e.g., ["bash","-i"] or ["claude"]
    entrypoint_json=$(grep -oE '"entrypoint":[[:space:]]*\[[^]]*\]' "$EXTENSIONS_JSON" | head -1 | sed 's/.*"entrypoint":[[:space:]]*//')

    if [ -n "$entrypoint_json" ]; then
        # Parse JSON array: ["cmd", "arg1", "arg2"] -> cmd and args
        # Remove brackets and quotes, split by comma
        entrypoint_clean=$(echo "$entrypoint_json" | tr -d '[]"' | sed 's/,/ /g')
        read -ra entrypoint_parts <<< "$entrypoint_clean"

        if [ ${#entrypoint_parts[@]} -gt 0 ]; then
            ADDT_CMD="${entrypoint_parts[0]}"
            ADDT_CMD_ARGS=("${entrypoint_parts[@]:1}")
        fi
    fi
fi

# Fallback to claude if still not set
ADDT_CMD="${ADDT_CMD:-claude}"

# Find the extension directory for args.sh
EXTENSIONS_DIR="/usr/local/share/addt/extensions"
ARGS_SCRIPT=""

# Look for args.sh in the extension matching the command
for ext_dir in "$EXTENSIONS_DIR"/*/; do
    if [ -f "$ext_dir/config.yaml" ]; then
        # Get entrypoint command (first element if array)
        ep_line=$(grep "^entrypoint:" "$ext_dir/config.yaml" 2>/dev/null)
        if [[ "$ep_line" =~ \[ ]]; then
            # Array format - extract first element
            entrypoint=$(echo "$ep_line" | sed 's/^entrypoint:[[:space:]]*//' | tr -d '[]"' | cut -d',' -f1 | xargs)
        else
            # String format
            entrypoint=$(echo "$ep_line" | sed 's/^entrypoint:[[:space:]]*//' | tr -d '"')
        fi

        if [ "$entrypoint" = "$ADDT_CMD" ] && [ -f "$ext_dir/args.sh" ]; then
            ARGS_SCRIPT="$ext_dir/args.sh"
            break
        fi
    fi
done

# Transform args through extension's args.sh if it exists
if [ -n "$ARGS_SCRIPT" ] && [ -f "$ARGS_SCRIPT" ]; then
    # Run args.sh and read transformed args (one per line)
    mapfile -t TRANSFORMED_ARGS < <(bash "$ARGS_SCRIPT" "$@")
    FINAL_ARGS=("${ADDT_CMD_ARGS[@]}" "${TRANSFORMED_ARGS[@]}")
else
    # No args.sh - pass args directly
    FINAL_ARGS=("${ADDT_CMD_ARGS[@]}" "$@")
fi

# Execute with optional time limit
if [ -n "$ADDT_TIME_LIMIT_SECONDS" ] && [ "$ADDT_TIME_LIMIT_SECONDS" -gt 0 ]; then
    echo "Time limit: $((ADDT_TIME_LIMIT_SECONDS / 60)) minutes"
    exec timeout --signal=TERM "$ADDT_TIME_LIMIT_SECONDS" "$ADDT_CMD" "${FINAL_ARGS[@]}"
else
    exec "$ADDT_CMD" "${FINAL_ARGS[@]}"
fi
