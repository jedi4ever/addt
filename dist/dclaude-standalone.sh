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
#   ./dclaude.sh --update  - Check for and install updates
# Examples:
#   ./dclaude.sh --help
#   ./dclaude.sh --version
#   ./dclaude.sh "Fix the bug in app.js"
#   ./dclaude.sh --model opus "Explain this codebase"

DCLAUDE_VERSION="1.0.0"

set -e

# Array to track temporary directories for cleanup
SSH_SAFE_DIRS=()

# Cleanup function
cleanup() {
    # Clean up embedded files
    [ -f "$SCRIPT_DIR/.dclaude-Dockerfile.tmp" ] && rm -f "$SCRIPT_DIR/.dclaude-Dockerfile.tmp"
    [ -f "$SCRIPT_DIR/docker-entrypoint.sh" ] && rm -f "$SCRIPT_DIR/docker-entrypoint.sh"
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

# Function to check if a port is available (using bash built-in, no external dependencies)
is_port_available() {
    local port=$1
    # Try to connect to port; if it succeeds, port is in use (return false)
    # Use read with timeout to avoid hanging
    (bash -c "exec 3<>/dev/tcp/localhost/$port" 2>/dev/null && exec 3>&-) && return 1 || return 0
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

# Function to check for and install updates
update_dclaude() {
    echo "Checking for updates..."
    echo "Current version: $DCLAUDE_VERSION"

    # Download latest version info
    LATEST_VERSION=$(curl -s https://raw.githubusercontent.com/jedi4ever/dclaude/main/VERSION 2>/dev/null || echo "")

    if [ -z "$LATEST_VERSION" ]; then
        echo "Error: Could not check for updates (network issue or repository unavailable)"
        exit 1
    fi

    echo "Latest version:  $LATEST_VERSION"

    # Compare versions
    if [ "$DCLAUDE_VERSION" = "$LATEST_VERSION" ]; then
        echo "✓ You are already on the latest version"
        exit 0
    fi

    # Prompt for update
    echo ""
    echo "New version available!"
    read -p "Update now? [Y/n] " -n 1 -r
    echo

    if [[ ! $REPLY =~ ^[Yy]$ ]] && [[ -n $REPLY ]]; then
        echo "Update cancelled"
        exit 0
    fi

    # Determine script path
    SCRIPT_PATH="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/$(basename "${BASH_SOURCE[0]}")"

    # Download new version
    echo "Downloading version $LATEST_VERSION..."
    TMP_FILE=$(mktemp)

    if ! curl -s -o "$TMP_FILE" https://raw.githubusercontent.com/jedi4ever/dclaude/main/dist/dclaude-standalone.sh; then
        echo "Error: Failed to download update"
        rm -f "$TMP_FILE"
        exit 1
    fi

    # Replace current script
    chmod +x "$TMP_FILE"
    if ! mv "$TMP_FILE" "$SCRIPT_PATH"; then
        echo "Error: Failed to replace script (permission denied?)"
        rm -f "$TMP_FILE"
        exit 1
    fi

    echo "✓ Updated to version $LATEST_VERSION"
    echo ""
    echo "Run dclaude again to use the new version"
    exit 0
}

# Check for special flags and commands
OPEN_SHELL=false
REBUILD_IMAGE=false

# Check for --update flag
if [ "$1" = "--update" ]; then
    update_dclaude
fi

# Check for --rebuild flag
if [ "$1" = "--rebuild" ]; then
    REBUILD_IMAGE=true
    shift  # Remove "--rebuild" from arguments
fi

# Check for "shell" command
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

# Check if curl is installed (needed for npm registry queries)
if ! command -v curl &> /dev/null; then
    echo "Error: curl is not installed"
    echo "Please install curl (usually: apt-get install curl, brew install curl, or yum install curl)"
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

# Handle --rebuild flag
if [ "$REBUILD_IMAGE" = true ]; then
    echo "Rebuilding $IMAGE_NAME..."
    if docker image inspect "$IMAGE_NAME" &> /dev/null; then
        echo "Removing existing image..."
        docker rmi "$IMAGE_NAME" &> /dev/null || true
    fi
fi

# Check if Docker image exists, build if needed
if ! docker image inspect "$IMAGE_NAME" &> /dev/null; then
    echo "Building $IMAGE_NAME..."

    # INJECT:Dockerfile-start
    # Embedded Dockerfile (generated by build.sh)
    DOCKERFILE_PATH="$SCRIPT_DIR/.dclaude-Dockerfile.tmp"
    cat > "$DOCKERFILE_PATH" <<'DOCKERFILE_EOF'
ARG NODE_VERSION=20
FROM node:${NODE_VERSION}-slim

# Build arguments for user ID and group ID
ARG USER_ID=1000
ARG GROUP_ID=1000
ARG USERNAME=claude
ARG CLAUDE_VERSION=latest

# Install dependencies, GitHub CLI, Docker CLI, and Docker daemon (for DinD)
RUN apt-get update && apt-get install -y \
    curl \
    gnupg \
    git \
    sudo \
    ripgrep \
    ca-certificates \
    iptables \
    supervisor \
    && curl -fsSL https://cli.github.com/packages/githubcli-archive-keyring.gpg | gpg --dearmor -o /usr/share/keyrings/githubcli-archive-keyring.gpg \
    && echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/githubcli-archive-keyring.gpg] https://cli.github.com/packages stable main" | tee /etc/apt/sources.list.d/github-cli.list > /dev/null \
    && curl -fsSL https://download.docker.com/linux/debian/gpg | gpg --dearmor -o /usr/share/keyrings/docker-archive-keyring.gpg \
    && echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/docker-archive-keyring.gpg] https://download.docker.com/linux/debian bookworm stable" | tee /etc/apt/sources.list.d/docker.list > /dev/null \
    && apt-get update \
    && apt-get install -y gh docker-ce-cli docker-ce containerd.io \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

# Install Claude Code globally (specific version or latest)
RUN if [ "$CLAUDE_VERSION" = "latest" ]; then \
        npm install -g @anthropic-ai/claude-code; \
    else \
        npm install -g @anthropic-ai/claude-code@$CLAUDE_VERSION; \
    fi

# Create user with matching UID/GID from host
# Note: GID 20 may already exist (staff/dialout group), so we just add user to that group
RUN (groupadd -g ${GROUP_ID} ${USERNAME} 2>/dev/null || true) \
    && useradd -m -u ${USER_ID} -g ${GROUP_ID} -s /bin/bash ${USERNAME} 2>/dev/null || usermod -u ${USER_ID} ${USERNAME} \
    && echo "${USERNAME} ALL=(ALL) NOPASSWD:ALL" >> /etc/sudoers

# Set working directory
WORKDIR /workspace

# Change ownership of workspace (use numeric IDs to avoid group name issues)
RUN chown -R ${USER_ID}:${GROUP_ID} /workspace

# Copy entrypoint wrapper for DinD support
COPY docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh
RUN chmod +x /usr/local/bin/docker-entrypoint.sh

# Switch to non-root user
USER ${USERNAME}

# Add version labels for tracking
LABEL org.opencontainers.image.title="dclaude"
LABEL org.opencontainers.image.description="Claude Code with Git, GitHub CLI, and Ripgrep"
LABEL org.opencontainers.image.authors="https://github.com/anthropics/claude-code"

# Tool version labels (populated at build time)
LABEL tools.claude="installed"
LABEL tools.git="installed"
LABEL tools.gh="installed"
LABEL tools.ripgrep="installed"
LABEL tools.node="installed"

# Entry point will be our wrapper script (handles DinD mode)
# Empty CMD means interactive session by default
ENTRYPOINT ["/usr/local/bin/docker-entrypoint.sh"]
CMD []
DOCKERFILE_EOF

    # Embedded entrypoint script (generated by build.sh)
    # Note: Must write to docker-entrypoint.sh for Dockerfile COPY to work
    ENTRYPOINT_PATH="$SCRIPT_DIR/docker-entrypoint.sh"
    cat > "$ENTRYPOINT_PATH" <<'ENTRYPOINT_EOF'
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

# Execute claude with system prompt (if any) and all user arguments
exec claude "${CLAUDE_ARGS[@]}" "$@"
ENTRYPOINT_EOF
    chmod +x "$ENTRYPOINT_PATH"

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
