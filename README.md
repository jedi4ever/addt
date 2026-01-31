# Docker + Makefile Project for Claude Code

Run Claude Code in Docker containers with easy build and run orchestration via Makefile.

## Overview

This project provides a complete Docker-based setup for running Claude Code in an isolated container environment. It includes:

- **Dockerfile**: Container definition with Claude Code, Git, and GitHub CLI pre-installed
- **Makefile**: Simple build and run orchestration
- **dclaude.sh**: Convenient wrapper script (recommended - auto-loads .env)
- **run.sh**: Alternative script for running Claude Code with arguments
- **Volume mounting**: Access to your local files from within the container
- **Environment-based auth**: Secure API key handling via .env file
- **GitHub CLI**: Full gh CLI support for GitHub operations
- **Git Identity**: Automatic git configuration from your local machine

## Prerequisites

- Docker installed and running
- ANTHROPIC_API_KEY environment variable
- GH_TOKEN environment variable (optional, for GitHub CLI authentication)

## Quick Start

1. **Set your API keys in .env file:**
   ```bash
   # Create or edit .env file
   echo "ANTHROPIC_API_KEY=your-anthropic-api-key" >> .env
   echo "GH_TOKEN=your-github-token" >> .env  # Optional, for GitHub operations
   ```

   Or export them in your shell:
   ```bash
   export ANTHROPIC_API_KEY='your-anthropic-api-key'
   export GH_TOKEN='your-github-token'  # Optional
   ```

2. **Build the Docker image:**
   ```bash
   make build
   ```

3. **Run Claude Code:**
   ```bash
   # Recommended: Use dclaude.sh (auto-loads .env)
   ./dclaude.sh

   # Or use make (requires exported variables)
   make run
   ```

## Usage

### Using dclaude.sh (Recommended)

The `dclaude.sh` script is the easiest way to run Claude Code. It automatically loads your `.env` file and passes all arguments to Claude:

```bash
# Interactive mode (default)
./dclaude.sh

# Display help
./dclaude.sh --help

# Check version
./dclaude.sh --version

# Run with a specific prompt
./dclaude.sh "Fix the bug in app.js"

# Use different model
./dclaude.sh --model opus "Explain this codebase"

# Continue previous conversation
./dclaude.sh --continue

# Non-interactive mode (for scripts/automation)
./dclaude.sh --print "List all files"

# Non-interactive with file write permissions
./dclaude.sh --print --permission-mode acceptEdits "Create a config.json file"

# Open a bash shell in the container
./dclaude.sh shell
```

**Special Commands:**
- `./dclaude.sh shell` - Opens a bash shell in the container for debugging and manual operations

**Permission Modes for Non-Interactive Use:**
When using `--print` mode for scripting/automation, Claude Code can't ask for permissions interactively. Use these flags:
- `--permission-mode acceptEdits` - Automatically accept file edits (recommended)
- `--permission-mode dontAsk` - Don't ask for permissions
- `--dangerously-skip-permissions` - Skip all permission checks (works with non-root user)
- For interactive use, permissions are prompted normally

**Benefits:**
- Automatically loads `.env` file (no need to export variables)
- Mounts current directory and `.gitconfig`
- Passes through all Claude Code arguments
- Built-in shell access for debugging
- Simple and convenient

### Using Makefile

The Makefile provides convenient targets for common operations:

```bash
# Display help
make help

# Build the Docker image
make build

# Run Claude Code interactively
make run

# Open a shell in the container for debugging
make shell

# Remove the Docker image
make clean
```

### Using run.sh Script

The `run.sh` script provides more flexibility for passing arguments:

```bash
# Make the script executable (first time only)
chmod +x run.sh

# Run interactively (default)
./run.sh

# Display Claude Code help
./run.sh --help

# Run with a specific prompt
./run.sh "Fix the bug in app.js"

# Run with additional options
./run.sh --model opus "Refactor the authentication module"
```

### Direct Docker Commands

You can also run Docker directly:

```bash
# Run interactively
docker run -it --rm \
  -v $(pwd):/workspace \
  -e ANTHROPIC_API_KEY=$ANTHROPIC_API_KEY \
  -e GH_TOKEN=$GH_TOKEN \
  dclaude:latest

# Run with specific command
docker run -it --rm \
  -v $(pwd):/workspace \
  -e ANTHROPIC_API_KEY=$ANTHROPIC_API_KEY \
  -e GH_TOKEN=$GH_TOKEN \
  dclaude:latest "List all files in the project"
```

## Architecture

### Base Image
- Uses `node:20-slim` for lightweight and reliable Node.js environment
- Debian-based for easy package installation

### Non-Root User Setup
- Container runs as your local user (not root) for security
- Build-time arguments automatically set UID/GID to match your local user
- Files created in container have correct ownership on host
- Enables use of `--dangerously-skip-permissions` flag if needed
- Git config and file permissions work seamlessly

### Installed Tools
- **Claude Code**: Global npm installation of `@anthropic-ai/claude-code`
- **Git**: Version control system for repository operations
- **GitHub CLI (gh)**: Official GitHub CLI for PR management, issues, and more
- **Ripgrep (rg)**: Fast search tool for code exploration and file searching

### Volume Mounting
- Current directory mounted to `/workspace` in container
- Local `~/.gitconfig` mounted to `/root/.gitconfig` (read-only)
  - Preserves your git identity (name and email) in commits
  - All git aliases and configurations available in container
- Claude Code can read and write files in your project
- Changes persist to your local filesystem

### Authentication & Identity
- **ANTHROPIC_API_KEY**: Required for Claude Code API access
- **GH_TOKEN**: Optional, enables GitHub CLI authentication for private repos and API operations
- **Git Identity**: Automatically uses your local git configuration
  - Commits made in the container will use your name and email
  - No need to configure git identity separately
- Keys and config passed securely (not stored in the image)

## File Structure

```
/Users/patrickdebois/dev/dclaude/
├── Dockerfile          # Container definition with Claude Code, Git, and gh CLI
├── Makefile           # Build and run orchestration
├── dclaude.sh         # Recommended wrapper script (auto-loads .env)
├── run.sh             # Alternative script for running Claude Code
├── .dockerignore      # Exclude unnecessary files from build context
├── .env               # Environment variables (ANTHROPIC_API_KEY, GH_TOKEN)
└── README.md          # This file
```

## Configuration

### Environment Variables

- **ANTHROPIC_API_KEY** (required): Your Anthropic API key for authentication
- **GH_TOKEN** (optional): GitHub personal access token for gh CLI authentication
  - Required for private repository access
  - Required for creating PRs, issues, and other write operations
  - Get yours at: https://github.com/settings/tokens

### GitHub CLI Integration

The container includes the official GitHub CLI (`gh`) for seamless GitHub operations. Claude Code can use it to:

- Create and manage pull requests
- View and create issues
- Check CI/CD status
- Clone and manage repositories
- And more

To enable GitHub CLI authentication, set the `GH_TOKEN` environment variable:

```bash
export GH_TOKEN='ghp_your_github_personal_access_token'
```

Inside the container, you can use gh commands:

```bash
# Example: Check gh CLI status
docker run --rm -e GH_TOKEN=$GH_TOKEN --entrypoint gh dclaude:latest auth status

# Example: List PRs in current repo
docker run --rm -v $(pwd):/workspace -e GH_TOKEN=$GH_TOKEN --entrypoint gh dclaude:latest pr list
```

### Volume Mounts

By default, the current directory is mounted to `/workspace`. You can add additional mounts:

```bash
# Example: Mount Claude session directory for persistence
docker run -it --rm \
  -v $(pwd):/workspace \
  -v ~/.claude:/root/.claude \
  -e ANTHROPIC_API_KEY=$ANTHROPIC_API_KEY \
  -e GH_TOKEN=$GH_TOKEN \
  dclaude:latest
```

### Image Customization

Edit the `Dockerfile` to customize the image:

```dockerfile
# Add additional tools (git and gh are already included)
RUN apt-get update && apt-get install -y vim

# Install additional npm packages
RUN npm install -g typescript
```

Then rebuild:
```bash
make build
```

## Examples

### Example 1: Quick Start with dclaude.sh
```bash
# Build the image (one time)
make build

# Run interactively (automatically loads .env)
./dclaude.sh
```

### Example 2: One-off Command
```bash
# Analyze code
./dclaude.sh "Analyze the Dockerfile and suggest improvements"

# Check for bugs
./dclaude.sh --print "Review app.js for potential bugs"
```

### Example 3: Creating Files (Non-Interactive)
```bash
# Create a file in non-interactive mode
./dclaude.sh --print --permission-mode acceptEdits "Create a hello.json file with a greeting message"

# Generate configuration files
./dclaude.sh --print --permission-mode acceptEdits "Create a package.json for a Node.js project"
```

### Example 4: Using Different Models
```bash
# Use Opus for complex tasks
./dclaude.sh --model opus "Design a new authentication system"

# Use Haiku for quick tasks
./dclaude.sh --model haiku "Fix the typo in README.md"
```

### Example 4: Shell Access for Debugging
```bash
# Open a shell using dclaude.sh
./dclaude.sh shell

# Or use make
make shell

# Now you're in a bash shell inside the container
# You can explore the environment, test commands, etc.
git config --global user.name  # Check git identity
gh --version                    # Check GitHub CLI
claude --version                # Check Claude Code
ls -la /workspace              # View mounted files
```

### Example 4: Using GitHub CLI
```bash
# Set both API keys
export ANTHROPIC_API_KEY='your-anthropic-key'
export GH_TOKEN='your-github-token'

# Run Claude and ask it to create a PR
./run.sh "Create a pull request for the current branch"

# Or use gh CLI directly
docker run --rm -v $(pwd):/workspace -e GH_TOKEN=$GH_TOKEN --entrypoint gh dclaude:latest pr list
```

### Example 5: Verify Git Identity
```bash
# Check that your git identity is correctly configured in the container
docker run --rm \
  -v $HOME/.gitconfig:/root/.gitconfig:ro \
  --entrypoint git dclaude:latest config --global user.name

docker run --rm \
  -v $HOME/.gitconfig:/root/.gitconfig:ro \
  --entrypoint git dclaude:latest config --global user.email
```

### Example 6: Custom Volume Mounts
```bash
docker run -it --rm \
  -v $(pwd):/workspace \
  -v ~/my-project:/external \
  -e ANTHROPIC_API_KEY=$ANTHROPIC_API_KEY \
  -e GH_TOKEN=$GH_TOKEN \
  dclaude:latest
```

## Troubleshooting

### API Key Not Set
```
Error: ANTHROPIC_API_KEY environment variable is not set
```
**Solution**: Export your API key:
```bash
export ANTHROPIC_API_KEY='your-key'
```

### Image Not Found
```
Error: Docker image 'dclaude:latest' not found
```
**Solution**: Build the image first:
```bash
make build
```

### Permission Issues
If you encounter permission errors with files:
```bash
# Check file ownership
ls -la

# If needed, run with user mapping
docker run -it --rm \
  -v $(pwd):/workspace \
  -u $(id -u):$(id -g) \
  -e ANTHROPIC_API_KEY=$ANTHROPIC_API_KEY \
  dclaude:latest
```

### Git Identity Not Set
If commits in the container don't have your identity:
```bash
# Check if .gitconfig is mounted correctly
docker run --rm \
  -v $HOME/.gitconfig:/root/.gitconfig:ro \
  --entrypoint ls dclaude:latest -la /root/.gitconfig

# Verify git configuration
make shell
git config --global user.name
git config --global user.email
```

The Makefile and run.sh automatically mount your `~/.gitconfig`, so commits will use your local git identity.

### Debugging Container Issues
```bash
# Open a shell to inspect the container
make shell

# Check if Claude Code is installed
which claude
claude --version

# Test Claude Code manually
claude --help
```

## Advanced Usage

### Building with Different Node Version
Edit `Dockerfile` and change the base image:
```dockerfile
FROM node:18-slim  # or node:22-slim
```

### Using Alpine for Smaller Image
```dockerfile
FROM node:20-alpine
```

Note: Alpine may have compatibility issues with some native modules.

### Persisting Claude Sessions
Mount the Claude config directory:
```bash
docker run -it --rm \
  -v $(pwd):/workspace \
  -v ~/.claude:/root/.claude \
  -e ANTHROPIC_API_KEY=$ANTHROPIC_API_KEY \
  dclaude:latest
```

## Contributing

Feel free to submit issues or pull requests to improve this setup.

## License

This project is provided as-is for use with Claude Code.
