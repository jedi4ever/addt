package assets

import (
	_ "embed"
)

// Embedded assets organized by provider

// Docker provider assets
//
//go:embed docker/Dockerfile
var DockerDockerfile []byte

//go:embed docker/docker-entrypoint.sh
var DockerEntrypoint []byte

// Daytona provider assets
//
//go:embed daytona/Dockerfile
var DaytonaDockerfile []byte

//go:embed daytona/daytona-entrypoint.sh
var DaytonaEntrypoint []byte
