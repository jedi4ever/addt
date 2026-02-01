package cmd

import (
	"fmt"

	"github.com/jedi4ever/dclaude/assets"
	"github.com/jedi4ever/dclaude/provider"
	"github.com/jedi4ever/dclaude/provider/daytona"
	"github.com/jedi4ever/dclaude/provider/docker"
)

// NewProvider creates a new provider based on the specified type
func NewProvider(providerType string, cfg *provider.Config) (provider.Provider, error) {
	switch providerType {
	case "docker", "":
		return docker.NewDockerProvider(cfg, assets.DockerDockerfile, assets.DockerEntrypoint)
	case "daytona":
		return daytona.NewDaytonaProvider(cfg, assets.DaytonaDockerfile, assets.DaytonaEntrypoint)
	default:
		return nil, fmt.Errorf("unknown provider type: %s (supported: docker, daytona)", providerType)
	}
}
