# addt Extensions

Every AI agent in addt is loaded as an **extension**. Extensions define how to install and run agents.

## Available Extensions

### AI Coding Agents

| Extension | Description | API Key |
|-----------|-------------|---------|
| `claude` | Claude Code by Anthropic | `ANTHROPIC_API_KEY` |
| `codex` | OpenAI Codex CLI | `OPENAI_API_KEY` |
| `gemini` | Google Gemini CLI | `GEMINI_API_KEY` |
| `copilot` | GitHub Copilot CLI | `GH_TOKEN` |
| `amp` | Sourcegraph Amp | - |
| `cursor` | Cursor CLI Agent | - |
| `kiro` | AWS Kiro CLI | `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY` |

### Claude Ecosystem

| Extension | Description | Requires |
|-----------|-------------|----------|
| `claude-flow` | Multi-agent orchestration | claude |
| `claude-sneakpeek` | Preview tool | claude |
| `openclaw` | Open source assistant | claude |
| `tessl` | AI skills package manager | claude |
| `gastown` | Multi-agent orchestration | claude, beads |

### Utilities

| Extension | Description |
|-----------|-------------|
| `beads` | Git-backed issue tracker |
| `backlog-md` | Markdown backlog management |

---

## Using Extensions

### Run an Agent

```bash
addt run claude "Fix this bug"
addt run codex "Explain this code"
addt run gemini "Review this PR"
```

### List Available Extensions

```bash
addt extensions list
```

### Extension Info

```bash
addt extensions info claude
```

---

## Configuration

### Version Pinning

```bash
# Via config
addt config extension claude set version 1.0.5

# Via environment
export ADDT_CLAUDE_VERSION=1.0.5
```

### Config Mounting & Session Resumption

By default, extension config directories (like `~/.claude` and `~/.claude.json`) are mounted from your host into the container. This enables:

- **Session resumption** - Use `--continue` or `--resume` to pick up where you left off, even though each container run is ephemeral
- **Persistent settings** - Your agent configuration persists across runs
- **No re-authentication** - Login once on host, works in every container

To disable auto-mounting (full isolation):

```bash
addt config extension claude set automount false
```

### API Keys

Extensions automatically forward their required API keys from your host. Just set them:

```bash
export ANTHROPIC_API_KEY="sk-ant-..."   # Claude
export OPENAI_API_KEY="sk-..."          # Codex
export GEMINI_API_KEY="..."             # Gemini
export GH_TOKEN="ghp_..."               # Copilot
```

---

## Creating Extensions

Create custom extensions in `~/.addt/extensions/`. Local extensions override built-in ones.

### Override Built-in Extensions

Clone a built-in extension to customize it:

```bash
# Clone to local (overrides built-in)
addt extensions clone claude

# Or clone as a new extension name
addt extensions clone claude my-claude

# Edit the local copy
vim ~/.addt/extensions/claude/install.sh

# Rebuild to apply changes
addt build claude --force
```

To remove a local extension:

```bash
addt extensions remove claude
addt extensions remove my-claude
```

### Scaffold a New Extension

```bash
addt extensions new myagent
```

This creates:
```
~/.addt/extensions/myagent/
├── config.yaml    # Metadata (required)
├── install.sh     # Build-time installation
└── setup.sh       # Runtime initialization
```

### Build and Run

```bash
addt build myagent
addt run myagent "Hello!"
```

---

## Extension Structure

### config.yaml (required)

```yaml
name: myagent
description: My custom AI agent
entrypoint: myagent-cli
default_version: latest

# Optional
dependencies:
  - claude              # Other extensions required
env_vars:
  - MY_API_KEY          # Auto-forwarded from host
mounts:
  - source: ~/.myagent
    target: /home/addt/.myagent
```

| Field | Required | Description |
|-------|----------|-------------|
| `name` | Yes | Extension identifier |
| `description` | Yes | Brief description |
| `entrypoint` | Yes | Command to run |
| `default_version` | No | Default version (`latest`, `stable`, or specific) |
| `dependencies` | No | Required extensions |
| `env_vars` | No | Environment variables to forward |
| `mounts` | No | Directories to mount |

### install.sh (optional)

Runs at **build time** to install packages:

```bash
#!/bin/bash
set -e

# System packages
sudo apt-get update && sudo apt-get install -y some-package

# Node.js packages
sudo npm install -g @some/package

# Go packages
go install github.com/user/tool@latest

# Python packages
uv pip install some-package
```

### setup.sh (optional)

Runs at **container startup**:

```bash
#!/bin/bash
echo "Initializing myagent..."

if [ -z "$MY_API_KEY" ]; then
    echo "Warning: MY_API_KEY not set"
fi
```

### args.sh (optional)

Transforms CLI arguments before execution:

```bash
#!/bin/bash
# Transform --yolo to agent-specific flag
ARGS=("$@")
for i in "${!ARGS[@]}"; do
    if [[ "${ARGS[$i]}" == "--yolo" ]]; then
        ARGS[$i]="--skip-permissions"
    fi
done
echo "${ARGS[@]}"
```

---

## Examples

### Multiple AI Agents

```bash
# Build with multiple agents
ADDT_EXTENSIONS=claude,codex,gemini addt build claude

# Run different agents
addt run claude "Fix this"
addt run codex "Explain this"
addt run gemini "Review this"
```

### Gastown (Multi-Agent)

```bash
# Build (auto-includes claude and beads)
addt build gastown

# Run
addt run gastown
```

### Tessl (Skills Manager)

```bash
addt build tessl
addt run tessl
# Then: tessl init, tessl skill search
```

---

## Troubleshooting

### Extension Not Found

```bash
# Check available extensions
addt extensions list

# Local extensions must be in ~/.addt/extensions/<name>/
```

### API Key Issues

```bash
# Verify key is set
echo $ANTHROPIC_API_KEY

# Check it's forwarded to container
addt shell claude -c "echo \$ANTHROPIC_API_KEY"
```

### Permission Errors

- Use `sudo` for `apt-get` and global `npm install`
- Go packages don't need sudo (install to `~/go/bin`)
