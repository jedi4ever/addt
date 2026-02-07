package config

import (
	"fmt"
	"strings"

	cfgtypes "github.com/jedi4ever/addt/config"
)

// GetPortsKeys returns all valid ports config keys
func GetPortsKeys() []KeyInfo {
	return []KeyInfo{
		{Key: "ports.forward", Description: "Enable port forwarding (default: true)", Type: "bool", EnvVar: "ADDT_PORTS_FORWARD"},
		{Key: "ports.expose", Description: "Container ports to expose (comma-separated)", Type: "string", EnvVar: "ADDT_PORTS"},
		{Key: "ports.inject_system_prompt", Description: "Inject port mappings into AI system prompt (default: true)", Type: "bool", EnvVar: "ADDT_PORTS_INJECT_SYSTEM_PROMPT"},
		{Key: "ports.range_start", Description: "Starting port for auto allocation", Type: "int", EnvVar: "ADDT_PORT_RANGE_START"},
	}
}

// GetPortsValue retrieves a ports config value
func GetPortsValue(p *cfgtypes.PortsSettings, key string) string {
	if p == nil {
		return ""
	}
	switch key {
	case "ports.forward":
		if p.Forward != nil {
			return fmt.Sprintf("%v", *p.Forward)
		}
	case "ports.expose":
		return strings.Join(p.Expose, ",")
	case "ports.inject_system_prompt":
		if p.InjectSystemPrompt != nil {
			return fmt.Sprintf("%v", *p.InjectSystemPrompt)
		}
	case "ports.range_start":
		if p.RangeStart != nil {
			return fmt.Sprintf("%d", *p.RangeStart)
		}
	}
	return ""
}

// SetPortsValue sets a ports config value
func SetPortsValue(p *cfgtypes.PortsSettings, key, value string) {
	switch key {
	case "ports.forward":
		b := value == "true"
		p.Forward = &b
	case "ports.expose":
		if value == "" {
			p.Expose = nil
		} else {
			parts := strings.Split(value, ",")
			for i := range parts {
				parts[i] = strings.TrimSpace(parts[i])
			}
			p.Expose = parts
		}
	case "ports.inject_system_prompt":
		b := value == "true"
		p.InjectSystemPrompt = &b
	case "ports.range_start":
		var i int
		fmt.Sscanf(value, "%d", &i)
		p.RangeStart = &i
	}
}

// UnsetPortsValue clears a ports config value
func UnsetPortsValue(p *cfgtypes.PortsSettings, key string) {
	switch key {
	case "ports.forward":
		p.Forward = nil
	case "ports.expose":
		p.Expose = nil
	case "ports.inject_system_prompt":
		p.InjectSystemPrompt = nil
	case "ports.range_start":
		p.RangeStart = nil
	}
}
