package podman

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
			p.Start(spec.Name)
			ctx.useExistingContainer = true
		}
	} else if spec.Persistent {
		fmt.Printf("Creating new persistent container: %s\n", spec.Name)
	}

	return ctx, nil
}

// buildBasePodmanArgs creates the base podman arguments for run/exec commands
func (p *PodmanProvider) buildBasePodmanArgs(spec *provider.RunSpec, ctx *containerContext) []string {
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
		if !ctx.useExistingContainer {
			podmanArgs = append(podmanArgs, "--init")
		}
	} else {
		podmanArgs = append(podmanArgs, "-i")
	}

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

	// Mount .gitconfig
	gitconfigPath := fmt.Sprintf("%s/.gitconfig", ctx.homeDir)
	if _, err := os.Stat(gitconfigPath); err == nil {
		podmanArgs = append(podmanArgs, "-v", fmt.Sprintf("%s:/home/%s/.gitconfig:ro", gitconfigPath, ctx.username))
	}

	// Add env file if exists
	if spec.Env["ADDT_ENV_FILE"] != "" {
		podmanArgs = append(podmanArgs, "--env-file", spec.Env["ADDT_ENV_FILE"])
	}

	// SSH forwarding
	podmanArgs = append(podmanArgs, p.HandleSSHForwarding(spec.SSHForward, ctx.homeDir, ctx.username)...)

	// GPG forwarding
	podmanArgs = append(podmanArgs, p.HandleGPGForwarding(spec.GPGForward, ctx.homeDir, ctx.username)...)

	// Firewall configuration with pasta network backend
	if p.config.FirewallEnabled {
		// Use pasta network backend for better firewall support in rootless mode
		// pasta handles network namespaces efficiently and supports filtering
		if p.CheckPastaAvailable() {
			podmanArgs = append(podmanArgs, "--network=pasta")
		}

		// Requires NET_ADMIN capability for iptables/nftables inside container
		podmanArgs = append(podmanArgs, "--cap-add", "NET_ADMIN")

		// Mount firewall config directory
		firewallConfigDir := filepath.Join(ctx.homeDir, ".addt", "firewall")
		if _, err := os.Stat(firewallConfigDir); err == nil {
			podmanArgs = append(podmanArgs, "-v", fmt.Sprintf("%s:/home/%s/.addt/firewall", firewallConfigDir, ctx.username))
		}
	}

	// Podman-in-Podman support (similar to DinD)
	podmanArgs = append(podmanArgs, p.HandlePodmanForwarding(spec.DindMode, spec.Name)...)

	// Add ports
	for _, port := range spec.Ports {
		podmanArgs = append(podmanArgs, "-p", fmt.Sprintf("%d:%d", port.Host, port.Container))
	}

	// Handle secrets_to_files: write extension env vars to files instead of passing as -e
	if p.config.Security.SecretsToFiles {
		secretsDir, secretVarNames, err := p.writeSecretsToFiles(spec.ImageName, spec.Env)
		if err == nil && secretsDir != "" {
			podmanArgs = p.addSecretsMount(podmanArgs, secretsDir)
			p.filterSecretEnvVars(spec.Env, secretVarNames)
			podmanArgs = append(podmanArgs, "-e", "ADDT_SECRETS_DIR=/run/secrets")
			// Set cleanup to remove secrets directory after container exits
			cleanup = func() { os.RemoveAll(secretsDir) }
		}
	}

	// Add environment variables
	for k, v := range spec.Env {
		podmanArgs = append(podmanArgs, "-e", fmt.Sprintf("%s=%s", k, v))
	}

	// Add resource limits
	if spec.CPUs != "" {
		podmanArgs = append(podmanArgs, "--cpus", spec.CPUs)
	}
	if spec.Memory != "" {
		podmanArgs = append(podmanArgs, "--memory", spec.Memory)
	}

	// Add security settings
	podmanArgs = p.addSecuritySettings(podmanArgs)

	return podmanArgs, cleanup
}

// executePodmanCommand runs the podman command with standard I/O
func (p *PodmanProvider) executePodmanCommand(podmanArgs []string) error {
	cmd := exec.Command("podman", podmanArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Run runs a new container
func (p *PodmanProvider) Run(spec *provider.RunSpec) error {
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

	// Handle shell mode or normal mode
	if ctx.useExistingContainer {
		podmanArgs = append(podmanArgs, spec.Name)
		// Call entrypoint with args for existing containers
		podmanArgs = append(podmanArgs, "/usr/local/bin/podman-entrypoint.sh")
		podmanArgs = append(podmanArgs, spec.Args...)
	} else {
		podmanArgs = append(podmanArgs, spec.ImageName)
		podmanArgs = append(podmanArgs, spec.Args...)
	}

	return p.executePodmanCommand(podmanArgs)
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
		podmanArgs = append(podmanArgs, spec.Name, "/bin/bash")
		podmanArgs = append(podmanArgs, spec.Args...)
	} else {
		// Override entrypoint to bash for shell mode
		// Need to handle firewall initialization and nested Podman initialization
		needsInit := spec.DindMode == "isolated" || spec.DindMode == "true" || p.config.FirewallEnabled

		if needsInit {
			// Create initialization script that runs before bash
			script := `
# Initialize firewall if enabled
if [ "${ADDT_FIREWALL_ENABLED}" = "true" ] && [ -f /usr/local/bin/init-firewall.sh ]; then
    sudo /usr/local/bin/init-firewall.sh
fi

# Start nested Podman if in isolated mode (Podman-in-Podman)
if [ "$ADDT_DIND" = "true" ]; then
    echo 'Nested Podman mode enabled'
    # Podman doesn't need a daemon, but we ensure socket is available
    if [ -S /run/podman/podman.sock ]; then
        echo '✓ Podman socket available'
    fi
fi

exec /bin/bash "$@"
`
			podmanArgs = append(podmanArgs, "--entrypoint", "/bin/bash", spec.ImageName, "-c", script, "bash")
			podmanArgs = append(podmanArgs, spec.Args...)
		} else {
			podmanArgs = append(podmanArgs, "--entrypoint", "/bin/bash", spec.ImageName)
			podmanArgs = append(podmanArgs, spec.Args...)
		}
	}

	return p.executePodmanCommand(podmanArgs)
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
		podmanArgs = append(podmanArgs, "--tmpfs", fmt.Sprintf("/home/addt:rw,noexec,nosuid,size=%s", sec.TmpfsHomeSize))
	}

	// Seccomp profile
	if sec.SeccompProfile != "" {
		switch sec.SeccompProfile {
		case "unconfined":
			podmanArgs = append(podmanArgs, "--security-opt", "seccomp=unconfined")
		case "restrictive":
			// Write embedded restrictive profile to temp file
			profilePath := filepath.Join(os.TempDir(), "addt-seccomp-restrictive.json")
			if err := os.WriteFile(profilePath, assets.SeccompRestrictive, 0644); err == nil {
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

// HandlePodmanForwarding configures Podman-in-Podman (nested containers) support
func (p *PodmanProvider) HandlePodmanForwarding(mode string, containerName string) []string {
	var args []string

	switch mode {
	case "isolated", "true":
		// Podman doesn't need a daemon, so "isolated" mode is simpler
		// We enable fuse-overlayfs and podman socket
		args = append(args,
			"-e", "ADDT_DIND=true",
			"--device", "/dev/fuse",
			"--security-opt", "label=disable",
		)
	case "host":
		// Share host's Podman socket (dangerous but useful for some workflows)
		podmanSocket := os.Getenv("XDG_RUNTIME_DIR")
		if podmanSocket == "" {
			podmanSocket = fmt.Sprintf("/run/user/%d", os.Getuid())
		}
		socketPath := filepath.Join(podmanSocket, "podman", "podman.sock")

		if _, err := os.Stat(socketPath); err == nil {
			args = append(args,
				"-v", fmt.Sprintf("%s:/run/podman/podman.sock", socketPath),
				"-e", "DOCKER_HOST=unix:///run/podman/podman.sock",
			)
		}
	}

	return args
}
