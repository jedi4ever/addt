package cmd

import (
	"fmt"

	"github.com/jedi4ever/addt/assets"
	"github.com/jedi4ever/addt/extensions"
	"github.com/jedi4ever/addt/provider"
	"github.com/jedi4ever/addt/provider/daytona"
	"github.com/jedi4ever/addt/provider/docker"
)

// NewProvider creates a new provider based on the specified type
func NewProvider(providerType string, cfg *provider.Config) (provider.Provider, error) {
	switch providerType {
	case "docker", "":
		return docker.NewDockerProvider(cfg, assets.DockerDockerfile, assets.DockerDockerfileBase, assets.DockerEntrypoint, assets.DockerInitFirewall, assets.DockerInstallSh, extensions.FS)
	case "daytona":
		return daytona.NewDaytonaProvider(cfg, assets.DaytonaDockerfile, assets.DaytonaEntrypoint)
	default:
		return nil, fmt.Errorf("unknown provider type: %s (supported: docker, daytona)", providerType)
	}
}
