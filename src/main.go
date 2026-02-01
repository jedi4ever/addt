package main

import (
	"fmt"
	"os"
	"os/exec"
)

// Version can be overridden at build time with -ldflags "-X main.Version=x.y.z"
var Version = "1.0.0"

const (
	DefaultNodeVersion    = "20"
	DefaultPortRangeStart = 30000
)

func main() {
	// Setup cleanup on exit
	SetupCleanup()

	// Parse command line arguments
	args := os.Args[1:]

	// Check for special commands
	if len(args) > 0 {
		switch args[0] {
		case "--update":
			UpdateDClaude()
			return
		case "--version":
			fmt.Printf("dclaude version %s\n", Version)
			return
		case "--help", "-h":
			printHelp()
			return
		}
	}

	// Load configuration
	cfg := LoadConfig()

	// Check for --rebuild flag
	rebuildImage := false
	if len(args) > 0 && args[0] == "--rebuild" {
		rebuildImage = true
		args = args[1:]
	}

	// Check for "shell" command
	openShell := false
	if len(args) > 0 && args[0] == "shell" {
		openShell = true
		args = args[1:]
	}

	// Replace --yolo with --dangerously-skip-permissions
	for i, arg := range args {
		if arg == "--yolo" {
			args[i] = "--dangerously-skip-permissions"
		}
	}

	// Check prerequisites
	CheckPrerequisites()

	// Determine image name and Claude version
	cfg.ImageName = DetermineImageName(cfg)

	// Handle --rebuild flag
	if rebuildImage {
		fmt.Printf("Rebuilding %s...\n", cfg.ImageName)
		if ImageExists(cfg.ImageName) {
			fmt.Println("Removing existing image...")
			exec.Command("docker", "rmi", cfg.ImageName).Run()
		}
	}

	// Build image if needed
	if !ImageExists(cfg.ImageName) {
		BuildImage(cfg)
	}

	// Auto-detect GitHub token if enabled
	if cfg.GitHubDetect && os.Getenv("GH_TOKEN") == "" {
		if token := DetectGitHubToken(); token != "" {
			os.Setenv("GH_TOKEN", token)
		}
	}

	// Load env file if exists
	LoadEnvFile(cfg.EnvFile)

	// Build and run docker command
	RunDocker(cfg, args, openShell)
}

func printHelp() {
	fmt.Printf(`dclaude - Run Claude Code in Docker container

Version: %s

Usage: dclaude [options] [prompt]

Commands:
  shell              Open bash shell in container
  --update           Check for and install updates
  --rebuild          Rebuild the Docker image
  --version          Show version
  --help             Show this help

Options (passed to claude):
  --yolo             Bypass all permission checks (alias for --dangerously-skip-permissions)
  --model <model>    Specify model to use

Environment Variables:
  DCLAUDE_CLAUDE_VERSION    Claude Code version (default: latest)
  DCLAUDE_NODE_VERSION      Node.js version (default: 20)
  DCLAUDE_ENV_VARS          Comma-separated env vars to pass (default: ANTHROPIC_API_KEY,GH_TOKEN)
  DCLAUDE_GITHUB_DETECT     Auto-detect GitHub token from gh CLI (default: false)
  DCLAUDE_PORTS             Comma-separated container ports to expose
  DCLAUDE_PORT_RANGE_START  Starting port for auto allocation (default: 30000)
  DCLAUDE_SSH_FORWARD       SSH forwarding mode: agent, keys, or empty
  DCLAUDE_GPG_FORWARD       Enable GPG forwarding (true/false)
  DCLAUDE_DOCKER_FORWARD    Docker mode: host, isolated, or empty
  DCLAUDE_ENV_FILE          Path to .env file (default: .env)
  DCLAUDE_LOG               Enable command logging (default: false)
  DCLAUDE_LOG_FILE          Log file path

Examples:
  dclaude --help
  dclaude "Fix the bug in app.js"
  dclaude --model opus "Explain this codebase"
  dclaude --yolo "Refactor this entire codebase"
  dclaude shell
`, Version)
}
