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
		AddtVersion:           cfg.AddtVersion,
		ExtensionVersions:     cfg.ExtensionVersions,
		ExtensionAutomount:    cfg.ExtensionAutomount,
		ExtensionFlagSettings: cfg.ExtensionFlagSettings,
		NodeVersion:           cfg.NodeVersion,
		GoVersion:             cfg.GoVersion,
		UvVersion:             cfg.UvVersion,
		EnvVars:               cfg.EnvVars,
		GitHubDetect:          cfg.GitHubDetect,
		Ports:                 cfg.Ports,
		PortRangeStart:        cfg.PortRangeStart,
		SSHForward:            cfg.SSHForward,
		SSHAllowedKeys:        cfg.SSHAllowedKeys,
		GPGForward:            cfg.GPGForward,
		GPGAllowedKeyIDs:      cfg.GPGAllowedKeyIDs,
		TmuxForward:           cfg.TmuxForward,
		HistoryPersist:        cfg.HistoryPersist,
		DockerDindMode:        cfg.DockerDindMode,
		EnvFile:               cfg.EnvFile,
		LogEnabled:            cfg.LogEnabled,
		LogFile:               cfg.LogFile,
		Persistent:            cfg.Persistent,
		WorkdirAutomount:      cfg.WorkdirAutomount,
		WorkdirReadonly:       cfg.WorkdirReadonly,
		Workdir:               cfg.Workdir,
		FirewallEnabled:       cfg.FirewallEnabled,
		FirewallMode:          cfg.FirewallMode,
		Mode:                  cfg.Mode,
		Provider:              cfg.Provider,
		Extensions:            cfg.Extensions,
		Command:               cfg.Command,
		DockerCPUs:            cfg.DockerCPUs,
		DockerMemory:          cfg.DockerMemory,
		Security:              cfg.Security,
		Otel:                  cfg.Otel,
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
