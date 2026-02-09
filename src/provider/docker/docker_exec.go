package docker

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

var dockerLogger = util.Log("docker")

// containerContext holds common container setup information
type containerContext struct {
	homeDir              string
	username             string
	useExistingContainer bool
}

// setupContainerContext prepares common container context and checks for existing containers
func (p *DockerProvider) setupContainerContext(spec *provider.RunSpec) (*containerContext, error) {
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

// buildBaseDockerArgs creates the base docker arguments for run/exec commands
func (p *DockerProvider) buildBaseDockerArgs(spec *provider.RunSpec, ctx *containerContext) []string {
	var dockerArgs []string

	if ctx.useExistingContainer {
		dockerArgs = []string{"exec"}
	} else {
		if spec.Persistent {
			dockerArgs = []string{"run", "--name", spec.Name}
		} else {
			dockerArgs = []string{"run", "--rm", "--name", spec.Name}
		}
	}

	// Interactive mode
	if spec.Interactive {
		dockerArgs = append(dockerArgs, "-it")
		if !ctx.useExistingContainer {
			dockerArgs = append(dockerArgs, "--init")
		}
	} else {
		dockerArgs = append(dockerArgs, "-i")
	}

	return dockerArgs
}

// addContainerVolumesAndEnv adds volumes, mounts, and environment variables for new containers
func (p *DockerProvider) addContainerVolumesAndEnv(dockerArgs []string, spec *provider.RunSpec, ctx *containerContext) ([]string, func()) {
	// Cleanup function for secrets directory (caller should defer this)
	cleanup := func() {}
	// Add volumes
	for _, vol := range spec.Volumes {
		mount := fmt.Sprintf("%s:%s", vol.Source, vol.Target)
		if vol.ReadOnly {
			mount += ":ro"
		}
		dockerArgs = append(dockerArgs, "-v", mount)
	}

	// Add extension mounts
	dockerArgs = p.AddExtensionMounts(dockerArgs, spec.ImageName, ctx.homeDir)

	// Mount .gitconfig (if forwarding enabled)
	if p.config.GitForwardConfig {
		gitconfigPath := p.config.GitConfigPath
		if gitconfigPath == "" {
			gitconfigPath = filepath.Join(ctx.homeDir, ".gitconfig")
		} else {
			gitconfigPath = util.ExpandTilde(gitconfigPath)
		}
		if _, err := os.Stat(gitconfigPath); err == nil {
			dockerArgs = append(dockerArgs, "-v", fmt.Sprintf("%s:/home/%s/.gitconfig.host:ro", gitconfigPath, ctx.username))
		}
	}

	// Note: Claude config mounts (~/.claude, ~/.claude.json) are now handled
	// by the claude extension via AddExtensionMounts above.
	// Use ADDT_MOUNT_CLAUDE_CONFIG=false to disable them.

	// Env file vars are loaded into spec.Env by BuildRunOptions (see core/options.go)
	// so they go through the same -e mechanism as other env vars.

	// SSH forwarding
	sshDir := p.config.SSHDir
	if sshDir == "" {
		sshDir = filepath.Join(ctx.homeDir, ".ssh")
	} else {
		sshDir = util.ExpandTilde(sshDir)
	}
	dockerArgs = append(dockerArgs, p.HandleSSHForwarding(spec.SSHForwardKeys, spec.SSHForwardMode, sshDir, ctx.username, spec.SSHAllowedKeys)...)

	// GPG forwarding
	gpgDir := p.config.GPGDir
	if gpgDir == "" {
		gpgDir = filepath.Join(ctx.homeDir, ".gnupg")
	} else {
		gpgDir = util.ExpandTilde(gpgDir)
	}
	dockerArgs = append(dockerArgs, p.HandleGPGForwarding(spec.GPGForward, gpgDir, ctx.username, spec.GPGAllowedKeyIDs)...)

	// Tmux forwarding
	dockerArgs = append(dockerArgs, p.HandleTmuxForwarding(spec.TmuxForward)...)

	// History persistence
	dockerArgs = append(dockerArgs, p.HandleHistoryPersist(spec.HistoryPersist, spec.WorkDir, ctx.username)...)

	// Firewall configuration
	if p.config.FirewallEnabled {
		// Start as root so entrypoint can apply iptables rules without sudo,
		// then drop to addt via gosu (compatible with no-new-privileges)
		dockerArgs = append(dockerArgs, "--user", "root")
		// Capabilities for the root phase (dropped after gosu switches to addt):
		// NET_ADMIN: iptables/nftables rules
		// DAC_OVERRIDE: create dirs/files in addt's home
		// CHOWN: fix file ownership
		// SETUID/SETGID: gosu needs these to switch from root to addt
		dockerArgs = append(dockerArgs, "--cap-add", "NET_ADMIN")
		dockerArgs = append(dockerArgs, "--cap-add", "DAC_OVERRIDE")
		dockerArgs = append(dockerArgs, "--cap-add", "CHOWN")
		dockerArgs = append(dockerArgs, "--cap-add", "SETUID")
		dockerArgs = append(dockerArgs, "--cap-add", "SETGID")

		// Mount firewall config directory
		addtHome := util.GetAddtHome()
		if addtHome != "" {
			firewallConfigDir := filepath.Join(addtHome, "firewall")
			if _, err := os.Stat(firewallConfigDir); err == nil {
				dockerArgs = append(dockerArgs, "-v", fmt.Sprintf("%s:/home/%s/.addt/firewall", firewallConfigDir, ctx.username))
			}
		}
	}

	// Docker forwarding (DinD)
	dindArgs := p.HandleDockerForwarding(spec.DockerDindMode, spec.Name)
	dockerArgs = append(dockerArgs, dindArgs...)

	// Start as root for DinD so entrypoint can start dockerd without sudo
	if spec.DockerDindMode == "isolated" || spec.DockerDindMode == "true" {
		dockerArgs = append(dockerArgs, "--user", "root")
	}

	// Add ports
	for _, port := range spec.Ports {
		dockerArgs = append(dockerArgs, "-p", fmt.Sprintf("%d:%d", port.Host, port.Container))
	}

	// Handle isolate_secrets: add tmpfs mount for secrets
	// Secrets will be copied via docker cp after container starts
	if p.config.Security.IsolateSecrets {
		dockerArgs = p.addTmpfsSecretsMount(dockerArgs)
	}

	// Handle OTEL: add host alias so container can reach host's OTEL collector
	if p.config.Otel.Enabled {
		dockerArgs = append(dockerArgs, "--add-host=host.docker.internal:host-gateway")
	}

	// Add environment variables
	for k, v := range spec.Env {
		dockerArgs = append(dockerArgs, "-e", fmt.Sprintf("%s=%s", k, v))
	}

	// Add resource limits
	if spec.ContainerCPUs != "" {
		dockerArgs = append(dockerArgs, "--cpus", spec.ContainerCPUs)
	}
	if spec.ContainerMemory != "" {
		dockerArgs = append(dockerArgs, "--memory", spec.ContainerMemory)
	}

	// Add security settings
	dockerArgs = p.addSecuritySettings(dockerArgs)

	return dockerArgs, cleanup
}

// executeDockerCommand runs the docker command with standard I/O
func (p *DockerProvider) executeDockerCommand(dockerArgs []string) error {
	dockerLogger.Debugf("Executing: docker %v", dockerArgs)
	cmd := exec.Command("docker", dockerArgs...)

	// Check if -it flag is present (fully interactive mode)
	hasItFlag := false
	hasIFlag := false
	isAttach := false
	for _, arg := range dockerArgs {
		if arg == "-it" {
			hasItFlag = true
			break
		}
		if arg == "-i" {
			hasIFlag = true
		}
		if arg == "attach" {
			isAttach = true
			dockerLogger.Debug("Detected attach command")
		}
	}
	dockerLogger.Debugf("Flag check: hasItFlag=%v, hasIFlag=%v, isAttach=%v", hasItFlag, hasIFlag, isAttach)

	if hasItFlag {
		// Fully interactive: connect to terminal stdin
		cmd.Stdin = os.Stdin
		dockerLogger.Debug("Connecting stdin to terminal (interactive mode with -it)")
	} else if hasIFlag {
		// Has -i but not -it: still connect to terminal stdin for interactive commands
		// This allows commands like "addt run claude" to receive input
		cmd.Stdin = os.Stdin
		dockerLogger.Debug("Connecting stdin to terminal (interactive mode with -i)")
	} else if isAttach {
		// Attach command: connect stdin (container was started with -i, so attach inherits it)
		cmd.Stdin = os.Stdin
		dockerLogger.Debug("Connecting stdin to terminal (attach command)")
	} else {
		// No -i flag: don't connect stdin
		cmd.Stdin = nil
		dockerLogger.Debug("Not connecting stdin (no -i flag)")
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	dockerLogger.Debug("Starting docker command execution")
	err := cmd.Run()
	if err != nil {
		dockerLogger.Debugf("Docker command failed: %v", err)
	} else {
		dockerLogger.Debug("Docker command completed successfully")
	}
	return err
}

// Run runs a new container
func (p *DockerProvider) Run(spec *provider.RunSpec) error {
	ctx, err := p.setupContainerContext(spec)
	if err != nil {
		return err
	}

	// Prepare secrets if enabled (before building args so we can filter env)
	var secretsJSON string
	if p.config.Security.IsolateSecrets && !ctx.useExistingContainer {
		json, secretVarNames, err := p.prepareSecretsJSON(spec.ImageName, spec.Env)
		if err == nil && json != "" {
			secretsJSON = json
			p.filterSecretEnvVars(spec.Env, secretVarNames)
			// ADDT_CREDENTIAL_VARS is no longer needed — secrets are in the file
			delete(spec.Env, "ADDT_CREDENTIAL_VARS")
		}
	}

	dockerArgs := p.buildBaseDockerArgs(spec, ctx)

	// Only add volumes and environment when creating a new container
	cleanup := func() {}
	if !ctx.useExistingContainer {
		dockerArgs, cleanup = p.addContainerVolumesAndEnv(dockerArgs, spec, ctx)
	}
	defer cleanup()

	// Handle existing container
	if ctx.useExistingContainer {
		dockerArgs = append(dockerArgs, spec.Name)
		dockerArgs = append(dockerArgs, "/usr/local/bin/docker-entrypoint.sh")
		dockerArgs = append(dockerArgs, spec.Args...)
		return p.executeDockerCommand(dockerArgs)
	}

	// New persistent container: detached keep-alive + exec entrypoint
	if spec.Persistent {
		return p.runPersistent(dockerArgs, spec, secretsJSON)
	}

	// New container with secrets: use two-step process
	// 1. Start container detached with wait script
	// 2. Copy secrets via docker cp
	// 3. Signal container to continue and attach
	if secretsJSON != "" {
		return p.runWithSecrets(dockerArgs, spec, secretsJSON)
	}

	// Normal run without secrets
	dockerArgs = append(dockerArgs, spec.ImageName)
	dockerArgs = append(dockerArgs, spec.Args...)
	return p.executeDockerCommand(dockerArgs)
}

// runPersistent creates a persistent container with sleep infinity as PID 1,
// then execs the entrypoint. This ensures the container stays alive after
// the agent exits, so subsequent runs can reuse it via docker exec.
func (p *DockerProvider) runPersistent(baseArgs []string, spec *provider.RunSpec, secretsJSON string) error {
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
	dockerLogger.Debugf("Starting persistent container: docker %v", runArgs)

	cmd := exec.Command("docker", runArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to start persistent container: %w\n%s", err, string(output))
	}

	// Copy secrets if needed
	if secretsJSON != "" {
		dockerLogger.Debug("Copying secrets to persistent container")
		if err := p.copySecretsToContainer(spec.Name, secretsJSON); err != nil {
			dockerLogger.Debugf("Failed to copy secrets, cleaning up container %s", spec.Name)
			exec.Command("docker", "rm", "-f", spec.Name).Run()
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
	execArgs = append(execArgs, spec.Name, "/usr/local/bin/docker-entrypoint.sh")
	execArgs = append(execArgs, spec.Args...)

	dockerLogger.Debugf("Executing entrypoint in persistent container: docker %v", execArgs)
	return p.executeDockerCommand(execArgs)
}

// runWithSecrets starts a container, copies secrets, then execs the entrypoint.
// Uses a simple approach: start with sleep, copy secrets, exec entrypoint.
// Entrypoint output goes directly to terminal via exec (no attach needed).
func (p *DockerProvider) runWithSecrets(baseArgs []string, spec *provider.RunSpec, secretsJSON string) error {
	// Strip interactive flags from run args — they'll be added to exec instead.
	// The detached sleep process doesn't need stdin or TTY.
	// Track -i vs -it separately: Docker requires a real TTY for -it.
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
	dockerLogger.Debugf("Starting detached container: docker %v", runArgs)

	cmd := exec.Command("docker", runArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to start container: %w\n%s", err, string(output))
	}

	// Copy secrets to container tmpfs
	dockerLogger.Debug("Copying secrets to container")
	if err := p.copySecretsToContainer(spec.Name, secretsJSON); err != nil {
		dockerLogger.Debugf("Failed to copy secrets, cleaning up container %s", spec.Name)
		exec.Command("docker", "rm", "-f", spec.Name).Run()
		return fmt.Errorf("failed to copy secrets: %w", err)
	}

	// Exec entrypoint — output goes directly to terminal
	// Use -it only when original run had TTY; otherwise just -i
	execArgs := []string{"exec"}
	if needsTTY {
		execArgs = append(execArgs, "-it")
	} else if needsStdin {
		execArgs = append(execArgs, "-i")
	}
	execArgs = append(execArgs, spec.Name, "/usr/local/bin/docker-entrypoint.sh")
	execArgs = append(execArgs, spec.Args...)

	dockerLogger.Debugf("Executing entrypoint: docker %v", execArgs)
	execErr := p.executeDockerCommand(execArgs)

	// On failure, dump container logs for debugging
	if execErr != nil {
		dockerLogger.Debugf("Entrypoint failed, fetching container logs for %s", spec.Name)
		if logsOutput, err := exec.Command("docker", "logs", spec.Name).CombinedOutput(); err == nil && len(logsOutput) > 0 {
			dockerLogger.Debugf("Container logs:\n%s", string(logsOutput))
		}
	}

	// Clean up non-persistent containers (stop sleep, triggers --rm if set)
	if !spec.Persistent {
		dockerLogger.Debugf("Removing non-persistent container %s", spec.Name)
		exec.Command("docker", "rm", "-f", spec.Name).Run()
	}

	return execErr
}

// Shell opens a shell in a container
func (p *DockerProvider) Shell(spec *provider.RunSpec) error {
	ctx, err := p.setupContainerContext(spec)
	if err != nil {
		return err
	}

	dockerArgs := p.buildBaseDockerArgs(spec, ctx)

	// Only add volumes and environment when creating a new container
	cleanup := func() {}
	if !ctx.useExistingContainer {
		dockerArgs, cleanup = p.addContainerVolumesAndEnv(dockerArgs, spec, ctx)
	}
	defer cleanup()

	// Open shell
	fmt.Println("Opening bash shell in container...")
	if ctx.useExistingContainer {
		// Run through entrypoint so init (socat, firewall, DinD) works
		dockerArgs = append(dockerArgs, "-e", "ADDT_COMMAND=/bin/bash")
		dockerArgs = append(dockerArgs, spec.Name, "/usr/local/bin/docker-entrypoint.sh")
		dockerArgs = append(dockerArgs, spec.Args...)
	} else if spec.Persistent {
		return p.shellPersistent(dockerArgs, spec, ctx)
	} else {
		// Override entrypoint to bash for shell mode
		// Need to handle firewall initialization and DinD initialization
		needsInit := spec.DockerDindMode == "isolated" || spec.DockerDindMode == "true" || p.config.FirewallEnabled

		if needsInit {
			// Start as root so init script can run privileged ops, then drop to addt
			dockerArgs = append(dockerArgs, "--user", "root")
			// Create initialization script that runs before bash
			script := `
# Initialize firewall if enabled
if [ "${ADDT_FIREWALL_ENABLED}" = "true" ] && [ -f /usr/local/bin/init-firewall.sh ]; then
    /usr/local/bin/init-firewall.sh
fi

# Start Docker daemon if in DinD mode
if [ "$ADDT_DOCKER_DIND_ENABLE" = "true" ]; then
    echo 'Starting Docker daemon in isolated mode...'
    dockerd --host=unix:///var/run/docker.sock >/tmp/docker.log 2>&1 &
    echo 'Waiting for Docker daemon...'
    for i in $(seq 1 30); do
        if [ -S /var/run/docker.sock ]; then
            chmod 666 /var/run/docker.sock
            if docker info >/dev/null 2>&1; then
                echo 'Docker daemon ready (isolated environment)'
                break
            fi
        fi
        sleep 1
    done
fi

exec gosu addt /bin/bash "$@"
`
			dockerArgs = append(dockerArgs, "--entrypoint", "/bin/bash", spec.ImageName, "-c", script, "bash")
			dockerArgs = append(dockerArgs, spec.Args...)
		} else {
			dockerArgs = append(dockerArgs, "--entrypoint", "/bin/bash", spec.ImageName)
			dockerArgs = append(dockerArgs, spec.Args...)
		}
	}

	return p.executeDockerCommand(dockerArgs)
}

// shellPersistent creates a persistent container with sleep infinity as PID 1,
// then execs the entrypoint with ADDT_COMMAND=/bin/bash for shell access.
func (p *DockerProvider) shellPersistent(baseArgs []string, spec *provider.RunSpec, ctx *containerContext) error {
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
	dockerLogger.Debugf("Starting persistent container for shell: docker %v", runArgs)

	cmd := exec.Command("docker", runArgs...)
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
	execArgs = append(execArgs, spec.Name, "/usr/local/bin/docker-entrypoint.sh")
	execArgs = append(execArgs, spec.Args...)

	dockerLogger.Debugf("Executing shell in persistent container: docker %v", execArgs)
	return p.executeDockerCommand(execArgs)
}

// addSecuritySettings adds container security hardening options
func (p *DockerProvider) addSecuritySettings(dockerArgs []string) []string {
	sec := p.config.Security

	// Process limits
	if sec.PidsLimit > 0 {
		dockerArgs = append(dockerArgs, "--pids-limit", fmt.Sprintf("%d", sec.PidsLimit))
	}

	// Ulimits
	if sec.UlimitNofile != "" {
		dockerArgs = append(dockerArgs, "--ulimit", "nofile="+sec.UlimitNofile)
	}
	if sec.UlimitNproc != "" {
		dockerArgs = append(dockerArgs, "--ulimit", "nproc="+sec.UlimitNproc)
	}

	// Privilege escalation prevention
	if sec.NoNewPrivileges {
		dockerArgs = append(dockerArgs, "--security-opt", "no-new-privileges")
	}

	// Drop capabilities
	for _, cap := range sec.CapDrop {
		dockerArgs = append(dockerArgs, "--cap-drop", cap)
	}

	// Add capabilities back
	for _, cap := range sec.CapAdd {
		dockerArgs = append(dockerArgs, "--cap-add", cap)
	}

	// Read-only root filesystem
	if sec.ReadOnlyRootfs {
		dockerArgs = append(dockerArgs, "--read-only")
		// Add tmpfs mounts for writable directories when using read-only rootfs
		dockerArgs = append(dockerArgs, "--tmpfs", fmt.Sprintf("/tmp:rw,noexec,nosuid,size=%s", sec.TmpfsTmpSize))
		dockerArgs = append(dockerArgs, "--tmpfs", "/var/tmp:rw,noexec,nosuid,size=128m")
		// Home dir needs exec (npm installs executables there) and uid/gid
		// so the non-root container user owns the tmpfs (Docker supports uid/gid)
		homeOpts := fmt.Sprintf("/home/addt:rw,exec,nosuid,size=%s", sec.TmpfsHomeSize)
		if u, err := user.Current(); err == nil {
			homeOpts = fmt.Sprintf("/home/addt:rw,exec,nosuid,uid=%s,gid=%s,size=%s", u.Uid, u.Gid, sec.TmpfsHomeSize)
		}
		dockerArgs = append(dockerArgs, "--tmpfs", homeOpts)
	}

	// Seccomp profile
	if sec.SeccompProfile != "" {
		switch sec.SeccompProfile {
		case "unconfined":
			dockerArgs = append(dockerArgs, "--security-opt", "seccomp=unconfined")
		case "restrictive":
			// Write embedded restrictive profile to temp file with restrictive permissions
			profilePath := filepath.Join(os.TempDir(), "addt-seccomp-restrictive.json")
			if err := os.WriteFile(profilePath, assets.SeccompRestrictive, 0600); err == nil {
				dockerArgs = append(dockerArgs, "--security-opt", "seccomp="+profilePath)
			}
		case "default":
			// Use Docker's default profile, no flag needed
		default:
			// Custom profile path
			dockerArgs = append(dockerArgs, "--security-opt", "seccomp="+sec.SeccompProfile)
		}
	}

	// Network mode (none = completely isolated, no network access)
	if sec.NetworkMode != "" {
		dockerArgs = append(dockerArgs, "--network", sec.NetworkMode)
	}

	// IPC namespace isolation
	if sec.DisableIPC {
		dockerArgs = append(dockerArgs, "--ipc", "none")
	}

	// Time limit - pass as env var for entrypoint to enforce with timeout command
	if sec.TimeLimit > 0 {
		dockerArgs = append(dockerArgs, "-e", fmt.Sprintf("ADDT_TIME_LIMIT_SECONDS=%d", sec.TimeLimit*60))
	}

	// User namespace remapping (requires Docker daemon config for "host")
	if sec.UserNamespace != "" {
		dockerArgs = append(dockerArgs, "--userns", sec.UserNamespace)
	}

	// Block mknod capability (prevents creating device files)
	if sec.DisableDevices {
		dockerArgs = append(dockerArgs, "--cap-drop", "MKNOD")
	}

	// Memory swap limit (-1 = disable swap entirely)
	if sec.MemorySwap != "" {
		dockerArgs = append(dockerArgs, "--memory-swap", sec.MemorySwap)
	}

	return dockerArgs
}
