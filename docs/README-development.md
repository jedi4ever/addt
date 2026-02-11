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

---

## Architecture

addt is written in Go and uses a provider-based architecture to support multiple container runtimes.

### High-Level Overview

```
┌─────────────────────────────────────────────────────────────┐
│                      CLI (cmd/root.go)                      │
│  - Parses arguments and flags                               │
│  - Routes to subcommands (run, build, shell, etc.)          │
│  - Detects binary name for symlink-based extension selection│
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                     Runner (core/)                          │
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

| Layer | Directory | Purpose |
|-------|-----------|---------|
| CLI | `cmd/` | Command parsing, subcommands, flag handling |
| Config | `cmd/config/` | Configuration subcommands (global, project, extension) |
| Extensions | `cmd/extensions/` | Extension management (list, info, new, clone, remove) |
| Firewall | `cmd/firewall/` | Network firewall rules (global, project, extension) |
| Core | `core/` | Business logic, runner, port mapping, volumes |
| Config | `config/` | Configuration loading and types |
| Provider | `provider/` | Container runtime abstraction |
| Extensions | `extensions/` | Embedded AI agent definitions |
| Assets | `assets/` | Docker files, scripts, resources |

---

## Project Structure

```
addt/
├── src/
│   ├── main.go                    # Entry point
│   ├── go.mod                     # Go module definition
│   │
│   ├── cmd/                       # CLI commands
│   │   ├── root.go                # Main CLI routing
│   │   ├── run.go                 # `addt run` command
│   │   ├── build.go               # `addt build` command
│   │   ├── shell.go               # `addt shell` command
│   │   ├── containers.go          # `addt containers` command
│   │   ├── help.go                # Help text generation
│   │   ├── version.go             # Version display
│   │   ├── provider_factory.go    # Provider instantiation
│   │   │
│   │   ├── config/                # Configuration subcommands
│   │   │   ├── handler.go         # `addt config` routing
│   │   │   ├── global.go          # Global config commands
│   │   │   ├── project.go         # Project config commands
│   │   │   ├── extension.go       # Extension config commands
│   │   │   └── keys.go            # Valid config keys
│   │   │
│   │   ├── extensions/            # Extension management
│   │   │   ├── handler.go         # `addt extensions` routing
│   │   │   ├── list.go            # List available extensions
│   │   │   ├── info.go            # Show extension details
│   │   │   ├── new.go             # Create new extension
│   │   │   ├── clone.go           # Clone built-in extension
│   │   │   ├── remove.go          # Remove local extension
│   │   │   └── helper.go          # Utility functions
│   │   │
│   │   └── firewall/              # Firewall commands
│   │       ├── firewall.go        # `addt firewall` routing
│   │       ├── global.go          # Global firewall rules
│   │       ├── project.go         # Project firewall rules
│   │       ├── extension.go       # Extension firewall rules
│   │       ├── check.go           # Domain checking logic
│   │       └── helpers.go         # Utility functions
│   │
│   ├── config/                    # Configuration loading
│   │   ├── types.go               # Config struct definitions
│   │   ├── loader.go              # LoadConfig, precedence logic
│   │   ├── file.go                # Config file I/O
│   │   ├── env.go                 # Environment file parsing
│   │   └── github.go              # GitHub token detection
│   │
│   ├── core/                      # Business logic
│   │   ├── runner.go              # Main runner orchestration
│   │   ├── ports.go               # Port availability checking
│   │   ├── ports_prompt.go        # Interactive port selection
│   │   ├── volumes.go             # Volume mounting logic
│   │   ├── env.go                 # Environment variable handling
│   │   ├── options.go             # RunSpec building
│   │   ├── status.go              # Status display
│   │   ├── logging.go             # Command logging
│   │   └── npm.go                 # NPM registry detection
│   │
│   ├── provider/                  # Provider implementations
│   │   ├── provider.go            # Provider interface
│   │   │
│   │   ├── docker/                # Docker provider
│   │   │   ├── docker.go          # Provider struct, Initialize
│   │   │   ├── docker_exec.go     # Run, Shell, argument building
│   │   │   ├── docker_build.go    # BuildIfNeeded, image naming
│   │   │   ├── docker_status.go   # GetStatus, status display
│   │   │   ├── images.go          # Image existence, inspection
│   │   │   ├── images_build.go    # BuildBaseImage, BuildExtensionImage
│   │   │   ├── extensions.go      # Extension metadata from images
│   │   │   ├── persistent.go      # Persistent container mode
│   │   │   ├── ssh.go             # SSH agent forwarding
│   │   │   ├── gpg.go             # GPG key forwarding
│   │   │   ├── dind.go            # Docker-in-Docker support
│   │   │   └── version.go         # Version detection
│   │   │
│   │   └── daytona/               # Daytona provider (experimental)
│   │       └── daytona.go
│   │
│   ├── extensions/                # Embedded extensions
│   │   ├── embed.go               # Go embed directive
│   │   ├── loader.go              # Extension loading logic
│   │   ├── types.go               # ExtensionConfig struct
│   │   ├── claude/                # Claude Code extension
│   │   ├── codex/                 # OpenAI Codex extension
│   │   ├── gemini/                # Google Gemini extension
│   │   ├── copilot/               # GitHub Copilot extension
│   │   ├── amp/                   # Sourcegraph Amp extension
│   │   ├── cursor/                # Cursor extension
│   │   ├── kiro/                  # AWS Kiro extension
│   │   ├── gastown/               # Multi-agent orchestration
│   │   ├── beads/                 # Git-backed issue tracker
│   │   └── ...                    # Other extensions
│   │
│   ├── assets/                    # Embedded assets
│   │   ├── embed.go               # Go embed directive
│   │   └── docker/                # Docker-specific assets
│   │       ├── Dockerfile.base    # Base image Dockerfile
│   │       ├── Dockerfile         # Extension image Dockerfile
│   │       ├── docker-entrypoint.sh
│   │       ├── init-firewall.sh
│   │       └── install.sh         # Extension installer
│   │
│   └── util/                      # Utilities
│       ├── cleanup.go             # Signal handling, cleanup
│       ├── files.go               # File operations
│       └── terminal/              # Terminal detection
│           ├── terminal.go
│           ├── terminal_unix.go
│           └── terminal_windows.go
│
├── dist/                          # Build output
├── docs/                          # Documentation
├── Makefile                       # Build automation
├── VERSION                        # Version file
└── CHANGELOG.md                   # Release notes
```

---

## Build System

### Prerequisites

- Go 1.21 or later
- Docker (for testing)
- Make

### Build Commands

```bash
# Format code and build for current platform
make build

# Build for all platforms
make dist

# Install to /usr/local/bin
make install

# Run tests
make test

# Clean build artifacts
make clean
```

### Cross-Platform Builds

```bash
make dist
# Creates:
#   dist/addt-darwin-amd64  (macOS Intel)
#   dist/addt-darwin-arm64  (macOS Apple Silicon)
#   dist/addt-linux-amd64   (Linux x86_64)
#   dist/addt-linux-arm64   (Linux ARM64)
```

### Embedding Assets

Go's `embed` directive includes assets in the binary:

```go
// src/assets/embed.go
//go:embed docker/*
var FS embed.FS

// src/extensions/embed.go
//go:embed */*
var FS embed.FS
```

---

## Extension System

Extensions add AI agents and tools to the container image.

### Extension Structure

```
extensions/myextension/
├── config.yaml    # Required: Extension metadata
├── install.sh     # Optional: Build-time installation
├── setup.sh       # Optional: Runtime initialization
└── args.sh        # Optional: Argument transformation
```

### config.yaml

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
flags:
  - flag: "--yolo"
    description: "Bypass permission checks"
```

### Extension Scripts

| Script | When | Purpose |
|--------|------|---------|
| `install.sh` | Docker build | Install packages, tools, dependencies |
| `setup.sh` | Container start | Initialize runtime environment |
| `args.sh` | Before execution | Transform CLI arguments |

### Extension Commands

```bash
# List available extensions
addt extensions list

# Show extension details
addt extensions info claude

# Create new extension from scratch
addt extensions new myagent

# Clone built-in extension for customization
addt extensions clone claude
addt extensions clone claude my-claude  # with different name

# Remove local extension
addt extensions remove myagent
```

### Local Extensions

Local extensions in `~/.addt/extensions/` override built-in ones:

```bash
# Clone and customize
addt extensions clone claude
vim ~/.addt/extensions/claude/install.sh
addt build claude --force
```

---

## Provider Architecture

### Provider Interface

```go
type Provider interface {
    // Lifecycle
    Initialize(cfg *Config) error
    Run(args []string) error
    Shell(args []string) error
    Cleanup() error

    // Container management
    Exists(name string) bool
    IsRunning(name string) bool
    Start(name string) error
    Stop(name string) error
    Remove(name string) error
    List() ([]Container, error)

    // Image management
    BuildIfNeeded(rebuild, rebuildBase bool) error
    DetermineImageName() string

    // Metadata
    GetStatus(envName string) string
    GetExtensionMounts(imageName string) []ExtensionMountWithName
    GetExtensionEnvVars(imageName string) []string
}
```

### Docker Provider

Default provider with two-stage image building:

- **Base image**: `addt-base:node22-uid501` (Node, Go, UV, system packages)
- **Extension image**: `addt:claude-stable` (built FROM base)

Features: SSH forwarding, GPG forwarding, Docker-in-Docker, port mapping, firewall

### Daytona Provider (Experimental)

Cloud-based workspace provider. See [README-daytona.md](README-daytona.md).

---

## Docker Image Structure

### Two-Stage Build

1. **Base image** (`addt-base:nodeXX-uidXXX`)
   - Node.js, Go, UV (Python)
   - Git, GitHub CLI, Ripgrep
   - Docker CLI and daemon
   - Cached for fast rebuilds

2. **Extension image** (`addt:extension-version`)
   - Built FROM base image
   - Extension install scripts run
   - Takes ~10-30 seconds

### Non-Root User

Container runs as non-root user matching host UID/GID:

```dockerfile
ARG USER_ID=1000
ARG GROUP_ID=1000
RUN useradd -m -u ${USER_ID} -s /bin/bash addt
```

### Runtime Flow

1. `docker-entrypoint.sh` starts
2. Extension `setup.sh` scripts run
3. Docker daemon started (if DinD mode)
4. Firewall initialized (if enabled)
5. Agent command executed

---

## Development Workflow

### Local Development

```bash
# Make changes
vim src/cmd/root.go

# Format and build
make build

# Test locally
./dist/addt version
./dist/addt extensions list

# Test with Docker
./dist/addt build claude
./dist/addt run claude --help
```

### Testing Extensions

```bash
# Build with specific extensions
./dist/addt build claude,codex

# Verify installation
./dist/addt shell claude -c "which claude"

# Check extension metadata
./dist/addt extensions info claude
```

### Force Rebuild

```bash
# Rebuild extension image only
./dist/addt build claude --force

# Rebuild base image too
./dist/addt build claude --addt-rebuild-base
```

### Development with Nix

The project includes a `flake.nix` providing a complete development environment for NixOS users.

**Quick Start:**

Enter the development shell:

```bash
nix develop
```

This provides Go, Make, Git, Podman, and Docker from nixos-unstable. For better shell integration:

```bash
nix develop --command $SHELL
```

**Basic workflow:**

```bash
nix develop              # Enter dev environment
make build              # Build the addt binary
make test               # Run tests
```

**Updating Dependencies:**

Update all flake inputs (nixpkgs, flake-utils):

```bash
nix flake update
```

Update only nixpkgs:

```bash
nix flake lock --update-input nixpkgs
```

Check what will change before updating:

```bash
nix flake metadata
```

After updating, test the new environment:

```bash
nix develop
make build && make test
```

Commit the updated `flake.lock` if everything works correctly.

**When to update:** Security patches, new Go versions, or after extended periods without updates (monthly/quarterly).

**Troubleshooting:**

If you see `error: experimental feature 'nix-command' and 'flakes' is disabled`, enable in `~/.config/nix/nix.conf` or `/etc/nix/nix.conf`:

```
experimental-features = nix-command flakes
```

For dependency errors, refresh the lock file:

```bash
nix flake update          # Refresh lock file
nix develop --refresh     # Force re-evaluation
```

To verify you're in the Nix shell:

```bash
which go    # Should point to /nix/store/...
```

For non-Nix users, see [Build System](#build-system) for traditional setup.

---

## Testing

### Run Tests

```bash
# All tests
make test

# Specific package
cd src && go test ./cmd/...

# With verbose output
cd src && go test -v ./...

# Integration tests (require Docker)
cd src && go test -v -tags=integration ./...
```

### Testing Checklist

Before submitting a PR:

- [ ] `make build` succeeds
- [ ] `make test` passes
- [ ] `./dist/addt version` works
- [ ] Container starts: `./dist/addt shell claude -c "echo ok"`
- [ ] Extensions install correctly
- [ ] Documentation updated

---

## Contributing

### Code Style

- Go standard formatting (`go fmt`)
- Files under 200 lines (split if larger)
- Meaningful variable names
- Comments for non-obvious logic

### Commit Guidelines

```
feat: add new extension support
fix: correct port mapping on Linux
docs: update development guide
refactor: split large file into modules

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>
```

### Pull Request Process

1. Fork and create feature branch
2. Make changes with tests
3. Run `make build && make test`
4. Update documentation
5. Submit PR with clear description

---

## Debugging

### Common Issues

**Build fails:**
```bash
go version        # Check Go 1.21+
make clean && make build
```

**Container won't start:**
```bash
docker info       # Check Docker running
docker images | grep addt
./dist/addt build claude --force
```

**Extension not found:**
```bash
./dist/addt extensions list
ls src/extensions/myextension/config.yaml
```

### Debug Mode

```bash
export ADDT_LOG=true
export ADDT_LOG_FILE="/tmp/addt.log"
./dist/addt run claude "test"
tail -f /tmp/addt.log
```

---

## License

MIT License - See LICENSE file for details.
