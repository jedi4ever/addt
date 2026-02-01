package ports

import (
	"fmt"
	"net"
	"strings"
	"time"
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

// Config represents basic configuration needed for port handling
type Config interface {
	GetPorts() []string
	GetPortRangeStart() int
}

// HandlePortMappings configures port mappings and returns mapping strings for display
// Note: This function appends Docker-specific "-p" flags. It's currently used only by Docker provider.
func HandlePortMappings(cfg Config, providerArgs *[]string) (string, string) {
	ports := cfg.GetPorts()
	if len(ports) == 0 {
		return "", ""
	}

	var portMappings []string
	hostPort := cfg.GetPortRangeStart()

	for _, containerPort := range ports {
		containerPort = strings.TrimSpace(containerPort)
		hostPort = FindAvailablePort(hostPort)

		*providerArgs = append(*providerArgs, "-p", fmt.Sprintf("%d:%s", hostPort, containerPort))
		portMappings = append(portMappings, fmt.Sprintf("%s:%d", containerPort, hostPort))
		hostPort++
	}

	portMapString := strings.Join(portMappings, ",")
	portMapDisplay := strings.ReplaceAll(portMapString, ":", "→")

	return portMapString, portMapDisplay
}
