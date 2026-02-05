package podman

import (
	"crypto/md5"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/jedi4ever/addt/provider"
)

// Container lifecycle management for persistent and ephemeral containers

// Exists checks if a container exists (running or stopped)
func (p *PodmanProvider) Exists(name string) bool {
	cmd := exec.Command(GetPodmanPath(), "ps", "-a", "--filter", fmt.Sprintf("name=^%s$", name), "--format", "{{.Names}}")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(output)) == name
}

// IsRunning checks if a container is currently running
func (p *PodmanProvider) IsRunning(name string) bool {
	cmd := exec.Command(GetPodmanPath(), "ps", "--filter", fmt.Sprintf("name=^%s$", name), "--format", "{{.Names}}")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(output)) == name
}

// Start starts a stopped container
func (p *PodmanProvider) Start(name string) error {
	cmd := exec.Command(GetPodmanPath(), "start", name)
	return cmd.Run()
}

// Stop stops a running container
func (p *PodmanProvider) Stop(name string) error {
	cmd := exec.Command(GetPodmanPath(), "stop", name)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Remove removes a container
func (p *PodmanProvider) Remove(name string) error {
	cmd := exec.Command(GetPodmanPath(), "rm", "-f", name)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// List lists all persistent addt containers
func (p *PodmanProvider) List() ([]provider.Environment, error) {
	cmd := exec.Command(GetPodmanPath(), "ps", "-a", "--filter", "name=^addt-persistent-",
		"--format", "{{.Names}}\t{{.Status}}\t{{.CreatedAt}}")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var envs []provider.Environment
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) >= 3 {
			envs = append(envs, provider.Environment{
				Name:      parts[0],
				Status:    parts[1],
				CreatedAt: parts[2],
			})
		}
	}
	return envs, nil
}

// GenerateContainerName generates a persistent container name based on working directory and extensions
// The name format is: addt-persistent-<dirname>-<hash>
// The hash is based on workdir + extensions to ensure:
// - Same workdir + same extensions = same container
// - Same workdir + different extensions = different container
func (p *PodmanProvider) GenerateContainerName() string {
	// Use configured workdir or fall back to current directory
	workdir := p.config.Workdir
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

	// Sanitize directory name (lowercase, remove special chars, max 20 chars)
	re := regexp.MustCompile(`[^a-z0-9-]+`)
	dirname = strings.ToLower(dirname)
	dirname = re.ReplaceAllString(dirname, "-")
	dirname = strings.Trim(dirname, "-")
	if len(dirname) > 20 {
		dirname = dirname[:20]
	}

	// Get sorted extensions for consistent naming
	extensions := strings.Split(p.config.Extensions, ",")
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

	// Create hash of workdir + extensions for uniqueness
	// Same workdir + same extensions = same container
	// Same workdir + different extensions = different container
	hashInput := workdir + "|" + extStr
	hash := md5.Sum([]byte(hashInput))
	hashStr := fmt.Sprintf("%x", hash)[:8]

	return fmt.Sprintf("addt-persistent-%s-%s", dirname, hashStr)
}

// GenerateEphemeralName generates a unique ephemeral container name
// The name format is: addt-<timestamp>-<pid>
func (p *PodmanProvider) GenerateEphemeralName() string {
	return fmt.Sprintf("addt-%s-%d", time.Now().Format("20060102-150405"), os.Getpid())
}

// GeneratePersistentName is an alias for GenerateContainerName to implement Provider interface
func (p *PodmanProvider) GeneratePersistentName() string {
	return p.GenerateContainerName()
}

// IsPersistentContainer checks if a container name matches the persistent naming pattern
func IsPersistentContainer(name string) bool {
	return strings.HasPrefix(name, "addt-persistent-")
}

// IsEphemeralContainer checks if a container name matches the ephemeral naming pattern
func IsEphemeralContainer(name string) bool {
	return strings.HasPrefix(name, "addt-") && !strings.HasPrefix(name, "addt-persistent-")
}

// GetContainerWorkdir extracts the workdir hint from a persistent container name
// Returns the sanitized directory name portion of the container name
func GetContainerWorkdir(name string) string {
	if !IsPersistentContainer(name) {
		return ""
	}
	// Format: addt-persistent-<dirname>-<hash>
	// Remove prefix and hash
	trimmed := strings.TrimPrefix(name, "addt-persistent-")
	if idx := strings.LastIndex(trimmed, "-"); idx > 0 {
		return trimmed[:idx]
	}
	return trimmed
}
