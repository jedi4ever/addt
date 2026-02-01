#!/bin/bash

# dclaude.sh - Wrapper script to run Claude Code in Docker container
#
# Build system placeholders:
# - INJECT:Dockerfile will be replaced with Dockerfile content
# - INJECT:docker-entrypoint.sh will be replaced with entrypoint script
#
# To build standalone version: make standalone
# Usage: ./dclaude.sh [claude-options] [prompt]
# Special commands:
#   ./dclaude.sh shell  - Open bash shell in container
# Examples:
#   ./dclaude.sh --help
#   ./dclaude.sh --version
#   ./dclaude.sh "Fix the bug in app.js"
#   ./dclaude.sh --model opus "Explain this codebase"


set -e

# Array to track temporary directories for cleanup
SSH_SAFE_DIRS=()

# Cleanup function
cleanup() {
    for dir in "${SSH_SAFE_DIRS[@]}"; do
        [ -d "$dir" ] && rm -rf "$dir"
    done
}
trap cleanup EXIT

# Default to latest Claude Code version, or use specified version
DCLAUDE_CLAUDE_VERSION="${DCLAUDE_CLAUDE_VERSION:-latest}"
# Default to Node 20, or use specified version (can be "20", "lts", "current", etc.)
DCLAUDE_NODE_VERSION="${DCLAUDE_NODE_VERSION:-20}"
# Default environment variables to pass (comma-separated list)
DCLAUDE_ENV_VARS="${DCLAUDE_ENV_VARS:-ANTHROPIC_API_KEY,GH_TOKEN}"
# Auto-detect GitHub token from gh CLI (default: false - opt-in)
DCLAUDE_GITHUB_DETECT="${DCLAUDE_GITHUB_DETECT:-false}"
# Port mappings (comma-separated list of container ports to expose)
DCLAUDE_PORTS="${DCLAUDE_PORTS:-}"
# Port range start for automatic allocation (default: 30000)
DCLAUDE_PORT_RANGE_START="${DCLAUDE_PORT_RANGE_START:-30000}"
IMAGE_NAME="dclaude:latest"

# Get the directory where this script is located
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Logging configuration (disabled by default, enable with DCLAUDE_LOG=true)
DCLAUDE_LOG="${DCLAUDE_LOG:-false}"
DCLAUDE_LOG_FILE="${DCLAUDE_LOG_FILE:-$SCRIPT_DIR/dclaude.log}"

# Function to log commands
log_command() {
    if [ "$DCLAUDE_LOG" = "true" ]; then
        local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
        echo "[$timestamp] $*" >> "$DCLAUDE_LOG_FILE"
    fi
}

# Function to check if a port is available
is_port_available() {
    local port=$1
    ! nc -z localhost "$port" 2>/dev/null
}

# Function to find next available port starting from a base
find_available_port() {
    local start_port=$1
    local port=$start_port
    while ! is_port_available "$port"; do
        port=$((port + 1))
    done
    echo "$port"
}

# Check for special "shell" command
OPEN_SHELL=false
if [ "$1" = "shell" ]; then
    OPEN_SHELL=true
    shift  # Remove "shell" from arguments
fi

# Check if Docker is installed
if ! command -v docker &> /dev/null; then
    echo "Error: Docker is not installed"
    echo "Please install Docker from: https://docs.docker.com/get-docker/"
    exit 1
fi

# Check if Docker daemon is running
if ! docker info &> /dev/null; then
    echo "Error: Docker daemon is not running"
    echo "Please start Docker and try again"
    exit 1
fi

# Check if we need a specific Claude Code version or latest from npm registry
if [ "$DCLAUDE_CLAUDE_VERSION" = "latest" ]; then
    # Query npm registry directly via HTTP (faster than npm CLI)
    NPM_LATEST=$(curl -s https://registry.npmjs.org/@anthropic-ai/claude-code 2>/dev/null | grep -o '"stable":"[^"]*"' | cut -d'"' -f4)

    if [ -n "$NPM_LATEST" ]; then
        # Check if we already have an image with this version (exclude dangling images)
        EXISTING_IMAGE=$(docker images --filter "label=tools.claude.version=$NPM_LATEST" --format "{{.Repository}}:{{.Tag}}" | grep -v "<none>" | head -1)

        if [ -n "$EXISTING_IMAGE" ]; then
            IMAGE_NAME="$EXISTING_IMAGE"
        else
            DCLAUDE_CLAUDE_VERSION="$NPM_LATEST"
            IMAGE_NAME="dclaude:claude-$NPM_LATEST"
        fi
    fi
else
    # Specific version requested - validate it exists in npm registry
    NPM_DATA=$(curl -s https://registry.npmjs.org/@anthropic-ai/claude-code 2>/dev/null)
    if ! echo "$NPM_DATA" | grep -q "\"$DCLAUDE_CLAUDE_VERSION\":"; then
        echo "Error: Claude Code version $DCLAUDE_CLAUDE_VERSION does not exist in npm"
        echo "Available versions: https://www.npmjs.com/package/@anthropic-ai/claude-code?activeTab=versions"
        echo "Hint: Use 'latest' or check available versions at the link above"
        exit 1
    fi

    # Check if an image with this Claude version already exists (exclude dangling images)
    EXISTING_IMAGE=$(docker images --filter "label=tools.claude.version=$DCLAUDE_CLAUDE_VERSION" --format "{{.Repository}}:{{.Tag}}" | grep -v "<none>" | head -1)

    if [ -n "$EXISTING_IMAGE" ]; then
        IMAGE_NAME="$EXISTING_IMAGE"
    else
        IMAGE_NAME="dclaude:claude-$DCLAUDE_CLAUDE_VERSION"
    fi
fi

# Check if Docker image exists, build if needed
if ! docker image inspect "$IMAGE_NAME" &> /dev/null; then
    echo "Building $IMAGE_NAME..."

    # INJECT:Dockerfile-start
    # In standalone mode, Dockerfile will be embedded here as heredoc
    # For development, use external Dockerfile
    DOCKERFILE_PATH="$SCRIPT_DIR/Dockerfile"
    ENTRYPOINT_PATH="$SCRIPT_DIR/docker-entrypoint.sh"
    # INJECT:Dockerfile-end

    # Build the image with user's UID/GID, Node version, and Claude version
    if docker build \
        --build-arg NODE_VERSION="$DCLAUDE_NODE_VERSION" \
        --build-arg USER_ID=$(id -u) \
        --build-arg GROUP_ID=$(id -g) \
        --build-arg USERNAME=$(whoami) \
        --build-arg CLAUDE_VERSION="$DCLAUDE_CLAUDE_VERSION" \
        -t "$IMAGE_NAME" -f "$DOCKERFILE_PATH" "$SCRIPT_DIR"; then
        echo ""
        echo "✓ Image built successfully!"
        echo ""
        echo "Detecting tool versions..."

        # Get versions from the built image
        CLAUDE_VERSION=$(docker run --rm --entrypoint claude "$IMAGE_NAME" --version 2>/dev/null | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -1)
        GH_VERSION=$(docker run --rm --entrypoint gh "$IMAGE_NAME" --version 2>/dev/null | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -1)
        RG_VERSION=$(docker run --rm --entrypoint rg "$IMAGE_NAME" --version 2>/dev/null | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -1)
        GIT_VERSION=$(docker run --rm --entrypoint git "$IMAGE_NAME" --version 2>/dev/null | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -1)
        NODE_VERSION_ACTUAL=$(docker run --rm --entrypoint node "$IMAGE_NAME" --version 2>/dev/null | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -1)

        # Create a temporary Dockerfile to add version labels
        TEMP_DOCKERFILE=$(mktemp)
        cat > "$TEMP_DOCKERFILE" << EOF
FROM $IMAGE_NAME
LABEL tools.claude.version="$CLAUDE_VERSION"
LABEL tools.gh.version="$GH_VERSION"
LABEL tools.ripgrep.version="$RG_VERSION"
LABEL tools.git.version="$GIT_VERSION"
LABEL tools.node.version="$NODE_VERSION_ACTUAL"
EOF

        # Rebuild with version labels (fast - just adds metadata layer)
        echo "Adding version labels..."
        if docker build -f "$TEMP_DOCKERFILE" -t "$IMAGE_NAME" "$SCRIPT_DIR" &> /dev/null; then
            # Also tag as dclaude:latest if we built the latest version
            if [ "$DCLAUDE_CLAUDE_VERSION" = "latest" ]; then
                docker tag "$IMAGE_NAME" "dclaude:latest" &> /dev/null
            fi
            # Tag with claude version for easy reference
            if [ -n "$CLAUDE_VERSION" ]; then
                docker tag "$IMAGE_NAME" "dclaude:claude-$CLAUDE_VERSION" &> /dev/null
            fi

            rm "$TEMP_DOCKERFILE"
            echo ""
            echo "Installed versions:"
            [ -n "$NODE_VERSION_ACTUAL" ] && echo "  • Node.js:     $NODE_VERSION_ACTUAL"
            [ -n "$CLAUDE_VERSION" ] && echo "  • Claude Code: $CLAUDE_VERSION"
            [ -n "$GH_VERSION" ] && echo "  • GitHub CLI:  $GH_VERSION"
            [ -n "$RG_VERSION" ] && echo "  • Ripgrep:     $RG_VERSION"
            [ -n "$GIT_VERSION" ] && echo "  • Git:         $GIT_VERSION"
            echo ""
            echo "Image tagged as: $IMAGE_NAME"
            [ "$DCLAUDE_CLAUDE_VERSION" = "latest" ] && echo "Also tagged as: dclaude:latest"
            [ -n "$CLAUDE_VERSION" ] && echo "Also tagged as: dclaude:claude-$CLAUDE_VERSION"
            echo ""
            echo "View labels: docker inspect $IMAGE_NAME --format '{{json .Config.Labels}}' | jq"
            echo ""
        else
            rm "$TEMP_DOCKERFILE"
            echo "Warning: Could not add version labels, but image is functional"
            echo ""
        fi
    else
        echo ""
        echo "Error: Failed to build Docker image"
        echo "Please check the Dockerfile and try again"
        exit 1
    fi
fi

# Auto-detect GitHub token from gh CLI if enabled and not already set
if [ "$DCLAUDE_GITHUB_DETECT" = "true" ] && [ -z "$GH_TOKEN" ]; then
    if command -v gh &> /dev/null; then
        GH_TOKEN_FROM_CLI=$(gh auth token 2>/dev/null)
        if [ -n "$GH_TOKEN_FROM_CLI" ]; then
            export GH_TOKEN="$GH_TOKEN_FROM_CLI"
        fi
    fi
fi

# Prepare env file for Docker
# Use DCLAUDE_ENV_FILE if specified, otherwise default to .env
ENV_FILE="${DCLAUDE_ENV_FILE:-.env}"
ENV_FILE_FOR_DOCKER=""

if [ -f "$ENV_FILE" ]; then
    # Use absolute path for Docker --env-file
    if [[ "$ENV_FILE" = /* ]]; then
        ENV_FILE_FOR_DOCKER="$ENV_FILE"
    else
        ENV_FILE_FOR_DOCKER="$(pwd)/$ENV_FILE"
    fi
    # Also source it for script variables like DCLAUDE_*
    set -a
    source "$ENV_FILE"
    set +a
elif [ -n "$DCLAUDE_ENV_FILE" ]; then
    echo "Warning: Specified env file not found: $ENV_FILE"
fi

# Check if ANTHROPIC_API_KEY is set (not required for shell mode)
#if [ "$OPEN_SHELL" = false ] && [ -z "$ANTHROPIC_API_KEY" ]; then
#    echo "Error: ANTHROPIC_API_KEY environment variable is not set"
#    echo "Please set it with: export ANTHROPIC_API_KEY='your-key'"
#    echo "Or add it to your .env file"
#    exit 1
#fi

# Generate unique container name
CONTAINER_NAME="dclaude-$(date +%Y%m%d-%H%M%S)-$$"

# Build docker run command using array for proper argument escaping
DOCKER_CMD=(docker run --rm --name "$CONTAINER_NAME")

# Detect if we're running in an interactive terminal
if [ -t 0 ] && [ -t 1 ]; then
    DOCKER_CMD+=(-it)
else
    DOCKER_CMD+=(-i)
fi

# Mount current directory
DOCKER_CMD+=(-v "$(pwd):/workspace")

# Add env file if it exists
if [ -n "$ENV_FILE_FOR_DOCKER" ]; then
    DOCKER_CMD+=(--env-file "$ENV_FILE_FOR_DOCKER")
fi

# Mount .gitconfig for git identity
if [ -f "$HOME/.gitconfig" ]; then
    DOCKER_CMD+=(-v "$HOME/.gitconfig:/home/$(whoami)/.gitconfig:ro")
fi

# Mount .claude directory for session persistence
if [ -d "$HOME/.claude" ]; then
    DOCKER_CMD+=(-v "$HOME/.claude:/home/$(whoami)/.claude")
fi

# Mount .claude.json file for configuration persistence
if [ -f "$HOME/.claude.json" ]; then
    DOCKER_CMD+=(-v "$HOME/.claude.json:/home/$(whoami)/.claude.json")
fi

# Mount .gnupg directory for GPG commit signing support (opt-in)
if [ "$DCLAUDE_GPG_FORWARD" = "true" ] && [ -d "$HOME/.gnupg" ]; then
    DOCKER_CMD+=(-v "$HOME/.gnupg:/home/$(whoami)/.gnupg")
    DOCKER_CMD+=(-e "GPG_TTY=/dev/console")
fi

# SSH forwarding (opt-in with configurable security levels)
if [ "$DCLAUDE_SSH_FORWARD" = "agent" ] || [ "$DCLAUDE_SSH_FORWARD" = "true" ]; then
    # Agent mode: Forward SSH agent socket (works well on Linux)
    if [ -n "$SSH_AUTH_SOCK" ] && [ -S "$SSH_AUTH_SOCK" ]; then
        # Check if socket is accessible (macOS launchd sockets won't work)
        if [[ "$SSH_AUTH_SOCK" =~ /private/tmp/com.apple.launchd ]] || [[ "$SSH_AUTH_SOCK" =~ /var/folders/.*/T/com.apple.launchd ]]; then
            echo "Warning: SSH agent forwarding not supported on macOS (use DCLAUDE_SSH_FORWARD=keys)"
        else
            DOCKER_CMD+=(-v "$SSH_AUTH_SOCK:/ssh-agent")
            DOCKER_CMD+=(-e "SSH_AUTH_SOCK=/ssh-agent")

            # Mount only public keys and config (not private keys)
            if [ -d "$HOME/.ssh" ]; then
                SSH_SAFE_DIR=$(mktemp -d)
                [ -f "$HOME/.ssh/config" ] && cp "$HOME/.ssh/config" "$SSH_SAFE_DIR/"
                [ -f "$HOME/.ssh/known_hosts" ] && cp "$HOME/.ssh/known_hosts" "$SSH_SAFE_DIR/"
                cp "$HOME/.ssh"/*.pub "$SSH_SAFE_DIR/" 2>/dev/null || true
                DOCKER_CMD+=(-v "$SSH_SAFE_DIR:/home/$(whoami)/.ssh:ro")
                SSH_SAFE_DIRS+=("$SSH_SAFE_DIR")
            fi
        fi
    fi
elif [ "$DCLAUDE_SSH_FORWARD" = "keys" ]; then
    # Keys mode: Mount entire .ssh directory
    if [ -d "$HOME/.ssh" ]; then
        DOCKER_CMD+=(-v "$HOME/.ssh:/home/$(whoami)/.ssh:ro")
    fi
fi

# Mount Docker socket for Docker-in-Docker support (opt-in)
if [ "$DCLAUDE_DOCKER_FORWARD" = "host" ]; then
    # Host mode: Mount host Docker socket (see all host containers)
    if [ -e "/var/run/docker.sock" ] || [ -L "/var/run/docker.sock" ]; then
        DOCKER_CMD+=(-v "/var/run/docker.sock:/var/run/docker.sock")

        # Dynamically detect Docker socket group ID
        DOCKER_SOCK_GID=$(stat -f "%g" /var/run/docker.sock 2>/dev/null || stat -c "%g" /var/run/docker.sock 2>/dev/null)
        if [ -n "$DOCKER_SOCK_GID" ]; then
            # Add the actual socket group, plus common fallback GIDs
            DOCKER_CMD+=(--group-add "$DOCKER_SOCK_GID")
            # Also add 102 (Rancher Desktop) and 999 (standard Docker) as fallbacks if different
            [ "$DOCKER_SOCK_GID" != "102" ] && DOCKER_CMD+=(--group-add 102)
            [ "$DOCKER_SOCK_GID" != "999" ] && DOCKER_CMD+=(--group-add 999)
        else
            echo "Warning: Could not detect Docker socket group, using common defaults"
            DOCKER_CMD+=(--group-add 102 --group-add 999)
        fi
    else
        echo "Warning: DCLAUDE_DOCKER_FORWARD=host but /var/run/docker.sock not found"
    fi
elif [ "$DCLAUDE_DOCKER_FORWARD" = "isolated" ] || [ "$DCLAUDE_DOCKER_FORWARD" = "true" ]; then
    # Isolated mode: Run separate Docker daemon (Docker-in-Docker)
    # Requires privileged mode for the daemon
    DOCKER_CMD+=(--privileged)
    # Mount a volume for the Docker daemon's data
    DOCKER_CMD+=(-v "dclaude-docker-${CONTAINER_NAME}:/var/lib/docker")
    # Set environment variable to signal DinD mode
    DOCKER_CMD+=(-e "DCLAUDE_DIND=true")
fi

# Handle port mappings if specified
PORT_MAP_STRING=""
PORT_MAP_DISPLAY=""
if [ -n "$DCLAUDE_PORTS" ]; then
    IFS=',' read -ra PORT_ARRAY <<< "$DCLAUDE_PORTS"
    HOST_PORT=$DCLAUDE_PORT_RANGE_START
    PORT_MAPPINGS=()

    for container_port in "${PORT_ARRAY[@]}"; do
        # Trim whitespace
        container_port=$(echo "$container_port" | xargs)

        # Find next available host port
        HOST_PORT=$(find_available_port "$HOST_PORT")

        # Add port mapping to docker command
        DOCKER_CMD+=(-p "$HOST_PORT:$container_port")

        # Track mapping for environment variable and display
        PORT_MAPPINGS+=("$container_port:$HOST_PORT")

        # Move to next port for next iteration
        HOST_PORT=$((HOST_PORT + 1))
    done

    # Build comma-separated string for environment variable
    PORT_MAP_STRING=$(IFS=','; echo "${PORT_MAPPINGS[*]}")

    # Build display string
    PORT_MAP_DISPLAY=$(echo "$PORT_MAP_STRING" | sed 's/:/→/g')
fi

# Pass port mapping info to container
if [ -n "$PORT_MAP_STRING" ]; then
    DOCKER_CMD+=(-e "DCLAUDE_PORT_MAP=$PORT_MAP_STRING")
fi

# Pass environment variables specified in DCLAUDE_ENV_VARS
IFS=',' read -ra ENV_VAR_ARRAY <<< "$DCLAUDE_ENV_VARS"
for var_name in "${ENV_VAR_ARRAY[@]}"; do
    # Trim whitespace
    var_name=$(echo "$var_name" | xargs)
    # Get the value of the variable
    var_value="${!var_name}"
    # Add to docker command if set
    if [ -n "$var_value" ]; then
        DOCKER_CMD+=(-e "$var_name=$var_value")
    fi
done

# Build and display concise status line
build_status_line() {
    local status=""

    # Image name (includes Claude version in tag)
    status="$IMAGE_NAME"

    # Node version (from image labels)
    NODE_VERSION=$(docker inspect "$IMAGE_NAME" --format '{{index .Config.Labels "tools.node.version"}}' 2>/dev/null)
    [ -n "$NODE_VERSION" ] && status="$status | Node ${NODE_VERSION}"

    # GitHub token status
    if [ -n "$GH_TOKEN" ]; then
        status="$status | GH:✓"
    else
        status="$status | GH:-"
    fi

    # SSH forwarding status
    if [ "$DCLAUDE_SSH_FORWARD" = "agent" ]; then
        status="$status | SSH:agent"
    elif [ "$DCLAUDE_SSH_FORWARD" = "keys" ]; then
        status="$status | SSH:keys"
    else
        status="$status | SSH:-"
    fi

    # GPG forwarding status
    if [ "$DCLAUDE_GPG_FORWARD" = "true" ]; then
        status="$status | GPG:✓"
    else
        status="$status | GPG:-"
    fi

    # Docker forwarding status
    if [ "$DCLAUDE_DOCKER_FORWARD" = "isolated" ] || [ "$DCLAUDE_DOCKER_FORWARD" = "true" ]; then
        status="$status | Docker:isolated"
    elif [ "$DCLAUDE_DOCKER_FORWARD" = "host" ]; then
        status="$status | Docker:host"
    else
        status="$status | Docker:-"
    fi

    # Port mappings
    if [ -n "$PORT_MAP_DISPLAY" ]; then
        status="$status | Ports:$PORT_MAP_DISPLAY"
    fi

    echo "✓ $status"
}

# Display status line
build_status_line

# Handle shell mode or normal mode
if [ "$OPEN_SHELL" = true ]; then
    echo "Opening bash shell in container..."
    # If using isolated Docker mode, we need to start dockerd first
    if [ "$DCLAUDE_DOCKER_FORWARD" = "isolated" ] || [ "$DCLAUDE_DOCKER_FORWARD" = "true" ]; then
        # Use a wrapper that starts dockerd then opens shell
        DOCKER_CMD+=("$IMAGE_NAME" /bin/bash -c "
            if [ \"\$DCLAUDE_DIND\" = \"true\" ]; then
                echo 'Starting Docker daemon in isolated mode...'
                sudo dockerd --host=unix:///var/run/docker.sock >/tmp/docker.log 2>&1 &
                echo 'Waiting for Docker daemon...'
                for i in {1..30}; do
                    if [ -S /var/run/docker.sock ]; then
                        sudo chmod 666 /var/run/docker.sock
                        if docker info >/dev/null 2>&1; then
                            echo '✓ Docker daemon ready (isolated environment)'
                            break
                        fi
                    fi
                    sleep 1
                done
            fi
            exec /bin/bash \"\$@\"
        " bash "$@")
    else
        # Normal shell mode without DinD
        DOCKER_CMD+=(--entrypoint /bin/bash "$IMAGE_NAME")
        DOCKER_CMD+=("$@")
    fi
else
    # Normal mode - run claude command (entrypoint handles DinD if needed)
    DOCKER_CMD+=("$IMAGE_NAME")
    DOCKER_CMD+=("$@")
fi

# Log the command
log_command "PWD: $(pwd) | Container: $CONTAINER_NAME | Command: $@"

# Execute the command with proper argument escaping
"${DOCKER_CMD[@]}"
