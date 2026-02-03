package assets

import _ "embed"

// Embedded assets organized by provider

// Docker provider assets
//
//go:embed docker/Dockerfile
var DockerDockerfile []byte

//go:embed docker/Dockerfile.base
var DockerDockerfileBase []byte

//go:embed docker/docker-entrypoint.sh
var DockerEntrypoint []byte

//go:embed docker/init-firewall.sh
var DockerInitFirewall []byte

//go:embed docker/install.sh
var DockerInstallSh []byte

// Daytona provider assets
//
//go:embed daytona/Dockerfile
var DaytonaDockerfile []byte

//go:embed daytona/daytona-entrypoint.sh
var DaytonaEntrypoint []byte
