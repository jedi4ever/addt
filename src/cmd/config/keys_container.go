package config

import (
	cfgtypes "github.com/jedi4ever/addt/config"
)

// GetContainerKeys returns all valid container resource config keys
func GetContainerKeys() []KeyInfo {
	return []KeyInfo{
		{Key: "container.cpus", Description: "CPU limit for container (e.g., \"2\", \"0.5\")", Type: "string", EnvVar: "ADDT_CONTAINER_CPUS"},
		{Key: "container.memory", Description: "Memory limit for container (e.g., \"512m\", \"2g\")", Type: "string", EnvVar: "ADDT_CONTAINER_MEMORY"},
	}
}

// GetContainerValue retrieves a container config value
func GetContainerValue(c *cfgtypes.ContainerSettings, key string) string {
	if c == nil {
		return ""
	}
	switch key {
	case "container.cpus":
		return c.CPUs
	case "container.memory":
		return c.Memory
	}
	return ""
}

// SetContainerValue sets a container config value
func SetContainerValue(c *cfgtypes.ContainerSettings, key, value string) {
	switch key {
	case "container.cpus":
		c.CPUs = value
	case "container.memory":
		c.Memory = value
	}
}

// UnsetContainerValue clears a container config value
func UnsetContainerValue(c *cfgtypes.ContainerSettings, key string) {
	switch key {
	case "container.cpus":
		c.CPUs = ""
	case "container.memory":
		c.Memory = ""
	}
}
