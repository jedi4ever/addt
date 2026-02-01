# DClaude - Containerized Claude Code

**dclaude is a containerized Claude Code runner.** It wraps the Claude CLI in Docker so your AI agent can read, write, and execute code in complete isolation.

**Same commands. Same workflows. No surprises on your host machine.**

```bash
# Instead of:        Use:
claude              dclaude
claude --help       dclaude --help
claude -p "prompt"  dclaude -p "prompt"
claude --continue   dclaude --continue
claude --model opus dclaude --model opus
```

## Overview

### What We've Built

**✓ Drop-in Replacement**
- [x] All `claude` CLI arguments, flags, and options work identically
- [x] Interactive mode, print mode, continue mode supported
- [x] Model selection (sonnet, opus, haiku)
- [x] Session history and conversation persistence

**✓ Extra Features (Enable What You Need)**
- [x] **GitHub Token Forwarding** - *Opt-in:* Auto-pass `GH_TOKEN` for private repos
- [x] **SSH Key Forwarding** - *Opt-in:* Mount SSH keys for git over SSH (agent mode recommended, keys mode exposes all private keys)
- [x] **GPG Key Forwarding** - *Opt-in:* Mount GPG keys for signed commits
- [x] **Docker-in-Docker** - *Opt-in:* Run Docker commands inside container (isolated or host mode)
- [x] **Automatic Port Mapping** - *Opt-in:* Maps container ports to host, Claude knows the URLs
- [x] **Version Pinning** - Pin to specific Claude Code or Node.js versions
- [x] **Custom Environment Variables** - Pass any env vars to container

**✓ Smart Automation**
- [x] **Auto-build on First Run** - No manual docker build needed
- [x] **Image Reuse** - Automatically reuses existing images with matching versions
- [x] **Version Detection** - Queries npm registry via HTTP for latest stable version
- [x] **Auto-mount Volumes** - Current directory, git config, Claude settings
- [x] **Git Identity Inheritance** - Uses your local git name/email automatically
- [x] **Environment File Loading** - Auto-loads `.env` file if present

**✓ Pre-installed Tools in Container**
- [x] Claude Code (configurable version)
- [x] Node.js (configurable version)
- [x] Git
- [x] GitHub CLI (gh)
- [x] Ripgrep (rg)
- [x] Docker CLI (for DinD support)
- [x] curl, bash, sudo

**✓ Distribution**
- [x] **Standalone Binary** - Single-file script with embedded Dockerfile
- [x] **No External Dependencies** - Only requires Docker and curl on host

## Providers

### Docker Provider (Default)
Runs Claude Code in local Docker containers:
- ✅ Complete isolation
- ✅ Full control over resources
- ✅ Docker-in-Docker support
- ⚙️ Requires Docker installed locally

## Prerequisites

### For Docker Provider (Default)
- **Docker** - Container runtime ([installation guide](https://docs.docker.com/get-docker/))
- **curl** - For npm registry queries (usually pre-installed on macOS/Linux)
- **Bash 4+** - For built-in port checking (standard on macOS/Linux)
- **Claude Code configured** - Must have run `claude login` locally, OR set `ANTHROPIC_API_KEY`

### Authentication (Choose One)
- **Option 1 (Recommended):** Configure Claude Code locally with `claude login`
  - Your `~/.claude` directory is automatically mounted
  - No need to set ANTHROPIC_API_KEY
- **Option 2:** Set `ANTHROPIC_API_KEY` environment variable
  - Get your API key from [console.anthropic.com](https://console.anthropic.com)
  - Pass via environment variable

### Optional
- **GH_TOKEN** - GitHub personal access token for private repos and write operations ([create one](https://github.com/settings/tokens))

## Installation

### Option 1: Download Pre-built Binary (Recommended)

Download the latest release for your platform:

```bash
# macOS Apple Silicon (M1/M2/M3)
curl -fsSL https://github.com/jedi4ever/dclaude/releases/latest/download/dclaude-darwin-arm64 -o dclaude
chmod +x dclaude
xattr -c dclaude && codesign --sign - --force dclaude
sudo mv dclaude /usr/local/bin/

# macOS Intel
curl -fsSL https://github.com/jedi4ever/dclaude/releases/latest/download/dclaude-darwin-amd64 -o dclaude
chmod +x dclaude
xattr -c dclaude && codesign --sign - --force dclaude
sudo mv dclaude /usr/local/bin/

# Linux x86_64
curl -fsSL https://github.com/jedi4ever/dclaude/releases/latest/download/dclaude-linux-amd64 -o dclaude
chmod +x dclaude
sudo mv dclaude /usr/local/bin/

# Linux ARM64
curl -fsSL https://github.com/jedi4ever/dclaude/releases/latest/download/dclaude-linux-arm64 -o dclaude
chmod +x dclaude
sudo mv dclaude /usr/local/bin/

# Verify installation
dclaude --dversion
```

**Install a specific version:**

If you need a specific version for reproducibility, use the version tag:

```bash
# Example: Install v1.4.3 specifically (macOS)
curl -fsSL https://github.com/jedi4ever/dclaude/releases/download/v1.4.3/dclaude-darwin-arm64 -o dclaude
chmod +x dclaude
xattr -c dclaude && codesign --sign - --force dclaude
sudo mv dclaude /usr/local/bin/
```

**Upgrading to a newer version:**

Simply re-run the installation command (the codesign step is important to avoid corruption):

```bash
# Upgrade to latest version (macOS)
curl -fsSL https://github.com/jedi4ever/dclaude/releases/latest/download/dclaude-darwin-arm64 -o dclaude
chmod +x dclaude
xattr -c dclaude && codesign --sign - --force dclaude
sudo mv dclaude /usr/local/bin/
```

See all releases at: https://github.com/jedi4ever/dclaude/releases

### Option 2: Build from Source

```bash
# Clone the repository
git clone https://github.com/jedi4ever/dclaude.git
cd dclaude

# Build
make build

# Install
sudo cp dist/dclaude /usr/local/bin/

# Or use make install
make install
```

## Quick Start

### Option 1: Use Docker with Local Claude Configuration (Recommended)

If you've already run `claude login` on your machine:

```bash
# Just run it - your ~/.claude config is automatically mounted!
dclaude
```

### Option 2: Use Docker with API Key

If you haven't configured Claude locally, set your API key:

```bash
# Export in your shell
export ANTHROPIC_API_KEY='your-anthropic-api-key'
export GH_TOKEN='your-github-token'  # Optional

# Run dclaude
dclaude
```

**Note:** The Docker image automatically builds on first run. No manual build step needed!

## Usage

### Drop-in Replacement

Use `dclaude` exactly like you would use `claude`:

```bash
# Interactive mode (default)
dclaude                                    # Same as: claude

# One-off command
dclaude "Fix the bug in app.js"            # Same as: claude "Fix the bug in app.js"

# Print mode (non-interactive)
dclaude -p "Explain this function"         # Same as: claude -p "Explain this function"

# Continue previous conversation
dclaude --continue                         # Same as: claude --continue

# Use different model
dclaude --model opus "Refactor this"       # Same as: claude --model opus "Refactor this"

# Help and options
dclaude --help                             # Same as: claude --help (shows Claude's help)

# Check version
dclaude --version                          # Same as: claude --version (shows Claude's version)
```

### DClaude-Specific Commands

DClaude adds special commands and flags:

```bash
# Show DClaude version
dclaude --dversion

# Show DClaude help
dclaude --dhelp

# Check for and install updates
dclaude --update

# Rebuild the Docker image (removes and rebuilds)
dclaude --rebuild

# Can combine with other commands
dclaude --rebuild --dversion

# YOLO mode - bypass all permission checks (shorthand for --dangerously-skip-permissions)
dclaude --yolo "Refactor this entire codebase"

# Open bash shell inside the container
dclaude shell

# Run a specific command in the container
dclaude shell -c "git config --list"
```

### Container Management

Manage persistent containers (see Persistent Mode section below):

```bash
# List all persistent containers with their status
dclaude containers list
dclaude containers ls              # Short form

# Stop a running persistent container
dclaude containers stop dclaude-persistent-myproject-a1b2c3d4

# Remove a persistent container
dclaude containers remove dclaude-persistent-myproject-a1b2c3d4
dclaude containers rm dclaude-persistent-myproject-a1b2c3d4  # Short form

# Remove all persistent containers
dclaude containers clean
```

**Note:** Container management commands only work with persistent containers created using `DCLAUDE_PERSISTENT=true`. Ephemeral containers are automatically removed after each run.



### Persistent Mode

By default, dclaude uses ephemeral containers that are removed after each run. Enable persistent mode to keep a long-running container per directory:

```bash
# Enable persistent mode
export DCLAUDE_PERSISTENT=true

# First run creates a persistent container
dclaude "Add a new feature"

# Subsequent runs reuse the same container (much faster!)
dclaude "Continue working"

# The container name is shown in the status line
# ✓ dclaude:claude-2.1.17 | Node 20.20.0 | Container:dclaude-persistent-myproject-a1b2c3d4

# List your persistent containers
dclaude containers list

# Clean up when done with the project
dclaude containers clean
```

**Benefits of persistent mode:**
- **Faster startup** - Container stays running, reconnection is instant
- **State preservation** - Docker images, installed packages, and files persist
- **Per-directory isolation** - Each directory gets its own container
- **Development continuity** - Pick up exactly where you left off

**Container naming:**
- Format: `dclaude-persistent-<dirname>-<hash>`
- Example: `dclaude-persistent-myproject-a1b2c3d4`
- Hash is based on full directory path for uniqueness

### Example Session

```bash
$ dclaude "Create a simple Express server on port 3000"

✓ dclaude:claude-2.1.17 | Node 20.20.0 | GH:- | SSH:- | GPG:- | Docker:-

I'll create an Express server for you.

[Claude creates server.js with Express app]

Server created! Run it with:
  node server.js

The server will be available at http://localhost:3000
```

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| **ANTHROPIC_API_KEY** | *(optional)* | Your Anthropic API key for authentication. Not needed if you've already run `claude login` locally (uses `~/.claude` config) |
| **GH_TOKEN** | *(optional)* | GitHub personal access token for gh CLI. Required for private repos, PRs, and write operations. Get yours at [github.com/settings/tokens](https://github.com/settings/tokens) |
| **DCLAUDE_CLAUDE_VERSION** | `latest` | Pin to a specific Claude Code version. Set to `latest` for newest stable, or specific version like `2.1.27`. Automatically checks npm registry and reuses existing images |
| **DCLAUDE_NODE_VERSION** | `20` | Node.js version for the container. Use major version (`18`, `20`, `22`), `lts`, or `current` |
| **DCLAUDE_GPG_FORWARD** | `false` | Enable GPG commit signing. Set to `true` to mount `~/.gnupg` |
| **DCLAUDE_SSH_FORWARD** | `false` | Enable SSH forwarding. Use `agent` or `true` for agent forwarding (recommended - secure), or `keys` to mount entire `~/.ssh` directory (⚠️ exposes all private keys) |
| **DCLAUDE_DOCKER_FORWARD** | `false` | Enable Docker support. Use `isolated` or `true` for isolated environment (recommended), or `host` to access host Docker daemon |
| **DCLAUDE_ENV_VARS** | `ANTHROPIC_API_KEY,GH_TOKEN` | Comma-separated list of environment variables to pass to container. Example: `ANTHROPIC_API_KEY,AWS_ACCESS_KEY_ID,AWS_SECRET_ACCESS_KEY` |
| **DCLAUDE_ENV_FILE** | `.env` | Path to environment file. Example: `.env.production` or `/path/to/config.env` |
| **DCLAUDE_GITHUB_DETECT** | `false` | Auto-detect GitHub token from `gh` CLI. Set to `true` to use token from `gh auth login` |
| **DCLAUDE_PORTS** | *(none)* | Comma-separated list of container ports to expose. Example: `3000,8080,5432`. Automatically maps to available host ports and tells Claude the correct URLs |
| **DCLAUDE_PORT_RANGE_START** | `30000` | Starting port number for automatic port allocation. Useful to avoid conflicts with other services |
| **DCLAUDE_LOG** | `false` | Enable command logging. Set to `true` to log all commands with timestamps, working directory, and container info |
| **DCLAUDE_LOG_FILE** | `dclaude.log` | Log file location (only used when `DCLAUDE_LOG=true`). Example: `/tmp/dclaude.log` or `~/logs/dclaude.log` |
| **DCLAUDE_PERSISTENT** | `false` | Enable persistent container mode. Set to `true` to keep containers running across sessions. Each directory gets its own persistent container with preserved state, Docker images, and installed packages |
| **DCLAUDE_MOUNT_PWD** | `true` | Mount present working directory to `/workspace` in container. Set to `false` to run without mounting the current directory (useful for isolated tasks) |
| **DCLAUDE_MOUNT_CLAUDE_CONFIG** | `true` | Mount `~/.claude` configuration directory (authentication and session history). Set to `false` to run without Claude config (requires `ANTHROPIC_API_KEY` environment variable) |
| **DCLAUDE_MODE** | `container` | Execution mode: `container` (Docker-based, default) or `shell` (direct host execution - not yet implemented) |
| **DCLAUDE_PROVIDER** | `docker` | Provider type: `docker` (default) or `daytona` (experimental, see [docs/README-daytona.md](docs/README-daytona.md)) |

### Quick Configuration Examples

```bash
# Basic usage
export ANTHROPIC_API_KEY="your-key"
export GH_TOKEN="your-github-token"
dclaude

# With port mapping for web development
export DCLAUDE_PORTS="3000,8080,5432"
dclaude "Create an Express app"

# With SSH and Docker support
export DCLAUDE_SSH_FORWARD=agent
export DCLAUDE_DOCKER_FORWARD=isolated
dclaude

# Pin to specific versions
export DCLAUDE_CLAUDE_VERSION=2.1.27
export DCLAUDE_NODE_VERSION=18
dclaude
```

## Common Use Cases

### Port Mapping for Web Development

When Claude starts web services, it needs to tell you the correct host ports:

```bash
# Enable port mapping
export DCLAUDE_PORTS="3000,8080,5432"
dclaude "Create a web server on port 3000"

# Status line shows: Ports:3000→30000,8080→30001
# Claude will say: "Visit http://localhost:30000 in your browser"
```

**How it works:**
- You specify container ports to expose
- DClaude automatically maps them to available host ports
- Port mappings are passed to Claude via `--append-system-prompt`
- Claude knows the correct URLs and tells you the host ports
- Internal testing uses container ports, user URLs use host ports

**Common scenarios:**
```bash
# Web development
DCLAUDE_PORTS="3000,5173,8080" dclaude

# Full stack (frontend, backend, database)
DCLAUDE_PORTS="3000,8000,5432" dclaude
```

### SSH Key Forwarding

For git operations over SSH, pushing to private repos, etc:

```bash
# Agent forwarding (recommended - more secure)
export DCLAUDE_SSH_FORWARD=agent
dclaude

# Or mount SSH keys directly (exposes all keys)
export DCLAUDE_SSH_FORWARD=keys
dclaude
```

**⚠️ Security Warning:**
- **`agent` mode (recommended)**: Only forwards SSH agent socket, keys stay on host
- **`keys` mode**: Mounts entire `~/.ssh` directory - **exposes ALL private keys** to the container
- Use `agent` mode unless you have specific compatibility issues

**When you need this:**
- Pushing to GitHub/GitLab over SSH
- Accessing private repositories
- SSH into remote servers
- Git operations that require authentication

### Docker-in-Docker Support

Let Claude run Docker commands:

```bash
# Isolated Docker environment (recommended)
export DCLAUDE_DOCKER_FORWARD=isolated
dclaude "Build and run a Docker container"

# Or access host Docker socket
export DCLAUDE_DOCKER_FORWARD=host
dclaude
```

**Isolated vs Host mode:**
- **Isolated**: Claude gets its own Docker environment (can't see your host containers)
- **Host**: Claude can see and manage all your Docker containers

### GPG Commit Signing

For signed commits:

```bash
export DCLAUDE_GPG_FORWARD=true
dclaude
```

Your GPG keys are mounted and commits will be signed automatically.

### Version Pinning

Pin to specific tool versions:

```bash
# Use specific Claude Code version
export DCLAUDE_CLAUDE_VERSION=2.1.27
dclaude

# Use specific Node.js version
export DCLAUDE_NODE_VERSION=18
dclaude

# Use latest stable (default)
dclaude  # Automatically uses latest stable version
```

### Custom Environment Variables

Pass additional environment variables to the container:

```bash
# Pass AWS credentials
export DCLAUDE_ENV_VARS="ANTHROPIC_API_KEY,AWS_ACCESS_KEY_ID,AWS_SECRET_ACCESS_KEY,AWS_REGION"
export AWS_ACCESS_KEY_ID="your-key"
export AWS_SECRET_ACCESS_KEY="your-secret"
export AWS_REGION="us-east-1"
dclaude
```

### Aliases and Shortcuts

Create shell aliases for common dclaude configurations:

```bash
# Add to ~/.bashrc or ~/.zshrc

# YOLO mode - bypasses all permission prompts (use with caution!)
# Note: dclaude has built-in --yolo flag, this alias just makes it shorter
alias dclaude-yolo='dclaude --yolo'

# Dev mode - with Docker and port mapping
alias dclaude-dev='DCLAUDE_DOCKER_FORWARD=isolated DCLAUDE_PORTS="3000,8080,5432" dclaude'

# Opus mode - always use Claude Opus
alias dclaude-opus='dclaude --model opus'

# Quick shell access
alias dshell='dclaude shell'
```

**⚠️ Warning about YOLO mode:**
- `--yolo` (shorthand for `--dangerously-skip-permissions`) bypasses ALL Claude Code permission checks
- Claude can execute any command, read any file, make any change without asking
- **Only use in trusted, sandboxed environments** (like throwaway containers or VMs)
- Never use with sensitive data or production systems
- Great for demos, experimentation, or CI/CD pipelines

**Usage:**
```bash
# Use --yolo flag directly (no alias needed)
dclaude --yolo "Refactor this entire codebase"

# Or set up aliases for convenience
source ~/.bashrc  # or source ~/.zshrc

# Then use them
dclaude-yolo "Refactor this entire codebase"
dclaude-dev "Create a full-stack app"
dclaude-opus "Complex architectural question"
dshell  # Quick shell access
```

## Troubleshooting

### Binary Killed with Signal 9 (macOS)

If you see `Killed: 9` when running dclaude on macOS, the binary needs to be code-signed:

```bash
# If you forgot to run codesign during installation
codesign --sign - --force /usr/local/bin/dclaude

# Or re-download with proper signing
curl -fsSL https://github.com/jedi4ever/dclaude/releases/latest/download/dclaude-darwin-arm64 -o dclaude
chmod +x dclaude
xattr -c dclaude && codesign --sign - --force dclaude
sudo mv dclaude /usr/local/bin/
```

**Why this happens:**
- macOS requires binaries to be properly signed
- Re-downloading without signing corrupts the signature
- The `codesign --sign - --force` command ad-hoc signs the binary

### Authentication Issues

If Claude Code reports authentication errors:

**Option 1: Use local Claude configuration (recommended)**
```bash
# Configure Claude locally (one-time setup)
claude login

# Then dclaude will automatically use your credentials
dclaude
```

**Option 2: Use API key**
```bash
# Set API key in environment
export ANTHROPIC_API_KEY='your-key'
dclaude
```

### Image Not Found

The image is built automatically on first run. If you see "image not found":

```bash
dclaude  # Will auto-build
```

Force rebuild:
```bash
# Easiest way - use --rebuild flag
dclaude --rebuild

# Or manually remove and rebuild
docker rmi dclaude:latest
dclaude  # Rebuilds automatically
```

### Permission Issues

If you get permission errors:

```bash
# Check file ownership
ls -la

# The container runs as your user, so files should have correct ownership
# If not, the UID/GID mapping might have failed
```

### Git Identity Not Set

Your local `.gitconfig` is automatically mounted. Check it exists:

```bash
ls -la ~/.gitconfig

# Test inside container
dclaude shell -c "git config --global user.name"
```

### Port Conflicts

If ports are already in use:

```bash
# Use different port range
export DCLAUDE_PORT_RANGE_START=40000
export DCLAUDE_PORTS="3000,8080"
dclaude
```

### Debugging

```bash
# Enable logging
export DCLAUDE_LOG=true
dclaude

# Check logs
cat dclaude.log

# Open shell to inspect container
dclaude shell
```

## How It Works

DClaude is a Go binary that:

1. **Checks Requirements** - Verifies Docker is available
2. **Version Detection** - Queries npm registry for Claude Code versions via HTTP
3. **Image Management** - Builds Docker image if needed, reuses existing images
4. **Volume Mounting** - Automatically mounts your project, git config, and Claude settings
5. **Port Mapping** - Finds available host ports and configures container
6. **System Prompt Injection** - Tells Claude about port mappings via `--append-system-prompt`
7. **Environment Setup** - Passes environment variables and configuration
8. **Container Execution** - Runs Claude Code in container with all settings applied

**What gets mounted:**
- Current directory → `/workspace` (your project files)
- `~/.gitconfig` → Git identity
- `~/.claude` → Authentication and session history
- `~/.gnupg` → GPG keys (opt-in)
- `~/.ssh` → SSH keys (opt-in)

**Pre-installed in container:**
- Claude Code
- Node.js
- Git
- GitHub CLI (gh)
- Ripgrep (rg)
- Docker CLI

For technical details, architecture, and development guide, see [docs/README-development.md](docs/README-development.md).

## Examples

### Create a Web Application

```bash
export DCLAUDE_PORTS="3000"
dclaude "Create a simple Express server with a /hello endpoint"
```

### Work with Private GitHub Repos

```bash
export GH_TOKEN="your-token"
export DCLAUDE_SSH_FORWARD=agent
dclaude "Clone my private repo and analyze the code structure"
```

### Build Docker Images

```bash
export DCLAUDE_DOCKER_FORWARD=isolated
dclaude "Create a Dockerfile for this Node.js app and build it"
```

### Sign Git Commits

```bash
export DCLAUDE_GPG_FORWARD=true
dclaude "Make a commit with GPG signature"
```

### Debug Container Environment

```bash
# Open shell
dclaude shell

# Inside container
echo $ANTHROPIC_API_KEY  # Check env vars
ls -la ~/.claude/         # Check mounted dirs
git config --list         # Check git config
claude --version          # Check Claude version
```

## Contributing

Contributions are welcome! Please see [docs/README-development.md](docs/README-development.md) for:
- Architecture details
- Build system
- Development workflow
- Testing guidelines
- Code style guide

## License

MIT License - See LICENSE file for details.

## Links

- [Claude Code](https://github.com/anthropics/claude-code)
- [Anthropic Console](https://console.anthropic.com)
- [Docker Installation](https://docs.docker.com/get-docker/)
- [GitHub Token Guide](https://github.com/settings/tokens)
