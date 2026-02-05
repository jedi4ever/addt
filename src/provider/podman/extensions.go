package podman

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/jedi4ever/addt/extensions"
)

// GetExtensionMounts reads extension metadata from image and returns all mounts
func (p *PodmanProvider) GetExtensionMounts(imageName string) []extensions.ExtensionMount {
	mountsWithNames := p.GetExtensionMountsWithNames(imageName)
	var mounts []extensions.ExtensionMount
	for _, m := range mountsWithNames {
		mounts = append(mounts, extensions.ExtensionMount{
			Source: m.Source,
			Target: m.Target,
		})
	}
	return mounts
}

// GetExtensionMountsWithNames reads extension metadata and returns mounts with extension names
func (p *PodmanProvider) GetExtensionMountsWithNames(imageName string) []extensions.ExtensionMountWithName {
	var mounts []extensions.ExtensionMountWithName

	// Read extensions.json from the image
	cmd := exec.Command("podman", "run", "--rm", "--entrypoint", "cat", imageName,
		"/home/addt/.addt/extensions.json")
	output, err := cmd.Output()
	if err != nil {
		// Extension metadata not available - this is normal for images without extensions
		// or when the extensions.json file doesn't exist yet. Not an error condition.
		return mounts
	}

	// Parse the JSON
	var config extensions.ExtensionsJSONConfig
	if err := json.Unmarshal(output, &config); err != nil {
		return mounts
	}

	// Collect all mounts from all extensions, with extension name and auto_mount
	for extName, ext := range config.Extensions {
		for _, mount := range ext.Mounts {
			mounts = append(mounts, extensions.ExtensionMountWithName{
				Source:        mount.Source,
				Target:        mount.Target,
				ExtensionName: extName,
				AutoMount:     ext.AutoMount, // from extension level
			})
		}
	}

	return mounts
}

// AddExtensionMounts adds extension mount volumes to podman args
func (p *PodmanProvider) AddExtensionMounts(podmanArgs []string, imageName, homeDir string) []string {
	extMounts := p.GetExtensionMountsWithNames(imageName)
	for _, extMount := range extMounts {
		// Determine if mount should be enabled based on auto_mount and explicit config
		// Default is false - extensions must explicitly set auto_mount: true
		autoMount := extMount.AutoMount != nil && *extMount.AutoMount

		if p.config.ExtensionAutomount != nil {
			if mountEnabled, exists := p.config.ExtensionAutomount[extMount.ExtensionName]; exists {
				if !mountEnabled {
					// Mount explicitly disabled by user config
					continue
				}
				// Mount explicitly enabled by user config - proceed even if auto_mount is false
			} else if !autoMount {
				// No user config and auto_mount not enabled in extension - skip
				continue
			}
		} else if !autoMount {
			// No user config and auto_mount not enabled in extension - skip
			continue
		}

		// Expand ~ to home directory
		source := extMount.Source
		if strings.HasPrefix(source, "~/") {
			source = filepath.Join(homeDir, source[2:])
		}

		// Check if source exists, create if it's a directory path
		if info, err := os.Stat(source); err == nil {
			// Source exists (file or directory)
			podmanArgs = append(podmanArgs, "-v", source+":"+extMount.Target)
		} else if os.IsNotExist(err) {
			// Source doesn't exist - create directory if path doesn't look like a file
			if !strings.Contains(filepath.Base(source), ".") {
				// Looks like a directory (no extension)
				if err := os.MkdirAll(source, 0755); err == nil {
					podmanArgs = append(podmanArgs, "-v", source+":"+extMount.Target)
				}
			}
			// Skip files that don't exist (e.g., ~/.claude.json on fresh install)
		} else if info != nil {
			podmanArgs = append(podmanArgs, "-v", source+":"+extMount.Target)
		}
	}
	return podmanArgs
}

// GetExtensionMetadata reads all extension metadata from the image
func (p *PodmanProvider) GetExtensionMetadata(imageName string) map[string]extensions.ExtensionMetadata {
	// Read extensions.json from the image
	cmd := exec.Command("podman", "run", "--rm", "--entrypoint", "cat", imageName,
		"/home/addt/.addt/extensions.json")
	output, err := cmd.Output()
	if err != nil {
		// Extension metadata not available - this is normal for images without extensions
		// or when the extensions.json file doesn't exist yet. Not an error condition.
		return nil
	}

	var config extensions.ExtensionsJSONConfig
	if err := json.Unmarshal(output, &config); err != nil {
		return nil
	}

	return config.Extensions
}

// GetExtensionFlags returns flags for a specific extension by entrypoint command
func (p *PodmanProvider) GetExtensionFlags(imageName, command string) []extensions.ExtensionFlag {
	metadata := p.GetExtensionMetadata(imageName)
	if metadata == nil {
		return nil
	}

	// Find extension by entrypoint command (first element of entrypoint array)
	for _, ext := range metadata {
		if ext.Entrypoint.Command() == command {
			return ext.Flags
		}
	}

	return nil
}

// GetExtensionEnvVars returns all unique environment variables needed by installed extensions
func (p *PodmanProvider) GetExtensionEnvVars(imageName string) []string {
	metadata := p.GetExtensionMetadata(imageName)
	if metadata == nil {
		return nil
	}

	// Use a map to deduplicate env vars
	envVarSet := make(map[string]bool)
	for _, ext := range metadata {
		for _, envVar := range ext.EnvVars {
			envVarSet[envVar] = true
		}
	}

	// Convert to slice
	var envVars []string
	for envVar := range envVarSet {
		envVars = append(envVars, envVar)
	}

	return envVars
}
