package cmd

import (
	"fmt"
	"os"

	"github.com/jedi4ever/addt/config"
)

// handleCliCommand handles the "cli" subcommand
func handleCliCommand(args []string, version string) {
	if len(args) == 0 {
		fmt.Println("Usage: addt cli <command>")
		fmt.Println()
		fmt.Println("Commands:")
		fmt.Println("  update          Install addt updates")
		fmt.Println("  install-podman  Download and install Podman")
		return
	}
	switch args[0] {
	case "update":
		UpdateAddt(version)
	case "install-podman":
		installPodman()
	default:
		fmt.Printf("Unknown cli command: %s\n", args[0])
		os.Exit(1)
	}
}

// installPodman downloads and installs Podman
func installPodman() {
	// Check if Podman is already available
	if path := config.GetPodmanPath(); path != "" {
		fmt.Printf("Podman is already available at: %s\n", path)
		return
	}

	// Download and install
	if err := config.DownloadPodman(); err != nil {
		fmt.Printf("Failed to install Podman: %v\n", err)
		os.Exit(1)
	}

	fmt.Println()
	fmt.Println("Podman has been installed to ~/.addt/bin/")
	fmt.Println("You can now use Podman as your container runtime.")
}

// printAddtSubcommandUsage prints the "addt" namespace usage (when invoked via agent)
func printAddtSubcommandUsage() {
	fmt.Println("Usage: <agent> addt <command>")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  update <ext> [version]    Update extension to latest or specific version")
	fmt.Println("  build [--build-arg ...]   Build the container image")
	fmt.Println("  shell                     Open bash shell in container")
	fmt.Println("  containers <subcommand>   Manage containers (list, stop, rm, clean)")
	fmt.Println("  firewall <subcommand>     Manage firewall (list, add, remove, reset)")
	fmt.Println("  extensions <subcommand>   Manage extensions (list, info, new)")
	fmt.Println("  config <subcommand>       Manage config (global, project, extension)")
	fmt.Println("  cli <subcommand>          Manage addt CLI (update)")
	fmt.Println("  version                   Show version info")
}
