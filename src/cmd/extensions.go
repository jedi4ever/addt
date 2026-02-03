package cmd

import (
	"fmt"
	"io/fs"
	"os"
	"sort"
	"strings"

	"github.com/jedi4ever/addt/extensions"
	"gopkg.in/yaml.v3"
)

// ExtensionConfig represents the config.yaml structure
type ExtensionConfig struct {
	Name           string   `yaml:"name"`
	Description    string   `yaml:"description"`
	Entrypoint     string   `yaml:"entrypoint"`
	DefaultVersion string   `yaml:"default_version"`
	AutoMount      bool     `yaml:"auto_mount"`
	Dependencies   []string `yaml:"dependencies"`
}

// PrintVersion prints addt version and loaded extension version
func PrintVersion(version, defaultNodeVersion, defaultGoVersion, defaultUvVersion string) {
	fmt.Printf("addt %s\n", version)
	fmt.Println()

	// Default tool versions
	fmt.Println("Tools:")
	fmt.Printf("  Node.js:  %s\n", defaultNodeVersion)
	fmt.Printf("  Go:       %s\n", defaultGoVersion)
	fmt.Printf("  UV:       %s\n", defaultUvVersion)
	fmt.Println()

	// Get loaded extension (from env or binary name)
	extName := os.Getenv("ADDT_EXTENSIONS")
	if extName == "" {
		extName = os.Getenv("ADDT_COMMAND")
	}
	// No default - extension must be explicitly set via symlink or env
	// Take first extension if comma-separated
	if idx := strings.Index(extName, ","); idx != -1 {
		extName = extName[:idx]
	}

	// Get version for this extension
	extVersion := os.Getenv("ADDT_" + strings.ToUpper(extName) + "_VERSION")
	if extVersion == "" {
		// Look up default version from config
		exts, err := getExtensions()
		if err == nil {
			for _, ext := range exts {
				if ext.Name == extName {
					extVersion = ext.DefaultVersion
					break
				}
			}
		}
	}
	if extVersion == "" {
		extVersion = "latest"
	}

	fmt.Println("Extension:")
	fmt.Printf("  %-16s %s\n", extName, extVersion)
}

// ListExtensions prints all available extensions
func ListExtensions() {
	exts, err := getExtensions()
	if err != nil {
		fmt.Printf("Error reading extensions: %v\n", err)
		return
	}

	// Find max lengths for alignment
	maxName := 4   // "Name"
	maxEntry := 10 // "Entrypoint"
	maxVer := 7    // "Version"
	for _, ext := range exts {
		if len(ext.Name) > maxName {
			maxName = len(ext.Name)
		}
		if len(ext.Entrypoint) > maxEntry {
			maxEntry = len(ext.Entrypoint)
		}
		ver := ext.DefaultVersion
		if ver == "" {
			ver = "latest"
		}
		if len(ver) > maxVer {
			maxVer = len(ver)
		}
	}

	// Print header
	fmt.Printf("  #  %-*s  %-*s  %-*s  %s\n", maxName, "Name", maxEntry, "Entrypoint", maxVer, "Version", "Description")
	fmt.Printf("  -  %-*s  %-*s  %-*s  %s\n", maxName, strings.Repeat("-", maxName), maxEntry, strings.Repeat("-", maxEntry), maxVer, strings.Repeat("-", maxVer), "-----------")

	// Print rows
	for i, ext := range exts {
		version := ext.DefaultVersion
		if version == "" {
			version = "latest"
		}
		fmt.Printf("%3d  %-*s  %-*s  %-*s  %s\n", i+1, maxName, ext.Name, maxEntry, ext.Entrypoint, maxVer, version, ext.Description)
	}

	fmt.Println()
	fmt.Println("Usage: ln -s /usr/local/bin/addt ~/bin/<entrypoint>")
}

// GetEntrypointForExtension returns the entrypoint command for a given extension name
// If extension not found, returns the extension name itself as fallback
func GetEntrypointForExtension(extName string) string {
	exts, err := getExtensions()
	if err != nil {
		return extName
	}

	for _, ext := range exts {
		if ext.Name == extName {
			if ext.Entrypoint != "" {
				return ext.Entrypoint
			}
			return extName
		}
	}

	return extName
}

// getExtensions reads all extension configs from embedded filesystem
func getExtensions() ([]ExtensionConfig, error) {
	var configs []ExtensionConfig

	entries, err := fs.ReadDir(extensions.FS, ".")
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		configPath := entry.Name() + "/config.yaml"
		data, err := extensions.FS.ReadFile(configPath)
		if err != nil {
			continue // Skip directories without config.yaml
		}

		var cfg ExtensionConfig
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			continue // Skip invalid configs
		}

		configs = append(configs, cfg)
	}

	// Sort by name
	sort.Slice(configs, func(i, j int) bool {
		return configs[i].Name < configs[j].Name
	})

	return configs, nil
}
