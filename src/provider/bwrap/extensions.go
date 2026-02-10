package bwrap

import (
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/jedi4ever/addt/extensions"
)

// GetExtensionEnvVars returns environment variable names needed by active extensions.
// Unlike Docker/Podman, bwrap reads extension configs directly from the embedded FS
// since there are no container images to read metadata from.
func (b *BwrapProvider) GetExtensionEnvVars(imageName string) []string {
	activeExts := parseExtensionList(b.config.Extensions)
	if len(activeExts) == 0 {
		return nil
	}

	allExts, err := extensions.GetExtensions()
	if err != nil {
		return nil
	}

	envVarSet := make(map[string]bool)
	for _, ext := range allExts {
		if !isActiveExtension(ext.Name, activeExts) {
			continue
		}
		for _, v := range ext.EnvVars {
			envVarSet[v] = true
		}
		for _, v := range ext.OtelVars {
			envVarSet[v] = true
		}
	}

	var result []string
	for v := range envVarSet {
		result = append(result, v)
	}
	return result
}

// addExtensionMounts adds extension config directory mounts to bwrap args.
// Reads mount definitions from extension configs and applies automount/readonly settings.
func (b *BwrapProvider) addExtensionMounts(args []string) []string {
	activeExts := parseExtensionList(b.config.Extensions)
	if len(activeExts) == 0 {
		return args
	}

	allExts, err := extensions.GetExtensions()
	if err != nil {
		return args
	}

	currentUser, err := user.Current()
	if err != nil {
		return args
	}
	homeDir := currentUser.HomeDir

	for _, ext := range allExts {
		if !isActiveExtension(ext.Name, activeExts) {
			continue
		}

		autoMount := ext.Config.Automount

		// Check user config overrides
		if b.config.ExtensionConfigAutomount != nil {
			if enabled, exists := b.config.ExtensionConfigAutomount[ext.Name]; exists {
				if !enabled {
					continue
				}
				// Explicitly enabled — proceed even if extension default is false
			} else if !autoMount {
				continue
			}
		} else if !autoMount {
			continue
		}

		// Determine readonly
		// Precedence: per-extension user config > global config > extension default
		readonly := ext.Config.Readonly
		if b.config.ConfigReadonly {
			readonly = true
		}
		if b.config.ExtensionConfigReadonly != nil {
			if ro, exists := b.config.ExtensionConfigReadonly[ext.Name]; exists {
				readonly = ro
			}
		}

		for _, mount := range ext.Config.Mounts {
			source := mount.Source
			if strings.HasPrefix(source, "~/") {
				source = filepath.Join(homeDir, source[2:])
			}

			if _, err := os.Stat(source); err == nil {
				if readonly {
					args = append(args, "--ro-bind", source, mount.Target)
				} else {
					args = append(args, "--bind", source, mount.Target)
				}
			} else if os.IsNotExist(err) {
				// Create directory if path doesn't look like a file
				if !strings.Contains(filepath.Base(source), ".") {
					if err := os.MkdirAll(source, 0755); err == nil {
						if readonly {
							args = append(args, "--ro-bind", source, mount.Target)
						} else {
							args = append(args, "--bind", source, mount.Target)
						}
					}
				}
			}
		}
	}

	return args
}

func parseExtensionList(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

func isActiveExtension(name string, active []string) bool {
	for _, a := range active {
		if a == name {
			return true
		}
	}
	return false
}
