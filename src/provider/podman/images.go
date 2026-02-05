package podman

import (
	"fmt"
	"os/exec"
	"os/user"
	"strings"
)

// ImageExists checks if a Podman image exists
func (p *PodmanProvider) ImageExists(imageName string) bool {
	cmd := exec.Command("podman", "image", "inspect", imageName)
	return cmd.Run() == nil
}

// FindImageByLabel finds an image by a specific label value
func (p *PodmanProvider) FindImageByLabel(label, value string) string {
	cmd := exec.Command("podman", "images",
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
func (p *PodmanProvider) GetImageLabel(imageName, label string) string {
	cmd := exec.Command("podman", "inspect",
		"--format", fmt.Sprintf("{{index .Config.Labels %q}}", label),
		imageName)
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

// GetBaseImageName returns the base image name for the current config
func (p *PodmanProvider) GetBaseImageName() string {
	// Get current user info for UID/GID in tag
	currentUser, err := user.Current()
	if err != nil {
		return "addt-base:latest"
	}
	// Base image is tagged with addt version, tool versions, and UID to ensure compatibility
	// When any of these change, a new base image will be built
	// Note: Using same image names as Docker for cache sharing when possible
	return fmt.Sprintf("addt-base:v%s-node%s-go%s-uv%s-uid%s",
		p.config.AddtVersion, p.config.NodeVersion, p.config.GoVersion, p.config.UvVersion, currentUser.Uid)
}
