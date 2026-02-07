package config

import (
	"fmt"

	cfgtypes "github.com/jedi4ever/addt/config"
)

// GetDockerKeys returns all valid Docker config keys (DinD only)
func GetDockerKeys() []KeyInfo {
	return []KeyInfo{
		{Key: "docker.dind.enable", Description: "Enable Docker-in-Docker", Type: "bool", EnvVar: "ADDT_DOCKER_DIND_ENABLE"},
		{Key: "docker.dind.mode", Description: "Docker-in-Docker mode: host or isolated", Type: "string", EnvVar: "ADDT_DOCKER_DIND_MODE"},
	}
}

// GetDockerValue retrieves a Docker config value
func GetDockerValue(d *cfgtypes.DockerSettings, key string) string {
	if d == nil {
		return ""
	}
	switch key {
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
