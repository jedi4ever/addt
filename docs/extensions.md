# DClaude Extensions

Extensions allow you to add additional tools and capabilities to your DClaude container image.

## Available Extensions

| Extension | Description | Entrypoint |
|-----------|-------------|------------|
| `beads` | Git-backed issue tracker for AI agents | `bd` |
| `gastown` | Multi-agent orchestration for Claude Code | `gt` |
| `tessl` | Agent enablement platform - package manager for AI agent skills | `tessl` |

## Using Extensions

### Building with Extensions

Use the `build` command with `--build-arg` to include extensions:

```bash
# Single extension
dclaude build --build-arg DCLAUDE_EXTENSIONS=gastown

# Multiple extensions (comma-separated)
dclaude build --build-arg DCLAUDE_EXTENSIONS=beads,tessl

# Via environment variable
DCLAUDE_EXTENSIONS=gastown dclaude build
```

### Extension Dependencies

Extensions can depend on other extensions. Dependencies are automatically resolved and installed in the correct order.

For example, `gastown` depends on `beads`, so running:

```bash
dclaude build --build-arg DCLAUDE_EXTENSIONS=gastown
```

Will automatically install both `beads` and `gastown`.

### Checking Installed Extensions

After building, you can verify installed extensions:

```bash
# Check extension metadata
dclaude shell -c "cat ~/.dclaude/extensions.json"

# Check specific tools
dclaude shell -c "which gt bd tessl"
```

## Creating Extensions

Extensions are stored in `src/assets/docker/extensions/` as directories containing:

```
extensions/
└── myextension/
    ├── config.yaml    # Extension metadata
    └── install.sh     # Installation script
```

### config.yaml

Defines extension metadata:

```yaml
name: myextension
description: Short description of what the extension does
entrypoint: mycommand
dependencies:
  - beads           # Other extensions this depends on
mounts:
  - source: ~/.myextension
    target: /home/claude/.myextension
  - source: ~/.config/myextension
    target: /home/claude/.config/myextension
```

| Field | Required | Description |
|-------|----------|-------------|
| `name` | Yes | Extension identifier (should match directory name) |
| `description` | Yes | Brief description |
| `entrypoint` | Yes | Main command provided by extension |
| `dependencies` | No | List of other extensions required |
| `mounts` | No | Directories to mount from host to container |

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

### Testing Your Extension

1. Create the extension directory and files
2. Build dclaude: `make build`
3. Build image with extension: `./dist/dclaude build --build-arg DCLAUDE_EXTENSIONS=myextension`
4. Verify: `./dist/dclaude shell -c "which mycommand"`

## Extension Metadata

When extensions are installed, metadata is written to `~/.dclaude/extensions.json`:

```json
{
  "extensions": {
    "beads": {
      "name": "beads",
      "description": "Git-backed issue tracker for AI agents",
      "entrypoint": "bd",
      "mounts": [
        {"source": "~/.beads", "target": "/home/claude/.beads"}
      ]
    },
    "gastown": {
      "name": "gastown",
      "description": "Multi-agent orchestration for Claude Code",
      "entrypoint": "gt",
      "mounts": [
        {"source": "~/.gastown", "target": "/home/claude/.gastown"}
      ]
    }
  }
}
```

This metadata is used at runtime to:
- Mount extension directories from the host
- Discover available extensions and their entrypoints

## Examples

### Gastown Extension

Gastown provides multi-agent orchestration for Claude Code:

```bash
# Build with gastown
dclaude build --build-arg DCLAUDE_EXTENSIONS=gastown

# Run gastown instead of claude
DCLAUDE_COMMAND=gt dclaude

# Or use shell mode
dclaude shell
gt --help
```

### Tessl Extension

Tessl is an agent enablement platform with a skills package manager:

```bash
# Build with tessl
dclaude build --build-arg DCLAUDE_EXTENSIONS=tessl

# Use tessl
dclaude shell
tessl init           # Authenticate
tessl skill search   # Find skills
tessl mcp            # Start MCP server
```

## Troubleshooting

### Permission Errors

If you see permission errors during installation:
- Use `sudo` for `apt-get` and global `npm install`
- Go packages don't need sudo (install to user's `~/go/bin`)

### Extension Not Found

If an extension is not recognized:
- Ensure the directory name matches the extension name in `config.yaml`
- Check that both `config.yaml` and `install.sh` exist
- Rebuild dclaude with `make build` to embed the new extension
