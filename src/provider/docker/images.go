package docker

import (
	"fmt"
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

// BuildImage builds the Docker image
func (p *DockerProvider) BuildImage(embeddedDockerfile, embeddedEntrypoint []byte) error {
	fmt.Printf("Building %s...\n", p.config.ImageName)

	// Create temp directory for build context with embedded files
	buildDir, err := os.MkdirTemp("", "dclaude-build-*")
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

	scriptDir := buildDir

	// Get current user info
	currentUser, _ := user.Current()
	uid := currentUser.Uid
	gid := currentUser.Gid
	username := "claude" // Always use "claude" in container, but with host UID/GID

	// Build docker command
	args := []string{
		"build",
		"--build-arg", fmt.Sprintf("NODE_VERSION=%s", p.config.NodeVersion),
		"--build-arg", fmt.Sprintf("USER_ID=%s", uid),
		"--build-arg", fmt.Sprintf("GROUP_ID=%s", gid),
		"--build-arg", fmt.Sprintf("USERNAME=%s", username),
		"--build-arg", fmt.Sprintf("CLAUDE_VERSION=%s", p.config.ClaudeVersion),
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
	cmd.Run()

	// Tag as dclaude:latest if this is latest
	if p.config.ClaudeVersion == "latest" {
		exec.Command("docker", "tag", imageName, "dclaude:latest").Run()
	}

	// Tag with claude version
	if v, ok := versions["claude"]; ok && v != "" {
		exec.Command("docker", "tag", imageName, fmt.Sprintf("dclaude:claude-%s", v)).Run()
	}
}

// Config is a local copy to avoid circular import
type Config struct {
	ClaudeVersion string
	NodeVersion   string
	ImageName     string
}
