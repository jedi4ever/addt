package core

import (
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/jedi4ever/addt/provider"
)

// IsPortAvailable checks if a port is available for binding
func IsPortAvailable(port int) bool {
	addr := fmt.Sprintf("localhost:%d", port)
	conn, err := net.DialTimeout("tcp", addr, time.Second)
	if err != nil {
		return true
	}
	conn.Close()
	return false
}

// FindAvailablePort finds the next available port starting from startPort
func FindAvailablePort(startPort int) int {
	port := startPort
	for !IsPortAvailable(port) {
		port++
	}
	return port
}

// BuildPorts creates port mappings from the configuration
// It finds available host ports starting from PortRangeStart
func BuildPorts(cfg *provider.Config) []provider.PortMapping {
	var portsList []provider.PortMapping
	hostPort := cfg.PortRangeStart

	for _, containerPort := range cfg.Ports {
		containerPort = strings.TrimSpace(containerPort)
		hostPort = FindAvailablePort(hostPort)

		// Parse container port as int
		var containerPortInt int
		fmt.Sscanf(containerPort, "%d", &containerPortInt)

		portsList = append(portsList, provider.PortMapping{
			Container: containerPortInt,
			Host:      hostPort,
		})
		hostPort++
	}

	return portsList
}

// BuildPortMapString creates a comma-separated port map string
// Format: "containerPort:hostPort,containerPort:hostPort"
// This is used for the ADDT_PORT_MAP environment variable
func BuildPortMapString(cfg *provider.Config) string {
	if len(cfg.Ports) == 0 {
		return ""
	}

	var mappings []string
	hostPort := cfg.PortRangeStart

	for _, containerPort := range cfg.Ports {
		containerPort = strings.TrimSpace(containerPort)
		hostPort = FindAvailablePort(hostPort)
		mappings = append(mappings, fmt.Sprintf("%s:%d", containerPort, hostPort))
		hostPort++
	}

	return strings.Join(mappings, ",")
}

// BuildPortDisplayString creates a display-friendly port mapping string
// Format: "containerPort→hostPort,containerPort→hostPort"
func BuildPortDisplayString(cfg *provider.Config) string {
	if len(cfg.Ports) == 0 {
		return ""
	}

	var mappings []string
	hostPort := cfg.PortRangeStart

	for _, containerPort := range cfg.Ports {
		containerPort = strings.TrimSpace(containerPort)
		hostPort = FindAvailablePort(hostPort)
		mappings = append(mappings, fmt.Sprintf("%s→%d", containerPort, hostPort))
		hostPort++
	}

	return strings.Join(mappings, ",")
}
