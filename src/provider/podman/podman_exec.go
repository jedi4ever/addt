package podman

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"

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
func (p *PodmanProvider) addContainerVolumesAndEnv(podmanArgs []string, spec *provider.RunSpec, ctx *containerContext) []string {
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

	// Firewall configuration
	if p.config.FirewallEnabled {
		// Requires NET_ADMIN capability for iptables
		podmanArgs = append(podmanArgs, "--cap-add", "NET_ADMIN")

		// Mount firewall config directory
		firewallConfigDir := filepath.Join(ctx.homeDir, ".addt", "firewall")
		if _, err := os.Stat(firewallConfigDir); err == nil {
			podmanArgs = append(podmanArgs, "-v", fmt.Sprintf("%s:/home/%s/.addt/firewall", firewallConfigDir, ctx.username))
		}
	}

	// Docker/Podman forwarding
	podmanArgs = append(podmanArgs, p.HandleDockerForwarding(spec.DindMode, spec.Name)...)

	// Add ports
	for _, port := range spec.Ports {
		podmanArgs = append(podmanArgs, "-p", fmt.Sprintf("%d:%d", port.Host, port.Container))
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

	return podmanArgs
}

// executePodmanCommand runs the podman command with standard I/O
func (p *PodmanProvider) executePodmanCommand(podmanArgs []string) error {
	cmd := exec.Command(GetPodmanPath(), podmanArgs...)
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
	if !ctx.useExistingContainer {
		podmanArgs = p.addContainerVolumesAndEnv(podmanArgs, spec, ctx)
	}

	// Handle shell mode or normal mode
	if ctx.useExistingContainer {
		podmanArgs = append(podmanArgs, spec.Name)
		// Call entrypoint with args for existing containers
		podmanArgs = append(podmanArgs, "/usr/local/bin/docker-entrypoint.sh")
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
	if !ctx.useExistingContainer {
		podmanArgs = p.addContainerVolumesAndEnv(podmanArgs, spec, ctx)
	}

	// Open shell
	fmt.Println("Opening bash shell in container...")
	if ctx.useExistingContainer {
		podmanArgs = append(podmanArgs, spec.Name, "/bin/bash")
		podmanArgs = append(podmanArgs, spec.Args...)
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
			podmanArgs = append(podmanArgs, "--entrypoint", "/bin/bash", spec.ImageName, "-c", script, "bash")
			podmanArgs = append(podmanArgs, spec.Args...)
		} else {
			podmanArgs = append(podmanArgs, "--entrypoint", "/bin/bash", spec.ImageName)
			podmanArgs = append(podmanArgs, spec.Args...)
		}
	}

	return p.executePodmanCommand(podmanArgs)
}
