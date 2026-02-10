package cmd

import (
	"fmt"
	"os"
	"strings"

	extcmd "github.com/jedi4ever/addt/cmd/extensions"
	"github.com/jedi4ever/addt/config"
	"github.com/jedi4ever/addt/core"
	"github.com/jedi4ever/addt/provider"
	"github.com/jedi4ever/addt/util"
)

var shellLogger = util.Log("shell")

// HandleShellCommand handles the "addt shell <extension>" command.
// Opens an interactive shell in a container with the specified extension.
func HandleShellCommand(args []string, version, defaultNodeVersion, defaultGoVersion, defaultUvVersion string, defaultPortRangeStart int) {
	cfg := config.LoadConfig(version, defaultNodeVersion, defaultGoVersion, defaultUvVersion, defaultPortRangeStart)

	// Parse extension from args
	var shellArgs []string
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		cfg.Extensions = args[0]
		shellArgs = args[1:]
	} else {
		shellArgs = args
	}

	// Check if extension is specified
	if cfg.Extensions == "" {
		printShellHelp()
		os.Exit(1)
	}

	// Validate extension exists
	if !extcmd.Exists(cfg.Extensions) {
		fmt.Printf("Error: extension '%s' does not exist\n", cfg.Extensions)
		fmt.Println("Run 'addt extensions list' to see available extensions")
		os.Exit(1)
	}

	// Set command to the extension's entrypoint
	if cfg.Command == "" {
		cfg.Command = extcmd.GetEntrypoint(cfg.Extensions)
	}

	// Create provider config
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
		Ports:                     cfg.Ports,
		PortRangeStart:            cfg.PortRangeStart,
		PortsInjectSystemPrompt:   cfg.PortsInjectSystemPrompt,
		SSHForwardKeys:            cfg.SSHForwardKeys,
		SSHForwardMode:            cfg.SSHForwardMode,
		SSHAllowedKeys:            cfg.SSHAllowedKeys,
		SSHDir:                    cfg.SSHDir,
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

	// Create and initialize provider
	prov, err := NewProvider(cfg.Provider, providerCfg)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	if err := prov.Initialize(providerCfg); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// Determine image name and build if needed
	providerCfg.ImageName = prov.DetermineImageName()
	if err := prov.BuildIfNeeded(false, false); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// Run shell via runner
	runner := core.NewRunner(prov, providerCfg)
	if err := runner.Shell(shellArgs); err != nil {
		prov.Cleanup()
		os.Exit(1)
	}

	prov.Cleanup()
}

func printShellHelp() {
	fmt.Println("Usage: addt shell <extension> [args...]")
	fmt.Println()
	fmt.Println("Open an interactive shell in a container with the specified extension.")
	fmt.Println()
	fmt.Println("Arguments:")
	fmt.Println("  <extension>    Name of the extension to use")
	fmt.Println("  [args...]      Optional arguments to pass to the shell")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  addt shell claude")
	fmt.Println("  addt shell codex")
	fmt.Println("  addt shell gemini")
	fmt.Println()
	fmt.Println("Environment:")
	fmt.Println("  ADDT_EXTENSIONS    Extension name (alternative to positional arg)")
	fmt.Println()
	fmt.Println("To see available extensions:")
	fmt.Println("  addt extensions list")
}
