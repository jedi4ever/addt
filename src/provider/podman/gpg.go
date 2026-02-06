package podman

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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
func (p *PodmanProvider) HandleGPGForwarding(gpgForward, homeDir, username string, allowedKeyIDs []string) []string {
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
func (p *PodmanProvider) handleGPGProxyForwarding(homeDir, username string, allowedKeyIDs []string) []string {
	var args []string

	agentSocket := getGPGAgentSocket(homeDir)
	if agentSocket == "" {
		fmt.Println("Warning: GPG agent socket not found, cannot create GPG proxy")
		return args
	}

	// On macOS, podman runs in a VM and can't mount Unix sockets via virtiofs.
	// Use TCP mode: proxy listens on TCP, container connects via socat.
	if runtime.GOOS == "darwin" {
		return p.handleGPGProxyForwardingTCP(agentSocket, homeDir, username, allowedKeyIDs)
	}

	// Linux: use Unix socket (can be mounted directly)
	proxy, err := security.NewGPGProxyAgent(agentSocket, allowedKeyIDs)
	if err != nil {
		fmt.Printf("Warning: failed to create GPG proxy: %v\n", err)
		return args
	}

	if err := proxy.Start(); err != nil {
		fmt.Printf("Warning: failed to start GPG proxy: %v\n", err)
		return args
	}

	p.gpgProxy = proxy

	proxyDir := proxy.SocketDir()
	args = append(args, "-v", fmt.Sprintf("%s:/home/%s/.gnupg/S.gpg-agent", proxy.SocketPath(), username))
	args = append(args, p.mountSafeGPGFiles(homeDir, username)...)
	args = append(args, "-e", "GPG_TTY=/dev/console")

	if len(allowedKeyIDs) > 0 {
		fmt.Printf("GPG proxy active: only keys matching %v are accessible\n", allowedKeyIDs)
	} else {
		fmt.Printf("GPG proxy active: all keys accessible (socket: %s)\n", proxyDir)
	}

	return args
}

// handleGPGProxyForwardingTCP creates a TCP-based GPG agent proxy for macOS.
// The proxy listens on a TCP port on the host; the container connects via socat.
func (p *PodmanProvider) handleGPGProxyForwardingTCP(agentSocket, homeDir, username string, allowedKeyIDs []string) []string {
	var args []string

	proxy, err := security.NewGPGProxyAgentTCP(agentSocket, allowedKeyIDs)
	if err != nil {
		fmt.Printf("Warning: failed to create GPG TCP proxy: %v\n", err)
		return args
	}

	if err := proxy.Start(); err != nil {
		fmt.Printf("Warning: failed to start GPG TCP proxy: %v\n", err)
		return args
	}

	p.gpgProxy = proxy

	hostIP, err := getHostGatewayIP()
	if err != nil {
		fmt.Printf("Warning: could not detect host IP for GPG proxy: %v\n", err)
		proxy.Stop()
		return args
	}

	// Pass TCP connection info — entrypoint uses socat to bridge TCP→Unix socket
	args = append(args, "-e", fmt.Sprintf("ADDT_GPG_PROXY_HOST=%s", hostIP))
	args = append(args, "-e", fmt.Sprintf("ADDT_GPG_PROXY_PORT=%d", proxy.TCPPort()))

	// Mount safe GPG files writable (socat needs to create S.gpg-agent socket inside)
	args = append(args, p.mountSafeGPGFilesWritable(homeDir, username)...)

	args = append(args, "-e", "GPG_TTY=/dev/console")

	if len(allowedKeyIDs) > 0 {
		fmt.Printf("GPG proxy active (TCP): only keys matching %v are accessible\n", allowedKeyIDs)
	} else {
		fmt.Println("GPG proxy active (TCP): all keys accessible")
	}

	return args
}

// handleGPGAgentForwarding forwards the gpg-agent socket directly
func (p *PodmanProvider) handleGPGAgentForwarding(homeDir, username string) []string {
	var args []string

	agentSocket := getGPGAgentSocket(homeDir)
	if agentSocket == "" {
		fmt.Println("Warning: GPG agent socket not found")
		return args
	}

	// macOS: can't mount Unix sockets into podman containers
	if runtime.GOOS == "darwin" {
		fmt.Println("Warning: GPG agent forwarding not supported on macOS (use ADDT_GPG_FORWARD=proxy)")
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
func (p *PodmanProvider) handleGPGKeysForwarding(homeDir, username string) []string {
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
func (p *PodmanProvider) mountSafeGPGFiles(homeDir, username string) []string {
	var args []string

	gnupgDir := filepath.Join(homeDir, ".gnupg")
	if _, err := os.Stat(gnupgDir); err != nil {
		return args
	}

	tmpDir, err := os.MkdirTemp("", "gpg-safe-*")
	if err != nil {
		return args
	}

	// Set restrictive permissions and write PID file
	if err := os.Chmod(tmpDir, 0700); err != nil {
		os.RemoveAll(tmpDir)
		return args
	}
	if err := security.WritePIDFile(tmpDir); err != nil {
		os.RemoveAll(tmpDir)
		return args
	}

	p.tempDirs = append(p.tempDirs, tmpDir)

	// Copy safe files only (no private keys)
	safeFiles := []string{
		"pubring.kbx",    // Public keyring (GPG 2.1+)
		"pubring.gpg",    // Public keyring (legacy)
		"trustdb.gpg",    // Trust database
		"gpg.conf",       // GPG configuration
		"gpg-agent.conf", // Agent configuration
		"dirmngr.conf",   // Directory manager config
		"sshcontrol",     // SSH control file
		"tofu.db",        // TOFU database
		"crls.d",         // CRL cache (directory)
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

// mountSafeGPGFilesWritable is like mountSafeGPGFiles but mounts writable.
// Needed for TCP mode where socat creates the S.gpg-agent socket inside the directory.
func (p *PodmanProvider) mountSafeGPGFilesWritable(homeDir, username string) []string {
	var args []string

	gnupgDir := filepath.Join(homeDir, ".gnupg")
	if _, err := os.Stat(gnupgDir); err != nil {
		return args
	}

	tmpDir, err := os.MkdirTemp("", "gpg-safe-*")
	if err != nil {
		return args
	}

	if err := os.Chmod(tmpDir, 0700); err != nil {
		os.RemoveAll(tmpDir)
		return args
	}
	if err := security.WritePIDFile(tmpDir); err != nil {
		os.RemoveAll(tmpDir)
		return args
	}

	p.tempDirs = append(p.tempDirs, tmpDir)

	safeFiles := []string{
		"pubring.kbx", "pubring.gpg", "trustdb.gpg",
		"gpg.conf", "gpg-agent.conf", "dirmngr.conf",
		"sshcontrol", "tofu.db", "crls.d",
	}

	for _, file := range safeFiles {
		src := filepath.Join(gnupgDir, file)
		dst := filepath.Join(tmpDir, file)
		info, err := os.Stat(src)
		if err != nil {
			continue
		}
		if info.IsDir() {
			util.SafeCopyDir(src, dst)
		} else {
			util.SafeCopyFile(src, dst)
		}
	}

	// Mount writable (no :ro) so socat can create S.gpg-agent socket
	args = append(args, "-v", fmt.Sprintf("%s:/home/%s/.gnupg", tmpDir, username))

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
