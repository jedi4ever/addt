package docker

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"regexp"
	"strings"
)

// ImageExists checks if a Docker image exists
func (p *DockerProvider) ImageExists(imageName string) bool {
	cmd := exec.Command("docker", "image", "inspect", imageName)
	return cmd.Run() == nil
}

// FindImageByLabel finds an image by a specific label value
func (p *DockerProvider) FindImageByLabel(label, value string) string {
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

// GetImageLabel retrieves a specific label value from an image
func (p *DockerProvider) GetImageLabel(imageName, label string) string {
	cmd := exec.Command("docker", "inspect",
		"--format", fmt.Sprintf("{{index .Config.Labels %q}}", label),
		imageName)
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

// GetBaseImageName returns the base image name for the current config
func (p *DockerProvider) GetBaseImageName() string {
	// Get current user info for UID/GID in tag
	currentUser, err := user.Current()
	if err != nil {
		return "addt-base:latest"
	}
	// Base image is tagged with node version and UID to ensure compatibility
	return fmt.Sprintf("addt-base:node%s-uid%s", p.config.NodeVersion, currentUser.Uid)
}

// BuildBaseImage builds the base Docker image (contains Node, Go, UV, system packages)
func (p *DockerProvider) BuildBaseImage() error {
	baseImageName := p.GetBaseImageName()
	fmt.Printf("Building base image %s...\n", baseImageName)

	// Create temp directory for build context
	buildDir, err := os.MkdirTemp("", "addt-base-build-*")
	if err != nil {
		return fmt.Errorf("failed to create temp build directory: %w", err)
	}
	defer os.RemoveAll(buildDir)

	// Write embedded Dockerfile.base
	dockerfilePath := filepath.Join(buildDir, "Dockerfile.base")
	if err := os.WriteFile(dockerfilePath, p.embeddedDockerfileBase, 0644); err != nil {
		return fmt.Errorf("failed to write Dockerfile.base: %w", err)
	}

	// Write embedded entrypoint script
	entrypointPath := filepath.Join(buildDir, "docker-entrypoint.sh")
	if err := os.WriteFile(entrypointPath, p.embeddedEntrypoint, 0755); err != nil {
		return fmt.Errorf("failed to write docker-entrypoint.sh: %w", err)
	}

	// Write embedded firewall init script
	initFirewallPath := filepath.Join(buildDir, "init-firewall.sh")
	if err := os.WriteFile(initFirewallPath, p.embeddedInitFirewall, 0755); err != nil {
		return fmt.Errorf("failed to write init-firewall.sh: %w", err)
	}

	// Get current user info
	currentUser, err := user.Current()
	if err != nil {
		return fmt.Errorf("failed to get current user: %w", err)
	}
	uid := currentUser.Uid
	gid := currentUser.Gid

	// Build docker command for base image
	args := []string{
		"build",
		"--build-arg", fmt.Sprintf("NODE_VERSION=%s", p.config.NodeVersion),
		"--build-arg", fmt.Sprintf("GO_VERSION=%s", p.config.GoVersion),
		"--build-arg", fmt.Sprintf("UV_VERSION=%s", p.config.UvVersion),
		"--build-arg", fmt.Sprintf("USER_ID=%s", uid),
		"--build-arg", fmt.Sprintf("GROUP_ID=%s", gid),
		"--build-arg", "USERNAME=addt",
		"-t", baseImageName,
		"-f", dockerfilePath,
		buildDir,
	}

	cmd := exec.Command("docker", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to build base Docker image: %w", err)
	}

	fmt.Printf("✓ Base image built: %s\n\n", baseImageName)
	return nil
}

// EnsureBaseImage checks if base image exists and builds it if needed
func (p *DockerProvider) EnsureBaseImage(forceRebuild bool) error {
	baseImageName := p.GetBaseImageName()

	if forceRebuild || !p.ImageExists(baseImageName) {
		return p.BuildBaseImage()
	}

	fmt.Printf("✓ Using cached base image: %s\n", baseImageName)
	return nil
}

// BuildImage builds the Docker image (extension layer on top of base)
func (p *DockerProvider) BuildImage(embeddedDockerfile, embeddedEntrypoint []byte) error {
	// First ensure base image exists
	if err := p.EnsureBaseImage(false); err != nil {
		return fmt.Errorf("failed to ensure base image: %w", err)
	}

	baseImageName := p.GetBaseImageName()
	fmt.Printf("Building %s (from %s)...\n", p.config.ImageName, baseImageName)

	// Create temp directory for build context with embedded files
	buildDir, err := os.MkdirTemp("", "addt-build-*")
	if err != nil {
		return fmt.Errorf("failed to create temp build directory: %w", err)
	}
	defer os.RemoveAll(buildDir)

	// Write embedded Dockerfile
	dockerfilePath := filepath.Join(buildDir, "Dockerfile")
	if err := os.WriteFile(dockerfilePath, embeddedDockerfile, 0644); err != nil {
		return fmt.Errorf("failed to write Dockerfile: %w", err)
	}

	// Write embedded entrypoint script
	entrypointPath := filepath.Join(buildDir, "docker-entrypoint.sh")
	if err := os.WriteFile(entrypointPath, embeddedEntrypoint, 0755); err != nil {
		return fmt.Errorf("failed to write docker-entrypoint.sh: %w", err)
	}

	// Write embedded firewall init script
	initFirewallPath := filepath.Join(buildDir, "init-firewall.sh")
	if err := os.WriteFile(initFirewallPath, p.embeddedInitFirewall, 0755); err != nil {
		return fmt.Errorf("failed to write init-firewall.sh: %w", err)
	}

	// Write install.sh to build directory
	installShPath := filepath.Join(buildDir, "install.sh")
	if err := os.WriteFile(installShPath, p.embeddedInstallSh, 0755); err != nil {
		return fmt.Errorf("failed to write install.sh: %w", err)
	}

	// Write embedded extensions (preserving directory structure)
	extensionsDir := filepath.Join(buildDir, "extensions")
	if err := os.MkdirAll(extensionsDir, 0755); err != nil {
		return fmt.Errorf("failed to create extensions directory: %w", err)
	}
	err = fs.WalkDir(p.embeddedExtensions, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		// Skip the root directory and Go source files
		if path == "." || path == "embed.go" || path == "go.mod" {
			return nil
		}
		destPath := filepath.Join(extensionsDir, path)

		if d.IsDir() {
			return os.MkdirAll(destPath, 0755)
		}

		content, err := p.embeddedExtensions.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(destPath, content, 0755)
	})
	if err != nil {
		return fmt.Errorf("failed to write extensions: %w", err)
	}

	scriptDir := buildDir

	// Build EXTENSION_VERSIONS string from map (e.g., "claude:stable,codex:latest")
	var versionPairs []string
	for extName, version := range p.config.ExtensionVersions {
		versionPairs = append(versionPairs, fmt.Sprintf("%s:%s", extName, version))
	}
	extensionVersions := strings.Join(versionPairs, ",")

	// Build docker command - use base image and only pass extension args
	args := []string{
		"build",
		"--build-arg", fmt.Sprintf("BASE_IMAGE=%s", baseImageName),
		"--build-arg", fmt.Sprintf("ADDT_EXTENSIONS=%s", p.config.Extensions),
		"--build-arg", fmt.Sprintf("EXTENSION_VERSIONS=%s", extensionVersions),
		"-t", p.config.ImageName,
		"-f", dockerfilePath,
		scriptDir,
	}

	cmd := exec.Command("docker", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to build Docker image: %w", err)
	}

	fmt.Println("\n✓ Image built successfully!")
	fmt.Println()
	fmt.Println("Detecting tool versions...")

	// Get versions from the built image
	versions := p.detectToolVersions(p.config.ImageName)

	// Add version labels to image
	p.addVersionLabels(p.config, versions)

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
	fmt.Printf("Image tagged as: %s\n", p.config.ImageName)

	return nil
}

func (p *DockerProvider) detectToolVersions(imageName string) map[string]string {
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

func (p *DockerProvider) addVersionLabels(cfg interface{}, versions map[string]string) {
	// Get ImageName from config (using p.config directly)
	imageName := p.config.ImageName
	if imageName == "" {
		return
	}

	// Create temporary Dockerfile
	tmpFile, err := os.CreateTemp("", "Dockerfile-labels-*")
	if err != nil {
		return
	}
	defer os.Remove(tmpFile.Name())

	content := fmt.Sprintf("FROM %s\n", imageName)
	for tool, version := range versions {
		if version != "" {
			content += fmt.Sprintf("LABEL tools.%s.version=\"%s\"\n", tool, version)
		}
	}
	tmpFile.WriteString(content)
	tmpFile.Close()

	// Build with labels
	cmd := exec.Command("docker", "build", "-f", tmpFile.Name(), "-t", imageName, ".")
	if err := cmd.Run(); err != nil {
		fmt.Printf("Warning: failed to add version labels: %v\n", err)
	}

	// Tag as addt:latest if this is latest
	claudeVersion := p.getExtensionVersion("claude")
	if claudeVersion == "latest" {
		if err := exec.Command("docker", "tag", imageName, "addt:latest").Run(); err != nil {
			fmt.Printf("Warning: failed to tag as addt:latest: %v\n", err)
		}
	}

	// Tag with claude version
	if v, ok := versions["claude"]; ok && v != "" {
		if err := exec.Command("docker", "tag", imageName, fmt.Sprintf("addt:claude-%s", v)).Run(); err != nil {
			fmt.Printf("Warning: failed to tag with claude version: %v\n", err)
		}
	}
}
