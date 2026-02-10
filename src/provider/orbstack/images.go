package orbstack

import (
	"crypto/sha256"
	"fmt"
	"hash"
	"io/fs"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	profilecmd "github.com/jedi4ever/addt/cmd/profile"
	"github.com/jedi4ever/addt/extensions"
	"github.com/jedi4ever/addt/util"
)

// ImageExists checks if a Docker image exists
func (p *OrbStackProvider) ImageExists(imageName string) bool {
	cmd := p.dockerCmd("image", "inspect", imageName)
	return cmd.Run() == nil
}

// FindImageByLabel finds an image by a specific label value
func (p *OrbStackProvider) FindImageByLabel(label, value string) string {
	cmd := p.dockerCmd("images",
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
func (p *OrbStackProvider) GetImageLabel(imageName, label string) string {
	cmd := p.dockerCmd("inspect",
		"--format", fmt.Sprintf("{{index .Config.Labels %q}}", label),
		imageName)
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

// assetsHash returns a short hash of the base image assets (Dockerfile.base, entrypoint, firewall)
// Used in base image tags so that changes to these files trigger a base rebuild
func (p *OrbStackProvider) assetsHash() string {
	h := sha256.New()
	h.Write(p.embeddedDockerfileBase)
	h.Write(p.embeddedEntrypoint)
	h.Write(p.embeddedInitFirewall)
	return fmt.Sprintf("%x", h.Sum(nil))[:8]
}

// hashDir walks a filesystem directory and hashes each file's relative path and content.
// Non-existent or empty directories are silently skipped.
func hashDir(h hash.Hash, dir string, logger *util.ModuleLogger, fileCount *int, totalBytes *int) {
	if dir == "" {
		return
	}
	filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		relPath, _ := filepath.Rel(dir, path)
		h.Write([]byte(relPath))
		h.Write(content)
		*fileCount++
		*totalBytes += len(content)
		logger.Debugf("  hashing: %s (%d bytes)", relPath, len(content))
		return nil
	})
}

// extAssetsHash returns a short hash of the extension layer assets
// (Dockerfile, install.sh, extensions/) so changes trigger an extension image rebuild
func (p *OrbStackProvider) extAssetsHash() string {
	logger := util.Log("orbstack-hash")
	h := sha256.New()
	h.Write(p.embeddedDockerfile)
	h.Write(p.embeddedInstallSh)
	fileCount := 0
	totalBytes := 0
	fs.WalkDir(p.embeddedExtensions, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		content, err := p.embeddedExtensions.ReadFile(path)
		if err != nil {
			return err
		}
		h.Write([]byte(path))
		h.Write(content)
		fileCount++
		totalBytes += len(content)
		logger.Debugf("  hashing: %s (%d bytes)", path, len(content))
		return nil
	})

	// Hash local extensions (~/.addt/extensions/) so changes trigger rebuild
	hashDir(h, extensions.GetLocalExtensionsDir(), logger, &fileCount, &totalBytes)

	// Hash extra extensions (ADDT_EXTENSIONS_DIR) so changes trigger rebuild
	hashDir(h, extensions.GetExtraExtensionsDir(), logger, &fileCount, &totalBytes)

	// Hash profile presets so changes trigger rebuild
	presetsFS := profilecmd.GetPresetsFS()
	fs.WalkDir(presetsFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		content, err := presetsFS.ReadFile(path)
		if err != nil {
			return err
		}
		h.Write([]byte(path))
		h.Write(content)
		fileCount++
		totalBytes += len(content)
		logger.Debugf("  hashing: %s (%d bytes)", path, len(content))
		return nil
	})

	hash := fmt.Sprintf("%x", h.Sum(nil))[:8]
	logger.Debugf("extAssetsHash: %d files, %d bytes total -> %s", fileCount, totalBytes, hash)
	return hash
}

// GetBaseImageName returns the base image name for the current config
func (p *OrbStackProvider) GetBaseImageName() string {
	currentUser, err := user.Current()
	if err != nil {
		return "addt-base:latest"
	}
	return fmt.Sprintf("addt-base:v%s-node%s-go%s-uv%s-uid%s-%s",
		p.config.AddtVersion, p.config.NodeVersion, p.config.GoVersion, p.config.UvVersion, currentUser.Uid, p.assetsHash())
}
