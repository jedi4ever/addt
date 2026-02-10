package bwrap

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/jedi4ever/addt/config/security"
	"github.com/jedi4ever/addt/provider"
	"github.com/jedi4ever/addt/util"
)

// handleSSHForwarding configures SSH forwarding for bwrap.
// Since bwrap runs on the same host, SSH sockets can be bound directly —
// no TCP proxy or socat bridge needed (unlike Docker on macOS).
func (b *BwrapProvider) handleSSHForwarding(spec *provider.RunSpec, sshDir string) []string {
	if !spec.SSHForwardKeys {
		return nil
	}

	var args []string

	// If allowed keys are specified with agent/proxy mode, use filtering proxy
	if len(spec.SSHAllowedKeys) > 0 && (spec.SSHForwardMode == "agent" || spec.SSHForwardMode == "proxy") {
		return b.handleSSHProxyForwarding(sshDir, spec.SSHAllowedKeys)
	}

	switch spec.SSHForwardMode {
	case "proxy":
		return b.handleSSHProxyForwarding(sshDir, nil)
	case "agent":
		return b.handleSSHAgentForwarding(sshDir)
	case "keys":
		return b.handleSSHKeysForwarding(sshDir)
	}

	return args
}

// handleSSHProxyForwarding creates a filtering SSH agent proxy
func (b *BwrapProvider) handleSSHProxyForwarding(sshDir string, allowedKeys []string) []string {
	var args []string

	sshAuthSock := os.Getenv("SSH_AUTH_SOCK")
	if sshAuthSock == "" {
		fmt.Println("Warning: SSH_AUTH_SOCK not set, cannot create SSH proxy")
		return args
	}

	proxy, err := security.NewSSHProxyAgent(sshAuthSock, allowedKeys)
	if err != nil {
		fmt.Printf("Warning: failed to create SSH proxy: %v\n", err)
		return args
	}

	if err := proxy.Start(); err != nil {
		fmt.Printf("Warning: failed to start SSH proxy: %v\n", err)
		return args
	}

	b.sshProxy = proxy
	proxySocket := proxy.SocketPath()
	proxyDir := filepath.Dir(proxySocket)

	// Bind the proxy socket directory and set SSH_AUTH_SOCK
	args = append(args, "--bind", proxyDir, proxyDir)
	args = append(args, "--setenv", "SSH_AUTH_SOCK", proxySocket)
	args = append(args, b.mountSafeSSHFiles(sshDir)...)

	if len(allowedKeys) > 0 {
		fmt.Printf("SSH proxy active: only keys matching %v are accessible\n", allowedKeys)
	} else {
		fmt.Println("SSH proxy active: all keys accessible")
	}

	return args
}

// handleSSHAgentForwarding binds the SSH agent socket directly
func (b *BwrapProvider) handleSSHAgentForwarding(sshDir string) []string {
	var args []string

	sshAuthSock := os.Getenv("SSH_AUTH_SOCK")
	if sshAuthSock == "" {
		return args
	}

	if _, err := os.Stat(sshAuthSock); err != nil {
		return args
	}

	// Bind the SSH agent socket and set env var
	sockDir := filepath.Dir(sshAuthSock)
	args = append(args, "--bind", sockDir, sockDir)
	args = append(args, "--setenv", "SSH_AUTH_SOCK", sshAuthSock)
	args = append(args, b.mountSafeSSHFiles(sshDir)...)

	return args
}

// handleSSHKeysForwarding mounts the SSH directory read-only
func (b *BwrapProvider) handleSSHKeysForwarding(sshDir string) []string {
	var args []string
	if _, err := os.Stat(sshDir); err == nil {
		args = append(args, "--ro-bind", sshDir, "/home/addt/.ssh")
	}
	return args
}

// mountSafeSSHFiles creates a temp directory with only safe SSH files
func (b *BwrapProvider) mountSafeSSHFiles(sshDir string) []string {
	var args []string

	if _, err := os.Stat(sshDir); err != nil {
		return args
	}

	tmpDir, err := os.MkdirTemp("", "bwrap-ssh-safe-*")
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
	b.tempDirs = append(b.tempDirs, tmpDir)

	// Copy safe files only
	util.SafeCopyFile(filepath.Join(sshDir, "config"), filepath.Join(tmpDir, "config"))
	util.SafeCopyFile(filepath.Join(sshDir, "known_hosts"), filepath.Join(tmpDir, "known_hosts"))

	files, _ := filepath.Glob(filepath.Join(sshDir, "*.pub"))
	for _, f := range files {
		util.SafeCopyFile(f, filepath.Join(tmpDir, filepath.Base(f)))
	}

	args = append(args, "--ro-bind", tmpDir, "/home/addt/.ssh")
	return args
}

// handleGPGForwarding configures GPG forwarding for bwrap.
// Direct socket binding works on Linux without TCP proxies.
func (b *BwrapProvider) handleGPGForwarding(spec *provider.RunSpec, gpgDir string) []string {
	gpgForward := strings.ToLower(strings.TrimSpace(spec.GPGForward))

	if gpgForward == "" || gpgForward == "off" || gpgForward == "false" || gpgForward == "none" {
		return nil
	}

	// If allowed key IDs are specified, use proxy mode
	if len(spec.GPGAllowedKeyIDs) > 0 && (gpgForward == "agent" || gpgForward == "true" || gpgForward == "proxy") {
		return b.handleGPGProxyForwarding(gpgDir, spec.GPGAllowedKeyIDs)
	}

	switch gpgForward {
	case "proxy":
		return b.handleGPGProxyForwarding(gpgDir, nil)
	case "agent":
		return b.handleGPGAgentForwarding(gpgDir)
	case "keys", "true":
		return b.handleGPGKeysForwarding(gpgDir)
	}

	return nil
}

// handleGPGProxyForwarding creates a filtering GPG agent proxy
func (b *BwrapProvider) handleGPGProxyForwarding(gpgDir string, allowedKeyIDs []string) []string {
	var args []string

	agentSocket := getGPGAgentSocketPath(gpgDir)
	if agentSocket == "" {
		fmt.Println("Warning: GPG agent socket not found, cannot create GPG proxy")
		return args
	}

	proxy, err := security.NewGPGProxyAgent(agentSocket, allowedKeyIDs)
	if err != nil {
		fmt.Printf("Warning: failed to create GPG proxy: %v\n", err)
		return args
	}

	if err := proxy.Start(); err != nil {
		fmt.Printf("Warning: failed to start GPG proxy: %v\n", err)
		return args
	}

	b.gpgProxy = proxy

	// Bind the proxy socket
	args = append(args, "--bind", proxy.SocketPath(), "/home/addt/.gnupg/S.gpg-agent")
	args = append(args, b.mountSafeGPGFiles(gpgDir)...)
	args = append(args, "--setenv", "GPG_TTY", "/dev/console")

	if len(allowedKeyIDs) > 0 {
		fmt.Printf("GPG proxy active: only keys matching %v are accessible\n", allowedKeyIDs)
	} else {
		fmt.Printf("GPG proxy active: all keys accessible (socket: %s)\n", proxy.SocketDir())
	}

	return args
}

// handleGPGAgentForwarding binds the GPG agent socket directly
func (b *BwrapProvider) handleGPGAgentForwarding(gpgDir string) []string {
	var args []string

	agentSocket := getGPGAgentSocketPath(gpgDir)
	if agentSocket == "" {
		fmt.Println("Warning: GPG agent socket not found")
		return args
	}

	args = append(args, "--bind", agentSocket, "/home/addt/.gnupg/S.gpg-agent")
	args = append(args, b.mountSafeGPGFiles(gpgDir)...)
	args = append(args, "--setenv", "GPG_TTY", "/dev/console")

	fmt.Println("GPG agent forwarding active")
	return args
}

// handleGPGKeysForwarding mounts GPG directory read-only
func (b *BwrapProvider) handleGPGKeysForwarding(gpgDir string) []string {
	var args []string
	if _, err := os.Stat(gpgDir); err != nil {
		return args
	}
	args = append(args, "--ro-bind", gpgDir, "/home/addt/.gnupg")
	args = append(args, "--setenv", "GPG_TTY", "/dev/console")
	return args
}

// mountSafeGPGFiles creates a temp directory with only safe GPG files
func (b *BwrapProvider) mountSafeGPGFiles(gpgDir string) []string {
	var args []string

	if _, err := os.Stat(gpgDir); err != nil {
		return args
	}

	tmpDir, err := os.MkdirTemp("", "bwrap-gpg-safe-*")
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
	b.tempDirs = append(b.tempDirs, tmpDir)

	safeFiles := []string{
		"pubring.kbx", "pubring.gpg", "trustdb.gpg",
		"gpg.conf", "gpg-agent.conf", "dirmngr.conf",
		"sshcontrol", "tofu.db",
	}

	for _, file := range safeFiles {
		src := filepath.Join(gpgDir, file)
		dst := filepath.Join(tmpDir, file)
		if _, err := os.Stat(src); err == nil {
			util.SafeCopyFile(src, dst)
		}
	}

	args = append(args, "--ro-bind", tmpDir, "/home/addt/.gnupg")
	return args
}

// getGPGAgentSocketPath returns the path to the gpg-agent socket
func getGPGAgentSocketPath(gpgDir string) string {
	// Try gpgconf first (most reliable)
	cmd := exec.Command("gpgconf", "--list-dirs", "agent-socket")
	if output, err := cmd.Output(); err == nil {
		socket := strings.TrimSpace(string(output))
		if _, err := os.Stat(socket); err == nil {
			return socket
		}
	}

	// Fall back to standard locations
	standardPaths := []string{
		filepath.Join(gpgDir, "S.gpg-agent"),
		fmt.Sprintf("/run/user/%d/gnupg/S.gpg-agent", os.Getuid()),
	}

	for _, path := range standardPaths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	return ""
}

// handleTmuxForwarding binds the tmux socket into the sandbox.
// On Linux, the socket can be mounted directly — no TCP bridge needed.
func (b *BwrapProvider) handleTmuxForwarding(spec *provider.RunSpec) []string {
	if !spec.TmuxForward {
		return nil
	}

	tmuxEnv := os.Getenv("TMUX")
	if tmuxEnv == "" {
		return nil
	}

	parts := strings.Split(tmuxEnv, ",")
	if len(parts) < 1 {
		return nil
	}

	socketPath := parts[0]
	if socketPath == "" {
		return nil
	}

	if _, err := os.Stat(socketPath); os.IsNotExist(err) {
		return nil
	}

	var args []string
	socketDir := filepath.Dir(socketPath)

	// Bind the tmux socket directory
	args = append(args, "--bind", socketDir, socketDir)
	args = append(args, "--setenv", "TMUX", tmuxEnv)

	if tmuxPane := os.Getenv("TMUX_PANE"); tmuxPane != "" {
		args = append(args, "--setenv", "TMUX_PANE", tmuxPane)
	}

	return args
}

// handleHistoryPersist configures shell history persistence.
// Uses the same per-project history directory as other providers.
func (b *BwrapProvider) handleHistoryPersist(spec *provider.RunSpec) []string {
	if !spec.HistoryPersist {
		return nil
	}

	projectDir := spec.WorkDir
	if projectDir == "" {
		projectDir, _ = os.Getwd()
	}

	historyDir, err := getBwrapProjectHistoryDir(projectDir)
	if err != nil {
		fmt.Printf("Warning: failed to create history directory: %v\n", err)
		return nil
	}

	var args []string

	// Create and mount bash history
	bashHistory := filepath.Join(historyDir, "bash_history")
	if err := touchHistoryFile(bashHistory); err == nil {
		args = append(args, "--bind", bashHistory, "/home/addt/.bash_history")
	}

	// Create and mount zsh history
	zshHistory := filepath.Join(historyDir, "zsh_history")
	if err := touchHistoryFile(zshHistory); err == nil {
		args = append(args, "--bind", zshHistory, "/home/addt/.zsh_history")
	}

	return args
}

// getBwrapProjectHistoryDir returns the history directory for a project
func getBwrapProjectHistoryDir(projectDir string) (string, error) {
	addtHome := util.GetAddtHome()
	if addtHome == "" {
		return "", fmt.Errorf("failed to determine addt home directory")
	}

	hash := sha256.Sum256([]byte(projectDir))
	projectHash := hex.EncodeToString(hash[:8])

	historyDir := filepath.Join(addtHome, "history", projectHash)
	if err := os.MkdirAll(historyDir, 0700); err != nil {
		return "", fmt.Errorf("failed to create history dir: %w", err)
	}

	return historyDir, nil
}

// touchHistoryFile creates an empty file if it doesn't exist
func touchHistoryFile(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			return err
		}
		f.Close()
	}
	return nil
}
