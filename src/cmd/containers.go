package cmd

import (
	"fmt"
	"os"

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
		HandleBuildCommand(prov, cfg, args[1:], false, false)
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
		printContainersHelp()
		os.Exit(1)
	}
}

func printContainersHelp() {
	fmt.Println(`Usage: addt containers [command]

Commands:
  list, ls      List all persistent containers
  stop <name>   Stop a persistent container
  rm <name>     Remove a persistent container
  clean         Remove all persistent containers

Examples:
  addt containers list
  addt containers stop my-container
  addt containers rm my-container
  addt containers clean`)
}
