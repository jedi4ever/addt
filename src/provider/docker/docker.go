package docker

import (
	"crypto/md5"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"regexp"
	"strings"
	"time"

	"github.com/jedi4ever/dclaude/provider"
)

// DockerProvider implements the Provider interface for Docker
type DockerProvider struct {
	config             *provider.Config
	tempDirs           []string
	embeddedDockerfile []byte
	embeddedEntrypoint []byte
}

// NewDockerProvider creates a new Docker provider
func NewDockerProvider(cfg *provider.Config, dockerfile, entrypoint []byte) (provider.Provider, error) {
	return &DockerProvider{
		config:             cfg,
		tempDirs:           []string{},
		embeddedDockerfile: dockerfile,
		embeddedEntrypoint: entrypoint,
	}, nil
}

// Initialize initializes the Docker provider
func (p *DockerProvider) Initialize(cfg *provider.Config) error {
	p.config = cfg
	return p.CheckPrerequisites()
}

// GetName returns the provider name
func (p *DockerProvider) GetName() string {
	return "docker"
}

// CheckPrerequisites verifies Docker is installed and running
func (p *DockerProvider) CheckPrerequisites() error {
	// Check Docker is installed
	if _, err := exec.LookPath("docker"); err != nil {
		return fmt.Errorf("Docker is not installed. Please install Docker from: https://docs.docker.com/get-docker/")
	}

	// Check Docker daemon is running
	cmd := exec.Command("docker", "info")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Docker daemon is not running. Please start Docker and try again")
	}

	return nil
}

// Exists checks if a container exists (running or stopped)
func (p *DockerProvider) Exists(name string) bool {
	cmd := exec.Command("docker", "ps", "-a", "--filter", fmt.Sprintf("name=^%s$", name), "--format", "{{.Names}}")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(output)) == name
}

// IsRunning checks if a container is currently running
func (p *DockerProvider) IsRunning(name string) bool {
	cmd := exec.Command("docker", "ps", "--filter", fmt.Sprintf("name=^%s$", name), "--format", "{{.Names}}")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(output)) == name
}

// Start starts a stopped container
func (p *DockerProvider) Start(name string) error {
	cmd := exec.Command("docker", "start", name)
	return cmd.Run()
}

// Stop stops a running container
func (p *DockerProvider) Stop(name string) error {
	cmd := exec.Command("docker", "stop", name)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Remove removes a container
func (p *DockerProvider) Remove(name string) error {
	cmd := exec.Command("docker", "rm", "-f", name)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// List lists all persistent dclaude containers
func (p *DockerProvider) List() ([]provider.Environment, error) {
	cmd := exec.Command("docker", "ps", "-a", "--filter", "name=^dclaude-persistent-",
		"--format", "{{.Names}}\t{{.Status}}\t{{.CreatedAt}}")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var envs []provider.Environment
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) >= 3 {
			envs = append(envs, provider.Environment{
				Name:      parts[0],
				Status:    parts[1],
				CreatedAt: parts[2],
			})
		}
	}
	return envs, nil
}

// Run runs a new container
func (p *DockerProvider) Run(spec *provider.RunSpec) error {
	currentUser, _ := user.Current()
	homeDir := currentUser.HomeDir
	username := currentUser.Username

	// Check if we should use existing container
	useExistingContainer := false
	if spec.Persistent && p.Exists(spec.Name) {
		fmt.Printf("Found existing persistent container: %s\n", spec.Name)
		if p.IsRunning(spec.Name) {
			fmt.Println("Container is running, connecting...")
			useExistingContainer = true
		} else {
			fmt.Println("Container is stopped, starting...")
			p.Start(spec.Name)
			useExistingContainer = true
		}
	} else if spec.Persistent {
		fmt.Printf("Creating new persistent container: %s\n", spec.Name)
	}

	// Build docker command
	var dockerArgs []string
	if useExistingContainer {
		// Use exec to connect to existing container
		dockerArgs = []string{"exec"}
	} else {
		// Create new container
		if spec.Persistent {
			dockerArgs = []string{"run", "--name", spec.Name}
		} else {
			dockerArgs = []string{"run", "--rm", "--name", spec.Name}
		}
	}

	// Detect if running in interactive terminal
	if spec.Interactive {
		dockerArgs = append(dockerArgs, "-it")
		if !useExistingContainer {
			dockerArgs = append(dockerArgs, "--init")
		}
	} else {
		dockerArgs = append(dockerArgs, "-i")
	}

	// Only add volumes and environment when creating a new container
	if !useExistingContainer {
		// Add volumes
		for _, vol := range spec.Volumes {
			mount := fmt.Sprintf("%s:%s", vol.Source, vol.Target)
			if vol.ReadOnly {
				mount += ":ro"
			}
			dockerArgs = append(dockerArgs, "-v", mount)
		}

		// Mount .gitconfig
		gitconfigPath := fmt.Sprintf("%s/.gitconfig", homeDir)
		if _, err := os.Stat(gitconfigPath); err == nil {
			dockerArgs = append(dockerArgs, "-v", fmt.Sprintf("%s:/home/%s/.gitconfig:ro", gitconfigPath, username))
		}

		// Mount .claude directory
		claudeDir := fmt.Sprintf("%s/.claude", homeDir)
		if _, err := os.Stat(claudeDir); err == nil {
			dockerArgs = append(dockerArgs, "-v", fmt.Sprintf("%s:/home/%s/.claude", claudeDir, username))
		}

		// Mount .claude.json
		claudeJson := fmt.Sprintf("%s/.claude.json", homeDir)
		if _, err := os.Stat(claudeJson); err == nil {
			dockerArgs = append(dockerArgs, "-v", fmt.Sprintf("%s:/home/%s/.claude.json", claudeJson, username))
		}

		// Add env file if exists
		if spec.Env["DCLAUDE_ENV_FILE"] != "" {
			dockerArgs = append(dockerArgs, "--env-file", spec.Env["DCLAUDE_ENV_FILE"])
		}

		// SSH forwarding
		dockerArgs = append(dockerArgs, p.HandleSSHForwarding(spec.SSHForward, homeDir, username)...)

		// GPG forwarding
		if spec.GPGForward {
			gnupgDir := fmt.Sprintf("%s/.gnupg", homeDir)
			if _, err := os.Stat(gnupgDir); err == nil {
				dockerArgs = append(dockerArgs, "-v", fmt.Sprintf("%s:/home/%s/.gnupg", gnupgDir, username))
				dockerArgs = append(dockerArgs, "-e", "GPG_TTY=/dev/console")
			}
		}

		// Docker forwarding
		dockerArgs = append(dockerArgs, p.HandleDockerForwarding(spec.DockerForward, spec.Name)...)

		// Add ports
		for _, port := range spec.Ports {
			dockerArgs = append(dockerArgs, "-p", fmt.Sprintf("%d:%d", port.Host, port.Container))
		}

		// Add environment variables
		for k, v := range spec.Env {
			dockerArgs = append(dockerArgs, "-e", fmt.Sprintf("%s=%s", k, v))
		}
	}

	// Handle shell mode or normal mode
	if useExistingContainer {
		dockerArgs = append(dockerArgs, spec.Name)
		dockerArgs = append(dockerArgs, spec.Args...)
	} else {
		dockerArgs = append(dockerArgs, spec.ImageName)
		dockerArgs = append(dockerArgs, spec.Args...)
	}

	// Execute docker command
	cmd := exec.Command("docker", dockerArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// Shell opens a shell in a container
func (p *DockerProvider) Shell(spec *provider.RunSpec) error {
	currentUser, _ := user.Current()
	homeDir := currentUser.HomeDir
	username := currentUser.Username

	// Check if we should use existing container
	useExistingContainer := false
	if spec.Persistent && p.Exists(spec.Name) {
		fmt.Printf("Found existing persistent container: %s\n", spec.Name)
		if p.IsRunning(spec.Name) {
			fmt.Println("Container is running, connecting...")
			useExistingContainer = true
		} else {
			fmt.Println("Container is stopped, starting...")
			p.Start(spec.Name)
			useExistingContainer = true
		}
	} else if spec.Persistent {
		fmt.Printf("Creating new persistent container: %s\n", spec.Name)
	}

	// Build docker command
	var dockerArgs []string
	if useExistingContainer {
		dockerArgs = []string{"exec"}
	} else {
		if spec.Persistent {
			dockerArgs = []string{"run", "--name", spec.Name}
		} else {
			dockerArgs = []string{"run", "--rm", "--name", spec.Name}
		}
	}

	// Interactive mode
	if spec.Interactive {
		dockerArgs = append(dockerArgs, "-it")
		if !useExistingContainer {
			dockerArgs = append(dockerArgs, "--init")
		}
	} else {
		dockerArgs = append(dockerArgs, "-i")
	}

	// Only add volumes and environment when creating a new container
	if !useExistingContainer {
		// Add volumes
		for _, vol := range spec.Volumes {
			mount := fmt.Sprintf("%s:%s", vol.Source, vol.Target)
			if vol.ReadOnly {
				mount += ":ro"
			}
			dockerArgs = append(dockerArgs, "-v", mount)
		}

		// Mount .gitconfig
		gitconfigPath := fmt.Sprintf("%s/.gitconfig", homeDir)
		if _, err := os.Stat(gitconfigPath); err == nil {
			dockerArgs = append(dockerArgs, "-v", fmt.Sprintf("%s:/home/%s/.gitconfig:ro", gitconfigPath, username))
		}

		// Mount .claude directory
		claudeDir := fmt.Sprintf("%s/.claude", homeDir)
		if _, err := os.Stat(claudeDir); err == nil {
			dockerArgs = append(dockerArgs, "-v", fmt.Sprintf("%s:/home/%s/.claude", claudeDir, username))
		}

		// Mount .claude.json
		claudeJson := fmt.Sprintf("%s/.claude.json", homeDir)
		if _, err := os.Stat(claudeJson); err == nil {
			dockerArgs = append(dockerArgs, "-v", fmt.Sprintf("%s:/home/%s/.claude.json", claudeJson, username))
		}

		// Add env file if exists
		if spec.Env["DCLAUDE_ENV_FILE"] != "" {
			dockerArgs = append(dockerArgs, "--env-file", spec.Env["DCLAUDE_ENV_FILE"])
		}

		// SSH forwarding
		dockerArgs = append(dockerArgs, p.HandleSSHForwarding(spec.SSHForward, homeDir, username)...)

		// GPG forwarding
		if spec.GPGForward {
			gnupgDir := fmt.Sprintf("%s/.gnupg", homeDir)
			if _, err := os.Stat(gnupgDir); err == nil {
				dockerArgs = append(dockerArgs, "-v", fmt.Sprintf("%s:/home/%s/.gnupg", gnupgDir, username))
				dockerArgs = append(dockerArgs, "-e", "GPG_TTY=/dev/console")
			}
		}

		// Docker forwarding
		dockerArgs = append(dockerArgs, p.HandleDockerForwarding(spec.DockerForward, spec.Name)...)

		// Add ports
		for _, port := range spec.Ports {
			dockerArgs = append(dockerArgs, "-p", fmt.Sprintf("%d:%d", port.Host, port.Container))
		}

		// Add environment variables
		for k, v := range spec.Env {
			dockerArgs = append(dockerArgs, "-e", fmt.Sprintf("%s=%s", k, v))
		}
	}

	// Open shell
	fmt.Println("Opening bash shell in container...")
	if useExistingContainer {
		dockerArgs = append(dockerArgs, spec.Name, "/bin/bash")
		dockerArgs = append(dockerArgs, spec.Args...)
	} else {
		if spec.DockerForward == "isolated" || spec.DockerForward == "true" {
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
			dockerArgs = append(dockerArgs, spec.ImageName, "/bin/bash", "-c", script, "bash")
			dockerArgs = append(dockerArgs, spec.Args...)
		} else {
			dockerArgs = append(dockerArgs, "--entrypoint", "/bin/bash", spec.ImageName)
			dockerArgs = append(dockerArgs, spec.Args...)
		}
	}

	// Execute docker command
	cmd := exec.Command("docker", dockerArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// Cleanup removes temporary directories
func (p *DockerProvider) Cleanup() error {
	for _, dir := range p.tempDirs {
		os.RemoveAll(dir)
	}
	p.tempDirs = []string{}
	return nil
}

// GetStatus returns a status string for display
func (p *DockerProvider) GetStatus(cfg *provider.Config, envName string) string {
	status := fmt.Sprintf("Provider:%s Mode:%s", p.GetName(), cfg.Mode)

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

	// Port mappings - will be added by orchestrator

	// Persistent container name
	if cfg.Persistent {
		status += fmt.Sprintf(" | Container:%s", envName)
	}

	return status
}

// GenerateContainerName generates a persistent container name based on working directory
func (p *DockerProvider) GenerateContainerName() string {
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

// GenerateEphemeralName generates a unique ephemeral container name
func (p *DockerProvider) GenerateEphemeralName() string {
	return fmt.Sprintf("dclaude-%s-%d", time.Now().Format("20060102-150405"), os.Getpid())
}

// GeneratePersistentName is an alias for GenerateContainerName to implement Provider interface
func (p *DockerProvider) GeneratePersistentName() string {
	return p.GenerateContainerName()
}

// BuildIfNeeded ensures the Docker image is ready
func (p *DockerProvider) BuildIfNeeded(rebuild bool) error {
	// Handle --rebuild flag
	if rebuild {
		fmt.Printf("Rebuilding %s...\n", p.config.ImageName)
		if p.ImageExists(p.config.ImageName) {
			fmt.Println("Removing existing image...")
			cmd := exec.Command("docker", "rmi", p.config.ImageName)
			cmd.Run()
		}
	}

	// Check if image exists
	if !p.ImageExists(p.config.ImageName) {
		return p.BuildImage(p.embeddedDockerfile, p.embeddedEntrypoint)
	}

	// Check if versions match (Claude and Node) using prefix matching
	claudeLabel := p.GetImageLabel(p.config.ImageName, "tools.claude.version")
	nodeLabel := p.GetImageLabel(p.config.ImageName, "tools.node.version")

	needsRebuild := false
	// Claude version: use prefix matching unless "latest"
	if p.config.ClaudeVersion != "latest" && claudeLabel != "" {
		if !strings.HasPrefix(claudeLabel, p.config.ClaudeVersion) {
			fmt.Printf("Claude version mismatch: image has %s, requested %s\n", claudeLabel, p.config.ClaudeVersion)
			needsRebuild = true
		}
	}
	// Node version: use prefix matching (e.g., "20" matches "20.20.0")
	if nodeLabel != "" && !strings.HasPrefix(nodeLabel, p.config.NodeVersion) {
		fmt.Printf("Node version mismatch: image has %s, requested %s\n", nodeLabel, p.config.NodeVersion)
		needsRebuild = true
	}

	if needsRebuild {
		fmt.Println("Rebuilding image with new versions...")
		cmd := exec.Command("docker", "rmi", p.config.ImageName)
		cmd.Run()
		return p.BuildImage(p.embeddedDockerfile, p.embeddedEntrypoint)
	}

	return nil
}

// DetermineImageName determines the appropriate Docker image name based on config
func (p *DockerProvider) DetermineImageName() string {
	if p.config.ClaudeVersion == "latest" {
		// Query npm registry for latest version
		npmLatest := p.getNpmLatestVersion()
		if npmLatest != "" {
			// Check if we already have an image with this version
			existingImage := p.FindImageByLabel("tools.claude.version", npmLatest)
			if existingImage != "" {
				return existingImage
			}
			p.config.ClaudeVersion = npmLatest
			return fmt.Sprintf("dclaude:claude-%s", npmLatest)
		}
		return "dclaude:latest"
	}

	// Specific version requested - validate it exists
	if !p.validateNpmVersion(p.config.ClaudeVersion) {
		fmt.Printf("Error: Claude Code version %s does not exist in npm\n", p.config.ClaudeVersion)
		fmt.Println("Available versions: https://www.npmjs.com/package/@anthropic-ai/claude-code?activeTab=versions")
		os.Exit(1)
	}

	// Check if image exists
	existingImage := p.FindImageByLabel("tools.claude.version", p.config.ClaudeVersion)
	if existingImage != "" {
		return existingImage
	}
	return fmt.Sprintf("dclaude:claude-%s", p.config.ClaudeVersion)
}
