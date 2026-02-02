package cmd

import (
	"fmt"
	"os"

	"github.com/jedi4ever/dclaude/provider/docker"
)

// PrintHelp displays usage information
func PrintHelp(version string) {
	PrintHelpWithFlags(version, "", "")
}

// PrintHelpWithFlags displays usage information with extension-specific flags
func PrintHelpWithFlags(version, imageName, command string) {
	fmt.Printf(`dclaude - Run AI coding agents in containerized environments

Version: %s

Usage: dclaude [options] [prompt]

Commands:
  shell                              Open bash shell in environment
  containers build [--build-arg ...] Build the container image
  containers [list|stop|rm|clean]    Manage persistent environments
  firewall [list|add|remove|reset]   Manage network firewall domains
  --update                           Check for and install updates
  --rebuild                          Rebuild the environment (Docker only)
  --dversion                         Show dclaude version
  --dhelp                            Show this help

`, version)

	// Try to get extension-specific flags
	if imageName != "" && command != "" {
		printExtensionFlags(imageName, command)
	} else {
		// Fallback to generic options
		fmt.Println(`Options:
  All options are passed to the agent. Generic flags transformed by extensions:
  --yolo                      Bypass permission checks (transformed by extension's args.sh)
`)
	}

	fmt.Print(`Environment Variables:
  DCLAUDE_PROVIDER            Provider type: docker or daytona (default: docker)
  DCLAUDE_NODE_VERSION        Node.js version (default: 22)
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
  DCLAUDE_FIREWALL            Enable network firewall (default: false, requires --cap-add=NET_ADMIN)
  DCLAUDE_FIREWALL_MODE       Firewall mode: strict, permissive, off (default: strict)
  DCLAUDE_MODE                Execution mode: container or shell (default: container)
  DCLAUDE_EXTENSIONS          Extensions to install at build time (e.g., claude,codex,gemini)
  DCLAUDE_COMMAND             Command to run instead of claude (e.g., codex, gemini)

Per-Extension Configuration:
  DCLAUDE_<EXT>_VERSION       Version for extension (e.g., DCLAUDE_CLAUDE_VERSION=1.0.5)
                              Default versions defined in each extension's config.yaml
  DCLAUDE_<EXT>_MOUNT_CONFIG  Mount extension config dir (e.g., DCLAUDE_CLAUDE_MOUNT_CONFIG=false)

Build Command:
  dclaude containers build [--build-arg KEY=VALUE]...
                              Build the container image with optional build args
                              Example: dclaude containers build --build-arg DCLAUDE_EXTENSIONS=gastown

Examples:
  dclaude --dhelp
  dclaude "Fix the bug in app.js"
  dclaude --yolo "Refactor this entire codebase"
  dclaude --help              # Shows agent's help
  dclaude shell
  DCLAUDE_COMMAND=codex dclaude   # Run Codex instead of Claude
  DCLAUDE_COMMAND=gemini dclaude  # Run Gemini instead of Claude
`)
}

// printExtensionFlags queries and prints flags for the active extension
func printExtensionFlags(imageName, command string) {
	// Create a minimal docker provider to query extension flags
	p := &docker.DockerProvider{}
	flags := p.GetExtensionFlags(imageName, command)

	if len(flags) > 0 {
		fmt.Printf("Options (%s):\n", command)
		fmt.Println("  Generic flags (transformed by extension's args.sh):")
		fmt.Println("  --yolo                      Bypass permission checks")
		fmt.Println()
		fmt.Printf("  Extension-specific flags (%s):\n", command)
		for _, flag := range flags {
			// Format flag with padding
			fmt.Printf("  %-25s %s\n", flag.Flag, flag.Description)
		}
		fmt.Println()
	} else {
		fmt.Println(`Options:
  All options are passed to the agent. Generic flags transformed by extensions:
  --yolo                      Bypass permission checks (transformed by extension's args.sh)
`)
	}
}

// GetActiveCommand returns the active command from env or default
func GetActiveCommand() string {
	if cmd := os.Getenv("DCLAUDE_COMMAND"); cmd != "" {
		return cmd
	}
	return "claude"
}
