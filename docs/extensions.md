# addt Extensions

Extensions allow you to add tools and AI agents to your addt container image. The base image provides infrastructure (Node.js, Go, Python/UV, Git, GitHub CLI), and extensions add the actual tools.

## Available Extensions

### AI Coding Agents

| Extension | Description | Entrypoint | Provider | API Key |
|-----------|-------------|------------|----------|---------|
| `claude` | Claude Code - AI coding assistant | `claude` | Anthropic | `ANTHROPIC_API_KEY` |
| `codex` | OpenAI Codex CLI - AI coding assistant | `codex` | OpenAI | `OPENAI_API_KEY` |
| `gemini` | Gemini CLI - AI coding agent | `gemini` | Google | `GEMINI_API_KEY`, `GOOGLE_API_KEY` |
| `copilot` | GitHub Copilot CLI - AI coding assistant | `copilot` | GitHub | `GH_TOKEN`, `GITHUB_TOKEN` |
| `amp` | Amp - AI coding agent | `amp` | Sourcegraph | - |
| `cursor` | Cursor CLI Agent - AI-powered code editor agent | `cursor` | Cursor | - |
| `kiro` | Kiro CLI - AI-powered development agent | `kiro-cli` | AWS | `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY` |

### Claude Ecosystem Extensions

| Extension | Description | Entrypoint | Dependencies |
|-----------|-------------|------------|--------------|
| `claude-flow` | Multi-agent orchestration platform for Claude | `claude-flow` | claude |
| `claude-sneakpeek` | Preview tool for Claude Code | `claudesp` | claude |
| `openclaw` | Open source personal AI assistant | `openclaw` | claude |
| `tessl` | Agent enablement platform - package manager for AI agent skills | `tessl` | claude |
| `gastown` | Multi-agent orchestration for Claude Code | `gt` | claude, beads |

### Utility Extensions

| Extension | Description | Entrypoint | Dependencies |
|-----------|-------------|------------|--------------|
| `beads` | Git-backed issue tracker for AI agents | `bd` | - |
| `backlog-md` | Markdown-based backlog management for AI agents | `backlog` | - |

**Note:** The `claude` extension is installed by default. When you build with extensions that have dependencies (like `gastown`), their dependencies are automatically installed.

## Using Extensions

### Building with Extensions

Use the `addt build` subcommand with `--build-arg` to include extensions:

```bash
# Default build (installs claude extension)
claude addt build

# Build with gastown (automatically includes claude and beads dependencies)
claude addt build --build-arg ADDT_EXTENSIONS=gastown

# Build with multiple extensions
claude addt build --build-arg ADDT_EXTENSIONS=claude,codex,gemini

# Build minimal image with only tessl (no claude)
claude addt build --build-arg ADDT_EXTENSIONS=tessl

# Via environment variable
ADDT_EXTENSIONS=gastown claude addt build
```

### Image Naming Convention

Docker images are automatically named based on the installed extensions and their versions:

```bash
# Single extension
addt:claude-stable

# Multiple extensions (sorted alphabetically)
addt:claude-stable_codex-latest

# Different combination = different image
addt:gemini-latest_tessl-latest
```

This ensures that different extension combinations always get their own isolated images.

### Extension Dependencies

Extensions can depend on other extensions. Dependencies are automatically resolved and installed in the correct order.

For example, `gastown` depends on `claude` and `beads`, so running:

```bash
claude addt build --build-arg ADDT_EXTENSIONS=gastown
```

Will automatically install `claude`, `beads`, and `gastown`.

### Checking Installed Extensions

After building, you can verify installed extensions:

```bash
# Check extension metadata
claude addt shell -c "cat ~/.addt/extensions.json"

# Check specific tools
claude addt shell -c "which claude gt bd tessl"

# List available extensions
addt extensions list
```

### Symlink-Based Extension Selection

You can create symlinks to the `addt` binary with names matching your extensions. When invoked via a symlink, addt automatically uses that extension:

```bash
# Create symlinks
ln -s /usr/local/bin/addt ~/bin/codex
ln -s /usr/local/bin/addt ~/bin/gemini
ln -s /usr/local/bin/addt ~/bin/claude-flow

# Now these are equivalent:
codex "help me with this code"           # Uses codex extension
ADDT_EXTENSIONS=codex addt "..."         # Same result

gemini "explain this function"           # Uses gemini extension
ADDT_EXTENSIONS=gemini addt "..."        # Same result
```

**How it works:**
- Detects the binary name from how it was invoked
- If not "addt", sets `ADDT_EXTENSIONS` and `ADDT_COMMAND` to match the binary name
- Environment variables can still override this behavior

This is useful for:
- Creating dedicated commands for different AI agents
- Simplifying workflows when you frequently use a specific agent
- Installing multiple "binaries" from a single addt installation

### Per-Extension Configuration

Each extension can be configured via config file or environment variables:

**Using config file (recommended for persistent settings):**

```bash
# Set version for a specific extension
addt config extension claude set version 2.0.0
addt config extension codex set version 0.1.0

# Disable config directory mounting
addt config extension claude set automount false

# View extension settings
addt config extension claude list
```

**Using environment variables (override config file):**

```bash
# Set version for a specific extension
ADDT_CLAUDE_VERSION=2.0.0 claude addt build
ADDT_CODEX_VERSION=0.1.0 claude addt build

# Disable config directory mounting for an extension
ADDT_CLAUDE_AUTOMOUNT=false addt

# Multiple extensions with specific versions
ADDT_EXTENSIONS=claude,codex \
  ADDT_CLAUDE_VERSION=2.1.0 \
  ADDT_CODEX_VERSION=latest \
  claude addt build
```

| Config Key | Env Variable Pattern | Description |
|------------|---------------------|-------------|
| `version` | `ADDT_<EXT>_VERSION` | Version to install (e.g., `2.1.0`, `latest`, `stable`) |
| `automount` | `ADDT_<EXT>_AUTOMOUNT` | Mount extension config dirs (`true`/`false`) |

**Configuration precedence:** Environment variables > Config file > Defaults

### Automatic Environment Variable Forwarding

Extensions can declare which environment variables they need in their `config.yaml`. When running addt, these variables are automatically forwarded from your host to the container - no need to specify them manually.

**Example extension configs:**

```yaml
# claude extension
env_vars:
  - ANTHROPIC_API_KEY

# codex extension
env_vars:
  - OPENAI_API_KEY

# gemini extension
env_vars:
  - GEMINI_API_KEY
  - GOOGLE_API_KEY

# kiro extension
env_vars:
  - AWS_ACCESS_KEY_ID
  - AWS_SECRET_ACCESS_KEY
  - AWS_SESSION_TOKEN
  - AWS_REGION
```

**How it works:**

1. When you build an image, each extension's `env_vars` are collected into `~/.addt/extensions.json`
2. At runtime, addt reads this metadata and automatically forwards listed variables from host to container
3. Variables are only forwarded if they're set on the host (empty values are skipped)

**Benefits:**

- No need to remember which API keys each tool needs
- Just set the variable on your host once, it's automatically available in containers
- Different extensions in the same image can have different env vars
- Users can still add additional variables via `ADDT_ENV_VARS`

**Example:**

```bash
# Just set your API keys on the host
export ANTHROPIC_API_KEY="sk-ant-..."
export OPENAI_API_KEY="sk-..."

# Build with both extensions
claude addt build --build-arg ADDT_EXTENSIONS=claude,codex

# Run - both API keys are automatically forwarded
addt "help me with this code"        # Uses ANTHROPIC_API_KEY
ADDT_COMMAND=codex addt "..."        # Uses OPENAI_API_KEY
```

## Local Extensions

You can create custom extensions in `~/.addt/extensions/` without modifying the addt source code. Local extensions override built-in extensions with the same name.

### Creating a Local Extension

Use the `addt extensions new` command to scaffold a new extension:

```bash
# Create a new extension
addt extensions new myagent

# This creates:
# ~/.addt/extensions/myagent/
#   ├── config.yaml    # Extension metadata (required)
#   ├── install.sh     # Installation script (runs at build time)
#   └── setup.sh       # Setup script (runs at container startup)
```

After editing the files, build and run your extension:

```bash
# Build image with your extension
addt build myagent

# Run your extension
addt run myagent "Hello!"

# Or create a symlink for direct access
ln -s /usr/local/bin/addt ~/bin/myagent
myagent "Hello!"
```

### Local Extension Priority

When an extension exists in both `~/.addt/extensions/` and the built-in extensions:
- Local extension takes priority
- Allows customizing built-in extensions without forking

```bash
# List extensions (shows source column)
addt extensions list

#   Name     Entrypoint   Version  Source    Description
#   myagent  myagent      latest   local     My custom agent
#   claude   claude       stable   built-in  Claude Code - AI coding assistant
```

## Creating Extensions

Extensions are stored in `src/extensions/` (built-in) or `~/.addt/extensions/` (local) as directories containing:

```
src/extensions/
└── myextension/
    ├── config.yaml    # Extension metadata (required)
    ├── install.sh     # Installation script (optional, runs at build time)
    ├── setup.sh       # Setup script (optional, runs at container startup)
    └── args.sh        # Argument transformation (optional, runs before command)
```

**Note:** Only `config.yaml` is required. Extensions can be metadata-only (no install.sh or setup.sh) if they just need to define mounts or dependencies.

### config.yaml

Defines extension metadata:

```yaml
name: myextension
description: Short description of what the extension does
entrypoint: mycommand
default_version: latest
auto_mount: true
dependencies:
  - beads           # Other extensions this depends on
env_vars:
  - MY_API_KEY      # Environment variables to forward from host
  - MY_SECRET_TOKEN
mounts:
  - source: ~/.myextension
    target: /home/addt/.myextension
  - source: ~/.config/myextension
    target: /home/addt/.config/myextension
flags:
  - flag: "--yolo"
    description: "Bypass permission checks"
```

| Field | Required | Description |
|-------|----------|-------------|
| `name` | Yes | Extension identifier (should match directory name) |
| `description` | Yes | Brief description |
| `entrypoint` | Yes | Main command provided by extension |
| `default_version` | No | Default version to install (`latest`, `stable`, or specific) |
| `auto_mount` | No | Whether to auto-mount config directories (default: true) |
| `dependencies` | No | List of other extensions required |
| `env_vars` | No | Environment variables to automatically forward from host |
| `mounts` | No | Directories to mount from host to container |
| `flags` | No | Extension-specific CLI flags |

### Extension Files

| File | Required | When it runs | Description |
|------|----------|--------------|-------------|
| `config.yaml` | Yes | Build time | Extension metadata and configuration |
| `install.sh` | No | Build time | Installs packages and tools into the image |
| `setup.sh` | No | Runtime | Runs at container startup for initialization |
| `args.sh` | No | Runtime | Transforms CLI arguments before command execution |

### Mounts

Extensions can specify directories to be mounted from the host into the container at runtime. This is useful for:

- Persisting extension configuration across container restarts
- Sharing data between host and container
- Caching extension data

Each mount entry requires:
- `source`: Path on the host (supports `~` for home directory)
- `target`: Path inside the container

The host directories are automatically created if they don't exist.

### install.sh

The installation script runs during Docker image build. It has access to:

- **apt** (via `sudo`) - for system packages
- **npm** (via `sudo`) - for Node.js packages
- **go** - for Go packages (installed to `~/go/bin`)
- **pip/uv** - for Python packages

Example install script:

```bash
#!/bin/bash
set -e

echo "Extension [myextension]: Installing..."

# System packages (requires sudo)
sudo apt-get update && sudo apt-get install -y --no-install-recommends \
    some-package

# Node.js packages (requires sudo for global)
sudo npm install -g @some/package

# Go packages (no sudo needed, installs to ~/go/bin)
/usr/local/go/bin/go install github.com/user/repo/cmd/tool@latest

# Python packages
uv pip install some-package

echo "Extension [myextension]: Done."
```

### setup.sh (Optional)

The setup script runs at container startup (runtime), not during image build. Use it for:

- Initializing runtime state
- Displaying welcome messages
- Checking for required environment variables
- Setting up runtime configuration

Example setup script:

```bash
#!/bin/bash
echo "Setup [myextension]: Initializing environment"

# Check for required API key
if [ -z "$MY_API_KEY" ]; then
    echo "Warning: MY_API_KEY not set"
fi
```

Setup scripts run once per container session. In persistent mode, they only run on the first start (a marker file prevents re-running).

### args.sh (Optional)

The args transformation script runs before command execution. Use it to:

- Transform flags (e.g., `--yolo` to agent-specific flags)
- Add default arguments
- Modify command behavior

Example args script:

```bash
#!/bin/bash
# Transform --yolo flag for this extension
ARGS=("$@")
for i in "${!ARGS[@]}"; do
    if [[ "${ARGS[$i]}" == "--yolo" ]]; then
        ARGS[$i]="--dangerously-skip-permissions"
    fi
done
echo "${ARGS[@]}"
```

### Testing Your Extension

1. Create the extension directory and files in `src/extensions/myextension/`
2. Rebuild addt: `make build`
3. Build image with extension: `./dist/addt addt build --build-arg ADDT_EXTENSIONS=myextension`
4. Verify: `./dist/addt addt shell -c "which mycommand"`

## Extension Metadata

When extensions are installed, metadata is written to `~/.addt/extensions.json`:

```json
{
  "extensions": {
    "claude": {
      "name": "claude",
      "description": "Claude Code - AI coding assistant by Anthropic",
      "entrypoint": "claude",
      "mounts": [
        {"source": "~/.claude", "target": "/home/addt/.claude"},
        {"source": "~/.claude.json", "target": "/home/addt/.claude.json"}
      ],
      "flags": [
        {"flag": "--yolo", "description": "Bypass permission checks"}
      ],
      "env_vars": ["ANTHROPIC_API_KEY"]
    },
    "gastown": {
      "name": "gastown",
      "description": "Multi-agent orchestration for Claude Code",
      "entrypoint": "gt",
      "mounts": [
        {"source": "~/.gastown", "target": "/home/addt/.gastown"}
      ],
      "env_vars": ["ANTHROPIC_API_KEY"]
    }
  }
}
```

This metadata is used at runtime to:
- Mount extension directories from the host
- Discover available extensions and their entrypoints
- Automatically forward required environment variables

## Examples

### AI Coding Agents

You can build images with different AI coding agents and switch between them:

```bash
# Build with multiple AI agents
claude addt build --build-arg ADDT_EXTENSIONS=claude,codex,gemini,copilot

# Run Claude (default)
addt

# Run OpenAI Codex
ADDT_COMMAND=codex addt

# Run Google Gemini
ADDT_COMMAND=gemini addt

# Run GitHub Copilot
ADDT_COMMAND=copilot addt

# Run Sourcegraph Amp
ADDT_COMMAND=amp addt

# Run Cursor Agent
ADDT_COMMAND=cursor addt

# Run AWS Kiro
ADDT_COMMAND=kiro-cli addt
```

**Using symlinks for dedicated agent commands:**

```bash
# Create symlinks for each agent
cd /usr/local/bin  # or wherever addt is installed
ln -s addt codex
ln -s addt gemini
ln -s addt copilot

# Build images for each (first run will auto-build)
codex addt build
gemini addt build

# Now use them directly
codex "refactor this function"
gemini "explain this code"
```

Each symlink automatically builds and uses its own isolated image (`addt:codex-latest`, `addt:gemini-latest`, etc.).

### Claude Ecosystem

Claude has several companion extensions:

```bash
# Build with Claude Flow for multi-agent orchestration
claude addt build --build-arg ADDT_EXTENSIONS=claude-flow
ADDT_COMMAND=claude-flow addt

# Build with OpenClaw (open source assistant)
claude addt build --build-arg ADDT_EXTENSIONS=openclaw
ADDT_COMMAND=openclaw addt

# Build with Claude Sneakpeek (preview tool)
claude addt build --build-arg ADDT_EXTENSIONS=claude-sneakpeek
ADDT_COMMAND=claudesp addt
```

### Gastown Extension

Gastown provides multi-agent orchestration for Claude Code:

```bash
# Build with gastown (includes claude and beads)
claude addt build --build-arg ADDT_EXTENSIONS=gastown

# Run gastown instead of claude
ADDT_COMMAND=gt addt

# Or use shell mode
claude addt shell
gt --help
```

### Tessl Extension

Tessl is an agent enablement platform with a skills package manager:

```bash
# Build with tessl
claude addt build --build-arg ADDT_EXTENSIONS=tessl

# Use tessl
claude addt shell
tessl init           # Authenticate
tessl skill search   # Find skills
tessl mcp            # Start MCP server
```

### Kiro Extension (AWS)

Kiro is AWS's AI-powered development agent:

```bash
# Build with kiro
claude addt build --build-arg ADDT_EXTENSIONS=kiro

# Set AWS credentials
export AWS_ACCESS_KEY_ID="your-key"
export AWS_SECRET_ACCESS_KEY="your-secret"
export AWS_REGION="us-east-1"

# Run kiro
ADDT_COMMAND=kiro-cli addt
```

## Troubleshooting

### Permission Errors

If you see permission errors during installation:
- Use `sudo` for `apt-get` and global `npm install`
- Go packages don't need sudo (install to user's `~/go/bin`)

### Extension Not Found

If an extension is not recognized:
- Ensure the directory name matches the extension name in `config.yaml`
- Check that `config.yaml` exists (install.sh and setup.sh are optional)
- For built-in extensions: Rebuild addt with `make build` to embed the new extension
- For local extensions: Ensure the extension is in `~/.addt/extensions/<name>/`
- Run `addt extensions list` to see available extensions and their source

### API Key Issues

If the agent can't authenticate:
- Check the required environment variables in the extension table above
- Verify the variable is set on your host: `echo $ANTHROPIC_API_KEY`
- Check it's being forwarded: `claude addt shell -c "echo \$ANTHROPIC_API_KEY"`
