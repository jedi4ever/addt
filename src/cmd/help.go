package cmd

import "fmt"

// PrintHelp displays usage information
func PrintHelp(version string) {
	fmt.Printf(`dclaude - Run Claude Code in containerized environments

Version: %s

Usage: dclaude [options] [prompt]

Commands:
  shell                       Open bash shell in environment
  containers [list|stop|rm|clean]  Manage persistent environments
  firewall [list|add|remove|reset] Manage network firewall domains
  --update                    Check for and install updates
  --rebuild                   Rebuild the environment (Docker only)
  --dversion                  Show dclaude version
  --dhelp                     Show this help

Options:
  All options are passed to Claude Code. Additionally:
  --yolo                      Bypass all permission checks (alias for --dangerously-skip-permissions)

Environment Variables:
  DCLAUDE_PROVIDER            Provider type: docker or daytona (default: docker)
  DCLAUDE_CLAUDE_VERSION      Claude Code version (default: stable)
  DCLAUDE_NODE_VERSION        Node.js version (default: 20)
  DCLAUDE_GO_VERSION          Go version (default: latest)
  DCLAUDE_UV_VERSION          UV Python package manager version (default: latest)
  DCLAUDE_ENV_VARS            Comma-separated env vars to pass (default: ANTHROPIC_API_KEY,GH_TOKEN)
  DCLAUDE_GITHUB_DETECT       Auto-detect GitHub token from gh CLI (default: false)
  DCLAUDE_PORTS               Comma-separated container ports to expose
  DCLAUDE_PORT_RANGE_START    Starting port for auto allocation (default: 30000)
  DCLAUDE_SSH_FORWARD         SSH forwarding mode: agent, keys, or empty
  DCLAUDE_GPG_FORWARD         Enable GPG forwarding (true/false)
  DCLAUDE_DIND_MODE           Docker-in-Docker mode: host, isolated (default: none)
  DCLAUDE_ENV_FILE            Path to .env file (default: .env)
  DCLAUDE_LOG                 Enable command logging (default: false)
  DCLAUDE_LOG_FILE            Log file path
  DCLAUDE_PERSISTENT          Enable persistent container mode (true/false)
  DCLAUDE_MOUNT_WORKDIR       Mount working directory to /workspace (default: true)
  DCLAUDE_MOUNT_CLAUDE_CONFIG Mount ~/.claude and ~/.claude.json (default: true)
  DCLAUDE_FIREWALL            Enable network firewall (default: false, requires --cap-add=NET_ADMIN)
  DCLAUDE_FIREWALL_MODE       Firewall mode: strict, permissive, off (default: strict)
  DCLAUDE_MODE                Execution mode: container or shell (default: container)
  DCLAUDE_EXTENSIONS          Extensions to install at build time (e.g., gastown,tessl)
  DCLAUDE_COMMAND             Command to run instead of claude (e.g., gt for gastown)

Build Command:
  dclaude build [--build-arg KEY=VALUE]...
                              Build the container image with optional build args
                              Example: dclaude build --build-arg DCLAUDE_EXTENSIONS=gastown

Examples:
  dclaude --dhelp
  dclaude "Fix the bug in app.js"
  dclaude --model opus "Explain this codebase"
  dclaude --yolo "Refactor this entire codebase"
  dclaude --help              # Shows Claude Code's help
  dclaude shell
  DCLAUDE_COMMAND=gt dclaude  # Run gastown instead of claude
`, version)
}
