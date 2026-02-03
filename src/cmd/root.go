package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jedi4ever/addt/config"
	"github.com/jedi4ever/addt/core"
	"github.com/jedi4ever/addt/internal/update"
	"github.com/jedi4ever/addt/provider"
)

// handleSubcommand handles addt subcommands (build, shell, containers, firewall)
func handleSubcommand(subCmd string, subArgs []string, defaultNodeVersion, defaultGoVersion, defaultUvVersion string, defaultPortRangeStart int) {
	cfg := config.LoadConfig(defaultNodeVersion, defaultGoVersion, defaultUvVersion, defaultPortRangeStart)

	switch subCmd {
	case "build":
		providerCfg := &provider.Config{
			ExtensionVersions: cfg.ExtensionVersions,
			NodeVersion:       cfg.NodeVersion,
			GoVersion:         cfg.GoVersion,
			UvVersion:         cfg.UvVersion,
			Provider:          cfg.Provider,
			Extensions:        cfg.Extensions,
		}
		prov, err := NewProvider(cfg.Provider, providerCfg)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		HandleBuildCommand(prov, providerCfg, subArgs)

	case "shell":
		providerCfg := &provider.Config{
			ExtensionVersions:  cfg.ExtensionVersions,
			ExtensionAutomount: cfg.ExtensionAutomount,
			NodeVersion:        cfg.NodeVersion,
			GoVersion:          cfg.GoVersion,
			UvVersion:          cfg.UvVersion,
			EnvVars:            cfg.EnvVars,
			GitHubDetect:       cfg.GitHubDetect,
			Ports:              cfg.Ports,
			PortRangeStart:     cfg.PortRangeStart,
			SSHForward:         cfg.SSHForward,
			GPGForward:         cfg.GPGForward,
			DindMode:           cfg.DindMode,
			EnvFile:            cfg.EnvFile,
			LogEnabled:         cfg.LogEnabled,
			LogFile:            cfg.LogFile,
			ImageName:          cfg.ImageName,
			Persistent:         cfg.Persistent,
			WorkdirAutomount:   cfg.WorkdirAutomount,
			Workdir:            cfg.Workdir,
			FirewallEnabled:    cfg.FirewallEnabled,
			FirewallMode:       cfg.FirewallMode,
			Mode:               cfg.Mode,
			Provider:           cfg.Provider,
			Extensions:         cfg.Extensions,
			Command:            cfg.Command,
		}
		prov, err := NewProvider(cfg.Provider, providerCfg)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		if err := prov.Initialize(providerCfg); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		providerCfg.ImageName = prov.DetermineImageName()
		if err := prov.BuildIfNeeded(false, false); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		orch := core.NewOrchestrator(prov, providerCfg)
		if err := orch.RunClaude(subArgs, true); err != nil {
			os.Exit(1)
		}
		prov.Cleanup()

	case "containers":
		providerCfg := &provider.Config{
			ExtensionVersions: cfg.ExtensionVersions,
			NodeVersion:       cfg.NodeVersion,
			GoVersion:         cfg.GoVersion,
			UvVersion:         cfg.UvVersion,
			Provider:          cfg.Provider,
			Extensions:        cfg.Extensions,
		}
		prov, err := NewProvider(cfg.Provider, providerCfg)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		HandleContainersCommand(prov, providerCfg, subArgs)

	case "firewall":
		HandleFirewallCommand(subArgs)

	default:
		fmt.Printf("Unknown command: %s\n", subCmd)
		os.Exit(1)
	}
}

// Execute is the main entry point for the CLI
func Execute(version, defaultNodeVersion, defaultGoVersion, defaultUvVersion string, defaultPortRangeStart int) {
	// Detect binary name for symlink-based extension selection
	// Supports: "claude", "codex", "addt-claude", "addt-codex", etc.
	binaryName := filepath.Base(os.Args[0])
	binaryName = strings.TrimSuffix(binaryName, filepath.Ext(binaryName)) // Remove .exe on Windows

	// Extract extension name from binary name
	// "addt-claude" -> "claude", "claude" -> "claude", "addt" -> ""
	extensionFromBinary := ""
	if strings.HasPrefix(binaryName, "addt-") {
		extensionFromBinary = strings.TrimPrefix(binaryName, "addt-")
	} else if binaryName != "addt" && binaryName != "" {
		extensionFromBinary = binaryName
	}

	if extensionFromBinary != "" {
		// Set extension and command based on binary name if not already set
		if os.Getenv("ADDT_EXTENSIONS") == "" {
			os.Setenv("ADDT_EXTENSIONS", extensionFromBinary)
		}
		if os.Getenv("ADDT_COMMAND") == "" {
			os.Setenv("ADDT_COMMAND", extensionFromBinary)
		}
	} else if os.Getenv("ADDT_EXTENSIONS") != "" && os.Getenv("ADDT_COMMAND") == "" {
		// If ADDT_EXTENSIONS is set but ADDT_COMMAND is not, look up the entrypoint
		extensions := os.Getenv("ADDT_EXTENSIONS")
		firstExt := strings.Split(extensions, ",")[0]
		// Get the actual entrypoint command (e.g., "kiro" -> "kiro-cli", "beads" -> "bd")
		entrypoint := GetEntrypointForExtension(firstExt)
		os.Setenv("ADDT_COMMAND", entrypoint)
	}

	// Parse command line arguments
	args := os.Args[1:]

	// If running as plain "addt" without extension, check if it's a known command
	// Otherwise show help - don't default to claude
	if extensionFromBinary == "" && os.Getenv("ADDT_EXTENSIONS") == "" {
		if len(args) == 0 {
			PrintHelp(version)
			return
		}
		// Check if first arg is a known addt command (matches switch cases below)
		switch args[0] {
		case "run", "build", "shell", "containers", "firewall",
			"--addt-version", "--addt-update", "--addt-list-extensions", "--addt-help":
			// Known command, continue processing
		default:
			// Unknown command, show help
			PrintHelp(version)
			return
		}
	}

	// Check for special commands
	if len(args) > 0 {
		switch args[0] {
		case "--addt-update":
			update.UpdateAddt(version)
			return
		case "--addt-version":
			PrintVersion(version, defaultNodeVersion, defaultGoVersion, defaultUvVersion)
			return
		case "--addt-list-extensions":
			ListExtensions()
			return
		case "--addt-help":
			// Try to show help with extension-specific flags
			cfg := config.LoadConfig(defaultNodeVersion, defaultGoVersion, defaultUvVersion, defaultPortRangeStart)
			providerCfg := &provider.Config{
				ExtensionVersions: cfg.ExtensionVersions,
				NodeVersion:       cfg.NodeVersion,
				Provider:          cfg.Provider,
				Extensions:        cfg.Extensions,
			}
			prov, err := NewProvider(cfg.Provider, providerCfg)
			if err == nil {
				prov.Initialize(providerCfg)
				imageName := prov.DetermineImageName()
				command := GetActiveCommand()
				PrintHelpWithFlags(version, imageName, command)
			} else {
				PrintHelp(version)
			}
			return
		case "run":
			// addt run <extension> [args...] - run a specific extension
			if len(args) < 2 {
				fmt.Println("Usage: addt run <extension> [args...]")
				fmt.Println()
				fmt.Println("Examples:")
				fmt.Println("  addt run claude \"Fix the bug\"")
				fmt.Println("  addt run codex --help")
				fmt.Println("  addt run gemini")
				fmt.Println()
				fmt.Println("Available extensions: addt --addt-list-extensions")
				return
			}
			// Set the extension and continue with normal execution
			extName := args[1]
			os.Setenv("ADDT_EXTENSIONS", extName)
			os.Setenv("ADDT_COMMAND", extName)
			args = args[2:] // Remove "run" and extension name, keep remaining args

		case "build", "shell", "containers", "firewall":
			// Top-level subcommands (work for both plain addt and via "addt" namespace)
			subCmd := args[0]
			subArgs := args[1:]
			handleSubcommand(subCmd, subArgs, defaultNodeVersion, defaultGoVersion, defaultUvVersion, defaultPortRangeStart)
			return

		case "addt":
			// addt subcommand namespace for container management (e.g., claude addt build)
			if len(args) < 2 {
				fmt.Println("Usage: <agent> addt <command>")
				fmt.Println()
				fmt.Println("Commands:")
				fmt.Println("  build [--build-arg ...]   Build the container image")
				fmt.Println("  shell                     Open bash shell in container")
				fmt.Println("  containers <subcommand>   Manage containers (list, stop, rm, clean)")
				fmt.Println("  firewall <subcommand>     Manage firewall (list, add, remove, reset)")
				return
			}
			handleSubcommand(args[1], args[2:], defaultNodeVersion, defaultGoVersion, defaultUvVersion, defaultPortRangeStart)
			return
		}
	}

	// Load configuration
	cfg := config.LoadConfig(defaultNodeVersion, defaultGoVersion, defaultUvVersion, defaultPortRangeStart)

	// Check for --addt-rebuild and --addt-rebuild-base flags
	rebuildImage := false
	rebuildBase := false
	for len(args) > 0 {
		if args[0] == "--addt-rebuild" {
			rebuildImage = true
			args = args[1:]
		} else if args[0] == "--addt-rebuild-base" {
			rebuildBase = true
			rebuildImage = true // Rebuilding base also requires extension rebuild
			args = args[1:]
		} else {
			break
		}
	}

	// Note: --yolo and other agent-specific arg transformations are handled
	// by each extension's args.sh script in the container

	// Convert main config to provider config
	providerCfg := &provider.Config{
		ExtensionVersions:  cfg.ExtensionVersions,
		ExtensionAutomount: cfg.ExtensionAutomount,
		NodeVersion:        cfg.NodeVersion,
		GoVersion:          cfg.GoVersion,
		UvVersion:          cfg.UvVersion,
		EnvVars:            cfg.EnvVars,
		GitHubDetect:       cfg.GitHubDetect,
		Ports:              cfg.Ports,
		PortRangeStart:     cfg.PortRangeStart,
		SSHForward:         cfg.SSHForward,
		GPGForward:         cfg.GPGForward,
		DindMode:           cfg.DindMode,
		EnvFile:            cfg.EnvFile,
		LogEnabled:         cfg.LogEnabled,
		LogFile:            cfg.LogFile,
		ImageName:          cfg.ImageName,
		Persistent:         cfg.Persistent,
		WorkdirAutomount:   cfg.WorkdirAutomount,
		Workdir:            cfg.Workdir,
		FirewallEnabled:    cfg.FirewallEnabled,
		FirewallMode:       cfg.FirewallMode,
		Mode:               cfg.Mode,
		Provider:           cfg.Provider,
		Extensions:         cfg.Extensions,
		Command:            cfg.Command,
	}

	// Create provider
	prov, err := NewProvider(cfg.Provider, providerCfg)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// Initialize provider (checks prerequisites)
	if err := prov.Initialize(providerCfg); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// Determine image name and build if needed (provider-specific)
	providerCfg.ImageName = prov.DetermineImageName()
	if err := prov.BuildIfNeeded(rebuildImage, rebuildBase); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// Create orchestrator
	orch := core.NewOrchestrator(prov, providerCfg)

	// Auto-detect GitHub token if enabled
	if cfg.GitHubDetect && os.Getenv("GH_TOKEN") == "" {
		if token := config.DetectGitHubToken(); token != "" {
			os.Setenv("GH_TOKEN", token)
		}
	}

	// Load env file if exists
	if err := config.LoadEnvFile(cfg.EnvFile); err != nil {
		fmt.Printf("Error loading env file: %v\n", err)
		os.Exit(1)
	}

	// Run via orchestrator
	if err := orch.RunClaude(args, false); err != nil {
		os.Exit(1)
	}

	// Cleanup
	prov.Cleanup()
}
