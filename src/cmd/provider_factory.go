package cmd

import (
	"fmt"

	"github.com/jedi4ever/addt/assets"
	"github.com/jedi4ever/addt/config"
	"github.com/jedi4ever/addt/extensions"
	"github.com/jedi4ever/addt/provider"
	"github.com/jedi4ever/addt/provider/bwrap"
	"github.com/jedi4ever/addt/provider/daytona"
	"github.com/jedi4ever/addt/provider/docker"
	"github.com/jedi4ever/addt/provider/orbstack"
	"github.com/jedi4ever/addt/provider/podman"
)

// NewProvider creates a new provider based on the specified type
// For podman/default, auto-downloads Podman if not available
func NewProvider(providerType string, cfg *provider.Config) (provider.Provider, error) {
	// Bwrap and daytona don't need a container runtime
	if providerType != "daytona" && providerType != "bwrap" {
		runtime, err := config.EnsureContainerRuntime()
		if err != nil {
			return nil, err
		}
		// Update provider type if it was auto-detected/downloaded
		if providerType == "" {
			providerType = runtime
		}
	}

	switch providerType {
	case "docker":
		return docker.NewDockerProvider(cfg, "desktop-linux", assets.DockerDockerfile, assets.DockerDockerfileBase, assets.DockerEntrypoint, assets.DockerInitFirewall, assets.DockerInstallSh, extensions.FS)
	case "rancher":
		return docker.NewDockerProvider(cfg, "rancher-desktop", assets.DockerDockerfile, assets.DockerDockerfileBase, assets.DockerEntrypoint, assets.DockerInitFirewall, assets.DockerInstallSh, extensions.FS)
	case "orbstack":
		return orbstack.NewOrbStackProvider(cfg, assets.OrbStackDockerfile, assets.OrbStackDockerfileBase, assets.OrbStackEntrypoint, assets.OrbStackInitFirewall, assets.OrbStackInstallSh, extensions.FS)
	case "podman", "":
		return podman.NewPodmanProvider(cfg, assets.PodmanDockerfile, assets.PodmanDockerfileBase, assets.PodmanEntrypoint, assets.PodmanInitFirewall, assets.PodmanInstallSh, extensions.FS)
	case "bwrap":
		return bwrap.NewBwrapProvider(cfg)
	case "daytona":
		return daytona.NewDaytonaProvider(cfg, assets.DaytonaDockerfile, assets.DaytonaEntrypoint)
	default:
		return nil, fmt.Errorf("unknown provider type: %s (supported: docker, rancher, podman, orbstack, bwrap, daytona)", providerType)
	}
}
