package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// ImageExists checks if a Docker image exists
func ImageExists(imageName string) bool {
	cmd := exec.Command("docker", "image", "inspect", imageName)
	return cmd.Run() == nil
}

// FindImageByLabel finds an image by a specific label value
func FindImageByLabel(label, value string) string {
	cmd := exec.Command("docker", "images",
		"--filter", fmt.Sprintf("label=%s=%s", label, value),
		"--format", "{{.Repository}}:{{.Tag}}")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if line != "" && !strings.Contains(line, "<none>") {
			return line
		}
	}
	return ""
}

// DetermineImageName determines the appropriate image name based on config
func DetermineImageName(cfg *Config) string {
	if cfg.ClaudeVersion == "latest" {
		// Query npm registry for latest version
		npmLatest := GetNpmLatestVersion()
		if npmLatest != "" {
			// Check if we already have an image with this version
			existingImage := FindImageByLabel("tools.claude.version", npmLatest)
			if existingImage != "" {
				return existingImage
			}
			cfg.ClaudeVersion = npmLatest
			return fmt.Sprintf("dclaude:claude-%s", npmLatest)
		}
		return "dclaude:latest"
	}

	// Specific version requested - validate it exists
	if !ValidateNpmVersion(cfg.ClaudeVersion) {
		fmt.Printf("Error: Claude Code version %s does not exist in npm\n", cfg.ClaudeVersion)
		fmt.Println("Available versions: https://www.npmjs.com/package/@anthropic-ai/claude-code?activeTab=versions")
		os.Exit(1)
	}

	// Check if image exists
	existingImage := FindImageByLabel("tools.claude.version", cfg.ClaudeVersion)
	if existingImage != "" {
		return existingImage
	}
	return fmt.Sprintf("dclaude:claude-%s", cfg.ClaudeVersion)
}

// BuildImage builds the Docker image
func BuildImage(cfg *Config) {
	fmt.Printf("Building %s...\n", cfg.ImageName)

	// Create temp directory for build context with embedded files
	buildDir, err := os.MkdirTemp("", "dclaude-build-*")
	if err != nil {
		fmt.Println("Error: Failed to create temp build directory")
		os.Exit(1)
	}
	defer os.RemoveAll(buildDir)

	// Write embedded Dockerfile
	dockerfilePath := filepath.Join(buildDir, "Dockerfile")
	if err := os.WriteFile(dockerfilePath, EmbeddedDockerfile, 0644); err != nil {
		fmt.Println("Error: Failed to write Dockerfile")
		os.Exit(1)
	}

	// Write embedded entrypoint script
	entrypointPath := filepath.Join(buildDir, "docker-entrypoint.sh")
	if err := os.WriteFile(entrypointPath, EmbeddedEntrypoint, 0755); err != nil {
		fmt.Println("Error: Failed to write docker-entrypoint.sh")
		os.Exit(1)
	}

	scriptDir := buildDir

	// Get current user info
	currentUser, _ := user.Current()
	uid := currentUser.Uid
	gid := currentUser.Gid
	username := currentUser.Username

	// Build docker command
	args := []string{
		"build",
		"--build-arg", fmt.Sprintf("NODE_VERSION=%s", cfg.NodeVersion),
		"--build-arg", fmt.Sprintf("USER_ID=%s", uid),
		"--build-arg", fmt.Sprintf("GROUP_ID=%s", gid),
		"--build-arg", fmt.Sprintf("USERNAME=%s", username),
		"--build-arg", fmt.Sprintf("CLAUDE_VERSION=%s", cfg.ClaudeVersion),
		"-t", cfg.ImageName,
		"-f", dockerfilePath,
		scriptDir,
	}

	cmd := exec.Command("docker", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Println("\nError: Failed to build Docker image")
		fmt.Println("Please check the Dockerfile and try again")
		os.Exit(1)
	}

	fmt.Println("\n✓ Image built successfully!")
	fmt.Println()
	fmt.Println("Detecting tool versions...")

	// Get versions from the built image
	versions := detectToolVersions(cfg.ImageName)

	// Add version labels to image
	addVersionLabels(cfg, versions)

	fmt.Println()
	fmt.Println("Installed versions:")
	if v, ok := versions["node"]; ok && v != "" {
		fmt.Printf("  • Node.js:     %s\n", v)
	}
	if v, ok := versions["claude"]; ok && v != "" {
		fmt.Printf("  • Claude Code: %s\n", v)
	}
	if v, ok := versions["gh"]; ok && v != "" {
		fmt.Printf("  • GitHub CLI:  %s\n", v)
	}
	if v, ok := versions["rg"]; ok && v != "" {
		fmt.Printf("  • Ripgrep:     %s\n", v)
	}
	if v, ok := versions["git"]; ok && v != "" {
		fmt.Printf("  • Git:         %s\n", v)
	}
	fmt.Println()
	fmt.Printf("Image tagged as: %s\n", cfg.ImageName)
}

func detectToolVersions(imageName string) map[string]string {
	versions := make(map[string]string)
	versionRegex := regexp.MustCompile(`[0-9]+\.[0-9]+\.[0-9]+`)

	tools := map[string][]string{
		"claude": {"claude", "--version"},
		"gh":     {"gh", "--version"},
		"rg":     {"rg", "--version"},
		"git":    {"git", "--version"},
		"node":   {"node", "--version"},
	}

	for name, cmdArgs := range tools {
		args := append([]string{"run", "--rm", "--entrypoint", cmdArgs[0], imageName}, cmdArgs[1:]...)
		cmd := exec.Command("docker", args...)
		output, err := cmd.Output()
		if err == nil {
			if match := versionRegex.FindString(string(output)); match != "" {
				versions[name] = match
			}
		}
	}

	return versions
}

func addVersionLabels(cfg *Config, versions map[string]string) {
	// Create temporary Dockerfile
	tmpFile, err := os.CreateTemp("", "Dockerfile-labels-*")
	if err != nil {
		return
	}
	defer os.Remove(tmpFile.Name())

	content := fmt.Sprintf("FROM %s\n", cfg.ImageName)
	for tool, version := range versions {
		if version != "" {
			content += fmt.Sprintf("LABEL tools.%s.version=\"%s\"\n", tool, version)
		}
	}
	tmpFile.WriteString(content)
	tmpFile.Close()

	// Build with labels
	cmd := exec.Command("docker", "build", "-f", tmpFile.Name(), "-t", cfg.ImageName, ".")
	cmd.Run()

	// Tag as dclaude:latest if this is latest
	if cfg.ClaudeVersion == "latest" {
		exec.Command("docker", "tag", cfg.ImageName, "dclaude:latest").Run()
	}

	// Tag with claude version
	if v, ok := versions["claude"]; ok && v != "" {
		exec.Command("docker", "tag", cfg.ImageName, fmt.Sprintf("dclaude:claude-%s", v)).Run()
	}
}

// RunDocker runs the Docker container with the specified configuration
func RunDocker(cfg *Config, args []string, openShell bool) {
	currentUser, _ := user.Current()
	homeDir := currentUser.HomeDir
	username := currentUser.Username
	cwd, _ := os.Getwd()

	// Generate container name (persistent or ephemeral)
	var containerName string
	useExistingContainer := false

	if cfg.Persistent {
		containerName = GenerateContainerName()

		// Check if persistent container exists
		if ContainerExists(containerName) {
			fmt.Printf("Found existing persistent container: %s\n", containerName)

			// Check if it's running
			if ContainerIsRunning(containerName) {
				fmt.Println("Container is running, connecting...")
				useExistingContainer = true
			} else {
				fmt.Println("Container is stopped, starting...")
				exec.Command("docker", "start", containerName).Run()
				useExistingContainer = true
			}
		} else {
			fmt.Printf("Creating new persistent container: %s\n", containerName)
		}
	} else {
		// Ephemeral mode - generate unique name
		containerName = fmt.Sprintf("dclaude-%s-%d", time.Now().Format("20060102-150405"), os.Getpid())
	}

	// Build docker command
	var dockerArgs []string
	if useExistingContainer {
		// Use exec to connect to existing container
		dockerArgs = []string{"exec"}
	} else {
		// Create new container
		if cfg.Persistent {
			// Persistent container - don't use --rm
			dockerArgs = []string{"run", "--name", containerName}
		} else {
			// Ephemeral container - use --rm
			dockerArgs = []string{"run", "--rm", "--name", containerName}
		}
	}

	// Detect if running in interactive terminal
	if IsTerminal() {
		dockerArgs = append(dockerArgs, "-it")
		// Use init to handle signals properly for interactive sessions
		if !useExistingContainer {
			dockerArgs = append(dockerArgs, "--init")
		}
	} else {
		dockerArgs = append(dockerArgs, "-i")
	}

	// Declare port mapping variables outside the block so they're accessible for status line
	var portMapString, portMapDisplay string

	// Only add volumes and environment when creating a new container
	if !useExistingContainer {
		// Mount current directory
		dockerArgs = append(dockerArgs, "-v", fmt.Sprintf("%s:/workspace", cwd))

	// Add env file if exists
	envFilePath := cfg.EnvFile
	if envFilePath == "" {
		envFilePath = ".env"
	}
	if !filepath.IsAbs(envFilePath) {
		envFilePath = filepath.Join(cwd, envFilePath)
	}
	if info, err := os.Stat(envFilePath); err == nil && !info.IsDir() {
		dockerArgs = append(dockerArgs, "--env-file", envFilePath)
	}

	// Mount .gitconfig
	gitconfigPath := filepath.Join(homeDir, ".gitconfig")
	if _, err := os.Stat(gitconfigPath); err == nil {
		dockerArgs = append(dockerArgs, "-v", fmt.Sprintf("%s:/home/%s/.gitconfig:ro", gitconfigPath, username))
	}

	// Mount .claude directory
	claudeDir := filepath.Join(homeDir, ".claude")
	if _, err := os.Stat(claudeDir); err == nil {
		dockerArgs = append(dockerArgs, "-v", fmt.Sprintf("%s:/home/%s/.claude", claudeDir, username))
	}

	// Mount .claude.json
	claudeJson := filepath.Join(homeDir, ".claude.json")
	if _, err := os.Stat(claudeJson); err == nil {
		dockerArgs = append(dockerArgs, "-v", fmt.Sprintf("%s:/home/%s/.claude.json", claudeJson, username))
	}

	// GPG forwarding
	if cfg.GPGForward {
		gnupgDir := filepath.Join(homeDir, ".gnupg")
		if _, err := os.Stat(gnupgDir); err == nil {
			dockerArgs = append(dockerArgs, "-v", fmt.Sprintf("%s:/home/%s/.gnupg", gnupgDir, username))
			dockerArgs = append(dockerArgs, "-e", "GPG_TTY=/dev/console")
		}
	}

	// SSH forwarding
	dockerArgs = append(dockerArgs, HandleSSHForwarding(cfg, homeDir, username)...)

	// Docker forwarding
	dockerArgs = append(dockerArgs, HandleDockerForwarding(cfg, containerName)...)

	// Port mappings
	portMapString, portMapDisplay = HandlePortMappings(cfg, &dockerArgs)
	if portMapString != "" {
		dockerArgs = append(dockerArgs, "-e", fmt.Sprintf("DCLAUDE_PORT_MAP=%s", portMapString))
	}

	// Pass terminal environment variables for proper paste handling
	if term := os.Getenv("TERM"); term != "" {
		dockerArgs = append(dockerArgs, "-e", fmt.Sprintf("TERM=%s", term))
	}
	if colorterm := os.Getenv("COLORTERM"); colorterm != "" {
		dockerArgs = append(dockerArgs, "-e", fmt.Sprintf("COLORTERM=%s", colorterm))
	}
	// Pass terminal size variables (critical for proper line handling in containers)
	cols, lines := GetTerminalSize()
	dockerArgs = append(dockerArgs, "-e", fmt.Sprintf("COLUMNS=%d", cols))
	dockerArgs = append(dockerArgs, "-e", fmt.Sprintf("LINES=%d", lines))

	// Pass environment variables
	for _, varName := range cfg.EnvVars {
		if value := os.Getenv(varName); value != "" {
			dockerArgs = append(dockerArgs, "-e", fmt.Sprintf("%s=%s", varName, value))
		}
	}
	} // End of useExistingContainer = false block

	// Display status line
	BuildStatusLine(cfg, portMapDisplay, containerName)

	// Handle shell mode or normal mode
	if useExistingContainer {
		// Exec into existing container
		dockerArgs = append(dockerArgs, containerName)
		if openShell {
			dockerArgs = append(dockerArgs, "/bin/bash")
			dockerArgs = append(dockerArgs, args...)
		} else {
			// Run claude command in existing container
			dockerArgs = append(dockerArgs, "claude")
			dockerArgs = append(dockerArgs, args...)
		}
	} else {
		// Create new container
		if openShell {
			fmt.Println("Opening bash shell in container...")
			if cfg.DockerForward == "isolated" || cfg.DockerForward == "true" {
				// DinD mode with shell
				script := `
if [ "$DCLAUDE_DIND" = "true" ]; then
    echo 'Starting Docker daemon in isolated mode...'
    sudo dockerd --host=unix:///var/run/docker.sock >/tmp/docker.log 2>&1 &
    echo 'Waiting for Docker daemon...'
    for i in $(seq 1 30); do
        if [ -S /var/run/docker.sock ]; then
            sudo chmod 666 /var/run/docker.sock
            if docker info >/dev/null 2>&1; then
                echo '✓ Docker daemon ready (isolated environment)'
                break
            fi
        fi
        sleep 1
    done
fi
exec /bin/bash "$@"
`
				dockerArgs = append(dockerArgs, cfg.ImageName, "/bin/bash", "-c", script, "bash")
				dockerArgs = append(dockerArgs, args...)
			} else {
				dockerArgs = append(dockerArgs, "--entrypoint", "/bin/bash", cfg.ImageName)
				dockerArgs = append(dockerArgs, args...)
			}
		} else {
			dockerArgs = append(dockerArgs, cfg.ImageName)
			dockerArgs = append(dockerArgs, args...)
		}
	}

	// Log the command
	if cfg.LogEnabled {
		LogCommand(cfg, cwd, containerName, args)
	}

	// Execute docker command
	cmd := exec.Command("docker", dockerArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	cmd.Run()
	Cleanup()
}

// BuildStatusLine displays the status line with configuration info
func BuildStatusLine(cfg *Config, portMapDisplay string, containerName string) {
	// Mode (container or shell)
	status := fmt.Sprintf("Mode:%s", cfg.Mode)

	// Image name
	status += fmt.Sprintf(" | %s", cfg.ImageName)

	// Get Node version from image labels
	cmd := exec.Command("docker", "inspect", cfg.ImageName, "--format", "{{index .Config.Labels \"tools.node.version\"}}")
	if output, err := cmd.Output(); err == nil {
		if nodeVersion := strings.TrimSpace(string(output)); nodeVersion != "" {
			status += fmt.Sprintf(" | Node %s", nodeVersion)
		}
	}

	// GitHub token status
	if os.Getenv("GH_TOKEN") != "" {
		status += " | GH:✓"
	} else {
		status += " | GH:-"
	}

	// SSH forwarding status
	switch cfg.SSHForward {
	case "agent":
		status += " | SSH:agent"
	case "keys":
		status += " | SSH:keys"
	default:
		status += " | SSH:-"
	}

	// GPG forwarding status
	if cfg.GPGForward {
		status += " | GPG:✓"
	} else {
		status += " | GPG:-"
	}

	// Docker forwarding status
	switch cfg.DockerForward {
	case "isolated", "true":
		status += " | Docker:isolated"
	case "host":
		status += " | Docker:host"
	default:
		status += " | Docker:-"
	}

	// Port mappings
	if portMapDisplay != "" {
		status += fmt.Sprintf(" | Ports:%s", portMapDisplay)
	}

	// Persistent container name
	if cfg.Persistent {
		status += fmt.Sprintf(" | Container:%s", containerName)
	}

	fmt.Printf("✓ %s\n", status)
}

// CheckPrerequisites verifies Docker is installed and running
func CheckPrerequisites() {
	// Check Docker is installed
	if _, err := exec.LookPath("docker"); err != nil {
		fmt.Println("Error: Docker is not installed")
		fmt.Println("Please install Docker from: https://docs.docker.com/get-docker/")
		os.Exit(1)
	}

	// Check Docker daemon is running
	cmd := exec.Command("docker", "info")
	if err := cmd.Run(); err != nil {
		fmt.Println("Error: Docker daemon is not running")
		fmt.Println("Please start Docker and try again")
		os.Exit(1)
	}
}
