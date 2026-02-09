package podman

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"time"

	"github.com/jedi4ever/addt/assets"
	"github.com/jedi4ever/addt/provider"
	"github.com/jedi4ever/addt/util"
)

var podmanLogger = util.Log("podman")

// containerContext holds common container setup information
type containerContext struct {
	homeDir              string
	username             string
	useExistingContainer bool
}

// setupContainerContext prepares common container context and checks for existing containers
func (p *PodmanProvider) setupContainerContext(spec *provider.RunSpec) (*containerContext, error) {
	currentUser, err := user.Current()
	if err != nil {
		return nil, fmt.Errorf("failed to get current user: %w", err)
	}

	ctx := &containerContext{
		homeDir:              currentUser.HomeDir,
		username:             "addt", // Always use "addt" in container, but with host UID/GID
		useExistingContainer: false,
	}

	// Check if we should use existing container
	if spec.Persistent && p.Exists(spec.Name) {
		fmt.Printf("Found existing persistent container: %s\n", spec.Name)
		if p.IsRunning(spec.Name) {
			fmt.Println("Container is running, connecting...")
			ctx.useExistingContainer = true
		} else {
			fmt.Println("Container is stopped, starting...")
			if err := p.Start(spec.Name); err != nil {
				fmt.Println("Failed to start container, removing and recreating...")
				p.Remove(spec.Name)
				// Fall through to create a new container
			} else {
				// Give container a moment to stabilize (entrypoint may exit immediately)
				time.Sleep(500 * time.Millisecond)
				if !p.IsRunning(spec.Name) {
					// Container started but exited immediately (entrypoint finished)
					fmt.Println("Container exited after start, removing and recreating...")
					p.Remove(spec.Name)
					// Fall through to create a new container
				} else {
					ctx.useExistingContainer = true
				}
			}
		}
	} else if spec.Persistent {
		fmt.Printf("Creating new persistent container: %s\n", spec.Name)
	}

	return ctx, nil
}

// buildBasePodmanArgs creates the base podman arguments for run/exec commands
func (p *PodmanProvider) buildBasePodmanArgs(spec *provider.RunSpec, ctx *containerContext) []string {
	podmanLogger.Debugf("buildBasePodmanArgs: spec.Interactive=%v, ctx.useExistingContainer=%v, spec.Persistent=%v",
		spec.Interactive, ctx.useExistingContainer, spec.Persistent)

	var podmanArgs []string

	if ctx.useExistingContainer {
		podmanArgs = []string{"exec"}
	} else {
		if spec.Persistent {
			podmanArgs = []string{"run", "--name", spec.Name}
		} else {
			podmanArgs = []string{"run", "--rm", "--name", spec.Name}
		}
	}

	// Interactive mode
	if spec.Interactive {
		podmanArgs = append(podmanArgs, "-it")
		podmanLogger.Debug("Added -it flag (interactive mode)")
		if !ctx.useExistingContainer {
			podmanArgs = append(podmanArgs, "--init")
		}
	} else {
		// Add -i flag for podman (needed for proper stdin handling)
		podmanArgs = append(podmanArgs, "-i")
		podmanLogger.Debug("Added -i flag (non-interactive mode)")
	}

	podmanLogger.Debugf("buildBasePodmanArgs returning: %v", podmanArgs)
	return podmanArgs
}

// addContainerVolumesAndEnv adds volumes, mounts, and environment variables for new containers
func (p *PodmanProvider) addContainerVolumesAndEnv(podmanArgs []string, spec *provider.RunSpec, ctx *containerContext) ([]string, func()) {
	// Cleanup function for secrets directory (caller should defer this)
	cleanup := func() {}

	// Add volumes
	for _, vol := range spec.Volumes {
		mount := fmt.Sprintf("%s:%s", vol.Source, vol.Target)
		if vol.ReadOnly {
			mount += ":ro"
		}
		podmanArgs = append(podmanArgs, "-v", mount)
	}

	// Add extension mounts
	podmanArgs = p.AddExtensionMounts(podmanArgs, spec.ImageName, ctx.homeDir)

	// Mount .gitconfig (if forwarding enabled)
	if p.config.GitForwardConfig {
		gitconfigPath := p.config.GitConfigPath
		if gitconfigPath == "" {
			gitconfigPath = filepath.Join(ctx.homeDir, ".gitconfig")
		} else {
			gitconfigPath = util.ExpandTilde(gitconfigPath)
		}
		if _, err := os.Stat(gitconfigPath); err == nil {
			podmanArgs = append(podmanArgs, "-v", fmt.Sprintf("%s:/home/%s/.gitconfig.host:ro", gitconfigPath, ctx.username))
		}
	}

	// Env file vars are loaded into spec.Env by BuildRunOptions (see core/options.go)
	// so they go through the same -e mechanism as other env vars.

	// SSH forwarding
	sshDir := p.config.SSHDir
	if sshDir == "" {
		sshDir = filepath.Join(ctx.homeDir, ".ssh")
	} else {
		sshDir = util.ExpandTilde(sshDir)
	}
	podmanArgs = append(podmanArgs, p.HandleSSHForwarding(spec.SSHForwardKeys, spec.SSHForwardMode, sshDir, ctx.username, spec.SSHAllowedKeys)...)

	// GPG forwarding
	gpgDir := p.config.GPGDir
	if gpgDir == "" {
		gpgDir = filepath.Join(ctx.homeDir, ".gnupg")
	} else {
		gpgDir = util.ExpandTilde(gpgDir)
	}
	podmanArgs = append(podmanArgs, p.HandleGPGForwarding(spec.GPGForward, gpgDir, ctx.username, spec.GPGAllowedKeyIDs)...)

	// Tmux forwarding
	podmanArgs = append(podmanArgs, p.HandleTmuxForwarding(spec.TmuxForward)...)

	// History persistence
	podmanArgs = append(podmanArgs, p.HandleHistoryPersist(spec.HistoryPersist, spec.WorkDir, ctx.username)...)

	// Firewall configuration with pasta network backend
	if p.config.FirewallEnabled {
		// Start as root so entrypoint can apply iptables rules without sudo,
		// then drop to addt via gosu (compatible with no-new-privileges)
		podmanArgs = append(podmanArgs, "--user", "root")

		// Use pasta network backend for better firewall support in rootless mode
		// pasta handles network namespaces efficiently and supports filtering
		if p.CheckPastaAvailable() {
			podmanArgs = append(podmanArgs, "--network=pasta")
		}

		// Capabilities for the root phase (dropped after gosu switches to addt):
		// NET_ADMIN: iptables/nftables rules
		// DAC_OVERRIDE: create dirs/files in addt's home
		// CHOWN: fix file ownership
		// SETUID/SETGID: gosu needs these to switch from root to addt
		podmanArgs = append(podmanArgs, "--cap-add", "NET_ADMIN")
		podmanArgs = append(podmanArgs, "--cap-add", "DAC_OVERRIDE")
		podmanArgs = append(podmanArgs, "--cap-add", "CHOWN")
		podmanArgs = append(podmanArgs, "--cap-add", "SETUID")
		podmanArgs = append(podmanArgs, "--cap-add", "SETGID")

		// Mount firewall config directory
		addtHome := util.GetAddtHome()
		if addtHome != "" {
			firewallConfigDir := filepath.Join(addtHome, "firewall")
			if _, err := os.Stat(firewallConfigDir); err == nil {
				podmanArgs = append(podmanArgs, "-v", fmt.Sprintf("%s:/home/%s/.addt/firewall", firewallConfigDir, ctx.username))
			}
		}
	}

	// Podman-in-Podman support (similar to DinD)
	podmanArgs = append(podmanArgs, p.HandlePodmanForwarding(spec.DockerDindMode, spec.Name)...)

	// Start as root for DinD so entrypoint can set up nested Podman
	if spec.DockerDindMode == "isolated" || spec.DockerDindMode == "true" {
		podmanArgs = append(podmanArgs, "--user", "root")
	}

	// Add ports
	for _, port := range spec.Ports {
		podmanArgs = append(podmanArgs, "-p", fmt.Sprintf("%d:%d", port.Host, port.Container))
	}

	// Handle isolate_secrets: add tmpfs mount for secrets
	// Secrets are copied via podman cp after container starts (see runWithSecrets)
	if p.config.Security.IsolateSecrets {
		podmanArgs = p.addTmpfsSecretsMount(podmanArgs)
	}

	// Handle OTEL: add host alias so container can reach host's OTEL collector
	// Podman's host-gateway can fail on macOS; use detected host IP instead
	if p.config.Otel.Enabled {
		if hostIP, err := getHostGatewayIP(); err == nil {
			podmanArgs = append(podmanArgs, fmt.Sprintf("--add-host=host.docker.internal:%s", hostIP))
		} else {
			fmt.Fprintf(os.Stderr, "Warning: could not detect host IP for OTEL: %v\n", err)
		}
	}

	// Add environment variables
	for k, v := range spec.Env {
		podmanArgs = append(podmanArgs, "-e", fmt.Sprintf("%s=%s", k, v))
	}

	// Add resource limits
	if spec.ContainerCPUs != "" {
		podmanArgs = append(podmanArgs, "--cpus", spec.ContainerCPUs)
	}
	if spec.ContainerMemory != "" {
		podmanArgs = append(podmanArgs, "--memory", spec.ContainerMemory)
	}

	// Add security settings
	podmanArgs = p.addSecuritySettings(podmanArgs)

	return podmanArgs, cleanup
}

// executePodmanCommand runs the podman command with standard I/O
func (p *PodmanProvider) executePodmanCommand(podmanArgs []string) error {
	podmanLogger.Debugf("Executing: podman %v", podmanArgs)
	cmd := exec.Command("podman", podmanArgs...)

	// Connect stdin if -it or -i flag is present
	hasInteractive := false
	for _, arg := range podmanArgs {
		if arg == "-it" || arg == "-i" {
			hasInteractive = true
			break
		}
	}

	if hasInteractive {
		cmd.Stdin = os.Stdin
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		podmanLogger.Debugf("Podman command failed: %v", err)
	}
	return err
}

// Run runs a new container
func (p *PodmanProvider) Run(spec *provider.RunSpec) error {
	podmanLogger.Debugf("PodmanProvider.Run called with spec: Name=%s, ImageName=%s, Args=%v, Interactive=%v",
		spec.Name, spec.ImageName, spec.Args, spec.Interactive)

	podmanLogger.Debug("Setting up container context")
	ctx, err := p.setupContainerContext(spec)
	if err != nil {
		podmanLogger.Debugf("Failed to setup container context: %v", err)
		return err
	}
	podmanLogger.Debugf("Container context: useExistingContainer=%v, homeDir=%s, username=%s",
		ctx.useExistingContainer, ctx.homeDir, ctx.username)

	// Prepare secrets if enabled (before building args so we can filter env)
	var secretsJSON string
	if p.config.Security.IsolateSecrets && !ctx.useExistingContainer {
		podmanLogger.Debug("Preparing secrets for isolated secrets mode")
		json, secretVarNames, err := p.prepareSecretsJSON(spec.ImageName, spec.Env)
		if err == nil && json != "" {
			secretsJSON = json
			p.filterSecretEnvVars(spec.Env, secretVarNames)
			// ADDT_CREDENTIAL_VARS is no longer needed — secrets are in the file
			delete(spec.Env, "ADDT_CREDENTIAL_VARS")
			podmanLogger.Debugf("Secrets prepared, %d secret variables filtered", len(secretVarNames))
		} else if err != nil {
			podmanLogger.Debugf("Failed to prepare secrets: %v", err)
		}
	}

	podmanLogger.Debug("Building base Podman arguments")
	podmanLogger.Debugf("Spec.Interactive=%v, ctx.useExistingContainer=%v", spec.Interactive, ctx.useExistingContainer)
	podmanArgs := p.buildBasePodmanArgs(spec, ctx)
	podmanLogger.Debugf("Base Podman args: %v", podmanArgs)

	// Only add volumes and environment when creating a new container
	cleanup := func() {}
	if !ctx.useExistingContainer {
		podmanLogger.Debug("Adding volumes and environment variables")
		podmanArgs, cleanup = p.addContainerVolumesAndEnv(podmanArgs, spec, ctx)
		podmanLogger.Debugf("Podman args after volumes/env (image name and args will be added next): %v", podmanArgs)
	}
	defer cleanup()

	// Handle existing container
	if ctx.useExistingContainer {
		podmanLogger.Debugf("Using existing container: %s", spec.Name)
		podmanArgs = append(podmanArgs, spec.Name)
		// Call entrypoint with args for existing containers
		podmanArgs = append(podmanArgs, "/usr/local/bin/podman-entrypoint.sh")
		podmanArgs = append(podmanArgs, spec.Args...)
		podmanLogger.Debugf("Executing podman exec with args: %v", podmanArgs)
		return p.executePodmanCommand(podmanArgs)
	}

	// New persistent container: detached keep-alive + exec entrypoint
	if spec.Persistent {
		return p.runPersistent(podmanArgs, spec, secretsJSON)
	}

	// New container with secrets: use two-step process
	// 1. Start container detached with wait script
	// 2. Copy secrets via podman cp
	// 3. Signal container to continue and attach
	if secretsJSON != "" {
		podmanLogger.Debug("Running with secrets (two-step process)")
		return p.runWithSecrets(podmanArgs, spec, secretsJSON)
	}

	// Normal run without secrets
	// Note: Image has default ENTRYPOINT ["/usr/local/bin/podman-entrypoint.sh"] set in Dockerfile.base
	podmanLogger.Debugf("Normal run without secrets, appending image: %s and args: %v", spec.ImageName, spec.Args)
	podmanArgs = append(podmanArgs, spec.ImageName)
	podmanArgs = append(podmanArgs, spec.Args...)
	podmanLogger.Debugf("Executing podman run with final args (entrypoint will be called from image): %v", podmanArgs)
	return p.executePodmanCommand(podmanArgs)
}

// runPersistent creates a persistent container with sleep infinity as PID 1,
// then execs the entrypoint. This ensures the container stays alive after
// the agent exits, so subsequent runs can reuse it via podman exec.
func (p *PodmanProvider) runPersistent(baseArgs []string, spec *provider.RunSpec, secretsJSON string) error {
	// Strip interactive/init flags — not needed for detached sleep process
	var runArgs []string
	needsTTY := false
	needsStdin := false
	for _, arg := range baseArgs {
		switch arg {
		case "-it":
			needsTTY = true
			needsStdin = true
		case "-i":
			needsStdin = true
		case "-t", "--init":
			// not needed for detached sleep process
		default:
			runArgs = append(runArgs, arg)
		}
	}

	// Start container detached with sleep as keep-alive PID 1
	runArgs = append(runArgs, "-d", "--entrypoint", "sleep", spec.ImageName, "infinity")
	podmanLogger.Debugf("Starting persistent container: podman %v", runArgs)

	cmd := exec.Command("podman", runArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to start persistent container: %w\n%s", err, string(output))
	}

	// Copy secrets if needed
	if secretsJSON != "" {
		podmanLogger.Debug("Copying secrets to persistent container")
		if err := p.copySecretsToContainer(spec.Name, secretsJSON); err != nil {
			podmanLogger.Debugf("Failed to copy secrets, cleaning up container %s", spec.Name)
			exec.Command("podman", "rm", "-f", spec.Name).Run()
			return fmt.Errorf("failed to copy secrets: %w", err)
		}
	}

	// Exec entrypoint — output goes directly to terminal
	execArgs := []string{"exec"}
	if needsTTY {
		execArgs = append(execArgs, "-it")
	} else if needsStdin {
		execArgs = append(execArgs, "-i")
	}
	execArgs = append(execArgs, spec.Name, "/usr/local/bin/podman-entrypoint.sh")
	execArgs = append(execArgs, spec.Args...)

	podmanLogger.Debugf("Executing entrypoint in persistent container: podman %v", execArgs)
	return p.executePodmanCommand(execArgs)
}

// runWithSecrets starts a container, copies secrets, then execs the entrypoint.
// Uses a simple approach: start with sleep, copy secrets, exec entrypoint.
// Entrypoint output goes directly to terminal via exec (no attach needed).
func (p *PodmanProvider) runWithSecrets(baseArgs []string, spec *provider.RunSpec, secretsJSON string) error {
	// Strip interactive flags from run args — they'll be added to exec instead.
	// The detached sleep process doesn't need stdin or TTY.
	var runArgs []string
	needsTTY := false
	needsStdin := false
	for _, arg := range baseArgs {
		switch arg {
		case "-it":
			needsTTY = true
			needsStdin = true
		case "-i":
			needsStdin = true
		case "-t", "--init":
			// not needed for detached sleep process
		default:
			runArgs = append(runArgs, arg)
		}
	}

	// Start container detached with sleep as keep-alive
	runArgs = append(runArgs, "-d", "--entrypoint", "sleep", spec.ImageName, "infinity")
	podmanLogger.Debugf("Starting detached container: podman %v", runArgs)

	cmd := exec.Command("podman", runArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to start container: %w\n%s", err, string(output))
	}

	// Copy secrets to container tmpfs
	podmanLogger.Debug("Copying secrets to container")
	if err := p.copySecretsToContainer(spec.Name, secretsJSON); err != nil {
		podmanLogger.Debugf("Failed to copy secrets, cleaning up container %s", spec.Name)
		exec.Command("podman", "rm", "-f", spec.Name).Run()
		return fmt.Errorf("failed to copy secrets: %w", err)
	}

	// Exec entrypoint — output goes directly to terminal
	// Note: secrets file ownership is fixed by root phase of entrypoint before dropping to addt
	execArgs := []string{"exec"}
	if needsTTY {
		execArgs = append(execArgs, "-it")
	} else if needsStdin {
		execArgs = append(execArgs, "-i")
	}
	execArgs = append(execArgs, spec.Name, "/usr/local/bin/podman-entrypoint.sh")
	execArgs = append(execArgs, spec.Args...)

	podmanLogger.Debugf("Executing entrypoint: podman %v", execArgs)
	execErr := p.executePodmanCommand(execArgs)

	// On failure, dump container logs for debugging
	if execErr != nil {
		podmanLogger.Debugf("Entrypoint failed, fetching container logs for %s", spec.Name)
		if logsOutput, err := exec.Command("podman", "logs", spec.Name).CombinedOutput(); err == nil && len(logsOutput) > 0 {
			podmanLogger.Debugf("Container logs:\n%s", string(logsOutput))
		}
	}

	// Clean up non-persistent containers (stop sleep, triggers --rm if set)
	if !spec.Persistent {
		podmanLogger.Debugf("Removing non-persistent container %s", spec.Name)
		exec.Command("podman", "rm", "-f", spec.Name).Run()
	}

	return execErr
}

// Shell opens a shell in a container
func (p *PodmanProvider) Shell(spec *provider.RunSpec) error {
	ctx, err := p.setupContainerContext(spec)
	if err != nil {
		return err
	}

	podmanArgs := p.buildBasePodmanArgs(spec, ctx)

	// Only add volumes and environment when creating a new container
	cleanup := func() {}
	if !ctx.useExistingContainer {
		podmanArgs, cleanup = p.addContainerVolumesAndEnv(podmanArgs, spec, ctx)
	}
	defer cleanup()

	// Open shell
	fmt.Println("Opening bash shell in container...")
	if ctx.useExistingContainer {
		// Run through entrypoint so socat bridges and debug logging work
		podmanArgs = append(podmanArgs, "-e", "ADDT_COMMAND=/bin/bash")
		podmanArgs = append(podmanArgs, spec.Name, "/usr/local/bin/podman-entrypoint.sh")
		podmanArgs = append(podmanArgs, spec.Args...)
	} else if spec.Persistent {
		return p.shellPersistent(podmanArgs, spec, ctx)
	} else {
		// Use default entrypoint with ADDT_COMMAND override to bash
		// The entrypoint handles all initialization: socat bridges, secrets,
		// firewall, DinD, extensions, and debug logging.
		podmanArgs = append(podmanArgs, "-e", "ADDT_COMMAND=/bin/bash")
		podmanArgs = append(podmanArgs, spec.ImageName)
		podmanArgs = append(podmanArgs, spec.Args...)
	}

	return p.executePodmanCommand(podmanArgs)
}

// shellPersistent creates a persistent container with sleep infinity as PID 1,
// then execs the entrypoint with ADDT_COMMAND=/bin/bash for shell access.
func (p *PodmanProvider) shellPersistent(baseArgs []string, spec *provider.RunSpec, ctx *containerContext) error {
	// Strip interactive/init flags — not needed for detached sleep process
	var runArgs []string
	needsTTY := false
	needsStdin := false
	for _, arg := range baseArgs {
		switch arg {
		case "-it":
			needsTTY = true
			needsStdin = true
		case "-i":
			needsStdin = true
		case "-t", "--init":
			// not needed for detached sleep process
		default:
			runArgs = append(runArgs, arg)
		}
	}

	// Start container detached with sleep as keep-alive PID 1
	runArgs = append(runArgs, "-d", "--entrypoint", "sleep", spec.ImageName, "infinity")
	podmanLogger.Debugf("Starting persistent container for shell: podman %v", runArgs)

	cmd := exec.Command("podman", runArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to start persistent container: %w\n%s", err, string(output))
	}

	// Exec entrypoint with bash override
	execArgs := []string{"exec"}
	if needsTTY {
		execArgs = append(execArgs, "-it")
	} else if needsStdin {
		execArgs = append(execArgs, "-i")
	}
	execArgs = append(execArgs, "-e", "ADDT_COMMAND=/bin/bash")
	execArgs = append(execArgs, spec.Name, "/usr/local/bin/podman-entrypoint.sh")
	execArgs = append(execArgs, spec.Args...)

	podmanLogger.Debugf("Executing shell in persistent container: podman %v", execArgs)
	return p.executePodmanCommand(execArgs)
}

// addSecuritySettings adds container security hardening options
func (p *PodmanProvider) addSecuritySettings(podmanArgs []string) []string {
	sec := p.config.Security

	// Process limits
	if sec.PidsLimit > 0 {
		podmanArgs = append(podmanArgs, "--pids-limit", fmt.Sprintf("%d", sec.PidsLimit))
	}

	// Ulimits
	if sec.UlimitNofile != "" {
		podmanArgs = append(podmanArgs, "--ulimit", "nofile="+sec.UlimitNofile)
	}
	if sec.UlimitNproc != "" {
		podmanArgs = append(podmanArgs, "--ulimit", "nproc="+sec.UlimitNproc)
	}

	// Privilege escalation prevention
	if sec.NoNewPrivileges {
		podmanArgs = append(podmanArgs, "--security-opt", "no-new-privileges")
	}

	// Drop capabilities
	for _, cap := range sec.CapDrop {
		podmanArgs = append(podmanArgs, "--cap-drop", cap)
	}

	// Add capabilities back
	for _, cap := range sec.CapAdd {
		podmanArgs = append(podmanArgs, "--cap-add", cap)
	}

	// Read-only root filesystem
	if sec.ReadOnlyRootfs {
		podmanArgs = append(podmanArgs, "--read-only")
		// Add tmpfs mounts for writable directories when using read-only rootfs
		podmanArgs = append(podmanArgs, "--tmpfs", fmt.Sprintf("/tmp:rw,noexec,nosuid,size=%s", sec.TmpfsTmpSize))
		podmanArgs = append(podmanArgs, "--tmpfs", "/var/tmp:rw,noexec,nosuid,size=128m")
		// Home dir needs exec (npm installs executables there).
		// Podman doesn't support uid/gid tmpfs params, so use mode=1777 instead.
		podmanArgs = append(podmanArgs, "--tmpfs", fmt.Sprintf("/home/addt:rw,exec,nosuid,mode=1777,size=%s", sec.TmpfsHomeSize))
	}

	// Seccomp profile
	if sec.SeccompProfile != "" {
		switch sec.SeccompProfile {
		case "unconfined":
			podmanArgs = append(podmanArgs, "--security-opt", "seccomp=unconfined")
		case "restrictive":
			// Write embedded restrictive profile to temp file with restrictive permissions
			profilePath := filepath.Join(os.TempDir(), "addt-seccomp-restrictive.json")
			if err := os.WriteFile(profilePath, assets.SeccompRestrictive, 0600); err == nil {
				podmanArgs = append(podmanArgs, "--security-opt", "seccomp="+profilePath)
			}
		case "default":
			// Use Podman's default profile, no flag needed
		default:
			// Custom profile path
			podmanArgs = append(podmanArgs, "--security-opt", "seccomp="+sec.SeccompProfile)
		}
	}

	// Network mode (none = completely isolated, no network access)
	// Note: If firewall with pasta is enabled, skip network mode override
	if sec.NetworkMode != "" && !p.config.FirewallEnabled {
		podmanArgs = append(podmanArgs, "--network", sec.NetworkMode)
	}

	// IPC namespace isolation
	if sec.DisableIPC {
		podmanArgs = append(podmanArgs, "--ipc", "private")
	}

	// Time limit - pass as env var for entrypoint to enforce with timeout command
	if sec.TimeLimit > 0 {
		podmanArgs = append(podmanArgs, "-e", fmt.Sprintf("ADDT_TIME_LIMIT_SECONDS=%d", sec.TimeLimit*60))
	}

	// User namespace remapping - Podman uses userns differently
	if sec.UserNamespace != "" {
		podmanArgs = append(podmanArgs, "--userns", sec.UserNamespace)
	}

	// Block mknod capability (prevents creating device files)
	if sec.DisableDevices {
		podmanArgs = append(podmanArgs, "--cap-drop", "MKNOD")
	}

	// Memory swap limit (-1 = disable swap entirely)
	if sec.MemorySwap != "" {
		podmanArgs = append(podmanArgs, "--memory-swap", sec.MemorySwap)
	}

	return podmanArgs
}
