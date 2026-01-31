#!/bin/bash

# dclaude.sh - Wrapper script to run Claude Code in Docker container
# Usage: ./dclaude.sh [claude-options] [prompt]
# Special commands:
#   ./dclaude.sh shell  - Open bash shell in container
# Examples:
#   ./dclaude.sh --help
#   ./dclaude.sh --version
#   ./dclaude.sh "Fix the bug in app.js"
#   ./dclaude.sh --model opus "Explain this codebase"

set -e

IMAGE_NAME="dclaude:latest"

# Check for special "shell" command
OPEN_SHELL=false
if [ "$1" = "shell" ]; then
    OPEN_SHELL=true
    shift  # Remove "shell" from arguments
fi

# Load .env file if it exists
if [ -f .env ]; then
    set -a
    source .env
    set +a
fi

# Check if ANTHROPIC_API_KEY is set (not required for shell mode)
if [ "$OPEN_SHELL" = false ] && [ -z "$ANTHROPIC_API_KEY" ]; then
    echo "Error: ANTHROPIC_API_KEY environment variable is not set"
    echo "Please set it with: export ANTHROPIC_API_KEY='your-key'"
    echo "Or add it to your .env file"
    exit 1
fi

# Check if Docker image exists
if ! docker image inspect "$IMAGE_NAME" >/dev/null 2>&1; then
    echo "Error: Docker image '$IMAGE_NAME' not found"
    echo "Please build it first with: make build"
    exit 1
fi

# Build docker run command
DOCKER_CMD="docker run -it --rm"

# Mount current directory
DOCKER_CMD="$DOCKER_CMD -v $(pwd):/workspace"

# Mount .gitconfig for git identity
if [ -f "$HOME/.gitconfig" ]; then
    DOCKER_CMD="$DOCKER_CMD -v $HOME/.gitconfig:/home/$(whoami)/.gitconfig:ro"
fi

# Pass environment variables (if set)
if [ -n "$ANTHROPIC_API_KEY" ]; then
    DOCKER_CMD="$DOCKER_CMD -e ANTHROPIC_API_KEY=$ANTHROPIC_API_KEY"
fi

# Add GH_TOKEN if it's set
if [ -n "$GH_TOKEN" ]; then
    DOCKER_CMD="$DOCKER_CMD -e GH_TOKEN=$GH_TOKEN"
fi

# Handle shell mode or normal mode
if [ "$OPEN_SHELL" = true ]; then
    # Override entrypoint for shell mode
    echo "Opening bash shell in container..."
    DOCKER_CMD="$DOCKER_CMD --entrypoint /bin/bash $IMAGE_NAME"
    # Add any remaining arguments
    if [ $# -gt 0 ]; then
        DOCKER_CMD="$DOCKER_CMD $@"
    fi
else
    # Normal mode - run claude command
    DOCKER_CMD="$DOCKER_CMD $IMAGE_NAME"
    # Add all arguments passed to this script
    if [ $# -gt 0 ]; then
        DOCKER_CMD="$DOCKER_CMD $@"
    fi
fi

# Execute the command
eval $DOCKER_CMD
