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

	// Mount .gitconfig
	gitconfigPath := fmt.Sprintf("%s/.gitconfig", ctx.homeDir)
	if _, err := os.Stat(gitconfigPath); err == nil {
		dockerArgs = append(dockerArgs, "-v", fmt.Sprintf("%s:/home/%s/.gitconfig:ro", gitconfigPath, ctx.username))
	}

	// Note: Claude config mounts (~/.claude, ~/.claude.json) are now handled
	// by the claude extension via AddExtensionMounts above.
	// Use ADDT_MOUNT_CLAUDE_CONFIG=false to disable them.

	// Add env file if exists (skip when isolate_secrets is on — values are
	// already in spec.Env and will go through the secrets file instead)
	if spec.Env["ADDT_ENV_FILE"] != "" && !p.config.Security.IsolateSecrets {
		dockerArgs = append(dockerArgs, "--env-file", spec.Env["ADDT_ENV_FILE"])
	}

	// SSH forwarding
	dockerArgs = append(dockerArgs, p.HandleSSHForwarding(spec.SSHForwardKeys, spec.SSHForwardMode, ctx.homeDir, ctx.username, spec.SSHAllowedKeys)...)

	// GPG forwarding
	dockerArgs = append(dockerArgs, p.HandleGPGForwarding(spec.GPGForward, ctx.homeDir, ctx.username, spec.GPGAllowedKeyIDs)...)

	// Tmux forwarding
	dockerArgs = append(dockerArgs, p.HandleTmuxForwarding(spec.TmuxForward)...)

	// History persistence
	dockerArgs = append(dockerArgs, p.HandleHistoryPersist(spec.HistoryPersist, spec.WorkDir, ctx.username)...)

	// Firewall configuration
	if p.config.FirewallEnabled {
		// Requires NET_ADMIN capability for iptables
		dockerArgs = append(dockerArgs, "--cap-add", "NET_ADMIN")

		// Mount firewall config directory
		firewallConfigDir := filepath.Join(ctx.homeDir, ".addt", "firewall")
		if _, err := os.Stat(firewallConfigDir); err == nil {
			dockerArgs = append(dockerArgs, "-v", fmt.Sprintf("%s:/home/%s/.addt/firewall", firewallConfigDir, ctx.username))
		}
	}

	// Docker forwarding
	dockerArgs = append(dockerArgs, p.HandleDockerForwarding(spec.DockerDindMode, spec.Name)...)

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
	interactive := false
	for _, arg := range baseArgs {
		switch arg {
		case "-it":
			interactive = true
		case "-i":
			interactive = true
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
	if interactive {
		execArgs = append(execArgs, "-it")
	} else {
		execArgs = append(execArgs, "-i")
	}
	execArgs = append(execArgs, spec.Name, "/usr/local/bin/docker-entrypoint.sh")
	execArgs = append(execArgs, spec.Args...)

	dockerLogger.Debugf("Executing entrypoint in persistent container: docker %v", execArgs)
	return p.executeDockerCommand(execArgs)
}

// runWithSecrets starts a container, copies secrets, then runs entrypoint
func (p *DockerProvider) runWithSecrets(baseArgs []string, spec *provider.RunSpec, secretsJSON string) error {
	// For detached start, we need to preserve -t (TTY) but remove -i (stdin)
	// This ensures attach will have a terminal, but detached start won't wait for stdin
	var runArgs []string
	interactive := false
	needsTTY := false
	for i := 0; i < len(baseArgs); i++ {
		arg := baseArgs[i]
		if arg == "-it" {
			interactive = true
			needsTTY = true
			// Add -t but not -i for detached run - TTY needed for attach, but no stdin during detached start
			runArgs = append(runArgs, "-t")
		} else if arg == "-i" {
			interactive = true
			// Don't add -i for detached run - we'll handle stdin on attach
		} else if arg == "-t" {
			interactive = true
			needsTTY = true
			// Keep -t for detached run - needed for terminal on attach
			runArgs = append(runArgs, "-t")
		} else {
			runArgs = append(runArgs, arg)
		}
	}
	dockerLogger.Debugf("Starting detached container (interactive=%v, needsTTY=%v), removed -i flag but kept -t", interactive, needsTTY)

	// Start container detached, waiting for secrets
	runArgs = append(runArgs, "-d")
	runArgs = append(runArgs, "--entrypoint", "/bin/sh")
	runArgs = append(runArgs, spec.ImageName)
	// Wait loop: check for .secrets file or .ready signal
	// Add echo to stderr so we can see the wait loop is running (helps debug)
	waitScript := `echo '[WAIT] Waiting for secrets...' >&2
debug_log() {
    if [ "${ADDT_LOG_LEVEL:-INFO}" = "DEBUG" ]; then
        echo "[$(date '+%Y-%m-%d %H:%M:%S')] [WAIT-DEBUG] $*" >&2
    fi
}
debug_log "Starting wait loop for /run/secrets/.secrets"
iter=0
while [ ! -f /run/secrets/.secrets ] && [ ! -f /run/secrets/.ready ]; do
    iter=$((iter + 1))
    # Check file existence and log the result
    if [ -f /run/secrets/.secrets ]; then
        debug_log "Check result: /run/secrets/.secrets EXISTS (iteration $iter)"
    else
        debug_log "Check result: /run/secrets/.secrets NOT FOUND (iteration $iter)"
    fi
    if [ -f /run/secrets/.ready ]; then
        debug_log "Check result: /run/secrets/.ready EXISTS (iteration $iter)"
    else
        debug_log "Check result: /run/secrets/.ready NOT FOUND (iteration $iter)"
    fi
    if [ $((iter % 20)) -eq 0 ]; then
        debug_log "Still waiting... (iteration $iter)"
        debug_log "Directory exists: $([ -d /run/secrets ] && echo yes || echo no)"
        debug_log "Directory readable: $([ -r /run/secrets ] && echo yes || echo no)"
        debug_log "Directory listing: $(ls -la /run/secrets 2>&1 | head -5)"
        debug_log "Test command result: test -f /run/secrets/.secrets returns $([ -f /run/secrets/.secrets ] && echo 0 || echo 1)"
    fi
    sleep 0.05
done
if [ -f /run/secrets/.secrets ]; then
    debug_log "Found /run/secrets/.secrets, file size: $(stat -c%s /run/secrets/.secrets 2>/dev/null || echo unknown)"
elif [ -f /run/secrets/.ready ]; then
    debug_log "Found /run/secrets/.ready signal"
fi
echo '[WAIT] Secrets found, starting entrypoint...' >&2
exec /usr/local/bin/docker-entrypoint.sh "$@"`
	runArgs = append(runArgs, "-c", waitScript, "--")
	runArgs = append(runArgs, spec.Args...)

	// Start the container
	cmd := exec.Command("docker", runArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to start container: %w\n%s", err, string(output))
	}

	// Copy secrets to container
	if err := p.copySecretsToContainer(spec.Name, secretsJSON); err != nil {
		// Cleanup: stop and remove container (it's waiting for secrets that will never arrive)
		// The container is stuck in a wait loop, so we need to clean it up
		dockerLogger.Debugf("Failed to copy secrets, cleaning up container %s", spec.Name)
		if rmErr := exec.Command("docker", "rm", "-f", spec.Name).Run(); rmErr != nil {
			dockerLogger.Debugf("Warning: failed to remove container %s during cleanup: %v", spec.Name, rmErr)
		}
		return fmt.Errorf("failed to copy secrets: %w", err)
	}

	// Wait for container to process secrets and start entrypoint
	// The wait loop in the container checks for .secrets file every 0.05s
	// Try to fetch debug log file from container (entrypoint writes to /tmp/addt-entrypoint-debug.log)
	dockerLogger.Debug("Waiting for container to process secrets and start entrypoint")
	var debugLogOutput []byte
	cpCmdStr := fmt.Sprintf("docker cp %s:/tmp/addt-entrypoint-debug.log", spec.Name)
	dockerLogger.Debugf("Fetching debug log with command: %s", cpCmdStr)

	// Retry copying debug log file for up to 4 seconds
	for i := 0; i < 40; i++ {
		time.Sleep(100 * time.Millisecond)
		dockerLogger.Debugf("Attempt %d/%d: Fetching debug log", i+1, 40)
		content, err := p.copyDebugLogFromContainer(spec.Name)
		if err != nil {
			dockerLogger.Debugf("Debug log copy error (attempt %d): %v", i+1, err)
		} else if len(content) > 0 {
			dockerLogger.Debugf("Found debug log on attempt %d (%d bytes)", i+1, len(content))
			debugLogOutput = content
			break
		} else {
			dockerLogger.Debugf("No debug log yet (attempt %d) - entrypoint may not have started", i+1)
		}
	}

	// Output debug log to stderr so user can see entrypoint output even when detached
	if len(debugLogOutput) > 0 {
		fmt.Fprintf(os.Stderr, "\n--- Entrypoint debug log (before attach) ---\n%s--- End of entrypoint debug log ---\n\n", string(debugLogOutput))
		dockerLogger.Debugf("Entrypoint debug log (before attach):\n%s", string(debugLogOutput))
	} else {
		dockerLogger.Debug("No debug log available yet - container may still be starting or ADDT_LOG_LEVEL is not DEBUG")
		// Also try to verify container exists and is running
		if p.Exists(spec.Name) {
			if p.IsRunning(spec.Name) {
				dockerLogger.Debug("Container exists and is running, but no debug log yet - entrypoint may not have started or ADDT_LOG_LEVEL is not DEBUG")
			} else {
				dockerLogger.Debug("Container exists but is not running")
			}
		} else {
			dockerLogger.Debug("Container does not exist")
		}
	}

	// Verify container is running before attaching
	if !p.IsRunning(spec.Name) {
		dockerLogger.Debugf("Container %s is not running, checking status", spec.Name)
		// Check if container exists and what state it's in
		if p.Exists(spec.Name) {
			return fmt.Errorf("container %s exists but is not running", spec.Name)
		}
		return fmt.Errorf("container %s does not exist", spec.Name)
	}
	dockerLogger.Debugf("Container %s is running, proceeding with attach", spec.Name)

	// Attach to container
	// Note: attach doesn't take -i flag; stdin is already configured from container start
	attachArgs := []string{"attach", spec.Name}
	dockerLogger.Debugf("Attaching to container with args: %v (interactive=%v)", attachArgs, interactive || spec.Interactive)

	return p.executeDockerCommand(attachArgs)
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
			// Create initialization script that runs before bash
			script := `
# Initialize firewall if enabled
if [ "${ADDT_FIREWALL_ENABLED}" = "true" ] && [ -f /usr/local/bin/init-firewall.sh ]; then
    sudo /usr/local/bin/init-firewall.sh
fi

# Start Docker daemon if in DinD mode
if [ "$ADDT_DOCKER_DIND_ENABLE" = "true" ]; then
    echo 'Starting Docker daemon in isolated mode...'
    sudo dockerd --host=unix:///var/run/docker.sock >/tmp/docker.log 2>&1 &
    echo 'Waiting for Docker daemon...'
    for i in $(seq 1 30); do
        if [ -S /var/run/docker.sock ]; then
            sudo chmod 666 /var/run/docker.sock
            if docker info >/dev/null 2>&1; then
                echo '✓ Docker daemon ready (isolated environment)'
                break
            fi
        fi
        sleep 1
    done
fi

exec /bin/bash "$@"
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
	interactive := false
	for _, arg := range baseArgs {
		switch arg {
		case "-it":
			interactive = true
		case "-i":
			interactive = true
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
	if interactive {
		execArgs = append(execArgs, "-it")
	} else {
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
		dockerArgs = append(dockerArgs, "--tmpfs", fmt.Sprintf("/home/addt:rw,noexec,nosuid,size=%s", sec.TmpfsHomeSize))
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
