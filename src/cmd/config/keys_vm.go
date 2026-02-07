package config

import (
	cfgtypes "github.com/jedi4ever/addt/config"
)

// GetVmKeys returns all valid VM resource config keys
func GetVmKeys() []KeyInfo {
	return []KeyInfo{
		{Key: "vm.cpus", Description: "VM CPU allocation (Podman machine/Docker Desktop)", Type: "string", EnvVar: "ADDT_VM_CPUS"},
		{Key: "vm.memory", Description: "VM memory in MB (Podman machine/Docker Desktop)", Type: "string", EnvVar: "ADDT_VM_MEMORY"},
	}
}

// GetVmValue retrieves a VM config value
func GetVmValue(v *cfgtypes.VmSettings, key string) string {
	if v == nil {
		return ""
	}
	switch key {
	case "vm.cpus":
		return v.CPUs
	case "vm.memory":
		return v.Memory
	}
	return ""
}

// SetVmValue sets a VM config value
func SetVmValue(v *cfgtypes.VmSettings, key, value string) {
	switch key {
	case "vm.cpus":
		v.CPUs = value
	case "vm.memory":
		v.Memory = value
	}
}

// UnsetVmValue clears a VM config value
func UnsetVmValue(v *cfgtypes.VmSettings, key string) {
	switch key {
	case "vm.cpus":
		v.CPUs = ""
	case "vm.memory":
		v.Memory = ""
	}
}
