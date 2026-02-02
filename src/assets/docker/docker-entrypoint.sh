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

# Build system prompt for port mappings
CLAUDE_ARGS=()

if [ -n "$DCLAUDE_PORT_MAP" ]; then
    # Parse port mappings (format: "3000:30000,8080:30001")
    SYSTEM_PROMPT="# Port Mapping Information

When the user starts a service inside this container on certain ports, you need to tell them the correct HOST port to access it from their browser.

Port mappings (container→host):
"
    IFS=',' read -ra MAPPINGS <<< "$DCLAUDE_PORT_MAP"
    for mapping in "${MAPPINGS[@]}"; do
        IFS=':' read -ra PORTS <<< "$mapping"
        CONTAINER_PORT="${PORTS[0]}"
        HOST_PORT="${PORTS[1]}"
        SYSTEM_PROMPT+="- Container port $CONTAINER_PORT → Host port $HOST_PORT (user accesses: http://localhost:$HOST_PORT)
"
    done

    SYSTEM_PROMPT+="
IMPORTANT:
- When testing/starting services inside the container, use the container ports (e.g., http://localhost:3000)
- When telling the USER where to access services in their browser, use the HOST ports (e.g., http://localhost:30000)
- Always remind the user to use the host port in their browser"

    # Add the system prompt to claude arguments
    CLAUDE_ARGS+=(--append-system-prompt "$SYSTEM_PROMPT")
fi

# Determine which command to run (default: claude)
DCLAUDE_CMD="${DCLAUDE_COMMAND:-claude}"

# Execute command with system prompt (if any) and all user arguments
exec "$DCLAUDE_CMD" "${CLAUDE_ARGS[@]}" "$@"
