# dclaude Go build Makefile

# Binary name
BINARY_NAME=dclaude

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=$(GOCMD) fmt
GOVET=$(GOCMD) vet

# Build flags
VERSION?=$(shell cat ../VERSION 2>/dev/null || echo "1.0.0")
LDFLAGS=-ldflags "-X main.Version=$(VERSION)"

# Output directory
BUILD_DIR=../dist

# Assets directory for embedded files
ASSETS_DIR=assets

# Installation directory
INSTALL_DIR?=/usr/local/bin

# Platforms for cross-compilation
PLATFORMS=linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64

.PHONY: all build clean test fmt vet lint install uninstall cross-compile help assets

# Default target
all: build

## assets: Copy Dockerfile and entrypoint for embedding
assets:
	@mkdir -p $(ASSETS_DIR)
	@cp ../Dockerfile $(ASSETS_DIR)/Dockerfile
	@cp ../docker-entrypoint.sh $(ASSETS_DIR)/docker-entrypoint.sh

## build: Build the binary for current platform
build: assets
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME) .

## build-release: Build optimized release binary
build-release: assets
	CGO_ENABLED=0 $(GOBUILD) $(LDFLAGS) -trimpath -ldflags "-s -w -X main.Version=$(VERSION)" -o $(BINARY_NAME) .

## clean: Remove build artifacts
clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_NAME).exe
	rm -rf $(BUILD_DIR)
	rm -rf $(ASSETS_DIR)

## test: Run tests
test:
	$(GOTEST) -v ./...

## test-coverage: Run tests with coverage
test-coverage:
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

## fmt: Format code
fmt:
	$(GOFMT) ./...

## vet: Run go vet
vet:
	$(GOVET) ./...

## lint: Run linter (requires golangci-lint)
lint:
	@which golangci-lint > /dev/null || (echo "golangci-lint not installed" && exit 1)
	golangci-lint run

## tidy: Tidy go modules
tidy:
	$(GOMOD) tidy

## deps: Download dependencies
deps:
	$(GOMOD) download

## install: Install binary to INSTALL_DIR
install: build
	install -d $(INSTALL_DIR)
	install -m 755 $(BINARY_NAME) $(INSTALL_DIR)/$(BINARY_NAME)

## uninstall: Remove binary from INSTALL_DIR
uninstall:
	rm -f $(INSTALL_DIR)/$(BINARY_NAME)

## cross-compile: Build for all platforms
cross-compile: clean assets
	@mkdir -p $(BUILD_DIR)
	@for platform in $(PLATFORMS); do \
		GOOS=$${platform%/*} GOARCH=$${platform#*/} \
		CGO_ENABLED=0 $(GOBUILD) $(LDFLAGS) -trimpath -ldflags "-s -w -X main.Version=$(VERSION)" \
		-o $(BUILD_DIR)/$(BINARY_NAME)-$${platform%/*}-$${platform#*/}$$([ "$${platform%/*}" = "windows" ] && echo ".exe") . ; \
		echo "Built: $(BUILD_DIR)/$(BINARY_NAME)-$${platform%/*}-$${platform#*/}" ; \
	done

## dist: Create distribution archives
dist: cross-compile
	@cd $(BUILD_DIR) && \
	for f in $(BINARY_NAME)-*; do \
		if [ -f "$$f" ]; then \
			if echo "$$f" | grep -q "windows"; then \
				zip "$${f%.exe}.zip" "$$f"; \
			else \
				tar -czf "$$f.tar.gz" "$$f"; \
			fi \
		fi \
	done
	@echo "Distribution archives created in $(BUILD_DIR)"

## run: Build and run with arguments
run: build
	./$(BINARY_NAME) $(ARGS)

## version: Show version
version:
	@echo $(VERSION)

## help: Show this help
help:
	@echo "dclaude Go Makefile"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## /  /' | sed 's/:/ - /'
