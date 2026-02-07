package cmd

import (
	"fmt"
	"os"

	extcmd "github.com/jedi4ever/addt/cmd/extensions"
	"github.com/jedi4ever/addt/config"
	"github.com/jedi4ever/addt/provider"
)

// HandleUpdateCommand handles the "addt update <extension> [version]" command.
// It force-rebuilds the extension image without cache, picking up the latest version.
func HandleUpdateCommand(args []string, version, defaultNodeVersion, defaultGoVersion, defaultUvVersion string, defaultPortRangeStart int) {
	if len(args) < 1 || args[0] == "--help" || args[0] == "-h" {
		printUpdateHelp()
		return
	}

	extName := args[0]

	// Validate extension exists
	if !extcmd.Exists(extName) {
		fmt.Printf("Error: extension '%s' does not exist\n", extName)
		fmt.Println("Run 'addt extensions list' to see available extensions")
		os.Exit(1)
	}

	// Load config
	cfg := config.LoadConfig(version, defaultNodeVersion, defaultGoVersion, defaultUvVersion, defaultPortRangeStart)
	cfg.Extensions = extName

	// If version argument provided, use it as a one-time override
	if len(args) > 1 {
		if cfg.ExtensionVersions == nil {
			cfg.ExtensionVersions = make(map[string]string)
		}
		cfg.ExtensionVersions[extName] = args[1]
		fmt.Printf("Updating %s to version %s...\n", extName, args[1])
	} else {
		ver := cfg.ExtensionVersions[extName]
		if ver == "" {
			ver = "latest"
		}
		fmt.Printf("Updating %s (%s)...\n", extName, ver)
	}

	// Create provider config (same minimal set as build command)
	providerCfg := &provider.Config{
		AddtVersion:       cfg.AddtVersion,
		ExtensionVersions: cfg.ExtensionVersions,
		NodeVersion:       cfg.NodeVersion,
		GoVersion:         cfg.GoVersion,
		UvVersion:         cfg.UvVersion,
		Provider:          cfg.Provider,
		Extensions:        cfg.Extensions,
		NoCache:           true,
	}

	prov, err := NewProvider(cfg.Provider, providerCfg)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	providerCfg.ImageName = prov.DetermineImageName()

	if err := prov.BuildIfNeeded(true, false); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully updated %s\n", extName)
}

func printUpdateHelp() {
	fmt.Println("Usage: addt update <extension> [version]")
	fmt.Println()
	fmt.Println("Update an extension by rebuilding its container image.")
	fmt.Println("Forces a fresh build without cache to pick up the latest version.")
	fmt.Println()
	fmt.Println("Arguments:")
	fmt.Println("  <extension>    Name of the extension to update")
	fmt.Println("  [version]      Optional version to install (one-time override)")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  addt update claude             # Rebuild with latest/configured version")
	fmt.Println("  addt update claude 1.0.5       # Rebuild with specific version")
	fmt.Println("  addt update codex              # Update codex extension")
	fmt.Println()
	fmt.Println("To see available extensions:")
	fmt.Println("  addt extensions list")
}
