package cmd

import (
	"fmt"
	"os"

	"github.com/jedi4ever/addt/provider/docker"
)

// PrintHelp displays usage information for plain addt (no extension)
func PrintHelp(version string) {
	fmt.Printf(`addt - Run AI coding agents in containerized environments

Version: %s

Commands:
  addt run <extension> [args...]     Run a specific extension
  addt build <extension>             Build the container image
  addt shell <extension>             Open bash shell in container
  addt containers [list|stop|rm]     Manage containers
  addt firewall [list|add|rm|reset]  Manage firewall
  addt extensions [list|info]        Manage extensions
  addt cli [update]                  Manage addt CLI
  addt version                       Show version info

Examples:
  addt run claude "Fix the bug"
  addt extensions list
  addt extensions info claude
  addt cli update
`, version)
}

// PrintHelpWithFlags displays usage information with extension-specific flags
func PrintHelpWithFlags(version, imageName, command string) {
	fmt.Printf(`addt - Run AI coding agents in containerized environments

Version: %s

Usage: <agent> [options] [prompt]

Container management (via agent):
  <agent> addt build                         Build the container image
  <agent> addt shell                         Open bash shell in container
  <agent> addt containers [list|stop|rm]     Manage persistent containers
  <agent> addt firewall [list|add|rm|reset]  Manage network firewall
  <agent> addt extensions [list|info]        Manage extensions
  <agent> addt cli [update]                  Manage addt CLI
  <agent> addt version                       Show version info

`, version)

	// Try to get extension-specific flags
	if imageName != "" && command != "" {
		printExtensionFlags(imageName, command)
	} else {
		// Fallback to generic options
		fmt.Println(`Options:
  All options are passed to the agent. Generic flags transformed by extensions:
  --yolo                      Bypass permission checks (transformed by extension's args.sh)`)
	}

	fmt.Print(`
Environment Variables:
  ADDT_PROVIDER            Provider type: docker or daytona (default: docker)
  ADDT_NODE_VERSION        Node.js version (default: 22)
  ADDT_GO_VERSION          Go version (default: latest)
  ADDT_UV_VERSION          UV Python package manager version (default: latest)
  ADDT_ENV_VARS            Comma-separated env vars to pass (default: ANTHROPIC_API_KEY,GH_TOKEN)
  ADDT_GITHUB_DETECT       Auto-detect GitHub token from gh CLI (default: false)
  ADDT_PORTS               Comma-separated container ports to expose
  ADDT_PORT_RANGE_START    Starting port for auto allocation (default: 30000)
  ADDT_SSH_FORWARD         SSH forwarding mode: agent, keys, or empty
  ADDT_GPG_FORWARD         Enable GPG forwarding (true/false)
  ADDT_DIND_MODE           Docker-in-Docker mode: host, isolated (default: none)
  ADDT_ENV_FILE            Path to .env file (default: .env)
  ADDT_LOG                 Enable command logging (default: false)
  ADDT_LOG_FILE            Log file path
  ADDT_PERSISTENT          Enable persistent container mode (true/false)
  ADDT_WORKDIR             Override working directory (default: current directory)
  ADDT_WORKDIR_AUTOMOUNT   Auto-mount working directory to /workspace (default: true)
  ADDT_FIREWALL            Enable network firewall (default: false, requires --cap-add=NET_ADMIN)
  ADDT_FIREWALL_MODE       Firewall mode: strict, permissive, off (default: strict)
  ADDT_MODE                Execution mode: container or shell (default: container)
  ADDT_EXTENSIONS          Extensions to install at build time (e.g., claude,codex,gemini)
  ADDT_COMMAND             Command to run instead of claude (e.g., codex, gemini)
  ADDT_CPUS                CPU limit for container (e.g., "2", "0.5", "1.5")
  ADDT_MEMORY              Memory limit for container (e.g., "512m", "2g", "4gb")

Per-Extension Configuration:
  ADDT_<EXT>_VERSION       Version for extension (e.g., ADDT_CLAUDE_VERSION=1.0.5)
                              Default versions defined in each extension's config.yaml
  ADDT_<EXT>_AUTOMOUNT     Auto-mount extension config (e.g., ADDT_CLAUDE_AUTOMOUNT=false)

Build Command:
  addt containers build [--build-arg KEY=VALUE]...
                              Build the container image with optional build args
                              Example: addt containers build --build-arg ADDT_EXTENSIONS=gastown

Examples:
  claude "Fix the bug in app.js"
  claude --yolo "Refactor this entire codebase"
  claude --help                    # Shows agent's help
  claude --addt-help               # Shows addt help
  claude --addt-list-extensions    # List available extensions
  claude addt build                # Build container image
  claude addt shell                # Open shell in container

Multiple agents via symlinks (avoids overriding real installs):
  mkdir -p ~/bin && ln -s /usr/local/bin/addt ~/bin/claude
  ln -s /usr/local/bin/addt ~/bin/codex
  ln -s /usr/local/bin/addt ~/bin/addt-claude   # Also supports addt-<extension> naming
`)
}

// printExtensionFlags queries and prints flags for the active extension
func printExtensionFlags(imageName, command string) {
	// Create a minimal docker provider to query extension flags
	p := &docker.DockerProvider{}
	flags := p.GetExtensionFlags(imageName, command)

	if len(flags) > 0 {
		fmt.Printf("Options (%s):\n", command)
		for _, flag := range flags {
			fmt.Printf("  %-25s %s\n", flag.Flag, flag.Description)
		}
		fmt.Println()
	} else {
		fmt.Println(`Options:
  All options are passed to the agent. Generic flags transformed by extensions:
  --yolo                      Bypass permission checks (transformed by extension's args.sh)`)
	}
}

// GetActiveCommand returns the active command from env or default
func GetActiveCommand() string {
	if cmd := os.Getenv("ADDT_COMMAND"); cmd != "" {
		return cmd
	}
	return "claude"
}
