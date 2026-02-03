# addt - Development Guide

Technical documentation for developers and contributors.

## Table of Contents

- [Architecture](#architecture)
- [Project Structure](#project-structure)
- [Build System](#build-system)
- [Extension System](#extension-system)
- [Provider Architecture](#provider-architecture)
- [Docker Image Structure](#docker-image-structure)
- [Development Workflow](#development-workflow)
- [Testing](#testing)
- [Contributing](#contributing)

## Architecture

addt is written in Go and uses a provider-based architecture to support multiple container runtimes.

### High-Level Overview

```
┌─────────────────────────────────────────────────────────────┐
│                      CLI (cmd/root.go)                      │
│  - Parses arguments and flags                               │
│  - Handles --addt-* special flags                           │
│  - Detects binary name for symlink-based extension selection│
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                  Orchestrator (core/)                       │
│  - Coordinates provider operations                          │
│  - Builds RunSpec from configuration                        │
│  - Handles port mapping, volumes, environment               │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                   Provider Interface                        │
├─────────────────────────┬───────────────────────────────────┤
│    Docker Provider      │      Daytona Provider             │
│  (provider/docker/)     │    (provider/daytona/)            │
│  - Builds images        │    - Manages workspaces           │
│  - Runs containers      │    - Cloud-based execution        │
│  - Manages lifecycle    │    - (Experimental)               │
└─────────────────────────┴───────────────────────────────────┘
```

### Key Components

1. **CLI Layer** (`cmd/`) - Command parsing, flag handling, subcommands
2. **Core Layer** (`core/`) - Business logic orchestration
3. **Provider Layer** (`provider/`) - Container runtime abstraction
4. **Config Layer** (`config/`) - Environment variable and configuration loading
5. **Extensions** (`extensions/`) - Embedded AI agent definitions
6. **Assets** (`assets/`) - Docker files, scripts, and resources

## Project Structure

```
addt/
├── src/
│   ├── main.go                    # Entry point
│   ├── go.mod                     # Go module definition
│   ├── cmd/                       # CLI commands
│   │   ├── root.go                # Main CLI logic
│   │   ├── commands.go            # Subcommand handlers
│   │   ├── extensions.go          # Extension listing
│   │   ├── factory.go             # Provider factory
│   │   ├── firewall.go            # Firewall commands
│   │   └── help.go                # Help text generation
│   ├── config/                    # Configuration
│   │   ├── config.go              # Config loading
│   │   ├── env.go                 # Environment file parsing
│   │   └── github.go              # GitHub token detection
│   ├── core/                      # Business logic
│   │   ├── orchestrator.go        # Main orchestrator
│   │   └── version.go             # Version utilities
│   ├── provider/                  # Provider implementations
│   │   ├── provider.go            # Provider interface
│   │   ├── docker/                # Docker provider
│   │   │   ├── docker.go          # Main Docker implementation
│   │   │   ├── images.go          # Image management
│   │   │   ├── extensions.go      # Extension metadata reading
│   │   │   ├── forwarding.go      # SSH/GPG forwarding
│   │   │   └── version.go         # Version detection
│   │   └── daytona/               # Daytona provider (experimental)
│   │       └── daytona.go
│   ├── extensions/                # Embedded extensions
│   │   ├── embed.go               # Go embed directive
│   │   ├── claude/                # Claude Code extension
│   │   │   ├── config.yaml        # Extension metadata
│   │   │   ├── install.sh         # Build-time installation
│   │   │   ├── setup.sh           # Runtime setup
│   │   │   └── args.sh            # Argument transformation
│   │   ├── codex/                 # OpenAI Codex extension
│   │   ├── gemini/                # Google Gemini extension
│   │   ├── copilot/               # GitHub Copilot extension
│   │   ├── amp/                   # Sourcegraph Amp extension
│   │   ├── cursor/                # Cursor extension
│   │   ├── gastown/               # Multi-agent orchestration
│   │   ├── beads/                 # Git-backed issue tracker
│   │   └── ...                    # Other extensions
│   ├── assets/                    # Embedded assets
│   │   ├── embed.go               # Go embed directive
│   │   └── docker/                # Docker-specific assets
│   │       ├── Dockerfile         # Main Dockerfile
│   │       ├── docker-entrypoint.sh
│   │       ├── init-firewall.sh
│   │       └── install.sh         # Extension installer
│   └── internal/                  # Internal utilities
│       ├── ports/                 # Port availability checking
│       ├── terminal/              # Terminal detection
│       ├── update/                # Self-update functionality
│       └── util/                  # General utilities
├── dist/                          # Build output
├── docs/                          # Documentation
├── legacy/                        # Old shell-based implementation
├── Makefile                       # Build automation
├── VERSION                        # Version file
└── CHANGELOG.md                   # Release notes
```

## Build System

addt uses a standard Go build system with Make for automation.

### Prerequisites

- Go 1.21 or later
- Docker (for testing)
- Make

### Build Commands

```bash
# Format code and build for current platform
make build

# Build for all platforms (darwin/amd64, darwin/arm64, linux/amd64, linux/arm64)
make dist

# Install to /usr/local/bin
make install

# Run tests
make test

# Clean build artifacts
make clean

# Create a release (updates VERSION, creates git tag)
make release
```

### Cross-Platform Builds

The `make dist` command builds binaries for all supported platforms:

```bash
make dist
# Creates:
#   dist/addt-darwin-amd64  (macOS Intel)
#   dist/addt-darwin-arm64  (macOS Apple Silicon)
#   dist/addt-linux-amd64   (Linux x86_64)
#   dist/addt-linux-arm64   (Linux ARM64)
```

### Embedding Assets

Go's `embed` directive is used to include assets in the binary:

```go
// src/assets/embed.go
//go:embed docker/*
var DockerAssets embed.FS

// src/extensions/embed.go
//go:embed */config.yaml */install.sh */setup.sh */args.sh
var ExtensionAssets embed.FS
```

This allows the binary to be distributed as a single file with all Docker files and extension definitions included.

## Extension System

Extensions add AI agents and tools to the container image.

### Extension Structure

Each extension is a directory containing:

```
extensions/myextension/
├── config.yaml    # Required: Extension metadata
├── install.sh     # Optional: Build-time installation
├── setup.sh       # Optional: Runtime initialization
└── args.sh        # Optional: Argument transformation
```

### config.yaml

Defines extension metadata:

```yaml
name: claude
description: Claude Code - AI coding assistant by Anthropic
entrypoint: claude
default_version: stable
auto_mount: true
dependencies: []
env_vars:
  - ANTHROPIC_API_KEY
mounts:
  - source: ~/.claude
    target: /home/addt/.claude
  - source: ~/.claude.json
    target: /home/addt/.claude.json
flags:
  - flag: "--yolo"
    description: "Bypass permission checks"
```

### Extension Scripts

| Script | When | Purpose |
|--------|------|---------|
| `install.sh` | Docker build | Install packages, tools, dependencies |
| `setup.sh` | Container start | Initialize runtime environment |
| `args.sh` | Before execution | Transform CLI arguments (e.g., `--yolo` expansion) |

### Adding a New Extension

1. Create directory: `src/extensions/myextension/`
2. Add `config.yaml` with metadata
3. Add `install.sh` if packages need to be installed
4. Add `setup.sh` if runtime initialization is needed
5. Rebuild: `make build`
6. Test: `./dist/addt containers build --build-arg ADDT_EXTENSIONS=myextension`

## Provider Architecture

The provider interface abstracts container runtime operations.

### Provider Interface

```go
type Provider interface {
    // Core lifecycle
    Initialize(cfg *Config) error
    Run(spec *RunSpec) error
    Shell(spec *RunSpec) error
    Cleanup() error

    // Environment management
    Exists(name string) bool
    IsRunning(name string) bool
    Start(name string) error
    Stop(name string) error
    Remove(name string) error
    List() ([]Environment, error)

    // Image/workspace management
    BuildIfNeeded(rebuild bool) error
    DetermineImageName() string

    // Status and metadata
    GetStatus(cfg *Config, envName string) string
    GetExtensionEnvVars(imageName string) []string
}
```

### Docker Provider

The default provider that builds and runs Docker containers:

- **Image naming**: `addt:claude-2.1.17_codex-latest` (based on extensions and versions)
- **Container naming**: `addt-YYYYMMDD-HHMMSS-PID` (ephemeral) or hash-based (persistent)
- **Features**: SSH forwarding, GPG forwarding, Docker-in-Docker, port mapping, firewall

### Daytona Provider (Experimental)

Cloud-based workspace provider using Daytona:

- Manages remote workspaces instead of local containers
- See [docs/README-daytona.md](README-daytona.md) for details

## Docker Image Structure

### Base Image

Uses `node:${NODE_VERSION}-slim` (Debian-based) with:

- Node.js (configurable version)
- Go (latest or pinned)
- UV (Python package manager)
- Git, GitHub CLI, Ripgrep
- Docker CLI and daemon (for DinD)

### Non-Root User

Container runs as a non-root user matching host UID/GID:

```dockerfile
ARG USER_ID=1000
ARG GROUP_ID=1000
ARG USERNAME=addt

RUN useradd -m -u ${USER_ID} -g ${GROUP_ID} -s /bin/bash ${USERNAME}
```

### Build Process

1. Base image with system packages
2. Go and UV installation
3. Extension scripts copied
4. Extensions installed via `install.sh`
5. Entrypoint and firewall scripts added

### Runtime Flow

1. `docker-entrypoint.sh` starts
2. Extension `setup.sh` scripts run (once per session)
3. Docker daemon started (if DinD mode)
4. Firewall initialized (if enabled)
5. Agent command executed with arguments

## Development Workflow

### Local Development

```bash
# Make changes to Go code
vim src/cmd/root.go

# Format and build
make build

# Test locally
./dist/addt --addt-version
./dist/addt --addt-help

# Test with Docker
./dist/addt shell -c "echo hello"
```

### Testing Extensions

```bash
# Build image with specific extensions
./dist/addt containers build --build-arg ADDT_EXTENSIONS=claude,codex

# Verify installation
./dist/addt shell -c "which claude codex"

# Check extension metadata
./dist/addt shell -c "cat ~/.addt/extensions.json"
```

### Debug Mode

```bash
# Enable logging
export ADDT_LOG=true
export ADDT_LOG_FILE="/tmp/addt-debug.log"
./dist/addt "test prompt"

# View logs
tail -f /tmp/addt-debug.log
```

### Force Rebuild

```bash
# Rebuild image from scratch
./dist/addt --addt-rebuild

# Or remove image manually
docker rmi addt:claude-stable
```

## Testing

### Unit Tests

```bash
make test
# Runs: cd src && go test -v ./...
```

### Integration Tests

```bash
# Test basic functionality
./dist/addt --addt-version
./dist/addt --addt-list-extensions

# Test container operations
./dist/addt shell -c "env | grep ADDT"
./dist/addt containers list

# Test port mapping
ADDT_PORTS="3000,8080" ./dist/addt shell -c "echo \$ADDT_PORT_MAP"

# Test SSH forwarding
ADDT_SSH_FORWARD=agent ./dist/addt shell -c "ssh-add -l"

# Test Docker-in-Docker
ADDT_DIND_MODE=isolated ./dist/addt shell -c "docker ps"
```

### Testing Checklist

Before submitting a PR:

- [ ] `make build` succeeds
- [ ] `make test` passes
- [ ] `./dist/addt --addt-version` works
- [ ] Container can start: `./dist/addt shell -c "echo ok"`
- [ ] Extensions install correctly
- [ ] Documentation updated if needed

## Contributing

### Code Style

- Go standard formatting (`go fmt`)
- Meaningful variable names
- Comments for non-obvious logic
- Keep functions focused

### Commit Guidelines

Use conventional commits:

```
feat: add new extension support
fix: correct port mapping on Linux
docs: update development guide
refactor: simplify provider interface
```

Include attribution:

```
feat: add gemini extension support

Co-Authored-By: Claude <noreply@anthropic.com>
```

### Pull Request Process

1. Fork and create feature branch
2. Make changes with tests
3. Run `make build && make test`
4. Update documentation
5. Submit PR with clear description

## Debugging

### Common Issues

**Build fails:**
```bash
# Check Go version
go version

# Clean and rebuild
make clean && make build
```

**Container won't start:**
```bash
# Check Docker
docker info

# Check image exists
docker images | grep addt

# Force rebuild
./dist/addt --addt-rebuild
```

**Extension not found:**
```bash
# Verify extension is embedded
./dist/addt --addt-list-extensions

# Check config.yaml exists
ls src/extensions/myextension/config.yaml
```

**Permission issues:**
```bash
# Check UID/GID matching
id
./dist/addt shell -c "id"
```

### Verbose Output

```bash
# Enable Go race detection during development
cd src && go build -race -o ../dist/addt .

# Add debug prints (temporary)
fmt.Fprintf(os.Stderr, "DEBUG: %v\n", variable)
```

## License

MIT License - See LICENSE file for details.
