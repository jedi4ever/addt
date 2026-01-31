.PHONY: help build run shell clean

IMAGE_NAME := dclaude
IMAGE_TAG := latest

help:
	@echo "Docker + Makefile for Claude Code"
	@echo ""
	@echo "Available targets:"
	@echo "  make build   - Build the Docker image"
	@echo "  make run     - Run Claude Code interactively in container"
	@echo "  make shell   - Drop into container shell for debugging"
	@echo "  make clean   - Remove Docker image"
	@echo "  make help    - Show this help message"
	@echo ""
	@echo "Prerequisites:"
	@echo "  - Docker installed"
	@echo "  - ANTHROPIC_API_KEY environment variable set"
	@echo "  - GH_TOKEN environment variable set (optional, for GitHub CLI)"
	@echo ""
	@echo "Usage:"
	@echo "  export ANTHROPIC_API_KEY='your-key'"
	@echo "  export GH_TOKEN='your-github-token'  # optional"
	@echo "  make build"
	@echo "  make run"

build:
	@echo "Building Docker image: $(IMAGE_NAME):$(IMAGE_TAG)"
	@echo "Using UID=$(shell id -u) GID=$(shell id -g) USER=$(shell whoami)"
	docker build \
		--build-arg USER_ID=$(shell id -u) \
		--build-arg GROUP_ID=$(shell id -g) \
		--build-arg USERNAME=$(shell whoami) \
		-t $(IMAGE_NAME):$(IMAGE_TAG) .

run:
	@if [ -z "$$ANTHROPIC_API_KEY" ]; then \
		echo "Error: ANTHROPIC_API_KEY environment variable is not set"; \
		echo "Please set it with: export ANTHROPIC_API_KEY='your-key'"; \
		exit 1; \
	fi
	@echo "Running Claude Code in container..."
	docker run -it --rm \
		-v $(PWD):/workspace \
		-v $(HOME)/.gitconfig:/home/$(shell whoami)/.gitconfig:ro \
		-e ANTHROPIC_API_KEY=$$ANTHROPIC_API_KEY \
		-e GH_TOKEN=$$GH_TOKEN \
		$(IMAGE_NAME):$(IMAGE_TAG)

shell:
	@echo "Starting interactive shell in container..."
	docker run -it --rm \
		-v $(PWD):/workspace \
		-v $(HOME)/.gitconfig:/home/$(shell whoami)/.gitconfig:ro \
		-e ANTHROPIC_API_KEY=$$ANTHROPIC_API_KEY \
		-e GH_TOKEN=$$GH_TOKEN \
		--entrypoint /bin/bash \
		$(IMAGE_NAME):$(IMAGE_TAG)

clean:
	@echo "Removing Docker image: $(IMAGE_NAME):$(IMAGE_TAG)"
	docker rmi $(IMAGE_NAME):$(IMAGE_TAG)
