# addt - AI Don't Do That

**Run AI coding agents safely in containers.** Your code stays isolated - no surprises on your host machine.

Supports **Podman** (default) and **Docker** as container runtimes.

```bash
# Install (macOS)
brew install jedi4ever/tap/addt

# Run Claude in a container
addt run claude "Fix the bug in app.js"
```

That's it. First run auto-downloads Podman (if needed) and builds the container (~2 min), then you're coding.

---

## Why?

AI agents can read, write, and execute code. Running them in containers means:
- **Isolation** - Agents can't accidentally modify your system
- **Reproducibility** - Same environment every time
- **Security** - Network firewall, resource limits, no host access
- **No daemon required** - Podman runs rootless without a background service

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

**Container runtime:** Podman is auto-downloaded if not available. You can also use Docker if preferred.

**Using Docker instead of Podman:**
```bash
export ADDT_PROVIDER=docker
addt run claude "Fix the bug"
```

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

## Project Setup

Use `addt init` to create a `.addt.yaml` config file for your project:

```bash
addt init           # Interactive setup
addt init -y        # Quick setup with smart defaults
addt init -y -f     # Overwrite existing config
```

The interactive setup asks:
1. Which AI agent to use (claude, codex, gemini, etc.)
2. Git operations needed (enables SSH forwarding)
3. Network access level (restricted, open, strict, air-gapped)
4. Workspace permissions (read-write or read-only)
5. Container persistence (ephemeral or persistent)

**Smart defaults** based on your project:
- Detects project type (Node.js, Python, Go, Rust, etc.)
- Enables SSH proxy if Git is detected
- Adds appropriate package registries to firewall allowlist
- Sets GitHub integration if `.github` or GitHub remote found

Example generated config:
```yaml
# .addt.yaml
extensions: claude
persistent: false
firewall: true
firewall_mode: strict
firewall_allowed:
  - api.anthropic.com
  - registry.npmjs.org
ssh:
  forward_keys: true
  forward_mode: proxy
github:
  forward_token: true
  token_source: gh_auth
node_version: "22"
```

Commit `.addt.yaml` to version control for team-wide consistency.

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

**For GitHub operations:** If the agent needs to create PRs, push commits, or access private repos, addt automatically picks up your token from `gh auth token` (requires [GitHub CLI](https://cli.github.com/) installed and `gh auth login` done). You can also set a token explicitly:
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

Or configure via YAML (`~/.addt/config.yaml` or `.addt.yaml`):
```yaml
ports:
  forward: true
  expose:
    - "3000"
    - "8080"
  range_start: 30000
```

Or via CLI:
```bash
addt config global set ports.expose "3000,8080"
addt config global set ports.range_start 40000
addt config global set ports.forward false   # disable port forwarding
```

### GitHub Access (private repos, PRs)

By default, addt auto-detects your GitHub token via `gh auth token` (requires [GitHub CLI](https://cli.github.com/) and `gh auth login`):

```bash
# Just works if gh CLI is installed and authenticated
addt run claude "Clone git@github.com:org/private-repo.git"
```

Or set a token explicitly:
```bash
export GH_TOKEN="ghp_..."
addt run claude "Clone the private repo and create a PR"
```

Token source options (`github.token_source`):
- **`gh_auth`** (default) — runs `gh auth token` on the host. Requires `gh` CLI installed and authenticated via `gh auth login`
- **`env`** — uses the `GH_TOKEN` environment variable as-is

To disable token forwarding entirely:
```bash
addt config set github.forward_token false
```

### SSH Keys (git over SSH)

SSH forwarding is enabled by default using agent mode. You can choose a forwarding mode:

```bash
# Default: agent mode (SSH agent socket forwarded)
addt run claude "Clone git@github.com:org/private-repo.git"

# Filter which keys are accessible (auto-enables proxy mode)
export ADDT_SSH_ALLOWED_KEYS="github,work"
addt run claude "Clone the repo"

# Alternative modes
export ADDT_SSH_FORWARD_MODE=proxy   # SSH proxy (keys never enter container)
export ADDT_SSH_FORWARD_MODE=keys    # Mount ~/.ssh read-only (less secure)
export ADDT_SSH_FORWARD_KEYS=false   # Disable SSH forwarding
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
export ADDT_CONTAINER_MEMORY=4g

# Project config (.addt.yaml)
addt config project set container.memory 4g

# Global config (~/.addt/config.yaml)
addt config global set container.memory 4g
```

All three set the same thing. Environment wins if multiple are set.

### Project Config File

Use `addt config project` to manage `.addt.yaml` (commit to git for team sharing):

```bash
addt config project set persistent true
addt config project set container.memory 4g
addt config project set firewall true
addt config project list
```

### Config Commands

```bash
# Global settings (all projects)
addt config global list
addt config global set container.memory 4g
addt config global unset container.memory

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
| `ADDT_PORTS_FORWARD=true` | Enable port forwarding (default: true) |
| `ADDT_PORTS=3000,8080` | Expose container ports |
| `ADDT_SSH_FORWARD_KEYS=true` | Enable SSH key forwarding (default: true) |
| `ADDT_SSH_FORWARD_MODE=proxy` | SSH forwarding mode: proxy, agent, or keys |
| `ADDT_SSH_ALLOWED_KEYS=github` | Filter SSH keys by comment |
| `ADDT_DOCKER_DIND_ENABLE=true` | Enable Docker-in-Docker |
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

### Shell History Persistence

Keep your bash and zsh history across container sessions:

```bash
export ADDT_HISTORY_PERSIST=true
addt run claude "Work on my project"
# Exit and re-run — your shell history is still there
```

History files are stored per-project at `~/.addt/history/<project-hash>/` on your host, and mounted as `~/.bash_history` and `~/.zsh_history` inside the container.

Configure via project config:
```bash
addt config project set history_persist true
```

### SSH Forwarding

SSH forwarding is controlled by two settings:
- `ssh.forward_keys` (bool): enable/disable SSH forwarding (default: true)
- `ssh.forward_mode` (string): forwarding method — `proxy` (default), `agent`, or `keys`

```bash
# Default: proxy mode (private keys never enter the container, works on macOS)
addt run claude "Clone the private repo"

# Agent mode: forward SSH agent socket directly (Linux only)
export ADDT_SSH_FORWARD_MODE=agent
addt run claude "Clone the private repo"

# Filter to specific keys by comment/name (auto-enables proxy mode)
export ADDT_SSH_ALLOWED_KEYS="github-personal"
addt run claude "Only github-personal key is accessible"

# Other modes
export ADDT_SSH_FORWARD_MODE=keys    # Mount ~/.ssh read-only
export ADDT_SSH_FORWARD_KEYS=false   # Disable SSH entirely
```

**Proxy mode benefits:**
- Private keys never enter the container
- Works on macOS (where agent forwarding doesn't work)
- Filter which keys are exposed with `ADDT_SSH_ALLOWED_KEYS`
- Keys matched by comment field (filename, email, etc.)

### Docker-in-Docker / Podman-in-Podman

```bash
export ADDT_DOCKER_DIND_ENABLE=true
addt run claude "Build a Docker image for this app"
```

With Podman, this enables nested Podman containers (Podman-in-Podman).

### GPG Signing

GPG forwarding supports multiple modes for different security levels:

```bash
# Agent mode - forward gpg-agent socket (most secure for signing)
export ADDT_GPG_FORWARD=agent
addt run claude "Create a signed commit"

# Proxy mode - filter which keys can sign
export ADDT_GPG_FORWARD=proxy
export ADDT_GPG_ALLOWED_KEY_IDS="ABC123,DEF456"
addt run claude "Sign with specific key only"

# Keys mode - mount ~/.gnupg read-only (legacy)
export ADDT_GPG_FORWARD=keys
addt run claude "Access GPG config"
```

**GPG mode benefits:**
- `agent`: Forward gpg-agent socket, private keys stay on host
- `proxy`: Filter which key IDs can sign operations
- `keys`: Mount entire ~/.gnupg read-only (backward compatible with `true`)

### Tmux Forwarding

Forward your host tmux session into the container for multi-pane workflows:

```bash
# Enable tmux forwarding (disabled by default)
export ADDT_TMUX_FORWARD=true
addt run claude "Work in tmux"
```

When enabled and you're running inside a tmux session, the container can:
- Access your host tmux socket
- Create new panes/windows visible on your host
- Use tmux commands to split terminals

**Note:** Only works when addt is run from within an active tmux session.

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

**Podman firewall:** When using Podman with firewall enabled, addt automatically uses the `pasta` network backend for efficient network namespace handling. The firewall works with both nftables (preferred) and iptables.

### Resource Limits

```bash
export ADDT_DOCKER_CPUS=2
export ADDT_CONTAINER_MEMORY=4g
addt run claude
```

### Security Hardening

Containers run with security defaults enabled:

| Setting | Default | Description |
|---------|---------|-------------|
| `pids_limit` | 200 | Max processes (prevents fork bombs) |
| `ulimit_nofile` | 4096:8192 | File descriptor limits |
| `ulimit_nproc` | 256:512 | Process limits |
| `no_new_privileges` | true | Prevents privilege escalation |
| `cap_drop` | [ALL] | Linux capabilities to drop |
| `cap_add` | [CHOWN, SETUID, SETGID] | Linux capabilities to add back |
| `read_only_rootfs` | false | Read-only root filesystem |
| `tmpfs_tmp_size` | 256m | Size of /tmp when read_only_rootfs is enabled |
| `tmpfs_home_size` | 512m | Size of /home/addt when read_only_rootfs is enabled |
| `network_mode` | bridge | Network mode: "bridge", "none" (air-gapped), "host" |
| `seccomp_profile` | default | Seccomp: "default", "restrictive", "unconfined", or path |
| `disable_ipc` | false | Disable IPC namespace sharing (`--ipc=none`) |
| `time_limit` | 0 | Auto-terminate after N minutes (0 = disabled) |
| `user_namespace` | "" | User namespace: "host" or "private" |
| `disable_devices` | false | Drop MKNOD capability (prevent device creation) |
| `memory_swap` | "" | Memory swap limit: "-1" to disable swap |
| `isolate_secrets` | false | Isolate secrets from child processes via tmpfs |

Configure in `~/.addt/config.yaml`:
```yaml
security:
  pids_limit: 200
  ulimit_nofile: "4096:8192"
  ulimit_nproc: "256:512"
  no_new_privileges: true
  cap_drop: [ALL]
  cap_add: [CHOWN, SETUID, SETGID]
  read_only_rootfs: true
  tmpfs_tmp_size: "100m"
  tmpfs_home_size: "500m"
  network_mode: none       # Completely disable networking (air-gapped)
  seccomp_profile: restrictive  # Use built-in restrictive syscall filter
  disable_ipc: true             # Isolate IPC namespace
  time_limit: 60                # Auto-terminate after 60 minutes
  disable_devices: true         # Prevent device file creation
  memory_swap: "-1"             # Disable swap entirely
  isolate_secrets: true         # Isolate secrets from child processes

# Mount workspace as read-only (agent can't modify your files)
workdir_readonly: true
```

Or via environment variables:
```bash
export ADDT_SECURITY_PIDS_LIMIT=500
export ADDT_SECURITY_READ_ONLY_ROOTFS=true
export ADDT_SECURITY_TMPFS_TMP_SIZE=100m
export ADDT_SECURITY_TMPFS_HOME_SIZE=500m
export ADDT_SECURITY_NETWORK_MODE=none
export ADDT_SECURITY_ISOLATE_SECRETS=true
export ADDT_WORKDIR_READONLY=true
```

### OpenTelemetry Support

Send telemetry data to an OTEL collector for observability:

| Setting | Default | Description |
|---------|---------|-------------|
| `enabled` | false | Enable OpenTelemetry |
| `endpoint` | http://host.docker.internal:4318 | OTLP endpoint URL |
| `protocol` | http/protobuf | Protocol: http/protobuf or grpc |
| `service_name` | addt | Service name for traces |
| `headers` | "" | OTLP headers (key=value,key2=value2) |

Configure in `~/.addt/config.yaml`:
```yaml
otel:
  enabled: true
  endpoint: http://host.docker.internal:4318
  protocol: http/protobuf
  service_name: my-project
```

Or via environment variables:
```bash
export ADDT_OTEL_ENABLED=true
export ADDT_OTEL_SERVICE_NAME=my-project
```

When enabled, the following environment variables are passed to the container:
- `CLAUDE_CODE_ENABLE_TELEMETRY=1` (enables Claude Code telemetry)
- `OTEL_EXPORTER_OTLP_ENDPOINT`
- `OTEL_EXPORTER_OTLP_PROTOCOL`
- `OTEL_SERVICE_NAME`
- `OTEL_EXPORTER_OTLP_HEADERS` (if configured)

The container can reach the host via `host.docker.internal` (automatically configured when OTEL is enabled).

Additional Claude Code telemetry options can be passed through from the host:
```bash
# Enable logging of user prompts (redacted by default)
export OTEL_LOG_USER_PROMPTS=1

# Enable logging of tool/MCP server names
export OTEL_LOG_TOOL_DETAILS=1

# Configure exporters
export OTEL_METRICS_EXPORTER=otlp
export OTEL_LOGS_EXPORTER=otlp
```

#### addt-otel: Simple OTEL Collector

A lightweight OTEL collector is included for debugging and development:

```bash
# Start the collector (listens on port 4318)
addt-otel

# With verbose output (show full payloads)
addt-otel --verbose

# Output as JSON lines
addt-otel --json

# Log to file
addt-otel --log /tmp/otel.log

# Custom port
addt-otel --port 4319
```

Example workflow:
```bash
# Terminal 1: Start the collector
addt-otel --verbose

# Terminal 2: Run addt with OTEL enabled
ADDT_OTEL_ENABLED=true addt run claude
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

# Developer tools
addt doctor                       # Check system health
addt completion bash              # Generate bash completions
addt completion zsh               # Generate zsh completions

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
| `ADDT_PROVIDER` | podman | Container runtime: `podman` (default), `docker`, or `daytona` |
| `ADDT_PERSISTENT` | false | Keep container running |
| `ADDT_PORTS_FORWARD` | true | Enable port forwarding |
| `ADDT_PORTS` | - | Ports to expose: `3000,8080` |
| `ADDT_DOCKER_CPUS` | - | CPU limit: `2` |
| `ADDT_CONTAINER_MEMORY` | - | Memory limit: `4g` |
| `ADDT_WORKDIR` | `.` | Working directory to mount |
| `ADDT_WORKDIR_READONLY` | false | Mount workspace as read-only |
| `ADDT_HISTORY_PERSIST` | false | Persist shell history between sessions |

### Forwarding
| Variable | Default | Description |
|----------|---------|-------------|
| `ADDT_SSH_FORWARD_KEYS` | true | Enable SSH key forwarding |
| `ADDT_SSH_FORWARD_MODE` | proxy | SSH mode: `proxy`, `agent`, or `keys` |
| `ADDT_SSH_ALLOWED_KEYS` | - | Filter SSH keys by comment: `github,work` |
| `ADDT_GPG_FORWARD` | - | GPG mode: `proxy`, `agent`, `keys`, or `off` |
| `ADDT_GPG_ALLOWED_KEY_IDS` | - | Filter GPG keys by ID: `ABC123,DEF456` |
| `ADDT_TMUX_FORWARD` | false | Forward tmux socket into container |
| `ADDT_DOCKER_DIND_ENABLE` | false | Enable Docker-in-Docker |
| `ADDT_DOCKER_DIND_MODE` | isolated | DinD mode: `isolated` or `host` |
| `ADDT_GITHUB_FORWARD_TOKEN` | true | Forward `GH_TOKEN` to container |
| `ADDT_GITHUB_TOKEN_SOURCE` | gh_auth | Token source: `gh_auth` (requires `gh` CLI) or `env` |

### Security
| Variable | Default | Description |
|----------|---------|-------------|
| `ADDT_FIREWALL` | false | Enable network firewall |
| `ADDT_FIREWALL_MODE` | strict | Mode: `strict`, `permissive`, `off` |
| `ADDT_SECURITY_PIDS_LIMIT` | 200 | Max processes in container |
| `ADDT_SECURITY_ULIMIT_NOFILE` | 4096:8192 | File descriptor limits |
| `ADDT_SECURITY_ULIMIT_NPROC` | 256:512 | Process limits |
| `ADDT_SECURITY_NO_NEW_PRIVILEGES` | true | Prevent privilege escalation |
| `ADDT_SECURITY_CAP_DROP` | ALL | Capabilities to drop (comma-separated) |
| `ADDT_SECURITY_CAP_ADD` | CHOWN,SETUID,SETGID | Capabilities to add back |
| `ADDT_SECURITY_READ_ONLY_ROOTFS` | false | Read-only root filesystem |
| `ADDT_SECURITY_TMPFS_TMP_SIZE` | 256m | Size of /tmp tmpfs |
| `ADDT_SECURITY_TMPFS_HOME_SIZE` | 512m | Size of /home/addt tmpfs |
| `ADDT_SECURITY_NETWORK_MODE` | bridge | Network mode: bridge, none, host |
| `ADDT_SECURITY_SECCOMP_PROFILE` | default | Seccomp profile to use |
| `ADDT_SECURITY_DISABLE_IPC` | false | Disable IPC namespace sharing |
| `ADDT_SECURITY_TIME_LIMIT` | 0 | Auto-terminate after N minutes |
| `ADDT_SECURITY_USER_NAMESPACE` | "" | User namespace mode |
| `ADDT_SECURITY_DISABLE_DEVICES` | false | Drop MKNOD capability |
| `ADDT_SECURITY_MEMORY_SWAP` | "" | Memory swap limit |

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

### Quick diagnostics
Run the built-in health check:
```bash
addt doctor
```
This checks Docker/Podman, API keys, disk space, and network connectivity.

### Shell completions
Enable tab completion for commands, extensions, and config keys (including namespaced keys like `github.token_source`, `security.pids_limit`, etc.):
```bash
# Bash (add to ~/.bashrc)
eval "$(addt completion bash)"

# Zsh (add to ~/.zshrc)
eval "$(addt completion zsh)"

# Fish (run once)
addt completion fish > ~/.config/fish/completions/addt.fish
```

Config keys use dot notation for namespaced settings:
```bash
addt config set github.token_source env
addt config set security.pids_limit 300
addt config get ports.forward
```

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
- [Podman](https://podman.io/getting-started/installation)
- [GitHub Tokens](https://github.com/settings/tokens)
