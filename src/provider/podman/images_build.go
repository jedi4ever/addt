package podman

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/jedi4ever/addt/extensions"
	"github.com/jedi4ever/addt/util"
)

// BuildBaseImage builds the base Podman image (contains Node, Go, UV, system packages)
func (p *PodmanProvider) BuildBaseImage() error {
	baseImageName := p.GetBaseImageName()
	startTime := time.Now()

	util.PrintBuildStart(baseImageName)
	util.PrintInfo("This may take a few minutes on first build...")

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
	entrypointPath := filepath.Join(buildDir, "podman-entrypoint.sh")
	if err := os.WriteFile(entrypointPath, p.embeddedEntrypoint, 0755); err != nil {
		return fmt.Errorf("failed to write podman-entrypoint.sh: %w", err)
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

	// Build podman command for base image
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

	// Run build with progress indication
	if err := util.RunBuildCommand("podman", args); err != nil {
		util.PrintError(fmt.Sprintf("Failed to build base image: %v", err))
		return fmt.Errorf("failed to build base Podman image: %w", err)
	}

	elapsed := time.Since(startTime)
	util.PrintBuildComplete(baseImageName, elapsed)
	fmt.Println()
	return nil
}

// EnsureBaseImage checks if base image exists and builds it if needed
func (p *PodmanProvider) EnsureBaseImage(forceRebuild bool) error {
	baseImageName := p.GetBaseImageName()

	if forceRebuild || !p.ImageExists(baseImageName) {
		return p.BuildBaseImage()
	}

	util.PrintCacheHit(baseImageName)
	return nil
}

// BuildImage builds the Podman image (extension layer on top of base)
func (p *PodmanProvider) BuildImage(embeddedDockerfile, embeddedEntrypoint []byte) error {
	// First ensure base image exists
	if err := p.EnsureBaseImage(false); err != nil {
		return fmt.Errorf("failed to ensure base image: %w", err)
	}

	baseImageName := p.GetBaseImageName()
	startTime := time.Now()

	util.PrintBuildStart(p.config.ImageName)
	util.PrintInfo(fmt.Sprintf("Building from base: %s", baseImageName))

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
	entrypointPath := filepath.Join(buildDir, "podman-entrypoint.sh")
	if err := os.WriteFile(entrypointPath, embeddedEntrypoint, 0755); err != nil {
		return fmt.Errorf("failed to write podman-entrypoint.sh: %w", err)
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

	// Copy local extensions (override embedded ones with same name)
	localExtsDir := extensions.GetLocalExtensionsDir()
	if localExtsDir != "" {
		if _, err := os.Stat(localExtsDir); err == nil {
			if err := p.copyLocalExtensions(localExtsDir, extensionsDir); err != nil {
				fmt.Printf("Warning: failed to copy local extensions: %v\n", err)
			}
		}
	}

	scriptDir := buildDir

	// Build EXTENSION_VERSIONS string from map (e.g., "claude:stable,codex:latest")
	var versionPairs []string
	for extName, version := range p.config.ExtensionVersions {
		versionPairs = append(versionPairs, fmt.Sprintf("%s:%s", extName, version))
	}
	extensionVersions := strings.Join(versionPairs, ",")

	// Build podman command - use base image and only pass extension args
	args := []string{"build"}

	// Add --no-cache if requested
	if p.config.NoCache {
		args = append(args, "--no-cache")
	}

	args = append(args,
		"--build-arg", fmt.Sprintf("BASE_IMAGE=%s", baseImageName),
		"--build-arg", fmt.Sprintf("ADDT_EXTENSIONS=%s", p.config.Extensions),
		"--build-arg", fmt.Sprintf("EXTENSION_VERSIONS=%s", extensionVersions),
		"-t", p.config.ImageName,
		"-f", dockerfilePath,
		scriptDir,
	)

	// Run build with progress indication
	if err := util.RunBuildCommand("podman", args); err != nil {
		util.PrintError(fmt.Sprintf("Failed to build image: %v", err))
		return fmt.Errorf("failed to build Podman image: %w", err)
	}

	elapsed := time.Since(startTime)
	util.PrintBuildComplete(p.config.ImageName, elapsed)
	fmt.Println()
	util.PrintInfo("Detecting tool versions...")

	// Get versions from the built image
	versions := p.detectToolVersions(p.config.ImageName)

	// Add version labels to image
	p.addVersionLabels(p.config, versions)

	fmt.Println()
	fmt.Println("Installed versions:")
	if v, ok := versions["node"]; ok && v != "" {
		fmt.Printf("  Node.js:     %s\n", v)
	}
	if v, ok := versions["claude"]; ok && v != "" {
		fmt.Printf("  Claude Code: %s\n", v)
	}
	if v, ok := versions["gh"]; ok && v != "" {
		fmt.Printf("  GitHub CLI:  %s\n", v)
	}
	if v, ok := versions["rg"]; ok && v != "" {
		fmt.Printf("  Ripgrep:     %s\n", v)
	}
	if v, ok := versions["git"]; ok && v != "" {
		fmt.Printf("  Git:         %s\n", v)
	}
	fmt.Println()
	fmt.Printf("Image tagged as: %s\n", p.config.ImageName)

	return nil
}

func (p *PodmanProvider) detectToolVersions(imageName string) map[string]string {
	versions := make(map[string]string)
	versionRegex := regexp.MustCompile(`[0-9]+\.[0-9]+\.[0-9]+`)

	tools := map[string][]string{
		"claude": {"claude", "--version"},
		"gh":     {"gh", "--version"},
		"rg":     {"rg", "--version"},
		"git":    {"git", "--version"},
		"node":   {"node", "--version"},
	}

	spinner := util.NewSpinner("Detecting versions...")
	spinner.Start()

	for name, cmdArgs := range tools {
		spinner.UpdateMessage(fmt.Sprintf("Detecting %s version...", name))
		args := append([]string{"run", "--rm", "--entrypoint", cmdArgs[0], imageName}, cmdArgs[1:]...)
		cmd := exec.Command("podman", args...)
		output, err := cmd.Output()
		if err == nil {
			if match := versionRegex.FindString(string(output)); match != "" {
				versions[name] = match
			}
		}
	}

	spinner.Stop()
	return versions
}

func (p *PodmanProvider) addVersionLabels(cfg interface{}, versions map[string]string) {
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
	cmd := exec.Command("podman", "build", "-f", tmpFile.Name(), "-t", imageName, ".")
	if err := cmd.Run(); err != nil {
		fmt.Printf("Warning: failed to add version labels: %v\n", err)
	}

	// Tag as addt:latest if this is latest
	claudeVersion := p.getExtensionVersion("claude")
	if claudeVersion == "latest" {
		if err := exec.Command("podman", "tag", imageName, "addt:latest").Run(); err != nil {
			fmt.Printf("Warning: failed to tag as addt:latest: %v\n", err)
		}
	}

	// Tag with claude version
	if v, ok := versions["claude"]; ok && v != "" {
		if err := exec.Command("podman", "tag", imageName, fmt.Sprintf("addt:claude-%s", v)).Run(); err != nil {
			fmt.Printf("Warning: failed to tag with claude version: %v\n", err)
		}
	}
}

// copyLocalExtensions copies local extensions to the build directory, overwriting embedded ones
func (p *PodmanProvider) copyLocalExtensions(srcDir, destDir string) error {
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		extName := entry.Name()
		srcExtDir := filepath.Join(srcDir, extName)
		destExtDir := filepath.Join(destDir, extName)

		// Check if this extension has a config.yaml (valid extension)
		if _, err := os.Stat(filepath.Join(srcExtDir, "config.yaml")); err != nil {
			continue // Skip directories without config.yaml
		}

		// Remove existing extension directory if it exists (to fully replace)
		os.RemoveAll(destExtDir)

		// Copy the entire extension directory
		if err := copyDir(srcExtDir, destExtDir); err != nil {
			return fmt.Errorf("failed to copy extension %s: %w", extName, err)
		}

		fmt.Printf("  Including local extension: %s\n", extName)
	}

	return nil
}

// copyDir recursively copies a directory
func copyDir(src, dst string) error {
	if err := os.MkdirAll(dst, 0755); err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			if err := copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}

	return nil
}

// copyFile copies a single file
func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	// Get source file info for permissions
	srcInfo, err := srcFile.Stat()
	if err != nil {
		return err
	}

	dstFile, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, srcInfo.Mode())
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}
