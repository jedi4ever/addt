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

**Requires:** [Docker](https://docs.docker.com/get-docker/) running locally.

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

**Available agents:** `claude` `codex` `gemini` `copilot` `amp` `cursor` `kiro` `claude-flow` `gastown` `beads` `tessl` `openclaw` and more. Run `addt extensions list` for details.

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

**Claude with Max subscription:** If you use Claude Max (not API), run `claude login` on your host machine. Your `~/.claude` config is auto-mounted into the container - no API key needed.

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

### Persistent Mode (faster startup)

Keep containers running between sessions:
```bash
export ADDT_PERSISTENT=true
claude "Start a feature"     # Creates container
claude "Continue working"    # Reuses container (instant!)
```

### Web Development (port mapping)

```bash
export ADDT_PORTS="3000,8080"
addt run claude "Create an Express server on port 3000"
# Agent tells you: "Visit http://localhost:30000"
```

### Rebuild Container

```bash
addt build claude --force    # Rebuild from scratch
```

---

## Configuration

Settings can be configured via environment variables, project config, or global config.

### Quick Config

```bash
# Global settings (all projects)
addt config global set persistent true
addt config global set docker_memory 4g

# Project settings (this directory only)
addt config project set firewall true

# Per-extension
addt config extension claude set version 1.0.5
```

### Project Config File

Create `.addt.yaml` in your project (commit to git for team sharing):

```yaml
persistent: true
docker_cpus: "2"
docker_memory: "4g"
firewall: true
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
