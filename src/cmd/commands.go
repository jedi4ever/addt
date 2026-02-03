package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/jedi4ever/addt/provider"
)

// HandleContainersCommand handles the containers subcommand using a provider
func HandleContainersCommand(prov provider.Provider, cfg *provider.Config, args []string) {
	if len(args) == 0 {
		args = []string{"list"}
	}

	action := args[0]
	switch action {
	case "build":
		// Redirect to addt build for backwards compatibility
		HandleBuildCommand(prov, cfg, args[1:])
	case "list", "ls":
		envs, err := prov.List()
		if err != nil {
			fmt.Printf("Error listing environments: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Persistent %s environments:\n", prov.GetName())
		fmt.Println("NAME\t\t\t\tSTATUS\t\tCREATED")
		for _, env := range envs {
			fmt.Printf("%s\t%s\t%s\n", env.Name, env.Status, env.CreatedAt)
		}
	case "stop":
		if len(args) < 2 {
			fmt.Println("Usage: addt containers stop <name>")
			os.Exit(1)
		}
		if err := prov.Stop(args[1]); err != nil {
			fmt.Printf("Error stopping environment: %v\n", err)
			os.Exit(1)
		}
	case "remove", "rm":
		if len(args) < 2 {
			fmt.Println("Usage: addt containers remove <name>")
			os.Exit(1)
		}
		if err := prov.Remove(args[1]); err != nil {
			fmt.Printf("Error removing environment: %v\n", err)
			os.Exit(1)
		}
	case "clean":
		envs, err := prov.List()
		if err != nil {
			fmt.Printf("Error listing environments: %v\n", err)
			os.Exit(1)
		}
		if len(envs) == 0 {
			fmt.Println("No persistent environments found")
			return
		}
		fmt.Println("Removing all persistent environments...")
		var failed []string
		for _, env := range envs {
			if err := prov.Remove(env.Name); err != nil {
				failed = append(failed, env.Name)
				fmt.Printf("Failed to remove: %s (%v)\n", env.Name, err)
			} else {
				fmt.Printf("Removed: %s\n", env.Name)
			}
		}
		if len(failed) > 0 {
			fmt.Printf("Failed to remove %d container(s)\n", len(failed))
			os.Exit(1)
		}
		fmt.Println("✓ Cleaned")
	default:
		fmt.Println(`Usage: <agent> addt containers [list|stop|rm|clean]

Commands:
  list, ls      - List all persistent containers
  stop <name>   - Stop a persistent container
  rm <name>     - Remove a persistent container
  clean         - Remove all persistent containers`)
		os.Exit(1)
	}
}

// HandleBuildCommand handles the build command
func HandleBuildCommand(prov provider.Provider, cfg *provider.Config, args []string) {
	// Parse --build-arg flags
	for i := 0; i < len(args); i++ {
		if args[i] == "--build-arg" && i+1 < len(args) {
			parts := strings.SplitN(args[i+1], "=", 2)
			if len(parts) == 2 {
				key, val := parts[0], parts[1]
				switch {
				case key == "ADDT_EXTENSIONS":
					cfg.Extensions = val
				case key == "NODE_VERSION":
					cfg.NodeVersion = val
				case key == "GO_VERSION":
					cfg.GoVersion = val
				case key == "UV_VERSION":
					cfg.UvVersion = val
				case strings.HasSuffix(key, "_VERSION"):
					// Per-extension versions (e.g., CLAUDE_VERSION, CODEX_VERSION)
					extName := strings.TrimSuffix(key, "_VERSION")
					extName = strings.ToLower(extName)
					if cfg.ExtensionVersions == nil {
						cfg.ExtensionVersions = make(map[string]string)
					}
					cfg.ExtensionVersions[extName] = val
				}
			}
			i++ // Skip next arg
		}
	}

	// Determine image name
	cfg.ImageName = prov.DetermineImageName()

	// Always rebuild extension image when using build command
	// Base image is only rebuilt if it doesn't exist
	if err := prov.BuildIfNeeded(true, false); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
