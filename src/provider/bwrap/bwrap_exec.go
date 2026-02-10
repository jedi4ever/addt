package bwrap

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/jedi4ever/addt/provider"
	"github.com/jedi4ever/addt/util"
)

var bwrapLogger = util.Log("bwrap")

// Run runs a command in a bwrap sandbox
func (b *BwrapProvider) Run(spec *provider.RunSpec) error {
	bwrapLogger.Debugf("BwrapProvider.Run called: Name=%s, Args=%v, Interactive=%v",
		spec.Name, spec.Args, spec.Interactive)

	bwrapArgs, err := b.buildBwrapArgs(spec)
	if err != nil {
		return err
	}

	// Determine the command to run
	bwrapArgs = append(bwrapArgs, "--")
	bwrapArgs = b.appendCommand(bwrapArgs, spec)

	return b.executeBwrapCommand(bwrapArgs)
}

// Shell opens a shell in a bwrap sandbox
func (b *BwrapProvider) Shell(spec *provider.RunSpec) error {
	fmt.Println("Opening bash shell in bwrap sandbox...")

	bwrapArgs, err := b.buildBwrapArgs(spec)
	if err != nil {
		return err
	}

	bwrapArgs = append(bwrapArgs, "--", "/bin/bash")
	if len(spec.Args) > 0 {
		bwrapArgs = append(bwrapArgs, spec.Args...)
	}
	return b.executeBwrapCommand(bwrapArgs)
}

// appendCommand adds the command and args to run inside the sandbox.
// Reads ADDT_COMMAND from spec.Env to determine the entrypoint.
func (b *BwrapProvider) appendCommand(bwrapArgs []string, spec *provider.RunSpec) []string {
	command := spec.Env["ADDT_COMMAND"]
	if command != "" {
		// Split command in case it contains spaces (e.g., "/usr/bin/node /path/to/script")
		parts := strings.Fields(command)
		bwrapArgs = append(bwrapArgs, parts...)
	} else if len(spec.Args) > 0 {
		// No ADDT_COMMAND, use args directly
		return append(bwrapArgs, spec.Args...)
	} else {
		bwrapArgs = append(bwrapArgs, "/bin/bash")
	}

	// Append remaining args
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

	// Mount essential host directories read-only
	for _, dir := range []string{"/usr", "/bin", "/lib", "/lib64", "/sbin", "/etc", "/opt"} {
		if info, err := os.Lstat(dir); err == nil {
			if info.Mode()&os.ModeSymlink != 0 {
				// Resolve symlinks (common on merged-/usr distros)
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

	// Mount proc and dev
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
			continue // Skip non-existent sources
		}
		if vol.ReadOnly {
			args = append(args, "--ro-bind", vol.Source, vol.Target)
		} else {
			args = append(args, "--bind", vol.Source, vol.Target)
		}
	}

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

	// Clear environment for isolation, then set specific vars
	args = append(args, "--clearenv")

	// Essential env vars
	args = append(args, "--setenv", "HOME", "/home/addt")
	args = append(args, "--setenv", "USER", "addt")
	args = append(args, "--setenv", "SHELL", "/bin/bash")
	args = append(args, "--setenv", "TERM", "xterm-256color")
	args = append(args, "--setenv", "PATH",
		"/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin")

	// Pass through locale settings from host
	for _, envVar := range []string{"LANG", "LC_ALL", "LC_CTYPE", "LC_MESSAGES"} {
		if val := os.Getenv(envVar); val != "" {
			args = append(args, "--setenv", envVar, val)
		}
	}

	// Pass through COLORTERM if present
	if ct := os.Getenv("COLORTERM"); ct != "" {
		args = append(args, "--setenv", "COLORTERM", ct)
	}

	// Set all spec env vars (these come from the core layer's BuildEnvironment)
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
