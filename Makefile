.PHONY: standalone clean test help build

help:
	@echo "Available targets:"
	@echo "  make build      - Build Go binary (dclaude-go)"
	@echo "  make standalone - Build single-file distributable version"
	@echo "  make test       - Test the standalone version"
	@echo "  make clean      - Remove generated files"

build: dclaude-go

dclaude-go: src/**/*.go src/assets/**/*
	@echo "Building dclaude-go..."
	@cd src && go build -o ../dclaude-go .
	@chmod +x dclaude-go
	@echo "✓ Built dclaude-go"

standalone: dist/dclaude-standalone.sh

dist/dclaude-standalone.sh: dclaude.sh Dockerfile docker-entrypoint.sh build.sh VERSION
	@./build.sh

test: dist/dclaude-standalone.sh
	@echo "Testing standalone version..."
	@./dist/dclaude-standalone.sh --version
	@echo "✓ Standalone version works!"

clean:
	@echo "Cleaning up..."
	@rm -rf dist
	@rm -f dclaude-go
	@rm -f .dclaude-Dockerfile.tmp
	@echo "✓ Cleaned"
