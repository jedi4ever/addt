# addt - AI Don't Do That

**Run AI coding agents safely in containers.** Your code stays isolated - no surprises on your host machine.

Supports **Podman** (default), **Docker**, and **OrbStack** as container runtimes.

```bash
# Install (macOS)
brew install jedi4ever/tap/addt

# Run Claude in a container
addt run claude "Fix the bug in app.js"
```

That's it. First run auto-downloads Podman (if needed) and builds the container, then you're coding.

**What happens:** addt mounts your current directory at `/workspace` inside a container, forwards your API keys, and runs the agent. Your files are editable by the agent, but your system is protected. All normal agent flags work - it's a drop-in replacement.

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

**Container runtime:** Podman is auto-downloaded if not available. To use Docker or OrbStack instead:
```bash
export ADDT_PROVIDER=docker    # or orbstack
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

**Available agents:** Built-in: `claude` `codex` `gemini` `copilot` `cursor` `tessl`. Experimental: `amp` `kiro` `claude-flow` `gastown` `beads` `openclaw` `claude-sneakpeek` `backlog-md`. Run `addt extensions list` for details.

### Set up aliases (recommended)

Add to your `~/.bashrc` or `~/.zshrc` for a seamless experience:
```bash
alias claude='addt run claude'
alias codex='addt run codex'
alias gemini='addt run gemini'

# Now use directly
claude "Fix the bug"
codex "Add unit tests"
```

Alternatively, create symlinks:
```bash
ln -s /usr/local/bin/addt /usr/local/bin/addt-claude
addt-claude "Fix the bug"
```

### Set up shell completions

```bash
# Bash (add to ~/.bashrc)
eval "$(addt completion bash)"

# Zsh (add to ~/.zshrc)
eval "$(addt completion zsh)"

# Fish (run once)
addt completion fish > ~/.config/fish/completions/addt.fish
```

---

## Authentication

Each agent uses its own API key via environment variable:

```bash
export ANTHROPIC_API_KEY="sk-ant-..."   # Claude
export OPENAI_API_KEY="sk-..."          # Codex
export GEMINI_API_KEY="..."             # Gemini
```

When `ANTHROPIC_API_KEY` is set, the container auto-configures Claude Code to skip onboarding and trust the workspace.

**Using a Claude subscription instead of API key?** Run `claude login` on your host first, then enable auto-mount:
```bash
addt config extension claude set automount true
```
This mounts `~/.claude` into the container. With auto-mount, `--continue` and `--resume` work for session resumption. If you see config conflicts, pin the container version with `ADDT_CLAUDE_VERSION` to match your local version.

---

## Common Workflows

Quick recipes for the most common scenarios:

| I want to... | Command |
|--------------|---------|
| Run Claude on my project | `addt run claude "Fix the bug"` |
| Give the agent GitHub access | `addt config set github.forward_token true` |
| Use git over SSH in container | `export ADDT_SSH_FORWARD_KEYS=true` |
| Expose a dev server port | `ADDT_PORTS=3000 addt run claude "Start the server"` |
| Keep the container between runs | `export ADDT_PERSISTENT=true` |
| Enable network firewall | `addt config set firewall.enabled true` |
| Restrict to specific domains | `addt firewall global allow api.example.com` |
| Enable yolo/auto-accept mode | `addt config set security.yolo true -g` |
| Limit CPU and memory | `addt config set container.cpus 2 -g && addt config set container.memory 4g -g` |
| Rebuild a stale container | `addt build claude --force` |
| Debug inside the container | `addt shell claude` |
| Check system health | `addt doctor` |

### GitHub Access (private repos, PRs)

Enable token forwarding to give the agent access to private repos and PRs:

```bash
addt config set github.forward_token true
addt run claude "Create a PR for this feature"
```

addt auto-detects your token via `gh auth token` (requires [GitHub CLI](https://cli.github.com/) and `gh auth login`). Or set a token explicitly:
```bash
export GH_TOKEN="ghp_..."
```

By default, `GH_TOKEN` is scoped to only the workspace repo. To allow additional repos:
```yaml
# .addt.yaml
github:
  scope_token: true
  scope_repos:
    - "myorg/shared-lib"
    - "myorg/common-config"
```

To disable scoping (allow all repos): `addt config set github.scope_token false`

Token source options (`github.token_source`): `gh_auth` (default, uses `gh` CLI) or `env` (uses `GH_TOKEN` directly).

### SSH Keys (git over SSH)

SSH forwarding uses proxy mode by default - private keys never enter the container:

```bash
export ADDT_SSH_FORWARD_KEYS=true
addt run claude "Clone git@github.com:org/private-repo.git"

# Filter which keys are accessible
export ADDT_SSH_ALLOWED_KEYS="github,work"
```

Other modes: `proxy` (default, most secure), `agent` (Linux only), `keys` (mounts ~/.ssh read-only).

### Web Development (port mapping)

```bash
ADDT_PORTS="3000,8080" addt run claude "Create an Express server on port 3000"
```

Or configure permanently:
```bash
addt config set ports.expose "3000,8080" -g
```

Container ports are mapped starting at host port 30000 by default (configurable with `ports.range_start`).

### Persistent Mode

By default, containers are ephemeral. For faster startup, keep them running:
```bash
export ADDT_PERSISTENT=true
claude "Start a feature"     # Creates container
claude "Continue working"    # Reuses container
```

### Network Firewall

Control which domains the agent can access:

```bash
addt config set firewall.enabled true -g

# Manage allowed/denied domains
addt firewall global allow api.example.com
addt firewall global deny malware.com
addt firewall global list
```

Project rules override global rules:
```bash
addt firewall global deny registry.npmjs.org    # Deny globally
addt firewall project allow registry.npmjs.org  # But allow for this project
```

Rule evaluation order: `Defaults -> Extension -> Global -> Project` (most specific wins).

---

## Project Setup

Use `addt init` to create a `.addt.yaml` config file for your project:

```bash
addt init           # Interactive setup
addt init -y        # Quick setup with smart defaults
addt init -y -f     # Overwrite existing config
```

The interactive setup asks about your agent, git needs, network access, workspace permissions, and container persistence.

**Smart defaults** detect your project type (Node.js, Python, Go, etc.) and configure appropriate package registries, SSH proxy, and GitHub integration.

Example generated config:
```yaml
# .addt.yaml
extensions: claude
persistent: false
firewall:
  enabled: true
  mode: strict
  allowed:
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

## Configuration

Three ways to configure addt:

| Method | Location | Use case |
|--------|----------|----------|
| **Environment variable** | Shell | Quick overrides, CI/CD |
| **Project config** | `.addt.yaml` in project | Team-shared settings, per-project defaults |
| **Global config** | `~/.addt/config.yaml` | Personal defaults across all projects |

**Precedence** (highest to lowest): Environment -> Project -> Global -> Defaults

### Config Commands

```bash
# Project settings (.addt.yaml)
addt config list
addt config set firewall.enabled true
addt config unset firewall.enabled

# Global settings (~/.addt/config.yaml)
addt config list -g
addt config set container.memory 4g -g
addt config unset container.memory -g

# Per-extension settings
addt config extension claude set version 1.0.5
addt config extension claude list
```

### Security Profiles

Apply preconfigured profiles to quickly set multiple settings:

```bash
addt profile list              # List available profiles
addt profile apply strict      # Apply a profile
```

Built-in profiles:
- **develop** -- Relaxed settings for development (firewall off, no read-only rootfs)
- **strict** -- Tighter security (firewall on, reduced capabilities, secrets isolation)
- **paranoia** -- Maximum lockdown (read-only rootfs, air-gapped network, time limits)

### Config Audit

Review your current security posture:
```bash
addt config audit
```

Shows which security settings are enabled/disabled with color-coded severity levels.

### Common Environment Variables

| Variable | Description |
|----------|-------------|
| `ADDT_PROVIDER=docker` | Container runtime: `podman`, `docker`, or `orbstack` |
| `ADDT_PERSISTENT=true` | Keep container running between sessions |
| `ADDT_PORTS=3000,8080` | Expose container ports |
| `ADDT_SSH_FORWARD_KEYS=true` | Enable SSH key forwarding |
| `ADDT_FIREWALL=true` | Enable network firewall |
| `ADDT_CONTAINER_MEMORY=4g` | Memory limit |
| `ADDT_CONTAINER_CPUS=2` | CPU limit |
| `ADDT_DOCKER_DIND_ENABLE=true` | Enable Docker-in-Docker |

See [Full Reference](#environment-variables-reference) for all options.

---

## Advanced Features

### Shell History Persistence

Keep your bash and zsh history across container sessions:

```bash
addt config set history_persist true
```

History files are stored per-project at `~/.addt/history/<project-hash>/` on your host.

### SSH Forwarding Modes

SSH forwarding is controlled by `ssh.forward_keys` and `ssh.forward_mode`:

| Mode | How it works | When to use |
|------|-------------|-------------|
| `proxy` (default) | Private keys never enter container | macOS, most secure |
| `agent` | Forwards SSH agent socket | Linux only |
| `keys` | Mounts ~/.ssh read-only | Legacy / fallback |

```bash
# Filter to specific keys by comment/name
export ADDT_SSH_ALLOWED_KEYS="github-personal"
```

### Docker-in-Docker / Podman-in-Podman

```bash
export ADDT_DOCKER_DIND_ENABLE=true
addt run claude "Build a Docker image for this app"
```

### GPG Signing

```bash
# Agent mode - forward gpg-agent socket (most secure)
export ADDT_GPG_FORWARD=agent
addt run claude "Create a signed commit"

# Proxy mode - filter which keys can sign
export ADDT_GPG_FORWARD=proxy
export ADDT_GPG_ALLOWED_KEY_IDS="ABC123,DEF456"

# Keys mode - mount ~/.gnupg read-only (legacy)
export ADDT_GPG_FORWARD=keys
```

### Git Config Forwarding

Your `.gitconfig` is automatically forwarded to the container (enabled by default):

```bash
addt config set git.forward_config false     # Disable
addt config set git.config_path /custom/path # Custom path
```

### Custom SSH/GPG Directories

```bash
addt config set ssh.dir /path/to/custom/.ssh
addt config set gpg.dir /path/to/custom/.gnupg
```

### Tmux Forwarding

Forward your host tmux session into the container for multi-pane workflows:

```bash
export ADDT_TMUX_FORWARD=true
addt run claude "Work in tmux"
```

Only works when addt is run from within an active tmux session.

### Terminal OSC Support

Enable terminal identification forwarding for clipboard access (OSC 52) and hyperlinks:

```bash
addt config set terminal.osc true
```

### Resource Limits

```bash
addt config set container.cpus 2 -g
addt config set container.memory 4g -g
```

Or via environment: `ADDT_CONTAINER_CPUS=2 ADDT_CONTAINER_MEMORY=4g`

### Complete Isolation (no workdir mount)

```bash
ADDT_WORKDIR_AUTOMOUNT=false addt run claude "Work without access to host files"
```

### Version Pinning

```bash
export ADDT_CLAUDE_VERSION=1.0.5
export ADDT_NODE_VERSION=20
addt run claude
```

### Experimental Extensions

8 additional extensions are available in `extensions_experimental/`: `amp`, `kiro`, `claude-flow`, `gastown`, `beads`, `openclaw`, `claude-sneakpeek`, `backlog-md`. To install one:

```bash
cp -r extensions_experimental/amp ~/.addt/extensions/amp
addt run amp "Hello!"
```

### Custom Extensions

```bash
addt extensions new myagent
# Edit ~/.addt/extensions/myagent/
addt build myagent
addt run myagent "Hello!"
```

See [docs/extensions.md](docs/extensions.md) for details.

### Security Hardening

Containers run with security defaults:

| Setting | Default | Description |
|---------|---------|-------------|
| `pids_limit` | 200 | Max processes (prevents fork bombs) |
| `no_new_privileges` | true | Prevents privilege escalation |
| `cap_drop` | [ALL] | Drop all Linux capabilities |
| `cap_add` | [CHOWN, SETUID, SETGID] | Add back minimal capabilities |
| `read_only_rootfs` | false | Read-only root filesystem |
| `network_mode` | "" | `bridge`, `none` (air-gapped), `host` |
| `seccomp_profile` | default | `default`, `restrictive`, `unconfined`, or path |
| `time_limit` | 0 | Auto-terminate after N minutes (0 = disabled) |
| `isolate_secrets` | false | Isolate secrets from child processes |
| `yolo` | false | Enable yolo mode globally for all extensions |
| `audit_log` | false | Enable security audit logging |

**Global yolo mode**: `addt config set security.yolo true -g` enables auto-accept across all extensions. Per-extension overrides take precedence:
```bash
addt config extension claude set yolo false   # Disable for claude only
```

**Git hooks neutralization** (enabled by default): Sets `core.hooksPath=/dev/null` via `GIT_CONFIG_COUNT` to prevent malicious git hooks. Disable with `addt config set git.disable_hooks false` if you need pre-commit/lint-staged hooks.

**Credential scrubbing**: API keys and secrets are overwritten with random data before being unset inside the container, preventing recovery from `/proc/*/environ` or memory dumps.

Configure in `~/.addt/config.yaml`:
```yaml
security:
  pids_limit: 200
  no_new_privileges: true
  cap_drop: [ALL]
  cap_add: [CHOWN, SETUID, SETGID]
  read_only_rootfs: true
  network_mode: none
  seccomp_profile: restrictive
  time_limit: 60
  isolate_secrets: true

workdir:
  readonly: true
```

### OpenTelemetry Support

Send telemetry data to an OTEL collector for observability:

```yaml
# ~/.addt/config.yaml
otel:
  enabled: true
  endpoint: http://host.docker.internal:4318
  protocol: http/json
  service_name: my-project
```

Or via environment: `ADDT_OTEL_ENABLED=true ADDT_OTEL_SERVICE_NAME=my-project`

A lightweight collector is included for debugging:
```bash
# Terminal 1: Start the collector
addt-otel --verbose

# Terminal 2: Run addt with OTEL enabled
ADDT_OTEL_ENABLED=true addt run claude
```

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
addt build claude --rebuild-base  # Rebuild base image too
addt shell <agent>                # Open shell in container
addt containers list              # List running containers
addt containers clean             # Remove all containers
addt update <agent> [version]     # Force-rebuild agent to version

# Configuration
addt config list                  # Show project settings
addt config list -g               # Show global settings
addt config set <k> <v>           # Set project setting
addt config set <k> <v> -g       # Set global setting
addt config extension <n> list    # Show extension settings
addt config audit                 # Review security posture

# Profiles
addt profile list                 # List available profiles
addt profile show <name>          # Show profile details
addt profile apply <name>         # Apply a security profile

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
addt extensions clone <src> [dst] # Clone extension from source
addt extensions remove <name>     # Remove local extension

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
| `ADDT_PROVIDER` | podman | Container runtime: `podman` (default), `docker`, or `orbstack` |
| `ADDT_PERSISTENT` | false | Keep container running |
| `ADDT_PORTS_FORWARD` | true | Enable port forwarding |
| `ADDT_PORTS` | - | Ports to expose: `3000,8080` |
| `ADDT_PORT_RANGE_START` | 30000 | Starting port for auto allocation |
| `ADDT_CONTAINER_CPUS` | 2 | CPU limit |
| `ADDT_CONTAINER_MEMORY` | 4g | Memory limit |
| `ADDT_WORKDIR` | `.` | Working directory to mount |
| `ADDT_WORKDIR_READONLY` | false | Mount workspace as read-only |
| `ADDT_HISTORY_PERSIST` | false | Persist shell history between sessions |
| `ADDT_VM_CPUS` | 4 | VM CPU allocation (Podman machine/Docker Desktop) |
| `ADDT_VM_MEMORY` | 8192 | VM memory in MB (Podman machine/Docker Desktop) |

### Forwarding
| Variable | Default | Description |
|----------|---------|-------------|
| `ADDT_SSH_FORWARD_KEYS` | false | Enable SSH key forwarding |
| `ADDT_SSH_FORWARD_MODE` | proxy | SSH mode: `proxy`, `agent`, or `keys` |
| `ADDT_SSH_ALLOWED_KEYS` | - | Filter SSH keys by comment: `github,work` |
| `ADDT_SSH_DIR` | - | Custom SSH directory path |
| `ADDT_GPG_FORWARD` | - | GPG mode: `proxy`, `agent`, `keys`, or `off` |
| `ADDT_GPG_ALLOWED_KEY_IDS` | - | Filter GPG keys by ID: `ABC123,DEF456` |
| `ADDT_GPG_DIR` | - | Custom GPG directory path |
| `ADDT_TMUX_FORWARD` | false | Forward tmux socket into container |
| `ADDT_TERMINAL_OSC` | false | Forward terminal identification for OSC support |
| `ADDT_DOCKER_DIND_ENABLE` | false | Enable Docker-in-Docker |
| `ADDT_DOCKER_DIND_MODE` | isolated | DinD mode: `isolated` or `host` |
| `ADDT_GITHUB_FORWARD_TOKEN` | false | Forward `GH_TOKEN` to container |
| `ADDT_GITHUB_TOKEN_SOURCE` | gh_auth | Token source: `gh_auth` (requires `gh` CLI) or `env` |
| `ADDT_GITHUB_SCOPE_TOKEN` | true | Scope `GH_TOKEN` to workspace repo via git credential-cache |
| `ADDT_GITHUB_SCOPE_REPOS` | - | Additional repos for scoping: `myorg/repo1,myorg/repo2` |

### Security
| Variable | Default | Description |
|----------|---------|-------------|
| `ADDT_GIT_DISABLE_HOOKS` | true | Neutralize git hooks inside container |
| `ADDT_GIT_FORWARD_CONFIG` | true | Forward .gitconfig to container |
| `ADDT_GIT_CONFIG_PATH` | - | Custom .gitconfig file path |
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
| `ADDT_SECURITY_NETWORK_MODE` | "" | Network mode: bridge, none, host (empty = provider default) |
| `ADDT_SECURITY_SECCOMP_PROFILE` | default | Seccomp profile to use |
| `ADDT_SECURITY_DISABLE_IPC` | false | Disable IPC namespace sharing |
| `ADDT_SECURITY_TIME_LIMIT` | 0 | Auto-terminate after N minutes |
| `ADDT_SECURITY_USER_NAMESPACE` | "" | User namespace mode |
| `ADDT_SECURITY_DISABLE_DEVICES` | false | Drop MKNOD capability |
| `ADDT_SECURITY_MEMORY_SWAP` | "" | Memory swap limit |
| `ADDT_SECURITY_YOLO` | false | Enable yolo mode globally for all extensions |
| `ADDT_SECURITY_ISOLATE_SECRETS` | true | Isolate secrets from child processes |
| `ADDT_SECURITY_AUDIT_LOG` | false | Enable security audit logging |
| `ADDT_SECURITY_AUDIT_LOG_FILE` | - | Path to audit log file (default: `~/.addt/audit.log`) |

### Paths & Logging
| Variable | Default | Description |
|----------|---------|-------------|
| `ADDT_ENV_FILE_LOAD` | true | Load .env file |
| `ADDT_ENV_FILE` | .env | Env file to load |
| `ADDT_ENV_VARS` | ANTHROPIC_API_KEY,GH_TOKEN | Vars to forward |
| `ADDT_LOG` | false | Enable logging |
| `ADDT_LOG_OUTPUT` | stderr | Output target: `stderr`, `stdout`, or `file` |
| `ADDT_LOG_FILE` | addt.log | Log file name |
| `ADDT_LOG_DIR` | ~/.addt/logs | Log directory |
| `ADDT_LOG_LEVEL` | INFO | Log level: `DEBUG`, `INFO`, `WARN`, `ERROR` |
| `ADDT_LOG_MODULES` | * | Comma-separated module filter |
| `ADDT_LOG_ROTATE` | false | Enable log rotation |
| `ADDT_LOG_MAX_SIZE` | 10m | Max file size before rotating |
| `ADDT_LOG_MAX_FILES` | 5 | Number of rotated files to keep |
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
```bash
addt doctor
```
Checks Docker/Podman, API keys, disk space, and network connectivity.

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

Credential scrubbing inspired by [IngmarKrusch/claude-docker](https://github.com/IngmarKrusch/claude-docker).

## License

MIT - See LICENSE file.

## Links

- [Claude Code](https://github.com/anthropics/claude-code)
- [Docker](https://docs.docker.com/get-docker/)
- [Podman](https://podman.io/getting-started/installation)
- [GitHub Tokens](https://github.com/settings/tokens)
