.PHONY: all build clean test help install dist fmt release build-otel build-orchestrator

# Variables
BINARY_NAME=addt
BUILD_DIR=dist
SRC_DIR=src
VERSION=$(shell cat VERSION 2>/dev/null || echo "dev")
GO_FILES=$(shell find $(SRC_DIR) -name '*.go')
ASSET_FILES=$(shell find $(SRC_DIR)/assets -type f 2>/dev/null)

# Build targets for different platforms
PLATFORMS=darwin/amd64 darwin/arm64 linux/amd64 linux/arm64

help:
	@echo "Available targets:"
	@echo "  make build              - Format and build main binary for current platform"
	@echo "  make build-otel         - Build addt-otel binary"
	@echo "  make build-orchestrator - Build addt-orchestrator binary"
	@echo "  make build-all          - Build all binaries"
	@echo "  make fmt                - Format Go code"
	@echo "  make dist               - Build binaries for all platforms"
	@echo "  make install            - Build and install to /usr/local/bin"
	@echo "  make clean              - Remove build artifacts"
	@echo "  make test               - Run tests"
	@echo "  make release            - Create and push a new release (requires VERSION and CHANGELOG.md updated)"
	@echo "  make help               - Show this help"

# Default target
all: build

# Format Go code
fmt:
	@echo "Formatting Go code..."
	@cd $(SRC_DIR) && go fmt ./...
	@echo "✓ Code formatted"

# Build main binary for current platform
build: fmt $(BUILD_DIR)/$(BINARY_NAME)

$(BUILD_DIR)/$(BINARY_NAME): $(GO_FILES) $(ASSET_FILES) VERSION
	@echo "Building $(BINARY_NAME) v$(VERSION)..."
	@mkdir -p $(BUILD_DIR)
	@cd $(SRC_DIR) && go build -ldflags "-X main.Version=$(VERSION)" -o ../$(BUILD_DIR)/$(BINARY_NAME) .
	@chmod +x $(BUILD_DIR)/$(BINARY_NAME)
	@echo "✓ Built $(BUILD_DIR)/$(BINARY_NAME)"

# Build addt-otel binary
build-otel: fmt
	@echo "Building addt-otel..."
	@mkdir -p $(BUILD_DIR)
	@cd $(SRC_DIR) && go build -o ../$(BUILD_DIR)/addt-otel ./cmd/addt-otel
	@chmod +x $(BUILD_DIR)/addt-otel
	@echo "✓ Built $(BUILD_DIR)/addt-otel"

# Build addt-orchestrator binary
build-orchestrator: fmt
	@echo "Building addt-orchestrator..."
	@mkdir -p $(BUILD_DIR)
	@cd $(SRC_DIR) && go build -o ../$(BUILD_DIR)/addt-orchestrator ./cmd/addt-orchestrator
	@chmod +x $(BUILD_DIR)/addt-orchestrator
	@echo "✓ Built $(BUILD_DIR)/addt-orchestrator"

# Build all binaries
build-all: build build-otel build-orchestrator
	@echo "✓ All binaries built"

# Build for all platforms
dist: clean fmt
	@echo "Building $(BINARY_NAME) v$(VERSION) for all platforms..."
	@mkdir -p $(BUILD_DIR)
	@for platform in $(PLATFORMS); do \
		os=$${platform%/*}; \
		arch=$${platform#*/}; \
		echo "Building for $$os/$$arch..."; \
		(cd $(SRC_DIR) && CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch go build \
			-ldflags "-X main.Version=$(VERSION)" \
			-o ../$(BUILD_DIR)/$(BINARY_NAME)-$$os-$$arch .); \
		echo "✓ Built $(BUILD_DIR)/$(BINARY_NAME)-$$os-$$arch"; \
	done
	@echo ""
	@echo "Build complete! Binaries in $(BUILD_DIR)/"
	@ls -lh $(BUILD_DIR)/

# Install to system
install: build
	@echo "Installing $(BINARY_NAME) to /usr/local/bin/..."
	@cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/$(BINARY_NAME)
	@echo "✓ Installed /usr/local/bin/$(BINARY_NAME)"

# Run tests
test:
	@echo "Running tests..."
	@cd $(SRC_DIR) && go test -v ./...

# Clean build artifacts
clean:
	@echo "Cleaning up..."
	@rm -rf $(BUILD_DIR)
	@echo "✓ Cleaned"

# Development shortcuts
dev: build
	@echo "Running in development mode..."
	@./$(BUILD_DIR)/$(BINARY_NAME)

# Check if source files have changed
.PHONY: check
check:
	@cd $(SRC_DIR) && go vet ./...
	@cd $(SRC_DIR) && go fmt ./...
	@echo "✓ Code checked"

# Create and push a new release
release:
	@if [ -z "$(VERSION)" ] || [ "$(VERSION)" = "dev" ]; then \
		echo "Error: VERSION file not found or invalid"; \
		exit 1; \
	fi
	@echo "Creating release v$(VERSION)..."
	@echo ""
	@echo "⚠️  Make sure you have:"
	@echo "  1. Updated VERSION file to $(VERSION)"
	@echo "  2. Updated CHANGELOG.md with release notes"
	@echo "  3. Committed all changes"
	@echo ""
	@read -p "Continue with release v$(VERSION)? [y/N] " -n 1 -r; \
	echo; \
	if [[ ! $$REPLY =~ ^[Yy]$$ ]]; then \
		echo "Release cancelled"; \
		exit 1; \
	fi
	@echo "Committing version bump..."
	@git add VERSION CHANGELOG.md
	@git commit -m "Bump version to $(VERSION)" || true
	@echo "Creating git tag v$(VERSION)..."
	@git tag -a v$(VERSION) -m "Release v$(VERSION)" || (echo "Tag already exists, skipping..."; true)
	@echo "Pushing to remote..."
	@git push
	@git push --tags
	@echo ""
	@echo "✓ Release v$(VERSION) created and pushed!"
	@echo "  GitHub Actions will build and publish the release"
	@echo "  Check status: gh run list"
	@echo "  View release: gh release view v$(VERSION)"
