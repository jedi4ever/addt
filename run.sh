#!/bin/bash

# run.sh - Script to run Claude Code in Docker container

set -e

IMAGE_NAME="dclaude:latest"

# Check if ANTHROPIC_API_KEY is set
if [ -z "$ANTHROPIC_API_KEY" ]; then
    echo "Error: ANTHROPIC_API_KEY environment variable is not set"
    echo "Please set it with: export ANTHROPIC_API_KEY='your-key'"
    exit 1
fi

# Check if Docker image exists
if ! docker image inspect "$IMAGE_NAME" >/dev/null 2>&1; then
    echo "Error: Docker image '$IMAGE_NAME' not found"
    echo "Please build it first with: make build"
    exit 1
fi

# Run Docker container with Claude Code
# - Interactive mode with TTY
# - Remove container after exit
# - Mount current directory to /workspace
# - Mount .gitconfig for git identity
# - Pass through ANTHROPIC_API_KEY and GH_TOKEN (if set)
# - Forward all script arguments to claude command

# Build docker run command with conditional GH_TOKEN
DOCKER_CMD="docker run -it --rm \
    -v $(pwd):/workspace \
    -v $HOME/.gitconfig:/root/.gitconfig:ro \
    -e ANTHROPIC_API_KEY=$ANTHROPIC_API_KEY"

# Add GH_TOKEN if it's set
if [ -n "$GH_TOKEN" ]; then
    DOCKER_CMD="$DOCKER_CMD -e GH_TOKEN=$GH_TOKEN"
fi

DOCKER_CMD="$DOCKER_CMD $IMAGE_NAME $@"

# Execute the command
eval $DOCKER_CMD
