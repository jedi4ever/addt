package cmd

import (
	"fmt"
	"os"

	"github.com/jedi4ever/dclaude/config"
	"github.com/jedi4ever/dclaude/core"
	"github.com/jedi4ever/dclaude/internal/update"
	"github.com/jedi4ever/dclaude/provider"
)

// Execute is the main entry point for the CLI
func Execute(version, defaultNodeVersion, defaultGoVersion, defaultUvVersion string, defaultPortRangeStart int) {
	// Parse command line arguments
	args := os.Args[1:]

	// Check for special commands
	if len(args) > 0 {
		switch args[0] {
		case "--update":
			update.UpdateDClaude(version)
			return
		case "--dversion":
			fmt.Printf("dclaude version %s\n", version)
			return
		case "--dhelp":
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
		case "build":
			HandleBuildCommand(args[1:], defaultNodeVersion, defaultGoVersion, defaultUvVersion, defaultPortRangeStart)
			return
		case "containers":
			// Load config for provider
			cfg := config.LoadConfig(defaultNodeVersion, defaultGoVersion, defaultUvVersion, defaultPortRangeStart)
			providerCfg := &provider.Config{
				Provider: cfg.Provider,
			}
			prov, err := NewProvider(cfg.Provider, providerCfg)
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}
			HandleContainersCommand(prov, args[1:])
			return
		case "firewall":
			HandleFirewallCommand(args[1:])
			return
		}
	}

	// Load configuration
	cfg := config.LoadConfig(defaultNodeVersion, defaultGoVersion, defaultUvVersion, defaultPortRangeStart)

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

	// Note: --yolo and other agent-specific arg transformations are handled
	// by each extension's args.sh script in the container

	// Convert main config to provider config
	providerCfg := &provider.Config{
		ExtensionVersions:    cfg.ExtensionVersions,
		MountExtensionConfig: cfg.MountExtensionConfig,
		NodeVersion:          cfg.NodeVersion,
		GoVersion:            cfg.GoVersion,
		UvVersion:            cfg.UvVersion,
		EnvVars:              cfg.EnvVars,
		GitHubDetect:         cfg.GitHubDetect,
		Ports:                cfg.Ports,
		PortRangeStart:       cfg.PortRangeStart,
		SSHForward:           cfg.SSHForward,
		GPGForward:           cfg.GPGForward,
		DindMode:             cfg.DindMode,
		EnvFile:              cfg.EnvFile,
		LogEnabled:           cfg.LogEnabled,
		LogFile:              cfg.LogFile,
		ImageName:            cfg.ImageName,
		Persistent:           cfg.Persistent,
		MountWorkdir:         cfg.MountWorkdir,
		FirewallEnabled:      cfg.FirewallEnabled,
		FirewallMode:         cfg.FirewallMode,
		Mode:                 cfg.Mode,
		Provider:             cfg.Provider,
		Extensions:           cfg.Extensions,
		Command:              cfg.Command,
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
	if err := prov.BuildIfNeeded(rebuildImage); err != nil {
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
	config.LoadEnvFile(cfg.EnvFile)

	// Run via orchestrator
	if err := orch.RunClaude(args, openShell); err != nil {
		os.Exit(1)
	}

	// Cleanup
	prov.Cleanup()
}
