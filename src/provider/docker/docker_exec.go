package docker

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"

	"github.com/jedi4ever/addt/assets"
	"github.com/jedi4ever/addt/provider"
)

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
			p.Start(spec.Name)
			ctx.useExistingContainer = true
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

	// Add env file if exists
	if spec.Env["ADDT_ENV_FILE"] != "" {
		dockerArgs = append(dockerArgs, "--env-file", spec.Env["ADDT_ENV_FILE"])
	}

	// SSH forwarding
	dockerArgs = append(dockerArgs, p.HandleSSHForwarding(spec.SSHForward, ctx.homeDir, ctx.username, spec.SSHAllowedKeys)...)

	// GPG forwarding
	dockerArgs = append(dockerArgs, p.HandleGPGForwarding(spec.GPGForward, ctx.homeDir, ctx.username, spec.GPGAllowedKeyIDs)...)

	// Tmux forwarding
	dockerArgs = append(dockerArgs, p.HandleTmuxForwarding(spec.TmuxForward)...)

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
	dockerArgs = append(dockerArgs, p.HandleDockerForwarding(spec.DindMode, spec.Name)...)

	// Add ports
	for _, port := range spec.Ports {
		dockerArgs = append(dockerArgs, "-p", fmt.Sprintf("%d:%d", port.Host, port.Container))
	}

	// Handle secrets_to_files: write extension env vars to files instead of passing as -e
	if p.config.Security.SecretsToFiles {
		secretsDir, secretVarNames, err := p.writeSecretsToFiles(spec.ImageName, spec.Env)
		if err == nil && secretsDir != "" {
			dockerArgs = p.addSecretsMount(dockerArgs, secretsDir)
			p.filterSecretEnvVars(spec.Env, secretVarNames)
			dockerArgs = append(dockerArgs, "-e", "ADDT_SECRETS_DIR=/run/secrets")
			// Set cleanup to remove secrets directory after container exits
			cleanup = func() { os.RemoveAll(secretsDir) }
		}
	}

	// Add environment variables
	for k, v := range spec.Env {
		dockerArgs = append(dockerArgs, "-e", fmt.Sprintf("%s=%s", k, v))
	}

	// Add resource limits
	if spec.CPUs != "" {
		dockerArgs = append(dockerArgs, "--cpus", spec.CPUs)
	}
	if spec.Memory != "" {
		dockerArgs = append(dockerArgs, "--memory", spec.Memory)
	}

	// Add security settings
	dockerArgs = p.addSecuritySettings(dockerArgs)

	return dockerArgs, cleanup
}

// executeDockerCommand runs the docker command with standard I/O
func (p *DockerProvider) executeDockerCommand(dockerArgs []string) error {
	cmd := exec.Command("docker", dockerArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Run runs a new container
func (p *DockerProvider) Run(spec *provider.RunSpec) error {
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

	// Handle shell mode or normal mode
	if ctx.useExistingContainer {
		dockerArgs = append(dockerArgs, spec.Name)
		// Call entrypoint with args for existing containers
		dockerArgs = append(dockerArgs, "/usr/local/bin/docker-entrypoint.sh")
		dockerArgs = append(dockerArgs, spec.Args...)
	} else {
		dockerArgs = append(dockerArgs, spec.ImageName)
		dockerArgs = append(dockerArgs, spec.Args...)
	}

	return p.executeDockerCommand(dockerArgs)
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
		dockerArgs = append(dockerArgs, spec.Name, "/bin/bash")
		dockerArgs = append(dockerArgs, spec.Args...)
	} else {
		// Override entrypoint to bash for shell mode
		// Need to handle firewall initialization and DinD initialization
		needsInit := spec.DindMode == "isolated" || spec.DindMode == "true" || p.config.FirewallEnabled

		if needsInit {
			// Create initialization script that runs before bash
			script := `
# Initialize firewall if enabled
if [ "${ADDT_FIREWALL_ENABLED}" = "true" ] && [ -f /usr/local/bin/init-firewall.sh ]; then
    sudo /usr/local/bin/init-firewall.sh
fi

# Start Docker daemon if in DinD mode
if [ "$ADDT_DIND" = "true" ]; then
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
