package docker

import (
	"crypto/md5"
	"embed"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/jedi4ever/dclaude/provider"
)

// DockerProvider implements the Provider interface for Docker
type DockerProvider struct {
	config               *provider.Config
	tempDirs             []string
	embeddedDockerfile   []byte
	embeddedEntrypoint   []byte
	embeddedInitFirewall []byte
	embeddedExtensions   embed.FS
}

// NewDockerProvider creates a new Docker provider
func NewDockerProvider(cfg *provider.Config, dockerfile, entrypoint, initFirewall []byte, extensions embed.FS) (provider.Provider, error) {
	return &DockerProvider{
		config:               cfg,
		tempDirs:             []string{},
		embeddedDockerfile:   dockerfile,
		embeddedEntrypoint:   entrypoint,
		embeddedInitFirewall: initFirewall,
		embeddedExtensions:   extensions,
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

// GetExtensionMounts reads extension metadata from image and returns all mounts
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
	username := "claude" // Always use "claude" in container, but with host UID/GID

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

		// Add extension mounts
		dockerArgs = p.AddExtensionMounts(dockerArgs, spec.ImageName, homeDir)

		// Mount .gitconfig
		gitconfigPath := fmt.Sprintf("%s/.gitconfig", homeDir)
		if _, err := os.Stat(gitconfigPath); err == nil {
			dockerArgs = append(dockerArgs, "-v", fmt.Sprintf("%s:/home/%s/.gitconfig:ro", gitconfigPath, username))
		}

		// Note: Claude config mounts (~/.claude, ~/.claude.json) are now handled
		// by the claude extension via AddExtensionMounts above.
		// Use DCLAUDE_MOUNT_CLAUDE_CONFIG=false to disable them.

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

		// Firewall configuration
		if p.config.FirewallEnabled {
			// Requires NET_ADMIN capability for iptables
			dockerArgs = append(dockerArgs, "--cap-add", "NET_ADMIN")

			// Mount firewall config directory
			firewallConfigDir := filepath.Join(homeDir, ".dclaude", "firewall")
			if _, err := os.Stat(firewallConfigDir); err == nil {
				dockerArgs = append(dockerArgs, "-v", fmt.Sprintf("%s:/home/%s/.dclaude/firewall", firewallConfigDir, username))
			}
		}

		// Docker forwarding
		dockerArgs = append(dockerArgs, p.HandleDockerForwarding(spec.DindMode, spec.Name)...)

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
		// Call entrypoint with args for existing containers
		dockerArgs = append(dockerArgs, "/usr/local/bin/docker-entrypoint.sh")
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
	username := "claude" // Always use "claude" in container, but with host UID/GID

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

		// Add extension mounts
		dockerArgs = p.AddExtensionMounts(dockerArgs, spec.ImageName, homeDir)

		// Mount .gitconfig
		gitconfigPath := fmt.Sprintf("%s/.gitconfig", homeDir)
		if _, err := os.Stat(gitconfigPath); err == nil {
			dockerArgs = append(dockerArgs, "-v", fmt.Sprintf("%s:/home/%s/.gitconfig:ro", gitconfigPath, username))
		}

		// Note: Claude config mounts (~/.claude, ~/.claude.json) are now handled
		// by the claude extension via AddExtensionMounts above.
		// Use DCLAUDE_MOUNT_CLAUDE_CONFIG=false to disable them.

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

		// Firewall configuration
		if p.config.FirewallEnabled {
			// Requires NET_ADMIN capability for iptables
			dockerArgs = append(dockerArgs, "--cap-add", "NET_ADMIN")

			// Mount firewall config directory
			firewallConfigDir := filepath.Join(homeDir, ".dclaude", "firewall")
			if _, err := os.Stat(firewallConfigDir); err == nil {
				dockerArgs = append(dockerArgs, "-v", fmt.Sprintf("%s:/home/%s/.dclaude/firewall", firewallConfigDir, username))
			}
		}

		// Docker forwarding
		dockerArgs = append(dockerArgs, p.HandleDockerForwarding(spec.DindMode, spec.Name)...)

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
		// Override entrypoint to bash for shell mode
		// Need to handle firewall initialization and DinD initialization
		needsInit := spec.DindMode == "isolated" || spec.DindMode == "true" || p.config.FirewallEnabled

		if needsInit {
			// Create initialization script that runs before bash
			script := `
# Initialize firewall if enabled
if [ "${DCLAUDE_FIREWALL_ENABLED}" = "true" ] && [ -f /usr/local/bin/init-firewall.sh ]; then
    sudo /usr/local/bin/init-firewall.sh
fi

# Start Docker daemon if in DinD mode
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
			dockerArgs = append(dockerArgs, "--entrypoint", "/bin/bash", spec.ImageName, "-c", script, "bash")
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
	switch cfg.DindMode {
	case "isolated", "true":
		status += " | Docker:isolated"
	case "host":
		status += " | Docker:host"
	default:
		status += " | Docker:-"
	}

	// Firewall status
	if cfg.FirewallEnabled {
		status += fmt.Sprintf(" | Firewall:%s", cfg.FirewallMode)
	} else {
		status += " | Firewall:-"
	}

	// Port mappings - will be added by orchestrator

	// Persistent container name
	if cfg.Persistent {
		status += fmt.Sprintf(" | Container:%s", envName)
	}

	return status
}

// GenerateContainerName generates a persistent container name based on working directory and extensions
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

	// Get sorted extensions for consistent naming
	extensions := strings.Split(p.config.Extensions, ",")
	for i := range extensions {
		extensions[i] = strings.TrimSpace(extensions[i])
	}
	var validExts []string
	for _, ext := range extensions {
		if ext != "" {
			validExts = append(validExts, ext)
		}
	}
	sort.Strings(validExts)
	extStr := strings.Join(validExts, ",")

	// Create hash of workdir + extensions for uniqueness
	// Same workdir + same extensions = same container
	// Same workdir + different extensions = different container
	hashInput := workdir + "|" + extStr
	hash := md5.Sum([]byte(hashInput))
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
	imageExists := p.ImageExists(p.config.ImageName)

	// Handle --rebuild flag
	if rebuild {
		if imageExists {
			fmt.Printf("Rebuilding %s...\n", p.config.ImageName)
			fmt.Println("Removing existing image...")
			cmd := exec.Command("docker", "rmi", p.config.ImageName)
			cmd.Run()
		}
		return p.BuildImage(p.embeddedDockerfile, p.embeddedEntrypoint)
	}

	// If image doesn't exist, build it
	if !imageExists {
		return p.BuildImage(p.embeddedDockerfile, p.embeddedEntrypoint)
	}

	// Image exists with matching tag - versions are encoded in tag, no rebuild needed
	return nil
}

// DetermineImageName determines the appropriate Docker image name based on installed extensions
func (p *DockerProvider) DetermineImageName() string {
	// Parse extensions list (comma-separated)
	extensions := strings.Split(p.config.Extensions, ",")
	for i := range extensions {
		extensions[i] = strings.TrimSpace(extensions[i])
	}

	// Filter empty entries and sort alphabetically for consistent naming
	var validExts []string
	for _, ext := range extensions {
		if ext != "" {
			validExts = append(validExts, ext)
		}
	}
	sort.Strings(validExts)

	// Build tag parts: ext1-version1_ext2-version2
	var tagParts []string
	for _, ext := range validExts {
		version := p.resolveExtensionVersion(ext)
		tagParts = append(tagParts, fmt.Sprintf("%s-%s", ext, version))
	}

	// Join with underscore
	tag := strings.Join(tagParts, "_")
	if tag == "" {
		tag = "base"
	}

	// Check if image already exists with this exact tag
	imageName := fmt.Sprintf("dclaude:%s", tag)
	if p.ImageExists(imageName) {
		return imageName
	}

	return imageName
}

// resolveExtensionVersion resolves the version for an extension, handling dist-tags
func (p *DockerProvider) resolveExtensionVersion(extName string) string {
	version := p.getExtensionVersion(extName)

	// For claude extension, handle npm dist-tags (latest, stable, next)
	if extName == "claude" && (version == "latest" || version == "stable" || version == "next") {
		npmVersion := p.getNpmVersionByTag(version)
		if npmVersion != "" {
			p.setExtensionVersion(extName, npmVersion)
			return npmVersion
		}
	}

	// For claude with specific version, validate it exists
	if extName == "claude" && version != "latest" && version != "stable" && version != "next" {
		if !p.validateNpmVersion(version) {
			fmt.Printf("Error: Claude Code version %s does not exist in npm\n", version)
			fmt.Println("Available versions: https://www.npmjs.com/package/@anthropic-ai/claude-code?activeTab=versions")
			os.Exit(1)
		}
	}

	return version
}

// getExtensionVersion returns the version for an extension, defaulting to "stable" for claude
func (p *DockerProvider) getExtensionVersion(extName string) string {
	if p.config.ExtensionVersions == nil {
		if extName == "claude" {
			return "stable"
		}
		return "latest"
	}
	if ver, ok := p.config.ExtensionVersions[extName]; ok {
		return ver
	}
	if extName == "claude" {
		return "stable"
	}
	return "latest"
}

// setExtensionVersion sets the version for an extension
func (p *DockerProvider) setExtensionVersion(extName, version string) {
	if p.config.ExtensionVersions == nil {
		p.config.ExtensionVersions = make(map[string]string)
	}
	p.config.ExtensionVersions[extName] = version
}
