package main

import (
	"crypto/md5"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

// GenerateContainerName generates a persistent container name based on working directory
func GenerateContainerName() string {
	workdir, err := os.Getwd()
	if err != nil {
		workdir = "/tmp"
	}

	// Get directory name
	dirname := workdir
	if idx := strings.LastIndex(workdir, "/"); idx != -1 {
		dirname = workdir[idx+1:]
	}

	// Sanitize directory name (lowercase, remove special chars, max 20 chars)
	re := regexp.MustCompile(`[^a-z0-9-]+`)
	dirname = strings.ToLower(dirname)
	dirname = re.ReplaceAllString(dirname, "-")
	dirname = strings.Trim(dirname, "-")
	if len(dirname) > 20 {
		dirname = dirname[:20]
	}

	// Create hash of full path for uniqueness
	hash := md5.Sum([]byte(workdir))
	hashStr := fmt.Sprintf("%x", hash)[:8]

	return fmt.Sprintf("dclaude-persistent-%s-%s", dirname, hashStr)
}

// ContainerExists checks if a container exists (running or stopped)
func ContainerExists(name string) bool {
	cmd := exec.Command("docker", "ps", "-a", "--filter", fmt.Sprintf("name=^%s$", name), "--format", "{{.Names}}")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(output)) == name
}

// ContainerIsRunning checks if a container is currently running
func ContainerIsRunning(name string) bool {
	cmd := exec.Command("docker", "ps", "--filter", fmt.Sprintf("name=^%s$", name), "--format", "{{.Names}}")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(output)) == name
}

// ListPersistentContainers lists all persistent dclaude containers
func ListPersistentContainers() {
	fmt.Println("Persistent dclaude containers:")
	cmd := exec.Command("docker", "ps", "-a", "--filter", "name=^dclaude-persistent-",
		"--format", "table {{.Names}}\t{{.Status}}\t{{.CreatedAt}}")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()
}

// StopContainer stops a persistent container
func StopContainer(name string) {
	if name == "" {
		fmt.Println("Usage: dclaude containers stop <container-name>")
		os.Exit(1)
	}
	cmd := exec.Command("docker", "stop", name)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Printf("Error stopping container: %v\n", err)
		os.Exit(1)
	}
}

// RemoveContainer removes a persistent container
func RemoveContainer(name string) {
	if name == "" {
		fmt.Println("Usage: dclaude containers remove <container-name>")
		os.Exit(1)
	}
	cmd := exec.Command("docker", "rm", "-f", name)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Printf("Error removing container: %v\n", err)
		os.Exit(1)
	}
}

// CleanPersistentContainers removes all persistent dclaude containers
func CleanPersistentContainers() {
	fmt.Println("Removing all persistent dclaude containers...")

	// Get list of containers
	cmd := exec.Command("docker", "ps", "-a", "--filter", "name=^dclaude-persistent-", "--format", "{{.Names}}")
	output, err := cmd.Output()
	if err != nil {
		fmt.Printf("Error listing containers: %v\n", err)
		os.Exit(1)
	}

	containers := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(containers) == 0 || containers[0] == "" {
		fmt.Println("No persistent containers found")
		return
	}

	// Remove each container
	for _, container := range containers {
		container = strings.TrimSpace(container)
		if container != "" {
			fmt.Println(container)
			exec.Command("docker", "rm", "-f", container).Run()
		}
	}
	fmt.Println("✓ Cleaned")
}

// HandleContainersCommand handles the containers subcommand
func HandleContainersCommand(args []string) {
	if len(args) == 0 {
		args = []string{"list"}
	}

	action := args[0]
	switch action {
	case "list", "ls":
		ListPersistentContainers()
	case "stop":
		if len(args) > 1 {
			StopContainer(args[1])
		} else {
			StopContainer("")
		}
	case "remove", "rm":
		if len(args) > 1 {
			RemoveContainer(args[1])
		} else {
			RemoveContainer("")
		}
	case "clean":
		CleanPersistentContainers()
	default:
		fmt.Println(`Usage: dclaude containers [list|stop|remove|clean]

Commands:
  list, ls    - List all persistent containers
  stop <name> - Stop a persistent container
  remove <name> - Remove a persistent container
  clean       - Remove all persistent containers`)
		os.Exit(1)
	}
}
