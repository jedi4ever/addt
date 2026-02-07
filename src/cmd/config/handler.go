package config

import (
	"fmt"
	"os"

	cfgtypes "github.com/jedi4ever/addt/config"
	"github.com/jedi4ever/addt/extensions"
)

// parseGlobalFlag extracts -g/--global flag from args and returns filtered args
func parseGlobalFlag(args []string) ([]string, bool) {
	useGlobal := false
	var filtered []string
	for _, arg := range args {
		if arg == "-g" || arg == "--global" {
			useGlobal = true
		} else {
			filtered = append(filtered, arg)
		}
	}
	return filtered, useGlobal
}

// parseVerboseFlag extracts -v/--verbose flag from args and returns filtered args
func parseVerboseFlag(args []string) ([]string, bool) {
	verbose := false
	var filtered []string
	for _, arg := range args {
		if arg == "-v" || arg == "--verbose" {
			verbose = true
		} else {
			filtered = append(filtered, arg)
		}
	}
	return filtered, verbose
}

// HandleCommand handles the config subcommand
func HandleCommand(args []string) {
	if len(args) == 0 {
		printHelp()
		return
	}

	// Parse -g/--global flag
	args, useGlobal := parseGlobalFlag(args)
	// Parse -v/--verbose flag
	args, verbose := parseVerboseFlag(args)
	if len(args) == 0 {
		printHelp()
		return
	}

	switch args[0] {
	case "list":
		if useGlobal {
			listGlobal(verbose)
		} else {
			listProject(verbose)
		}
	case "get":
		if len(args) < 2 {
			fmt.Println("Usage: addt config get <key> [-g]")
			os.Exit(1)
		}
		if useGlobal {
			getGlobal(args[1])
		} else {
			getProject(args[1])
		}
	case "set":
		if len(args) < 3 {
			fmt.Println("Usage: addt config set <key> <value> [-g]")
			os.Exit(1)
		}
		if useGlobal {
			setGlobal(args[1], args[2])
		} else {
			setProject(args[1], args[2])
		}
	case "unset":
		if len(args) < 2 {
			fmt.Println("Usage: addt config unset <key> [-g]")
			os.Exit(1)
		}
		if useGlobal {
			unsetGlobal(args[1])
		} else {
			unsetProject(args[1])
		}
	case "extension":
		handleExtension(args[1:], useGlobal)
	case "path":
		fmt.Printf("Global config:  %s\n", cfgtypes.GetGlobalConfigPath())
		fmt.Printf("Project config: %s\n", cfgtypes.GetProjectConfigPath())
	default:
		fmt.Printf("Unknown config command: %s\n", args[0])
		printHelp()
		os.Exit(1)
	}
}

// handleExtension handles extension-specific config subcommands
func handleExtension(args []string, useGlobal bool) {
	if len(args) == 0 {
		printExtensionHelp()
		return
	}

	// Parse -g/--global flag from remaining args (in case it comes after extension name)
	args, globalFromExt := parseGlobalFlag(args)
	if globalFromExt {
		useGlobal = true
	}
	// Parse -v/--verbose flag from remaining args
	args, verbose := parseVerboseFlag(args)

	if len(args) == 0 {
		printExtensionHelp()
		return
	}

	extName := args[0]

	// Check if first arg is a subcommand (user forgot extension name)
	if extName == "list" || extName == "get" || extName == "set" || extName == "unset" {
		fmt.Println("Error: extension name required")
		fmt.Println()
		printExtensionHelp()
		os.Exit(1)
	}

	// Validate that the extension exists
	if !extensionExists(extName) {
		fmt.Printf("Error: extension '%s' does not exist\n", extName)
		fmt.Println("Run 'addt extensions list' to see available extensions")
		os.Exit(1)
	}

	if len(args) < 2 {
		// Default to list for extension
		listExtension(extName, useGlobal, verbose)
		return
	}

	switch args[1] {
	case "list":
		listExtension(extName, useGlobal, verbose)
	case "get":
		if len(args) < 3 {
			fmt.Println("Usage: addt config extension <name> get <key> [-g]")
			os.Exit(1)
		}
		getExtension(extName, args[2], useGlobal)
	case "set":
		if len(args) < 4 {
			fmt.Println("Usage: addt config extension <name> set <key> <value> [-g]")
			os.Exit(1)
		}
		setExtension(extName, args[2], args[3], useGlobal)
	case "unset":
		if len(args) < 3 {
			fmt.Println("Usage: addt config extension <name> unset <key> [-g]")
			os.Exit(1)
		}
		unsetExtension(extName, args[2], useGlobal)
	default:
		fmt.Printf("Unknown extension config command: %s\n", args[1])
		printExtensionHelp()
		os.Exit(1)
	}
}

// extensionExists checks if an extension with the given name exists
func extensionExists(name string) bool {
	exts, err := extensions.GetExtensions()
	if err != nil {
		return false
	}
	for _, ext := range exts {
		if ext.Name == name {
			return true
		}
	}
	return false
}

func printHelp() {
	fmt.Println("Usage: addt config <command> [-g]")
	fmt.Println()
	fmt.Println("Manage configuration. Project config (.addt.yaml) is the default.")
	fmt.Println("Use -g or --global for global config (~/.addt/config.yaml).")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  list                                    List configuration values")
	fmt.Println("  get <key>                               Get a configuration value")
	fmt.Println("  set <key> <value>                       Set a configuration value")
	fmt.Println("  unset <key>                             Remove a configuration value")
	fmt.Println("  extension <name> list                   List extension config")
	fmt.Println("  extension <name> get <key>              Get extension config value")
	fmt.Println("  extension <name> set <key> <value>      Set extension config value")
	fmt.Println("  extension <name> unset <key>            Remove extension config value")
	fmt.Println("  path                                    Show config file paths")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("  -g, --global    Use global config instead of project config")
	fmt.Println("  -v, --verbose   Show descriptions for each config key")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  addt config list                                # project config")
	fmt.Println("  addt config list -g                             # global config")
	fmt.Println("  addt config set container.cpus 2")
	fmt.Println("  addt config set firewall.enabled true -g")
	fmt.Println()
	fmt.Println("  addt config extension claude list               # list extension config")
	fmt.Println("  addt config extension claude set version 1.0.5  # set extension version")
	fmt.Println("  addt config extension claude set yolo true      # set extension flag")
	fmt.Println("  addt config extension claude set version 1.0.5 -g")
	fmt.Println()
	fmt.Println("Precedence (highest to lowest):")
	fmt.Println("  1. Environment variables (e.g., ADDT_FIREWALL)")
	fmt.Println("  2. Project config (.addt.yaml)")
	fmt.Println("  3. Global config (~/.addt/config.yaml)")
	fmt.Println("  4. Default values")
}

func printExtensionHelp() {
	fmt.Println("Usage: addt config extension <name> <command> [-g]")
	fmt.Println()
	fmt.Println("Manage extension-specific configuration.")
	fmt.Println("Project config is the default. Use -g for global config.")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  list              List extension configuration")
	fmt.Println("  get <key>         Get a configuration value")
	fmt.Println("  set <key> <value> Set a configuration value")
	fmt.Println("  unset <key>       Remove a configuration value")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("  -g, --global      Use global config instead of project config")
	fmt.Println()
	fmt.Println("Available keys:")
	fmt.Println("  version     Extension version (e.g., \"1.0.5\", \"latest\", \"stable\")")
	fmt.Println("  automount   Auto-mount extension config directories (true/false)")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  addt config extension claude list")
	fmt.Println("  addt config extension claude set version 1.0.5")
	fmt.Println("  addt config extension claude set version 1.0.5 -g")
}
