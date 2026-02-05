# addt - AI Don't Do That

**Run AI coding agents safely in Docker containers.** Your code stays isolated - no surprises on your host machine.

```bash
# Install (macOS)
brew install jedi4ever/tap/addt

# Run Claude in a container
addt run claude "Fix the bug in app.js"
```

That's it. First run auto-builds the container (~2 min), then you're coding.

---

## Why?

AI agents can read, write, and execute code. Running them in containers means:
- **Isolation** - Agents can't accidentally modify your system
- **Reproducibility** - Same environment every time
- **Security** - Network firewall, resource limits, no host access

All your normal agent commands work identically - it's a drop-in replacement.

---

## Install

**macOS (Homebrew):**
```bash
brew install jedi4ever/tap/addt
```

**mise:**
```bash
mise use -g github:jedi4ever/addt
```

**macOS (manual):**
```bash
# Apple Silicon
curl -fsSL https://github.com/jedi4ever/addt/releases/latest/download/addt-darwin-arm64 -o addt
# Intel: use addt-darwin-amd64

chmod +x addt && xattr -c addt && codesign --sign - --force addt
sudo mv addt /usr/local/bin/
```

**Linux:**
```bash
curl -fsSL https://github.com/jedi4ever/addt/releases/latest/download/addt-linux-amd64 -o addt
# ARM64: use addt-linux-arm64

chmod +x addt && sudo mv addt /usr/local/bin/
```

**Verify:** `addt version`

**Requires:** [Docker](https://docs.docker.com/get-docker/) or [Podman](https://podman.io/docs/installation) running locally.

---

## Quick Start

```bash
# Run any supported agent
addt run claude "Explain this codebase"
addt run codex "Add unit tests"
addt run gemini "Review this PR"

# All agent flags work normally
addt run claude --model opus "Refactor this"
addt run claude --continue
```

**Available agents:** Every agent is loaded as an extension. Built-in: `claude` `codex` `gemini` `copilot` `amp` `cursor` `kiro` `claude-flow` `gastown` `beads` `tessl` `openclaw` and more. Run `addt extensions list` for details.

When the agent starts, your current directory is auto-mounted (read-write) at `/workspace` in the container.

---

## Authentication

Each agent uses its own API key via environment variable:

```bash
# Claude
export ANTHROPIC_API_KEY="sk-ant-..."

# Codex (OpenAI)
export OPENAI_API_KEY="sk-..."

# Gemini
export GEMINI_API_KEY="..."
```

**Claude with API key:** When `ANTHROPIC_API_KEY` is set, the container auto-configures Claude Code to skip onboarding and trust the workspace - no interactive prompts.

**Claude with a subscription:** If you use Claude with a subscription (OAuth, not API), you need to:
1. Run `claude login` on your host machine first
2. Enable auto-mount to share your Claude config with the container:

```bash
addt config extension claude set automount true
```

This mounts `~/.claude` and `~/.claude.json` into the container.

⚠️ **Version caveat:** Auto-mount shares your local Claude config with the container. If the Claude Code version in the container differs from your local version, you may see config conflicts or unexpected behavior. Use version pinning (`ADDT_CLAUDE_VERSION`) to match versions if needed.

**Session resumption:** With auto-mount enabled, Claude can resume previous sessions using `--continue` or `--resume`. Your session history in `~/.claude` is mounted into the container.

**Your code:** Your current directory is automatically mounted at `/workspace` in the container. The agent can read and edit your files directly.

**For GitHub operations:** If the agent needs to create PRs, push commits, or access private repos, pass your GitHub token:
```bash
export GH_TOKEN="ghp_..."
```

---

## Everyday Usage

### Aliases (recommended)

Add to your `~/.bashrc` or `~/.zshrc`:
```bash
alias claude='addt run claude'
alias codex='addt run codex'
alias gemini='addt run gemini'

# Now use directly
claude "Fix the bug"
```

### Symlinks

Alternatively, create symlinks. Use the `addt-` prefix to make it clear:
```bash
ln -s /usr/local/bin/addt /usr/local/bin/addt-claude
ln -s /usr/local/bin/addt /usr/local/bin/addt-codex

# Use with prefix
addt-claude "Fix the bug"
```

You can also symlink directly to the agent name (e.g., `claude`), but the prefix avoids confusion with the real CLI if installed.

### Web Development (port mapping)

```bash
export ADDT_PORTS="8080"
addt run claude "Create an Express server on port 8080"
# Agent tells you: "Visit http://localhost:30000"
```

### GitHub Access (private repos, PRs)

```bash
export GH_TOKEN="ghp_..."
addt run claude "Clone the private repo and create a PR"
```

Or auto-detect from `gh auth login`:
```bash
export ADDT_GITHUB_DETECT=true
addt run claude "Clone git@github.com:org/private-repo.git"
```

### SSH Keys (git over SSH)

```bash
export ADDT_SSH_FORWARD=keys
addt run claude "Clone git@github.com:org/private-repo.git"
```

### Rebuild Container

```bash
addt build claude --force    # Rebuild from scratch
```

### Complete Isolation (no workdir mount)

```bash
export ADDT_WORKDIR_AUTOMOUNT=false
addt run claude "Work without access to host files"
```

### Network Firewall

```bash
export ADDT_FIREWALL=true
addt run claude "Only allowed domains are accessible"
```

---

## Configuration

There are three ways to configure addt:

| Method | Location | Use case |
|--------|----------|----------|
| **Environment variable** | Shell | Quick overrides, CI/CD |
| **Project config** | `.addt.yaml` in project | Team-shared settings, per-project defaults |
| **Global config** | `~/.addt/config.yaml` | Personal defaults across all projects |

**Precedence** (highest to lowest): Environment → Project → Global → Defaults

### Example: Setting memory limit

```bash
# Environment variable (highest priority)
export ADDT_DOCKER_MEMORY=4g

# Project config (.addt.yaml)
addt config project set docker_memory 4g

# Global config (~/.addt/config.yaml)
addt config global set docker_memory 4g
```

All three set the same thing. Environment wins if multiple are set.

### Project Config File

Use `addt config project` to manage `.addt.yaml` (commit to git for team sharing):

```bash
addt config project set persistent true
addt config project set docker_memory 4g
addt config project set firewall true
addt config project list
```

### Config Commands

```bash
# Global settings (all projects)
addt config global list
addt config global set docker_memory 4g
addt config global unset docker_memory

# Project settings (this directory only)
addt config project list
addt config project set firewall true

# Per-extension
addt config extension claude set version 1.0.5
```

### Common Environment Variables

| Variable | Description |
|----------|-------------|
| `ADDT_PERSISTENT=true` | Keep container running between sessions |
| `ADDT_PORTS=3000,8080` | Expose container ports |
| `ADDT_SSH_FORWARD=agent` | Forward SSH agent for git |
| `ADDT_DIND=true` | Enable Docker-in-Docker |
| `ADDT_FIREWALL=true` | Enable network firewall |

See [Full Reference](#environment-variables-reference) for all options.

---

## Advanced Features

### Persistent Mode

By default, containers are ephemeral (destroyed after each run). For faster startup, keep them running:
```bash
export ADDT_PERSISTENT=true
claude "Start a feature"     # Creates container
claude "Continue working"    # Reuses container (instant!)
```

### SSH Forwarding

```bash
export ADDT_SSH_FORWARD=agent   # Forward SSH agent (secure)
addt run claude "Clone the private repo"
```

### Docker-in-Docker

```bash
export ADDT_DIND=true
addt run claude "Build a Docker image for this app"
```

### GPG Signing

```bash
export ADDT_GPG_FORWARD=true
addt run claude "Create a signed commit"
```

### Network Firewall

Control which domains the agent can access:

```bash
# Enable firewall
addt config global set firewall true

# Manage allowed/denied domains
addt firewall global allow api.example.com
addt firewall global deny malware.com
addt firewall global list
```

**Layered rules** - Project rules override global rules:
```bash
# Globally deny npm
addt firewall global deny registry.npmjs.org

# But allow it for this project
addt firewall project allow registry.npmjs.org
```

Rule evaluation: `Defaults → Extension → Global → Project` (most specific wins)

### Using Podman Instead of Docker

If you prefer Podman over Docker, simply set the provider:

```bash
export ADDT_PROVIDER=podman
addt run claude "Fix the bug"
```

Podman is a daemonless container runtime that's compatible with Docker images and commands. All features (firewall, SSH forwarding, etc.) work the same way with both runtimes.

### Resource Limits

```bash
export ADDT_DOCKER_CPUS=2
export ADDT_DOCKER_MEMORY=4g
addt run claude
```

### Version Pinning

```bash
export ADDT_CLAUDE_VERSION=1.0.5
export ADDT_NODE_VERSION=20
addt run claude
```

### Custom Extensions

Create your own agent extensions:

```bash
addt extensions new myagent
# Edit ~/.addt/extensions/myagent/
addt build myagent
addt run myagent "Hello!"
```

See [docs/extensions.md](docs/extensions.md) for details.

---

## Command Reference

```bash
# Run agents
addt run <agent> [args...]        # Run an agent
addt run claude "Fix bug"
addt run codex --help

# Container management
addt build <agent>                # Build container image
addt build claude --force         # Rebuild without cache
addt shell <agent>                # Open shell in container
addt containers list              # List running containers
addt containers clean             # Remove all containers

# Configuration
addt config global list           # Show global settings
addt config global set <k> <v>    # Set global setting
addt config project list          # Show project settings
addt config extension <n> list    # Show extension settings

# Firewall
addt firewall global list         # List global rules
addt firewall global allow <d>    # Allow domain globally
addt firewall global deny <d>     # Deny domain globally
addt firewall project allow <d>   # Allow domain for project
addt firewall project deny <d>    # Deny domain for project

# Extensions
addt extensions list              # List available agents
addt extensions info <name>       # Show agent details
addt extensions new <name>        # Create custom agent

# Meta
addt version                      # Show version
addt cli update                   # Update addt
```

---

## Environment Variables Reference

### Authentication
| Variable | Default | Description |
|----------|---------|-------------|
| `ANTHROPIC_API_KEY` | - | API key (not needed if `claude login` done locally) |
| `GH_TOKEN` | - | GitHub token for private repos |

### Agent Selection
| Variable | Default | Description |
|----------|---------|-------------|
| `ADDT_EXTENSIONS` | - | Agents to install: `claude,codex` |
| `ADDT_COMMAND` | auto | Override command to run |
| `ADDT_<EXT>_VERSION` | stable | Version per agent: `ADDT_CLAUDE_VERSION=1.0.5` |

### Container Behavior
| Variable | Default | Description |
|----------|---------|-------------|
| `ADDT_PROVIDER` | docker | Container runtime: `docker`, `podman`, or `daytona` |
| `ADDT_PERSISTENT` | false | Keep container running |
| `ADDT_PORTS` | - | Ports to expose: `3000,8080` |
| `ADDT_DOCKER_CPUS` | - | CPU limit: `2` |
| `ADDT_DOCKER_MEMORY` | - | Memory limit: `4g` |
| `ADDT_WORKDIR` | `.` | Working directory to mount |

### Forwarding
| Variable | Default | Description |
|----------|---------|-------------|
| `ADDT_SSH_FORWARD` | - | SSH mode: `agent` or `keys` |
| `ADDT_GPG_FORWARD` | false | Mount GPG keys |
| `ADDT_DIND` | false | Enable Docker-in-Docker |
| `ADDT_DIND_MODE` | isolated | DinD mode: `isolated` or `host` |
| `ADDT_GITHUB_DETECT` | false | Auto-detect GH token from `gh` CLI |

### Security
| Variable | Default | Description |
|----------|---------|-------------|
| `ADDT_FIREWALL` | false | Enable network firewall |
| `ADDT_FIREWALL_MODE` | strict | Mode: `strict`, `permissive`, `off` |

### Paths & Logging
| Variable | Default | Description |
|----------|---------|-------------|
| `ADDT_ENV_FILE` | .env | Env file to load |
| `ADDT_ENV_VARS` | ANTHROPIC_API_KEY,GH_TOKEN | Vars to forward |
| `ADDT_LOG` | false | Enable logging |
| `ADDT_LOG_FILE` | addt.log | Log file path |
| `ADDT_CONFIG_DIR` | ~/.addt | Config directory |

### Tool Versions
| Variable | Default | Description |
|----------|---------|-------------|
| `ADDT_NODE_VERSION` | 22 | Node.js version |
| `ADDT_GO_VERSION` | latest | Go version |
| `ADDT_UV_VERSION` | latest | UV (Python) version |

---

## Troubleshooting

### macOS: "Killed: 9"
Binary needs code-signing:
```bash
codesign --sign - --force /usr/local/bin/addt
```

### Authentication errors
Either run `claude login` locally, or set `ANTHROPIC_API_KEY`.

### Container issues
```bash
addt build claude --force     # Rebuild container
addt shell claude             # Debug inside container
export ADDT_LOG=true          # Enable logging
```

---

## Contributing

See [docs/README-development.md](docs/README-development.md) for development setup.

## Credits

Network firewall inspired by [claude-clamp](https://github.com/Richargh/claude-clamp).

## License

MIT - See LICENSE file.

## Links

- [Claude Code](https://github.com/anthropics/claude-code)
- [Docker](https://docs.docker.com/get-docker/)
- [GitHub Tokens](https://github.com/settings/tokens)
