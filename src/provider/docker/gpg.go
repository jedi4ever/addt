package docker

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/jedi4ever/addt/config/security"
	"github.com/jedi4ever/addt/util"
)

// HandleGPGForwarding configures GPG forwarding based on mode.
// Modes:
//   - "proxy": Forward gpg-agent socket through filtering proxy (most secure)
//   - "agent": Forward gpg-agent socket directly
//   - "keys" or "true": Mount ~/.gnupg read-only (legacy, less secure)
//   - "" or "off" or "false": No GPG forwarding
//
// If allowedKeyIDs is set, proxy mode is automatically enabled
func (p *DockerProvider) HandleGPGForwarding(gpgForward, homeDir, username string, allowedKeyIDs []string) []string {
	var args []string

	// Normalize boolean-like values
	gpgForward = strings.ToLower(strings.TrimSpace(gpgForward))

	// If disabled, return empty
	if gpgForward == "" || gpgForward == "off" || gpgForward == "false" || gpgForward == "none" {
		return args
	}

	// If allowed key IDs are specified, use proxy mode
	if len(allowedKeyIDs) > 0 && (gpgForward == "agent" || gpgForward == "true" || gpgForward == "proxy") {
		return p.handleGPGProxyForwarding(homeDir, username, allowedKeyIDs)
	}

	switch gpgForward {
	case "proxy":
		return p.handleGPGProxyForwarding(homeDir, username, nil)
	case "agent":
		return p.handleGPGAgentForwarding(homeDir, username)
	case "keys", "true":
		return p.handleGPGKeysForwarding(homeDir, username)
	default:
		// Unknown mode, treat as disabled
		return args
	}
}

// handleGPGProxyForwarding creates a filtering GPG agent proxy
func (p *DockerProvider) handleGPGProxyForwarding(homeDir, username string, allowedKeyIDs []string) []string {
	var args []string

	agentSocket := getGPGAgentSocket(homeDir)
	if agentSocket == "" {
		fmt.Println("Warning: GPG agent socket not found, cannot create GPG proxy")
		return args
	}

	// Create the proxy
	proxy, err := security.NewGPGProxyAgent(agentSocket, allowedKeyIDs)
	if err != nil {
		fmt.Printf("Warning: failed to create GPG proxy: %v\n", err)
		return args
	}

	// Start the proxy
	if err := proxy.Start(); err != nil {
		fmt.Printf("Warning: failed to start GPG proxy: %v\n", err)
		return args
	}

	// Store proxy for cleanup
	p.gpgProxy = proxy

	// Mount the proxy socket directory
	proxyDir := proxy.SocketDir()
	args = append(args, "-v", fmt.Sprintf("%s:/home/%s/.gnupg/S.gpg-agent", proxy.SocketPath(), username))

	// Mount safe GPG files only (public keys, config)
	args = append(args, p.mountSafeGPGFiles(homeDir, username)...)

	// Set GPG_TTY for interactive operations
	args = append(args, "-e", "GPG_TTY=/dev/console")

	if len(allowedKeyIDs) > 0 {
		fmt.Printf("GPG proxy active: only keys matching %v are accessible\n", allowedKeyIDs)
	} else {
		fmt.Printf("GPG proxy active: all keys accessible (socket: %s)\n", proxyDir)
	}

	return args
}

// handleGPGAgentForwarding forwards the gpg-agent socket directly
func (p *DockerProvider) handleGPGAgentForwarding(homeDir, username string) []string {
	var args []string

	agentSocket := getGPGAgentSocket(homeDir)
	if agentSocket == "" {
		fmt.Println("Warning: GPG agent socket not found")
		return args
	}

	// Mount the agent socket
	args = append(args, "-v", fmt.Sprintf("%s:/home/%s/.gnupg/S.gpg-agent", agentSocket, username))

	// Mount safe GPG files only
	args = append(args, p.mountSafeGPGFiles(homeDir, username)...)

	// Set GPG_TTY
	args = append(args, "-e", "GPG_TTY=/dev/console")

	fmt.Println("GPG agent forwarding active")

	return args
}

// handleGPGKeysForwarding mounts ~/.gnupg read-only (legacy mode)
func (p *DockerProvider) handleGPGKeysForwarding(homeDir, username string) []string {
	var args []string

	gnupgDir := filepath.Join(homeDir, ".gnupg")
	if _, err := os.Stat(gnupgDir); err != nil {
		return args
	}

	// Mount entire directory read-only
	args = append(args, "-v", fmt.Sprintf("%s:/home/%s/.gnupg:ro", gnupgDir, username))

	// Set GPG_TTY
	args = append(args, "-e", "GPG_TTY=/dev/console")

	return args
}

// mountSafeGPGFiles creates a temp directory with only safe GPG files
// and returns mount arguments
func (p *DockerProvider) mountSafeGPGFiles(homeDir, username string) []string {
	var args []string

	gnupgDir := filepath.Join(homeDir, ".gnupg")
	if _, err := os.Stat(gnupgDir); err != nil {
		return args
	}

	tmpDir, err := os.MkdirTemp("", "gpg-safe-*")
	if err != nil {
		return args
	}

	p.tempDirs = append(p.tempDirs, tmpDir)

	// Copy safe files only (no private keys)
	safeFiles := []string{
		"pubring.kbx",      // Public keyring (GPG 2.1+)
		"pubring.gpg",      // Public keyring (legacy)
		"trustdb.gpg",      // Trust database
		"gpg.conf",         // GPG configuration
		"gpg-agent.conf",   // Agent configuration
		"dirmngr.conf",     // Directory manager config
		"sshcontrol",       // SSH control file
		"tofu.db",          // TOFU database
		"crls.d",           // CRL cache (directory)
	}

	for _, file := range safeFiles {
		src := filepath.Join(gnupgDir, file)
		dst := filepath.Join(tmpDir, file)

		info, err := os.Stat(src)
		if err != nil {
			continue
		}

		if info.IsDir() {
			// Copy directory
			util.SafeCopyDir(src, dst)
		} else {
			util.SafeCopyFile(src, dst)
		}
	}

	// Mount the safe directory
	args = append(args, "-v", fmt.Sprintf("%s:/home/%s/.gnupg:ro", tmpDir, username))

	return args
}

// getGPGAgentSocket returns the path to the gpg-agent socket
func getGPGAgentSocket(homeDir string) string {
	// Try gpgconf first (most reliable)
	cmd := exec.Command("gpgconf", "--list-dirs", "agent-socket")
	output, err := cmd.Output()
	if err == nil {
		socket := strings.TrimSpace(string(output))
		if _, err := os.Stat(socket); err == nil {
			return socket
		}
	}

	// Fall back to standard locations
	standardPaths := []string{
		filepath.Join(homeDir, ".gnupg", "S.gpg-agent"),
		"/run/user/" + fmt.Sprint(os.Getuid()) + "/gnupg/S.gpg-agent",
	}

	for _, path := range standardPaths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	return ""
}
