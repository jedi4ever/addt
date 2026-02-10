package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	configcmd "github.com/jedi4ever/addt/cmd/config"
	extcmd "github.com/jedi4ever/addt/cmd/extensions"
	firewallcmd "github.com/jedi4ever/addt/cmd/firewall"
	profilecmd "github.com/jedi4ever/addt/cmd/profile"
	"github.com/jedi4ever/addt/config"
	"github.com/jedi4ever/addt/core"
	"github.com/jedi4ever/addt/provider"
	"github.com/jedi4ever/addt/util"
)

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
		entrypoint := extcmd.GetEntrypoint(firstExt)
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
		case "run", "build", "update", "shell", "containers", "firewall",
			"extensions", "cli", "config", "profile", "version", "completion", "doctor", "init":
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
		case "version":
			PrintVersion(version, defaultNodeVersion, defaultGoVersion, defaultUvVersion)
			return
		case "completion":
			HandleCompletionCommand(args[1:])
			return
		case "doctor":
			HandleDoctorCommand(args[1:])
			return
		case "init":
			HandleInitCommand(args[1:])
			return
		case "cli":
			handleCliCommand(args[1:], version)
			return
		case "config":
			configcmd.HandleCommand(args[1:])
			return
		case "profile":
			profilecmd.HandleCommand(args[1:])
			return
		case "extensions":
			extcmd.HandleCommand(args[1:])
			return
		case "run":
			// addt run <extension> [args...] - run a specific extension
			remainingArgs := HandleRunCommand(args[1:])
			if remainingArgs == nil {
				return // Help was printed or error occurred
			}
			args = remainingArgs

		case "update":
			HandleUpdateCommand(args[1:], version, defaultNodeVersion, defaultGoVersion, defaultUvVersion, defaultPortRangeStart)
			return

		case "build", "shell", "containers", "firewall":
			// Top-level subcommands (work for both plain addt and via "addt" namespace)
			subCmd := args[0]
			subArgs := args[1:]
			handleSubcommand(subCmd, subArgs, version, defaultNodeVersion, defaultGoVersion, defaultUvVersion, defaultPortRangeStart)
			return

		case "addt":
			// addt subcommand namespace for container management (e.g., claude addt build)
			if len(args) < 2 {
				printAddtSubcommandUsage()
				return
			}
			// Handle addt subcommands
			subCmd := args[1]
			subArgs := args[2:]
			switch subCmd {
			case "extensions":
				extcmd.HandleCommandAgent(subArgs)
			case "update":
				HandleUpdateCommand(subArgs, version, defaultNodeVersion, defaultGoVersion, defaultUvVersion, defaultPortRangeStart)
			case "cli":
				handleCliCommand(subArgs, version)
			case "config":
				configcmd.HandleCommand(subArgs)
			case "profile":
				profilecmd.HandleCommand(subArgs)
			case "version":
				PrintVersion(version, defaultNodeVersion, defaultGoVersion, defaultUvVersion)
			default:
				handleSubcommand(subCmd, subArgs, version, defaultNodeVersion, defaultGoVersion, defaultUvVersion, defaultPortRangeStart)
			}
			return
		}
	}

	// Load configuration
	cfg := config.LoadConfig(version, defaultNodeVersion, defaultGoVersion, defaultUvVersion, defaultPortRangeStart)

	// Resolve ~ in LogDir
	cfg.LogDir = util.ExpandTilde(cfg.LogDir)

	// Initialize logger
	util.InitLoggerFull(cfg.LogFile, cfg.LogDir, cfg.LogOutput, cfg.LogEnabled, cfg.LogLevel, cfg.LogModules, cfg.LogRotate, cfg.LogMaxSize, cfg.LogMaxFiles)
	logger := util.Log("root")
	logger.Debugf("Initializing logger with file: %s, dir: %s, enabled: %v, level: %s, modules: %s", cfg.LogFile, cfg.LogDir, cfg.LogEnabled, cfg.LogLevel, cfg.LogModules)
	// Note: --yolo and other agent-specific arg transformations are handled
	// by each extension's args.sh script in the container

	// Convert main config to provider config
	providerCfg := &provider.Config{
		AddtVersion:               cfg.AddtVersion,
		ExtensionVersions:         cfg.ExtensionVersions,
		ExtensionConfigAutomount:  cfg.ExtensionConfigAutomount,
		ExtensionConfigReadonly:   cfg.ExtensionConfigReadonly,
		ExtensionWorkdirAutotrust: cfg.ExtensionWorkdirAutotrust,
		ConfigAutomount:           cfg.ConfigAutomount,
		ConfigReadonly:            cfg.ConfigReadonly,
		AuthAutologin:             cfg.AuthAutologin,
		AuthMethod:                cfg.AuthMethod,
		ExtensionAuthAutologin:    cfg.ExtensionAuthAutologin,
		ExtensionAuthMethod:       cfg.ExtensionAuthMethod,
		ExtensionFlagSettings:     cfg.ExtensionFlagSettings,
		NodeVersion:               cfg.NodeVersion,
		GoVersion:                 cfg.GoVersion,
		UvVersion:                 cfg.UvVersion,
		EnvVars:                   cfg.EnvVars,
		GitHubForwardToken:        cfg.GitHubForwardToken,
		GitHubTokenSource:         cfg.GitHubTokenSource,
		GitHubScopeToken:          cfg.GitHubScopeToken,
		GitHubScopeRepos:          cfg.GitHubScopeRepos,
		Ports:                     cfg.Ports,
		PortRangeStart:            cfg.PortRangeStart,
		PortsInjectSystemPrompt:   cfg.PortsInjectSystemPrompt,
		SSHForwardKeys:            cfg.SSHForwardKeys,
		SSHForwardMode:            cfg.SSHForwardMode,
		SSHAllowedKeys:            cfg.SSHAllowedKeys,
		SSHDir:                    cfg.SSHDir,
		GitDisableHooks:           cfg.GitDisableHooks,
		GitForwardConfig:          cfg.GitForwardConfig,
		GitConfigPath:             cfg.GitConfigPath,
		GPGForward:                cfg.GPGForward,
		GPGAllowedKeyIDs:          cfg.GPGAllowedKeyIDs,
		GPGDir:                    cfg.GPGDir,
		TmuxForward:               cfg.TmuxForward,
		HistoryPersist:            cfg.HistoryPersist,
		TerminalOSC:               cfg.TerminalOSC,
		DockerDindMode:            cfg.DockerDindMode,
		EnvFileLoad:               cfg.EnvFileLoad,
		EnvFile:                   cfg.EnvFile,
		LogEnabled:                cfg.LogEnabled,
		LogFile:                   cfg.LogFile,
		ImageName:                 cfg.ImageName,
		Persistent:                cfg.Persistent,
		WorkdirAutomount:          cfg.WorkdirAutomount,
		WorkdirReadonly:           cfg.WorkdirReadonly,
		WorkdirAutotrust:          cfg.WorkdirAutotrust,
		Workdir:                   cfg.Workdir,
		FirewallEnabled:           cfg.FirewallEnabled,
		FirewallMode:              cfg.FirewallMode,
		GlobalFirewallAllowed:     cfg.GlobalFirewallAllowed,
		GlobalFirewallDenied:      cfg.GlobalFirewallDenied,
		ProjectFirewallAllowed:    cfg.ProjectFirewallAllowed,
		ProjectFirewallDenied:     cfg.ProjectFirewallDenied,
		ExtensionFirewallAllowed:  cfg.ExtensionFirewallAllowed,
		ExtensionFirewallDenied:   cfg.ExtensionFirewallDenied,
		Mode:                      cfg.Mode,
		Provider:                  cfg.Provider,
		Extensions:                cfg.Extensions,
		Command:                   cfg.Command,
		ContainerCPUs:             cfg.ContainerCPUs,
		ContainerMemory:           cfg.ContainerMemory,
		Security:                  cfg.Security,
		Otel:                      cfg.Otel,
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
	if err := prov.BuildIfNeeded(false, false); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// Create runner
	runner := core.NewRunner(prov, providerCfg)

	// Auto-detect GitHub token from gh CLI if configured
	config.HandleGitHubGhAuth(cfg.GitHubTokenSource)

	// Filter GH_TOKEN from env vars if forwarding is disabled
	providerCfg.EnvVars = config.HandleGitHubToken(cfg.GitHubForwardToken, providerCfg.EnvVars)

	// Load env file if enabled
	if cfg.EnvFileLoad {
		if err := config.LoadEnvFile(cfg.EnvFile); err != nil {
			fmt.Printf("Error loading env file: %v\n", err)
			os.Exit(1)
		}
	}

	// Run via runner
	if err := runner.Run(args); err != nil {
		os.Exit(1)
	}

	// Cleanup
	prov.Cleanup()
}

// handleSubcommand handles addt subcommands (build, shell, containers, firewall)
func handleSubcommand(subCmd string, subArgs []string, version, defaultNodeVersion, defaultGoVersion, defaultUvVersion string, defaultPortRangeStart int) {
	cfg := config.LoadConfig(version, defaultNodeVersion, defaultGoVersion, defaultUvVersion, defaultPortRangeStart)

	switch subCmd {
	case "build":
		// Check for --force and --rebuild-base flags
		forceNoCache := false
		rebuildBase := false
		var filteredArgs []string
		for _, arg := range subArgs {
			if arg == "--force" {
				forceNoCache = true
			} else if arg == "--rebuild-base" {
				rebuildBase = true
			} else {
				filteredArgs = append(filteredArgs, arg)
			}
		}
		subArgs = filteredArgs

		// Check if extension is passed as first arg (addt build claude)
		if len(subArgs) > 0 && !strings.HasPrefix(subArgs[0], "-") {
			cfg.Extensions = subArgs[0]
			subArgs = subArgs[1:]
		}
		// Check if extension is specified
		if cfg.Extensions == "" {
			fmt.Println("Error: No extension specified")
			fmt.Println()
			fmt.Println("Usage: addt build <extension> [--force] [--rebuild-base]")
			fmt.Println("       ADDT_EXTENSIONS=claude addt build")
			fmt.Println()
			fmt.Println("Options:")
			fmt.Println("  --force         Rebuild without using Docker cache")
			fmt.Println("  --rebuild-base  Rebuild the base image before building extension image")
			fmt.Println()
			fmt.Println("Examples:")
			fmt.Println("  addt build claude")
			fmt.Println("  addt build claude --force")
			fmt.Println("  addt build claude --rebuild-base")
			fmt.Println("  addt build claude --force --rebuild-base")
			fmt.Println("  addt build claude,codex")
			os.Exit(1)
		}
		providerCfg := &provider.Config{
			AddtVersion:       cfg.AddtVersion,
			ExtensionVersions: cfg.ExtensionVersions,
			NodeVersion:       cfg.NodeVersion,
			GoVersion:         cfg.GoVersion,
			UvVersion:         cfg.UvVersion,
			Provider:          cfg.Provider,
			Extensions:        cfg.Extensions,
			NoCache:           forceNoCache,
		}
		prov, err := NewProvider(cfg.Provider, providerCfg)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		HandleBuildCommand(prov, providerCfg, subArgs, forceNoCache, rebuildBase)

	case "shell":
		HandleShellCommand(subArgs, version, defaultNodeVersion, defaultGoVersion, defaultUvVersion, defaultPortRangeStart)

	case "containers":
		providerCfg := &provider.Config{
			AddtVersion:       cfg.AddtVersion,
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
		firewallcmd.HandleCommand(subArgs)

	default:
		fmt.Printf("Unknown command: %s\n", subCmd)
		os.Exit(1)
	}
}
