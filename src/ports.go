package main

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

// HandlePortMappings configures port mappings and returns mapping strings for display
func HandlePortMappings(cfg *Config, dockerArgs *[]string) (string, string) {
	if len(cfg.Ports) == 0 {
		return "", ""
	}

	var portMappings []string
	hostPort := cfg.PortRangeStart

	for _, containerPort := range cfg.Ports {
		containerPort = strings.TrimSpace(containerPort)
		hostPort = FindAvailablePort(hostPort)

		*dockerArgs = append(*dockerArgs, "-p", fmt.Sprintf("%d:%s", hostPort, containerPort))
		portMappings = append(portMappings, fmt.Sprintf("%s:%d", containerPort, hostPort))
		hostPort++
	}

	portMapString := strings.Join(portMappings, ",")
	portMapDisplay := strings.ReplaceAll(portMapString, ":", "→")

	return portMapString, portMapDisplay
}
