package config

import (
	"fmt"

	cfgtypes "github.com/jedi4ever/addt/config"
)

// GetDockerKeys returns all valid Docker config keys
func GetDockerKeys() []KeyInfo {
	return []KeyInfo{
		{Key: "docker.cpus", Description: "CPU limit for container (e.g., \"2\", \"0.5\")", Type: "string", EnvVar: "ADDT_DOCKER_CPUS"},
		{Key: "docker.dind.enable", Description: "Enable Docker-in-Docker", Type: "bool", EnvVar: "ADDT_DOCKER_DIND_ENABLE"},
		{Key: "docker.dind.mode", Description: "Docker-in-Docker mode: host or isolated", Type: "string", EnvVar: "ADDT_DOCKER_DIND_MODE"},
		{Key: "docker.memory", Description: "Memory limit for container (e.g., \"512m\", \"2g\")", Type: "string", EnvVar: "ADDT_DOCKER_MEMORY"},
	}
}

// GetDockerValue retrieves a Docker config value
func GetDockerValue(d *cfgtypes.DockerSettings, key string) string {
	if d == nil {
		return ""
	}
	switch key {
	case "docker.cpus":
		return d.CPUs
	case "docker.memory":
		return d.Memory
	case "docker.dind.enable":
		if d.Dind != nil && d.Dind.Enable != nil {
			return fmt.Sprintf("%v", *d.Dind.Enable)
		}
	case "docker.dind.mode":
		if d.Dind != nil {
			return d.Dind.Mode
		}
	}
	return ""
}

// SetDockerValue sets a Docker config value
func SetDockerValue(d *cfgtypes.DockerSettings, key, value string) {
	switch key {
	case "docker.cpus":
		d.CPUs = value
	case "docker.memory":
		d.Memory = value
	case "docker.dind.enable":
		if d.Dind == nil {
			d.Dind = &cfgtypes.DindSettings{}
		}
		b := value == "true"
		d.Dind.Enable = &b
	case "docker.dind.mode":
		if d.Dind == nil {
			d.Dind = &cfgtypes.DindSettings{}
		}
		d.Dind.Mode = value
	}
}

// UnsetDockerValue clears a Docker config value
func UnsetDockerValue(d *cfgtypes.DockerSettings, key string) {
	switch key {
	case "docker.cpus":
		d.CPUs = ""
	case "docker.memory":
		d.Memory = ""
	case "docker.dind.enable":
		if d.Dind != nil {
			d.Dind.Enable = nil
		}
	case "docker.dind.mode":
		if d.Dind != nil {
			d.Dind.Mode = ""
		}
	}
}
