package bwrap

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	"github.com/jedi4ever/addt/provider"
	"github.com/jedi4ever/addt/util"
)

var bwrapLogger = util.Log("bwrap")

// Run runs a command in a bwrap sandbox
func (b *BwrapProvider) Run(spec *provider.RunSpec) error {
	bwrapLogger.Debugf("BwrapProvider.Run called: Name=%s, Args=%v, Interactive=%v, Persistent=%v",
		spec.Name, spec.Args, spec.Interactive, spec.Persistent)

	// Check for existing persistent session
	if spec.Persistent && b.Exists(spec.Name) {
		fmt.Printf("Found existing persistent session: %s\n", spec.Name)
		fmt.Println("Session is running, connecting via nsenter...")
		return b.execInSession(spec.Name, spec)
	}

	// Prepare secrets before building args (modifies spec.Env)
	secretsDir := b.prepareAndFilterSecrets(spec)

	bwrapArgs, err := b.buildBwrapArgs(spec)
	if err != nil {
		return err
	}

	// Add secrets mount
	if secretsDir != "" {
		bwrapArgs = append(bwrapArgs, "--bind", secretsDir, "/run/secrets")
	}

	// Persistent: start sandbox with sleep infinity, then nsenter the command
	if spec.Persistent {
		fmt.Printf("Creating new persistent session: %s\n", spec.Name)
		return b.runPersistent(bwrapArgs, spec)
	}

	// Ephemeral: run command directly
	bwrapArgs = append(bwrapArgs, "--")
	if secretsDir != "" {
		bwrapArgs = append(bwrapArgs, "/run/secrets/.wrapper.sh")
	}
	bwrapArgs = b.appendCommand(bwrapArgs, spec)
	return b.executeBwrapCommand(bwrapArgs)
}

// Shell opens a shell in a bwrap sandbox
func (b *BwrapProvider) Shell(spec *provider.RunSpec) error {
	// Check for existing persistent session
	if spec.Persistent && b.Exists(spec.Name) {
		fmt.Println("Opening bash shell in existing bwrap session...")
		spec.Env["ADDT_COMMAND"] = "/bin/bash"
		return b.execInSession(spec.Name, spec)
	}

	fmt.Println("Opening bash shell in bwrap sandbox...")

	// Prepare secrets before building args
	secretsDir := b.prepareAndFilterSecrets(spec)

	bwrapArgs, err := b.buildBwrapArgs(spec)
	if err != nil {
		return err
	}

	// Add secrets mount
	if secretsDir != "" {
		bwrapArgs = append(bwrapArgs, "--bind", secretsDir, "/run/secrets")
	}

	if spec.Persistent {
		fmt.Printf("Creating new persistent session: %s\n", spec.Name)
		spec.Env["ADDT_COMMAND"] = "/bin/bash"
		return b.runPersistent(bwrapArgs, spec)
	}

	bwrapArgs = append(bwrapArgs, "--")
	if secretsDir != "" {
		bwrapArgs = append(bwrapArgs, "/run/secrets/.wrapper.sh")
	}
	bwrapArgs = append(bwrapArgs, "/bin/bash")
	if len(spec.Args) > 0 {
		bwrapArgs = append(bwrapArgs, spec.Args...)
	}
	return b.executeBwrapCommand(bwrapArgs)
}

// runPersistent starts a bwrap sandbox with "sleep infinity" as PID 1,
// saves the PID, then uses nsenter to execute the actual command.
func (b *BwrapProvider) runPersistent(bwrapArgs []string, spec *provider.RunSpec) error {
	// Start bwrap with sleep infinity in the background
	bwrapArgs = append(bwrapArgs, "--", "sleep", "infinity")
	bwrapLogger.Debugf("Starting persistent session: bwrap %v", bwrapArgs)

	cmd := exec.Command("bwrap", bwrapArgs...)
	// Don't connect stdin/stdout — this is a background keep-alive process
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Stdin = nil

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start persistent session: %w", err)
	}

	pid := cmd.Process.Pid
	if err := writePid(spec.Name, pid); err != nil {
		cmd.Process.Kill()
		return fmt.Errorf("failed to save session PID: %w", err)
	}

	bwrapLogger.Debugf("Persistent session started: PID=%d, name=%s", pid, spec.Name)

	// Wait briefly for the namespace to be ready
	time.Sleep(200 * time.Millisecond)

	if !isProcessAlive(pid) {
		removePid(spec.Name)
		return fmt.Errorf("persistent session exited immediately (PID %d)", pid)
	}

	// Now exec the actual command via nsenter
	return b.execInSession(spec.Name, spec)
}

// execInSession uses nsenter to execute a command inside an existing
// persistent bwrap session's namespaces.
func (b *BwrapProvider) execInSession(name string, spec *provider.RunSpec) error {
	pid := readPid(name)
	if pid == 0 {
		return fmt.Errorf("no PID found for session %s", name)
	}

	pidStr := fmt.Sprintf("%d", pid)

	// Build nsenter command: enter mount, pid, ipc, uts namespaces
	nsenterArgs := []string{
		"--target", pidStr,
		"--mount",
		"--pid",
		"--ipc",
		"--uts",
		"--",
	}

	// Set environment variables via env command inside the namespace
	var envCmd []string
	envCmd = append(envCmd, "env")

	// Clear environment and set our vars
	envCmd = append(envCmd, "-i")
	envCmd = append(envCmd, "HOME=/home/addt")
	envCmd = append(envCmd, "USER=addt")
	envCmd = append(envCmd, "SHELL=/bin/bash")
	envCmd = append(envCmd, "TERM=xterm-256color")
	envCmd = append(envCmd, "PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin")

	for k, v := range spec.Env {
		envCmd = append(envCmd, fmt.Sprintf("%s=%s", k, v))
	}

	// Determine the command to run
	command := spec.Env["ADDT_COMMAND"]
	if command != "" {
		parts := strings.Fields(command)
		envCmd = append(envCmd, parts...)
	} else if len(spec.Args) > 0 {
		envCmd = append(envCmd, spec.Args...)
	} else {
		envCmd = append(envCmd, "/bin/bash")
	}

	// Append remaining args
	if command != "" && len(spec.Args) > 0 {
		envCmd = append(envCmd, spec.Args...)
	}

	nsenterArgs = append(nsenterArgs, envCmd...)

	bwrapLogger.Debugf("Executing in session: nsenter %v", nsenterArgs)
	cmd := exec.Command("nsenter", nsenterArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		bwrapLogger.Debugf("nsenter command failed: %v", err)
	}
	return err
}

// appendCommand adds the command and args to run inside the sandbox.
// Reads ADDT_COMMAND from spec.Env to determine the entrypoint.
func (b *BwrapProvider) appendCommand(bwrapArgs []string, spec *provider.RunSpec) []string {
	command := spec.Env["ADDT_COMMAND"]
	if command != "" {
		parts := strings.Fields(command)
		bwrapArgs = append(bwrapArgs, parts...)
	} else if len(spec.Args) > 0 {
		return append(bwrapArgs, spec.Args...)
	} else {
		bwrapArgs = append(bwrapArgs, "/bin/bash")
	}

	if len(spec.Args) > 0 {
		bwrapArgs = append(bwrapArgs, spec.Args...)
	}
	return bwrapArgs
}

// buildBwrapArgs constructs the bwrap command arguments
func (b *BwrapProvider) buildBwrapArgs(spec *provider.RunSpec) ([]string, error) {
	if err := b.ensureHomeDir(); err != nil {
		return nil, fmt.Errorf("failed to create home directory: %w", err)
	}

	currentUser, err := user.Current()
	if err != nil {
		return nil, fmt.Errorf("failed to get current user: %w", err)
	}

	var args []string

	// ── Filesystem mounts ────────────────────────────────────────────

	for _, dir := range []string{"/usr", "/bin", "/lib", "/lib64", "/sbin", "/etc", "/opt"} {
		if info, err := os.Lstat(dir); err == nil {
			if info.Mode()&os.ModeSymlink != 0 {
				target, err := os.Readlink(dir)
				if err == nil {
					args = append(args, "--symlink", target, dir)
					continue
				}
			}
			args = append(args, "--ro-bind", dir, dir)
		}
	}

	// NixOS support
	if _, err := os.Stat("/nix"); err == nil {
		args = append(args, "--ro-bind", "/nix", "/nix")
	}

	args = append(args, "--proc", "/proc")
	args = append(args, "--dev", "/dev")

	// tmpfs for temporary directories
	args = append(args, "--tmpfs", "/tmp")
	args = append(args, "--tmpfs", "/var/tmp")
	args = append(args, "--tmpfs", "/run")

	// Home directory (persistent across runs)
	homeDir := b.getHomeDir()
	args = append(args, "--bind", homeDir, "/home/addt")

	// ── Working directory ────────────────────────────────────────────

	workDir := spec.WorkDir
	if workDir == "" {
		workDir, _ = os.Getwd()
	}

	if b.config.WorkdirAutomount && workDir != "" {
		if b.config.WorkdirReadonly {
			args = append(args, "--ro-bind", workDir, "/workspace")
		} else {
			args = append(args, "--bind", workDir, "/workspace")
		}
		args = append(args, "--chdir", "/workspace")
	}

	// ── Volume mounts from spec ──────────────────────────────────────

	for _, vol := range spec.Volumes {
		if _, err := os.Stat(vol.Source); err != nil {
			continue
		}
		if vol.ReadOnly {
			args = append(args, "--ro-bind", vol.Source, vol.Target)
		} else {
			args = append(args, "--bind", vol.Source, vol.Target)
		}
	}

	// ── Extension config mounts ──────────────────────────────────────

	args = b.addExtensionMounts(args)

	// ── Forwarding ───────────────────────────────────────────────────

	sshDir := b.config.SSHDir
	if sshDir == "" {
		sshDir = filepath.Join(currentUser.HomeDir, ".ssh")
	} else {
		sshDir = util.ExpandTilde(sshDir)
	}
	args = append(args, b.handleSSHForwarding(spec, sshDir)...)

	gpgDir := b.config.GPGDir
	if gpgDir == "" {
		gpgDir = filepath.Join(currentUser.HomeDir, ".gnupg")
	} else {
		gpgDir = util.ExpandTilde(gpgDir)
	}
	args = append(args, b.handleGPGForwarding(spec, gpgDir)...)
	args = append(args, b.handleTmuxForwarding(spec)...)
	args = append(args, b.handleHistoryPersist(spec)...)

	// ── Git config ───────────────────────────────────────────────────

	if b.config.GitForwardConfig {
		gitconfigPath := b.config.GitConfigPath
		if gitconfigPath == "" {
			gitconfigPath = filepath.Join(currentUser.HomeDir, ".gitconfig")
		} else {
			gitconfigPath = util.ExpandTilde(gitconfigPath)
		}
		if _, err := os.Stat(gitconfigPath); err == nil {
			args = append(args, "--ro-bind", gitconfigPath, "/home/addt/.gitconfig.host")
		}
	}

	// ── Environment variables ────────────────────────────────────────

	args = append(args, "--clearenv")

	args = append(args, "--setenv", "HOME", "/home/addt")
	args = append(args, "--setenv", "USER", "addt")
	args = append(args, "--setenv", "SHELL", "/bin/bash")
	args = append(args, "--setenv", "TERM", "xterm-256color")
	args = append(args, "--setenv", "PATH",
		"/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin")

	for _, envVar := range []string{"LANG", "LC_ALL", "LC_CTYPE", "LC_MESSAGES"} {
		if val := os.Getenv(envVar); val != "" {
			args = append(args, "--setenv", envVar, val)
		}
	}

	if ct := os.Getenv("COLORTERM"); ct != "" {
		args = append(args, "--setenv", "COLORTERM", ct)
	}

	for k, v := range spec.Env {
		args = append(args, "--setenv", k, v)
	}

	// ── Namespace isolation ──────────────────────────────────────────

	args = append(args, "--unshare-pid")
	args = append(args, "--unshare-ipc")
	args = append(args, "--die-with-parent")
	args = append(args, "--new-session")

	// ── Security settings ────────────────────────────────────────────

	args = b.addSecurityArgs(args)

	return args, nil
}

// executeBwrapCommand runs the bwrap command with standard I/O
func (b *BwrapProvider) executeBwrapCommand(bwrapArgs []string) error {
	bwrapLogger.Debugf("Executing: bwrap %v", bwrapArgs)
	cmd := exec.Command("bwrap", bwrapArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		bwrapLogger.Debugf("bwrap command failed: %v", err)
	}
	return err
}
