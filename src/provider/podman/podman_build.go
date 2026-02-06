package podman

import (
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"

	"github.com/jedi4ever/addt/util"
)

// BuildIfNeeded ensures the Podman image is ready
func (p *PodmanProvider) BuildIfNeeded(rebuild bool, rebuildBase bool) error {
	// Handle --addt-rebuild-base flag - rebuild base image first
	if rebuildBase {
		baseImageName := p.GetBaseImageName()
		fmt.Printf("Rebuilding base image %s...\n", baseImageName)
		if p.ImageExists(baseImageName) {
			cmd := exec.Command("podman", "rmi", baseImageName)
			cmd.Run()
		}
		if err := p.BuildBaseImage(); err != nil {
			return err
		}
	}

	logger := util.Log("podman-build")
	logger.Debugf("Checking image: %s", p.config.ImageName)
	imageExists := p.ImageExists(p.config.ImageName)
	logger.Debugf("Image exists: %v", imageExists)

	// Handle --addt-rebuild flag
	if rebuild {
		if imageExists {
			fmt.Printf("Rebuilding %s...\n", p.config.ImageName)
			fmt.Println("Removing existing image...")
			cmd := exec.Command("podman", "rmi", p.config.ImageName)
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

// DetermineImageName determines the appropriate Podman image name based on installed extensions
func (p *PodmanProvider) DetermineImageName() string {
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

	// Handle base image case (no extensions)
	if len(validExts) == 0 {
		return fmt.Sprintf("addt:v%s_base-%s", p.config.AddtVersion, p.assetsHash())
	}

	// Check if all extensions have explicit versions (not dist-tags)
	// If so, we can skip npm lookups and check for exact image match
	allExplicitVersions := true
	for _, ext := range validExts {
		version := p.getExtensionVersion(ext)
		if version == "latest" || version == "stable" || version == "next" {
			allExplicitVersions = false
			break
		}
	}

	// Build image name with resolved versions
	var tagParts []string
	for _, ext := range validExts {
		var version string
		if allExplicitVersions {
			// Use explicit version directly (no npm lookup needed)
			version = p.getExtensionVersion(ext)
		} else {
			// Resolve version (may do npm lookup for dist-tags)
			version = p.resolveExtensionVersion(ext)
		}
		tagParts = append(tagParts, fmt.Sprintf("%s-%s", ext, version))
	}

	// Join with underscore
	tag := strings.Join(tagParts, "_")

	// Prefix with addt version, base hash, and extension hash so images are rebuilt when assets change
	baseHash := p.assetsHash()
	extHash := p.extAssetsHash()
	imageName := fmt.Sprintf("addt:v%s_%s-%s-%s", p.config.AddtVersion, tag, baseHash, extHash)
	logger := util.Log("podman-build")
	logger.Debugf("assetsHash=%s extAssetsHash=%s imageName=%s", baseHash, extHash, imageName)
	return imageName
}

// resolveExtensionVersion resolves the version for an extension, handling dist-tags
func (p *PodmanProvider) resolveExtensionVersion(extName string) string {
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
func (p *PodmanProvider) getExtensionVersion(extName string) string {
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
func (p *PodmanProvider) setExtensionVersion(extName, version string) {
	if p.config.ExtensionVersions == nil {
		p.config.ExtensionVersions = make(map[string]string)
	}
	p.config.ExtensionVersions[extName] = version
}
