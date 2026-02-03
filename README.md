# nddt - Nope, Don't Do That

> **Warning:** This project is experimental and things are not perfect yet.

> **Note:** This project was formerly known as "dclaude". It has been renamed to "nddt" to reflect its support for multiple AI agents beyond Claude.

**Run AI coding agents in Docker containers.** Your agent can read, write, and execute code in complete isolation - no surprises on your host machine.

The binary name determines which agent runs. Symlink `nddt` to an extension name (e.g., `claude`, `codex`, `gemini`) and it auto-detects which agent to use. Run `nddt --nddt-list-extensions` to see all available extensions.

## Quick Start

```bash
# 1. Download (macOS Apple Silicon)
curl -fsSL https://github.com/jedi4ever/nddt/releases/latest/download/nddt-darwin-arm64 -o nddt
chmod +x nddt
xattr -c nddt && codesign --sign - --force nddt
sudo mv nddt /usr/local/bin/

# 2. Use it directly or via symlink
nddt "Fix the bug in app.js"           # Uses default (claude)
nddt --nddt-list-extensions            # See all available agents
```

**That's it.** First run auto-builds the container.

```bash
# Want to use as a specific agent? Create symlinks in ~/bin (won't override real installs):
mkdir -p ~/bin
ln -s /usr/local/bin/nddt ~/bin/claude
ln -s /usr/local/bin/nddt ~/bin/codex
ln -s /usr/local/bin/nddt ~/bin/gemini

# Add ~/bin to PATH (add to ~/.bashrc or ~/.zshrc)
export PATH="$HOME/bin:$PATH"

# Now use them:
claude "help me with this code"
codex "explain this function"
```

Each symlink name runs its own containerized agent with isolated config and Docker images.

## How It Works

1. **Name-based detection**: The binary checks its own filename (`claude`, `codex`, `gemini`, etc.)
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
curl -fsSL https://github.com/jedi4ever/nddt/releases/latest/download/nddt-darwin-arm64 -o nddt
chmod +x nddt
xattr -c nddt && codesign --sign - --force nddt
sudo mv nddt /usr/local/bin/
```

### macOS Intel

```bash
curl -fsSL https://github.com/jedi4ever/nddt/releases/latest/download/nddt-darwin-amd64 -o nddt
chmod +x nddt
xattr -c nddt && codesign --sign - --force nddt
sudo mv nddt /usr/local/bin/
```

### Linux x86_64

```bash
curl -fsSL https://github.com/jedi4ever/nddt/releases/latest/download/nddt-linux-amd64 -o nddt
chmod +x nddt
sudo mv nddt /usr/local/bin/
```

### Linux ARM64

```bash
curl -fsSL https://github.com/jedi4ever/nddt/releases/latest/download/nddt-linux-arm64 -o nddt
chmod +x nddt
sudo mv nddt /usr/local/bin/
```

### Homebrew

```bash
brew tap jedi4ever/tap
brew install nddt
```

### Verify Installation

```bash
nddt --nddt-version         # Shows nddt version
nddt --nddt-list-extensions # List available agents
```

**Upgrading:**

Re-run the installation command with codesign to avoid corruption:

```bash
curl -fsSL https://github.com/jedi4ever/nddt/releases/latest/download/nddt-darwin-arm64 -o nddt
chmod +x nddt
xattr -c nddt && codesign --sign - --force nddt
sudo mv nddt /usr/local/bin/
```

### Build from Source

```bash
git clone https://github.com/jedi4ever/nddt.git
cd nddt
make build
sudo cp dist/nddt /usr/local/bin/
```

## Usage

Use your containerized agent exactly like the real one:

```bash
# All normal commands work
claude "Fix the bug in app.js"
claude -p "Explain this function"
claude --continue
claude --model opus "Refactor this"
claude --help
```

### nddt-Specific Flags

These flags control the container, not the agent:

```bash
claude --nddt-version          # Show nddt version
claude --nddt-help             # Show nddt help (not agent help)
claude --nddt-rebuild          # Rebuild the Docker image
claude --nddt-update           # Check for nddt updates
claude --nddt-list-extensions  # List available extensions

# YOLO mode - bypass all permission checks
claude --yolo "Refactor this entire codebase"
```

### nddt Subcommands

Container management commands live under the `nddt` subcommand:

```bash
claude nddt build                    # Build the container image
claude nddt build --build-arg NDDT_EXTENSIONS=claude,codex

claude nddt shell                    # Open bash shell in container
claude nddt shell -c "git status"    # Run a command in container

claude nddt containers list          # List persistent containers
claude nddt containers stop <name>   # Stop a container
claude nddt containers rm <name>     # Remove a container
claude nddt containers clean         # Remove all persistent containers

claude nddt firewall list            # List allowed domains
claude nddt firewall add example.com # Add domain to whitelist
claude nddt firewall rm example.com  # Remove domain
claude nddt firewall reset           # Reset to defaults
```

Firewall config: `~/.nddt/firewall/allowed-domains.txt`



### Persistent Mode

Keep containers running across sessions for faster startup:

```bash
export NDDT_PERSISTENT=true
claude "Add a new feature"      # Creates container
claude "Continue working"       # Reuses same container (instant!)
```

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| **ANTHROPIC_API_KEY** | *(optional)* | Your Anthropic API key for authentication. Not needed if you've already run `claude login` locally (uses `~/.claude` config) |
| **GH_TOKEN** | *(optional)* | GitHub personal access token for gh CLI. Required for private repos, PRs, and write operations. Get yours at [github.com/settings/tokens](https://github.com/settings/tokens) |
| **NDDT_EXTENSIONS** | `claude` | Comma-separated list of extensions to install. Example: `claude,codex,gemini`. See [docs/extensions.md](docs/extensions.md) |
| **NDDT_COMMAND** | *(auto)* | Command to run instead of default. Example: `codex`, `gemini`, `gt` |
| **NDDT_<EXT>_VERSION** | `stable`/`latest` | Version for specific extension. Example: `NDDT_CLAUDE_VERSION=2.1.27`, `NDDT_CODEX_VERSION=latest` |
| **NDDT_<EXT>_AUTOMOUNT** | `true` | Mount extension config dirs. Example: `NDDT_CLAUDE_AUTOMOUNT=false` |
| **NDDT_NODE_VERSION** | `22` | Node.js version for the container. Use major version (`18`, `20`, `22`), `lts`, or `current` |
| **NDDT_GO_VERSION** | `latest` | Go version for the container. Use `latest` for newest stable, or specific version like `1.23.5`, `1.25.6`, etc. |
| **NDDT_UV_VERSION** | `latest` | UV (Python package manager) version. Use `latest` for newest stable, or specific version like `0.5.11`, `0.9.28`, etc. Supports `uv self update` inside containers. |
| **NDDT_GPG_FORWARD** | `false` | Enable GPG commit signing. Set to `true` to mount `~/.gnupg` |
| **NDDT_SSH_FORWARD** | `false` | Enable SSH forwarding. Use `agent` or `true` for agent forwarding (recommended - secure), or `keys` to mount entire `~/.ssh` directory (⚠️ exposes all private keys) |
| **NDDT_DIND_MODE** | *(none)* | Docker-in-Docker mode. Use `isolated` for own Docker daemon (recommended), or `host` to access host Docker socket |
| **NDDT_ENV_VARS** | `ANTHROPIC_API_KEY,GH_TOKEN` | Comma-separated list of environment variables to pass to container. Example: `ANTHROPIC_API_KEY,AWS_ACCESS_KEY_ID,AWS_SECRET_ACCESS_KEY` |
| **NDDT_ENV_FILE** | `.env` | Path to environment file. Example: `.env.production` or `/path/to/config.env` |
| **NDDT_GITHUB_DETECT** | `false` | Auto-detect GitHub token from `gh` CLI. Set to `true` to use token from `gh auth login` |
| **NDDT_PORTS** | *(none)* | Comma-separated list of container ports to expose. Example: `3000,8080,5432`. Automatically maps to available host ports and tells Claude the correct URLs |
| **NDDT_PORT_RANGE_START** | `30000` | Starting port number for automatic port allocation. Useful to avoid conflicts with other services |
| **NDDT_LOG** | `false` | Enable command logging. Set to `true` to log all commands with timestamps, working directory, and container info |
| **NDDT_LOG_FILE** | `nddt.log` | Log file location (only used when `NDDT_LOG=true`). Example: `/tmp/nddt.log` or `~/logs/nddt.log` |
| **NDDT_PERSISTENT** | `false` | Enable persistent container mode. Set to `true` to keep containers running across sessions. Each directory gets its own persistent container with preserved state, Docker images, and installed packages |
| **NDDT_WORKDIR_AUTOMOUNT** | `true` | Mount working directory to `/workspace` in container. Set to `false` to run without mounting the current directory (useful for isolated tasks) |
| **NDDT_CLAUDE_AUTOMOUNT** | `true` | Mount `~/.claude` directory and `~/.claude.json` file (authentication and session history). Set to `false` to run without Claude config (requires `ANTHROPIC_API_KEY` environment variable) |
| **NDDT_FIREWALL** | `false` | Enable network firewall (whitelist-based). Set to `true` to restrict outbound network access to allowed domains. **Requires `--cap-add=NET_ADMIN`** (automatically added when enabled). Particularly useful in CI/CD environments |
| **NDDT_FIREWALL_MODE** | `strict` | Firewall mode: `strict` (block non-whitelisted traffic), `permissive` (log but allow all traffic), or `off` (disable firewall). Default is `strict` when firewall is enabled |
| **NDDT_MODE** | `container` | Execution mode: `container` (Docker-based, default) or `shell` (direct host execution - not yet implemented) |
| **NDDT_PROVIDER** | `docker` | Provider type: `docker` (default) or `daytona` (experimental, see [docs/README-daytona.md](docs/README-daytona.md)) |

### Quick Examples

```bash
# Web development with port mapping
export NDDT_PORTS="3000,8080"
claude "Create an Express app"

# With SSH and Docker support
export NDDT_SSH_FORWARD=agent
export NDDT_DIND_MODE=isolated
claude

# Pin to specific versions
export NDDT_CLAUDE_VERSION=2.1.27
export NDDT_NODE_VERSION=18
claude
```

## Common Use Cases

### Port Mapping

Container ports are auto-mapped to available host ports. The agent is told the correct URLs.

```bash
export NDDT_PORTS="3000,8080"
claude "Create a web server on port 3000"
# Agent will tell you: "Visit http://localhost:30000 in your browser"
```

### SSH Forwarding

```bash
export NDDT_SSH_FORWARD=agent   # Recommended: forwards agent socket only
# export NDDT_SSH_FORWARD=keys  # Mounts ~/.ssh (exposes all keys)
claude
```

### Docker-in-Docker

```bash
export NDDT_DIND_MODE=isolated   # Own Docker environment
# export NDDT_DIND_MODE=host     # Access host Docker
claude "Build a Docker image"
```

### GPG Signing

```bash
export NDDT_GPG_FORWARD=true
claude
```

### Version Pinning

```bash
export NDDT_CLAUDE_VERSION=2.1.27
export NDDT_NODE_VERSION=18
claude
```

### Custom Env Vars

```bash
export NDDT_ENV_VARS="ANTHROPIC_API_KEY,AWS_ACCESS_KEY_ID,AWS_SECRET_ACCESS_KEY"
claude
```

### Aliases

```bash
# Add to ~/.bashrc or ~/.zshrc
alias claude-yolo='claude --yolo'

alias claude-dev='NDDT_DIND_MODE=isolated NDDT_PORTS="3000,8080" claude'
alias claude-opus='claude --model opus'
```

**YOLO mode** (`--yolo`) bypasses all permission checks. Only use in trusted environments.

## Troubleshooting

### macOS: Killed: 9

Binary needs code-signing. Re-run:
```bash
codesign --sign - --force /usr/local/bin/claude
```

### Authentication Errors

Either run `claude login` locally (config auto-mounted), or set `ANTHROPIC_API_KEY`.

### Force Rebuild

```bash
claude --nddt-rebuild
```

### Debug

```bash
export NDDT_LOG=true
claude
cat nddt.log

claude shell     # Open shell to inspect container
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
