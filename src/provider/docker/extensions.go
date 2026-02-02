package docker

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ExtensionMount represents a mount point for an extension
type ExtensionMount struct {
	Source string `json:"source"`
	Target string `json:"target"`
}

// ExtensionFlag represents a CLI flag for an extension
type ExtensionFlag struct {
	Flag        string `json:"flag"`
	Description string `json:"description"`
}

// ExtensionMetadata represents metadata for an installed extension
type ExtensionMetadata struct {
	Name        string           `json:"name"`
	Description string           `json:"description"`
	Entrypoint  string           `json:"entrypoint"`
	Mounts      []ExtensionMount `json:"mounts"`
	Flags       []ExtensionFlag  `json:"flags"`
}

// ExtensionsConfig represents the extensions.json file structure
type ExtensionsConfig struct {
	Extensions map[string]ExtensionMetadata `json:"extensions"`
}

// ExtensionMountWithName includes the extension name for mount filtering
type ExtensionMountWithName struct {
	ExtensionMount
	ExtensionName string
}

// GetExtensionMounts reads extension metadata from image and returns all mounts
func (p *DockerProvider) GetExtensionMounts(imageName string) []ExtensionMount {
	mountsWithNames := p.GetExtensionMountsWithNames(imageName)
	var mounts []ExtensionMount
	for _, m := range mountsWithNames {
		mounts = append(mounts, m.ExtensionMount)
	}
	return mounts
}

// GetExtensionMountsWithNames reads extension metadata and returns mounts with extension names
func (p *DockerProvider) GetExtensionMountsWithNames(imageName string) []ExtensionMountWithName {
	var mounts []ExtensionMountWithName

	// Read extensions.json from the image
	cmd := exec.Command("docker", "run", "--rm", "--entrypoint", "cat", imageName,
		"/home/claude/.dclaude/extensions.json")
	output, err := cmd.Output()
	if err != nil {
		// No extensions metadata or file doesn't exist
		return mounts
	}

	// Parse the JSON
	var config ExtensionsConfig
	if err := json.Unmarshal(output, &config); err != nil {
		return mounts
	}

	// Collect all mounts from all extensions, with extension name
	for extName, ext := range config.Extensions {
		for _, mount := range ext.Mounts {
			mounts = append(mounts, ExtensionMountWithName{
				ExtensionMount: mount,
				ExtensionName:  extName,
			})
		}
	}

	return mounts
}

// AddExtensionMounts adds extension mount volumes to docker args
func (p *DockerProvider) AddExtensionMounts(dockerArgs []string, imageName, homeDir string) []string {
	extMounts := p.GetExtensionMountsWithNames(imageName)
	for _, extMount := range extMounts {
		// Check per-extension mount config (defaults to true if not specified)
		if p.config.MountExtensionConfig != nil {
			if mountEnabled, exists := p.config.MountExtensionConfig[extMount.ExtensionName]; exists && !mountEnabled {
				// Mount explicitly disabled for this extension
				continue
			}
		}

		// Expand ~ to home directory
		source := extMount.Source
		if strings.HasPrefix(source, "~/") {
			source = filepath.Join(homeDir, source[2:])
		}

		// Check if source exists, create if it's a directory path
		if info, err := os.Stat(source); err == nil {
			// Source exists (file or directory)
			dockerArgs = append(dockerArgs, "-v", source+":"+extMount.Target)
		} else if os.IsNotExist(err) {
			// Source doesn't exist - create directory if path doesn't look like a file
			if !strings.Contains(filepath.Base(source), ".") {
				// Looks like a directory (no extension)
				if err := os.MkdirAll(source, 0755); err == nil {
					dockerArgs = append(dockerArgs, "-v", source+":"+extMount.Target)
				}
			}
			// Skip files that don't exist (e.g., ~/.claude.json on fresh install)
		} else if info != nil {
			dockerArgs = append(dockerArgs, "-v", source+":"+extMount.Target)
		}
	}
	return dockerArgs
}

// GetExtensionMetadata reads all extension metadata from the image
func (p *DockerProvider) GetExtensionMetadata(imageName string) map[string]ExtensionMetadata {
	// Read extensions.json from the image
	cmd := exec.Command("docker", "run", "--rm", "--entrypoint", "cat", imageName,
		"/home/claude/.dclaude/extensions.json")
	output, err := cmd.Output()
	if err != nil {
		return nil
	}

	var config ExtensionsConfig
	if err := json.Unmarshal(output, &config); err != nil {
		return nil
	}

	return config.Extensions
}

// GetExtensionFlags returns flags for a specific extension by entrypoint command
func (p *DockerProvider) GetExtensionFlags(imageName, command string) []ExtensionFlag {
	metadata := p.GetExtensionMetadata(imageName)
	if metadata == nil {
		return nil
	}

	// Find extension by entrypoint
	for _, ext := range metadata {
		if ext.Entrypoint == command {
			return ext.Flags
		}
	}

	return nil
}
