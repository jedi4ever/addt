package bwrap

import (
	"crypto/md5"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/jedi4ever/addt/provider"
)

// Bwrap sandboxes are ephemeral processes, not persistent containers.
// The methods below implement the Provider interface but persistent
// container lifecycle (Start/Stop/restart) is not supported.

// Exists always returns false — bwrap sandboxes are ephemeral processes
func (b *BwrapProvider) Exists(name string) bool {
	return false
}

// IsRunning always returns false — bwrap sandboxes are ephemeral processes
func (b *BwrapProvider) IsRunning(name string) bool {
	return false
}

// Start is not supported — bwrap sandboxes cannot be restarted
func (b *BwrapProvider) Start(name string) error {
	return fmt.Errorf("bwrap provider does not support persistent containers (start is not available)")
}

// Stop is not supported — bwrap sandboxes are ephemeral
func (b *BwrapProvider) Stop(name string) error {
	return fmt.Errorf("bwrap provider does not support persistent containers (stop is not available)")
}

// Remove is a no-op — nothing to remove for ephemeral sandboxes
func (b *BwrapProvider) Remove(name string) error {
	return nil
}

// List returns an empty list — bwrap does not track persistent environments
func (b *BwrapProvider) List() ([]provider.Environment, error) {
	return nil, nil
}

// GeneratePersistentName generates a name consistent with the project directory.
// While bwrap doesn't support persistent containers, the name is still used
// for status display and logging.
func (b *BwrapProvider) GeneratePersistentName() string {
	return b.generateName("addt-bwrap-persistent")
}

// GenerateEphemeralName generates a unique ephemeral sandbox name
func (b *BwrapProvider) GenerateEphemeralName() string {
	return fmt.Sprintf("addt-bwrap-%s-%d", time.Now().Format("20060102-150405"), os.Getpid())
}

// generateName creates a name based on the working directory and extensions
func (b *BwrapProvider) generateName(prefix string) string {
	workdir := b.config.Workdir
	if workdir == "" {
		var err error
		workdir, err = os.Getwd()
		if err != nil {
			workdir = "/tmp"
		}
	}

	// Get directory name
	dirname := workdir
	if idx := strings.LastIndex(workdir, "/"); idx != -1 {
		dirname = workdir[idx+1:]
	}

	// Sanitize
	re := regexp.MustCompile(`[^a-z0-9-]+`)
	dirname = strings.ToLower(dirname)
	dirname = re.ReplaceAllString(dirname, "-")
	dirname = strings.Trim(dirname, "-")
	if len(dirname) > 20 {
		dirname = dirname[:20]
	}

	// Extension hash for uniqueness
	extensions := strings.Split(b.config.Extensions, ",")
	for i := range extensions {
		extensions[i] = strings.TrimSpace(extensions[i])
	}
	var validExts []string
	for _, ext := range extensions {
		if ext != "" {
			validExts = append(validExts, ext)
		}
	}
	sort.Strings(validExts)
	extStr := strings.Join(validExts, ",")

	hashInput := workdir + "|" + extStr
	hash := md5.Sum([]byte(hashInput))
	hashStr := fmt.Sprintf("%x", hash)[:8]

	return fmt.Sprintf("%s-%s-%s", prefix, dirname, hashStr)
}
