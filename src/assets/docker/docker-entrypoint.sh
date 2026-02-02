#!/bin/bash
set -e

# Start Docker daemon if in DinD mode
if [ "$DCLAUDE_DIND" = "true" ]; then
    echo "Starting Docker daemon in isolated mode..."

    # Start dockerd in the background
    sudo dockerd --host=unix:///var/run/docker.sock >/tmp/docker.log 2>&1 &

    # Wait for Docker daemon to be ready
    echo "Waiting for Docker daemon..."
    for i in {1..30}; do
        if [ -S /var/run/docker.sock ]; then
            # Socket exists, fix permissions
            sudo chmod 666 /var/run/docker.sock
            if docker info >/dev/null 2>&1; then
                echo "✓ Docker daemon ready (isolated environment)"
                break
            fi
        fi
        if [ $i -eq 30 ]; then
            echo "Error: Docker daemon failed to start"
            cat /tmp/docker.log 2>/dev/null || echo "No log file available"
            exit 1
        fi
        sleep 1
    done
fi

# Initialize firewall if enabled
if [ "${DCLAUDE_FIREWALL_ENABLED}" = "true" ] && [ -f /usr/local/bin/init-firewall.sh ]; then
    sudo /usr/local/bin/init-firewall.sh
fi

# Run extension setup scripts (if not already run in this session)
EXTENSIONS_DIR="/usr/local/share/dclaude/extensions"
EXTENSIONS_JSON="$HOME/.dclaude/extensions.json"
SETUP_MARKER="$HOME/.dclaude/.setup-done"

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
export DCLAUDE_SYSTEM_PROMPT=""

if [ -n "$DCLAUDE_PORT_MAP" ]; then
    # Parse port mappings (format: "3000:30000,8080:30001")
    DCLAUDE_SYSTEM_PROMPT="# Port Mapping Information

When you start a service inside this container on certain ports, tell the user the correct HOST port to access it from their browser.

Port mappings (container→host):
"
    IFS=',' read -ra MAPPINGS <<< "$DCLAUDE_PORT_MAP"
    for mapping in "${MAPPINGS[@]}"; do
        IFS=':' read -ra PORTS <<< "$mapping"
        CONTAINER_PORT="${PORTS[0]}"
        HOST_PORT="${PORTS[1]}"
        DCLAUDE_SYSTEM_PROMPT+="- Container port $CONTAINER_PORT → Host port $HOST_PORT (user accesses: http://localhost:$HOST_PORT)
"
    done

    DCLAUDE_SYSTEM_PROMPT+="
IMPORTANT:
- When testing/starting services inside the container, use the container ports (e.g., http://localhost:3000)
- When telling the USER where to access services in their browser, use the HOST ports (e.g., http://localhost:30000)
- Always remind the user to use the host port in their browser"
fi

# Ensure ~/.local/bin and ~/go/bin are in PATH (for extensions installed there)
export PATH="$HOME/.local/bin:$HOME/go/bin:$PATH"

# Determine which command to run (default: claude)
DCLAUDE_CMD="${DCLAUDE_COMMAND:-claude}"

# Find the extension directory for args.sh
EXTENSIONS_DIR="/usr/local/share/dclaude/extensions"
ARGS_SCRIPT=""

# Look for args.sh in the extension matching the command
for ext_dir in "$EXTENSIONS_DIR"/*/; do
    if [ -f "$ext_dir/config.yaml" ]; then
        entrypoint=$(grep "^entrypoint:" "$ext_dir/config.yaml" 2>/dev/null | sed 's/^entrypoint:[[:space:]]*//' | tr -d '"')
        if [ "$entrypoint" = "$DCLAUDE_CMD" ] && [ -f "$ext_dir/args.sh" ]; then
            ARGS_SCRIPT="$ext_dir/args.sh"
            break
        fi
    fi
done

# Transform args through extension's args.sh if it exists
if [ -n "$ARGS_SCRIPT" ] && [ -f "$ARGS_SCRIPT" ]; then
    # Run args.sh and read transformed args (one per line)
    mapfile -t TRANSFORMED_ARGS < <(bash "$ARGS_SCRIPT" "$@")
    exec "$DCLAUDE_CMD" "${TRANSFORMED_ARGS[@]}"
else
    # No args.sh - pass args directly
    exec "$DCLAUDE_CMD" "$@"
fi
