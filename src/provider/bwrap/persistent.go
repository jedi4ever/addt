package bwrap

import (
	"crypto/md5"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/jedi4ever/addt/provider"
	"github.com/jedi4ever/addt/util"
)

// Persistent bwrap sessions work by:
//  1. Starting a bwrap sandbox with "sleep infinity" as the command
//  2. Saving the PID to ~/.addt/bwrap/pids/<name>.pid
//  3. Using nsenter to re-enter the same namespaces for subsequent commands
//  4. Killing the sleep process to stop the session

// getPidDir returns the directory for storing PID files
func getPidDir() string {
	addtHome := util.GetAddtHome()
	if addtHome == "" {
		return filepath.Join(os.TempDir(), "addt-bwrap-pids")
	}
	return filepath.Join(addtHome, "bwrap", "pids")
}

// getPidFile returns the PID file path for a named session
func getPidFile(name string) string {
	return filepath.Join(getPidDir(), name+".pid")
}

// readPid reads a PID from a file, returns 0 if not found or invalid
func readPid(name string) int {
	data, err := os.ReadFile(getPidFile(name))
	if err != nil {
		return 0
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0
	}
	return pid
}

// writePid stores a PID to file
func writePid(name string, pid int) error {
	dir := getPidDir()
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	return os.WriteFile(getPidFile(name), []byte(strconv.Itoa(pid)), 0600)
}

// removePid removes a PID file
func removePid(name string) {
	os.Remove(getPidFile(name))
}

// isProcessAlive checks whether a PID is still running
func isProcessAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	// Signal 0 checks process existence without actually signaling
	err := syscall.Kill(pid, 0)
	return err == nil
}

// Exists checks if a persistent session exists and is alive
func (b *BwrapProvider) Exists(name string) bool {
	pid := readPid(name)
	if pid == 0 {
		return false
	}
	if !isProcessAlive(pid) {
		// Stale PID file — clean up
		removePid(name)
		return false
	}
	return true
}

// IsRunning checks if a persistent session is currently running
func (b *BwrapProvider) IsRunning(name string) bool {
	return b.Exists(name) // For bwrap, exists == running (no stopped state)
}

// Start is not supported — bwrap sessions cannot be restarted once dead.
// The namespaces are destroyed when the sleep process exits.
func (b *BwrapProvider) Start(name string) error {
	return fmt.Errorf("bwrap sessions cannot be restarted (namespaces are gone); remove and re-run instead")
}

// Stop kills the persistent session's sleep process
func (b *BwrapProvider) Stop(name string) error {
	pid := readPid(name)
	if pid == 0 {
		return nil
	}
	if isProcessAlive(pid) {
		// Kill the process group to clean up children too
		syscall.Kill(-pid, syscall.SIGTERM)
		// Give it a moment, then force kill
		time.Sleep(500 * time.Millisecond)
		if isProcessAlive(pid) {
			syscall.Kill(-pid, syscall.SIGKILL)
		}
	}
	removePid(name)
	return nil
}

// Remove stops and removes a persistent session
func (b *BwrapProvider) Remove(name string) error {
	return b.Stop(name)
}

// List returns all active persistent bwrap sessions by scanning PID files
func (b *BwrapProvider) List() ([]provider.Environment, error) {
	dir := getPidDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var envs []provider.Environment
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".pid") {
			continue
		}
		name := strings.TrimSuffix(entry.Name(), ".pid")
		pid := readPid(name)

		status := "stopped"
		if pid > 0 && isProcessAlive(pid) {
			status = "running"
		} else {
			// Stale PID file
			removePid(name)
			continue
		}

		info, _ := entry.Info()
		created := ""
		if info != nil {
			created = info.ModTime().Format("2006-01-02 15:04:05")
		}

		envs = append(envs, provider.Environment{
			Name:      name,
			Status:    status,
			CreatedAt: created,
		})
	}
	return envs, nil
}

// GeneratePersistentName generates a name based on working directory + extensions
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

	dirname := workdir
	if idx := strings.LastIndex(workdir, "/"); idx != -1 {
		dirname = workdir[idx+1:]
	}

	re := regexp.MustCompile(`[^a-z0-9-]+`)
	dirname = strings.ToLower(dirname)
	dirname = re.ReplaceAllString(dirname, "-")
	dirname = strings.Trim(dirname, "-")
	if len(dirname) > 20 {
		dirname = dirname[:20]
	}

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
