# addt - AI Don't Do That

> **Warning:** This project is experimental and things are not perfect yet.

> **Note:** This project was formerly known as "dclaude" and then "nddt" (Nope, Don't Do That). It has been renamed to "addt" (AI Don't Do That) to reflect its support for multiple AI agents beyond Claude.

**Run AI coding agents in Docker containers.** Your agent can read, write, and execute code in complete isolation - no surprises on your host machine.

## Quick Start

```bash
# 1. Download (macOS Apple Silicon)
curl -fsSL https://github.com/jedi4ever/addt/releases/latest/download/addt-darwin-arm64 -o addt
chmod +x addt
xattr -c addt && codesign --sign - --force addt
sudo mv addt /usr/local/bin/

# 2. Run an agent
addt run claude "Fix the bug in app.js"
addt run codex "Explain this function"

# 3. See available extensions
addt extensions list
```

**That's it.** First run auto-builds the container.

### Using Symlinks (Optional)

For convenience, create symlinks to run agents directly:

```bash
mkdir -p ~/bin
ln -s /usr/local/bin/addt ~/bin/claude
ln -s /usr/local/bin/addt ~/bin/codex
ln -s /usr/local/bin/addt ~/bin/gemini

# Add ~/bin to PATH (add to ~/.bashrc or ~/.zshrc)
export PATH="$HOME/bin:$PATH"

# Now use them directly:
claude "help me with this code"
codex "explain this function"
```

Each symlink name runs its own containerized agent with isolated config and Docker images.

## How It Works

1. **Run command**: Use `addt run <extension>` or create symlinks for direct access
2. **Auto-builds container**: First run builds a Docker image with the right agent installed
3. **Runs in isolation**: Your code runs in a container with your project mounted at `/workspace`
4. **Same commands**: All CLI arguments pass through to the real agent

## Features

**Drop-in Replacement**
- All agent CLI arguments, flags, and options work identically
- Interactive mode, print mode, continue mode supported
- Session history and conversation persistence

**Extra Features (Opt-in)**
- **GitHub Token Forwarding** - Auto-pass `GH_TOKEN` for private repos
- **SSH Key Forwarding** - Mount SSH keys for git over SSH
- **GPG Key Forwarding** - Mount GPG keys for signed commits
- **Docker-in-Docker** - Run Docker commands inside container
- **Automatic Port Mapping** - Maps container ports to host, agent knows the URLs
- **Version Pinning** - Pin to specific agent or Node.js versions
- **Network Firewall** - Whitelist-based outbound traffic control

**Pre-installed Tools**
- Node.js, Go, UV (Python), Git, GitHub CLI, Ripgrep, Docker CLI

## Prerequisites

- **Docker** - [Install Docker](https://docs.docker.com/get-docker/)
- **Authentication** (choose one):
  - Run `claude login` locally (config auto-mounted), OR
  - Set `ANTHROPIC_API_KEY` environment variable
- **Optional**: `GH_TOKEN` for GitHub private repos ([create token](https://github.com/settings/tokens))

## Installation

### macOS Apple Silicon (M1/M2/M3)

```bash
curl -fsSL https://github.com/jedi4ever/addt/releases/latest/download/addt-darwin-arm64 -o addt
chmod +x addt
xattr -c addt && codesign --sign - --force addt
sudo mv addt /usr/local/bin/
```

### macOS Intel

```bash
curl -fsSL https://github.com/jedi4ever/addt/releases/latest/download/addt-darwin-amd64 -o addt
chmod +x addt
xattr -c addt && codesign --sign - --force addt
sudo mv addt /usr/local/bin/
```

### Linux x86_64

```bash
curl -fsSL https://github.com/jedi4ever/addt/releases/latest/download/addt-linux-amd64 -o addt
chmod +x addt
sudo mv addt /usr/local/bin/
```

### Linux ARM64

```bash
curl -fsSL https://github.com/jedi4ever/addt/releases/latest/download/addt-linux-arm64 -o addt
chmod +x addt
sudo mv addt /usr/local/bin/
```

### Homebrew

```bash
brew tap jedi4ever/tap
brew install addt
```

### mise

```bash
mise use -g github:jedi4ever/addt
```

### Verify Installation

```bash
addt version              # Shows addt version
addt extensions list      # List available agents
```

**Upgrading:**

```bash
addt cli update
```

Or re-run the installation command with codesign:

```bash
curl -fsSL https://github.com/jedi4ever/addt/releases/latest/download/addt-darwin-arm64 -o addt
chmod +x addt
xattr -c addt && codesign --sign - --force addt
sudo mv addt /usr/local/bin/
```

### Build from Source

```bash
git clone https://github.com/jedi4ever/addt.git
cd addt
make build
sudo cp dist/addt /usr/local/bin/
```

## Usage

### Running Agents

```bash
# Using addt run
addt run claude "Fix the bug in app.js"
addt run codex "Explain this function"
addt run gemini "Help me refactor"

# Using symlinks (if set up)
claude "Fix the bug in app.js"
claude -p "Explain this function"
claude --continue
claude --model opus "Refactor this"
claude --help
```

### addt Commands

```bash
# Run agents
addt run <extension> [args...]     # Run a specific extension
addt run claude "Fix the bug"
addt run codex --help

# Build and manage containers
addt build <extension>             # Build the container image
addt build claude --force          # Rebuild without cache
addt shell <extension>             # Open bash shell in container

# Container management
addt containers list               # List persistent containers
addt containers stop <name>        # Stop a container
addt containers rm <name>          # Remove a container
addt containers clean              # Remove all persistent containers

# Firewall management
addt firewall list                 # List allowed domains
addt firewall add example.com      # Add domain to whitelist
addt firewall rm example.com       # Remove domain
addt firewall reset                # Reset to defaults

# Extension management
addt extensions list               # List available extensions
addt extensions info <name>        # Show extension details
addt extensions new <name>         # Create a new local extension

# Configuration management
addt config global list            # List all global settings
addt config global set <key> <val> # Set a global config value
addt config extension <name> list  # List extension settings
addt config extension <name> set version 1.0.5  # Set extension version
addt config path                   # Show config file path

# CLI management
addt cli update                    # Check for and install updates
addt version                       # Show version info
```

### Via Symlink

When running via symlink (e.g., `claude`), use the `addt` subcommand:

```bash
claude addt build                  # Build the container image
claude addt shell                  # Open bash shell in container
claude addt containers list        # List persistent containers
claude addt extensions list        # List available extensions
claude addt version                # Show version info
```

Firewall config: `~/.addt/firewall/allowed-domains.txt`

### Persistent Mode

Keep containers running across sessions for faster startup:

```bash
export ADDT_PERSISTENT=true
claude "Add a new feature"      # Creates container
claude "Continue working"       # Reuses same container (instant!)
```

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| **ANTHROPIC_API_KEY** | *(optional)* | Your Anthropic API key for authentication. Not needed if you've already run `claude login` locally (uses `~/.claude` config) |
| **GH_TOKEN** | *(optional)* | GitHub personal access token for gh CLI. Required for private repos, PRs, and write operations. Get yours at [github.com/settings/tokens](https://github.com/settings/tokens) |
| **ADDT_EXTENSIONS** | *(none)* | Comma-separated list of extensions to install. Example: `claude,codex,gemini`. See [docs/extensions.md](docs/extensions.md) |
| **ADDT_COMMAND** | *(auto)* | Command to run instead of default. Example: `codex`, `gemini`, `gt` |
| **ADDT_<EXT>_VERSION** | `stable`/`latest` | Version for specific extension. Example: `ADDT_CLAUDE_VERSION=2.1.27`, `ADDT_CODEX_VERSION=latest` |
| **ADDT_<EXT>_AUTOMOUNT** | `true` | Mount extension config dirs. Example: `ADDT_CLAUDE_AUTOMOUNT=false` |
| **ADDT_NODE_VERSION** | `22` | Node.js version for the container. Use major version (`18`, `20`, `22`), `lts`, or `current` |
| **ADDT_GO_VERSION** | `latest` | Go version for the container. Use `latest` for newest stable, or specific version like `1.23.5`, `1.25.6`, etc. |
| **ADDT_UV_VERSION** | `latest` | UV (Python package manager) version. Use `latest` for newest stable, or specific version like `0.5.11`, `0.9.28`, etc. Supports `uv self update` inside containers. |
| **ADDT_GPG_FORWARD** | `false` | Enable GPG commit signing. Set to `true` to mount `~/.gnupg` |
| **ADDT_SSH_FORWARD** | `agent` | SSH forwarding mode. Use `agent` for agent forwarding (recommended - secure), or `keys` to mount entire `~/.ssh` directory (exposes all private keys) |
| **ADDT_DIND** | `false` | Enable Docker-in-Docker. Set to `true` to allow running Docker commands inside the container |
| **ADDT_DIND_MODE** | `isolated` | Docker-in-Docker mode. Use `isolated` for own Docker daemon (recommended), or `host` to access host Docker socket |
| **ADDT_DOCKER_CPUS** | *(none)* | CPU limit for container. Example: `2`, `0.5`, `1.5` |
| **ADDT_DOCKER_MEMORY** | *(none)* | Memory limit for container. Example: `512m`, `2g`, `4gb` |
| **ADDT_WORKDIR** | `.` | Override working directory. Default is current directory |
| **ADDT_ENV_VARS** | `ANTHROPIC_API_KEY,GH_TOKEN` | Comma-separated list of environment variables to pass to container. Example: `ANTHROPIC_API_KEY,AWS_ACCESS_KEY_ID,AWS_SECRET_ACCESS_KEY` |
| **ADDT_ENV_FILE** | `.env` | Path to environment file. Example: `.env.production` or `/path/to/config.env` |
| **ADDT_GITHUB_DETECT** | `false` | Auto-detect GitHub token from `gh` CLI. Set to `true` to use token from `gh auth login` |
| **ADDT_PORTS** | *(none)* | Comma-separated list of container ports to expose. Example: `3000,8080,5432`. Automatically maps to available host ports and tells Claude the correct URLs |
| **ADDT_PORT_RANGE_START** | `30000` | Starting port number for automatic port allocation. Useful to avoid conflicts with other services |
| **ADDT_LOG** | `false` | Enable command logging. Set to `true` to log all commands with timestamps, working directory, and container info |
| **ADDT_LOG_FILE** | `addt.log` | Log file location (only used when `ADDT_LOG=true`). Example: `/tmp/addt.log` or `~/logs/addt.log` |
| **ADDT_PERSISTENT** | `false` | Enable persistent container mode. Set to `true` to keep containers running across sessions. Each directory gets its own persistent container with preserved state, Docker images, and installed packages |
| **ADDT_WORKDIR_AUTOMOUNT** | `true` | Mount working directory to `/workspace` in container. Set to `false` to run without mounting the current directory (useful for isolated tasks) |
| **ADDT_CLAUDE_AUTOMOUNT** | `true` | Mount `~/.claude` directory and `~/.claude.json` file (authentication and session history). Set to `false` to run without Claude config (requires `ANTHROPIC_API_KEY` environment variable) |
| **ADDT_FIREWALL** | `false` | Enable network firewall (whitelist-based). Set to `true` to restrict outbound network access to allowed domains. **Requires `--cap-add=NET_ADMIN`** (automatically added when enabled). Particularly useful in CI/CD environments |
| **ADDT_FIREWALL_MODE** | `strict` | Firewall mode: `strict` (block non-whitelisted traffic), `permissive` (log but allow all traffic), or `off` (disable firewall). Default is `strict` when firewall is enabled |
| **ADDT_MODE** | `container` | Execution mode: `container` (Docker-based, default) or `shell` (direct host execution - not yet implemented) |
| **ADDT_PROVIDER** | `docker` | Provider type: `docker` (default) or `daytona` (experimental, see [docs/README-daytona.md](docs/README-daytona.md)) |
| **ADDT_CONFIG_DIR** | `~/.addt` | Directory for global config file. Config is stored as `config.yaml` in this directory |

### Persistent Configuration

Use `addt config` to manage settings that persist across sessions:

```bash
# Global settings (apply to all projects)
addt config global list                      # Show all settings with source
addt config global set docker_cpus 2         # Limit container to 2 CPUs
addt config global set docker_memory 4g      # Limit container memory
addt config global set dind true             # Enable Docker-in-Docker
addt config global unset docker_cpus         # Remove setting (use default)

# Project settings (apply to current directory only)
addt config project set persistent true      # Enable persistent mode for this project
addt config project set firewall true        # Enable firewall for this project
addt config project list                     # Show project-specific settings

# Per-extension settings
addt config extension claude list            # Show claude settings
addt config extension claude set version 1.0.5  # Pin claude version
addt config extension claude set automount false # Disable config mounting

# View config file paths
addt config path
```

**Configuration precedence** (highest to lowest):
1. Environment variables (e.g., `ADDT_DOCKER_CPUS`)
2. Project config (`.addt.yaml` in current directory)
3. Global config (`~/.addt/config.yaml`)
4. Default values

Project config is useful for:
- Team-shared settings (commit `.addt.yaml` to git)
- Project-specific resource limits
- Enabling features per-project (e.g., persistent mode, firewall)

### Quick Examples

```bash
# Web development with port mapping
export ADDT_PORTS="3000,8080"
addt run claude "Create an Express app"

# With SSH and Docker support
export ADDT_SSH_FORWARD=agent
export ADDT_DIND=true
addt run claude

# Pin to specific versions
export ADDT_CLAUDE_VERSION=2.1.27
export ADDT_NODE_VERSION=18
addt run claude

# Limit container resources
export ADDT_DOCKER_CPUS=2
export ADDT_DOCKER_MEMORY=4g
addt run claude
```

## Common Use Cases

### Port Mapping

Container ports are auto-mapped to available host ports. The agent is told the correct URLs.

```bash
export ADDT_PORTS="3000,8080"
addt run claude "Create a web server on port 3000"
# Agent will tell you: "Visit http://localhost:30000 in your browser"
```

### SSH Forwarding

```bash
export ADDT_SSH_FORWARD=agent   # Recommended: forwards agent socket only
# export ADDT_SSH_FORWARD=keys  # Mounts ~/.ssh (exposes all keys)
addt run claude
```

### Docker-in-Docker

```bash
export ADDT_DIND_MODE=isolated   # Own Docker environment
# export ADDT_DIND_MODE=host     # Access host Docker
addt run claude "Build a Docker image"
```

### GPG Signing

```bash
export ADDT_GPG_FORWARD=true
addt run claude
```

### Version Pinning

```bash
export ADDT_CLAUDE_VERSION=2.1.27
export ADDT_NODE_VERSION=18
addt run claude
```

### Custom Env Vars

```bash
export ADDT_ENV_VARS="ANTHROPIC_API_KEY,AWS_ACCESS_KEY_ID,AWS_SECRET_ACCESS_KEY"
addt run claude
```

### Aliases

```bash
# Add to ~/.bashrc or ~/.zshrc
alias claude='addt run claude'
alias codex='addt run codex'
alias claude-yolo='addt run claude --yolo'

alias claude-dev='ADDT_DIND_MODE=isolated ADDT_PORTS="3000,8080" addt run claude'
alias claude-opus='addt run claude --model opus'
```

**YOLO mode** (`--yolo`) bypasses all permission checks. Only use in trusted environments.

### Local Extensions

Create custom extensions in `~/.addt/extensions/`:

```bash
# Scaffold a new extension
addt extensions new myagent

# This creates:
# ~/.addt/extensions/myagent/
#   ├── config.yaml    # Extension metadata
#   ├── install.sh     # Build-time installation
#   └── setup.sh       # Runtime initialization

# Edit the files, then build
addt build myagent
addt run myagent "Hello!"
```

Local extensions override built-in extensions with the same name. See [docs/extensions.md](docs/extensions.md) for creating extensions.

## Troubleshooting

### macOS: Killed: 9

Binary needs code-signing. Re-run:
```bash
codesign --sign - --force /usr/local/bin/addt
```

### Authentication Errors

Either run `claude login` locally (config auto-mounted), or set `ANTHROPIC_API_KEY`.

### Force Rebuild

```bash
addt build claude --force
```

### Debug

```bash
export ADDT_LOG=true
addt run claude
cat addt.log

addt shell claude     # Open shell to inspect container
```

## Contributing

Contributions are welcome! Please see [docs/README-development.md](docs/README-development.md) for:
- Architecture details
- Build system
- Development workflow
- Testing guidelines
- Code style guide

## Credits

Network firewall implementation inspired by [claude-clamp](https://github.com/Richargh/claude-clamp) by Richargh. Thank you for pioneering the whitelist-based firewall approach for AI containerization!

## License

MIT License - See LICENSE file for details.

## Links

- [Claude Code](https://github.com/anthropics/claude-code)
- [Anthropic Console](https://console.anthropic.com)
- [Docker Installation](https://docs.docker.com/get-docker/)
- [GitHub Token Guide](https://github.com/settings/tokens)
