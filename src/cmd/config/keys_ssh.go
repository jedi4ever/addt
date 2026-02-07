package config

import (
	"fmt"
	"strings"

	cfgtypes "github.com/jedi4ever/addt/config"
)

// GetSSHKeys returns all valid SSH config keys
func GetSSHKeys() []KeyInfo {
	return []KeyInfo{
		{Key: "ssh.forward_keys", Description: "Enable SSH key forwarding (default: true)", Type: "bool", EnvVar: "ADDT_SSH_FORWARD_KEYS"},
		{Key: "ssh.forward_mode", Description: "SSH forwarding mode: agent, keys, or proxy (default: proxy)", Type: "string", EnvVar: "ADDT_SSH_FORWARD_MODE"},
		{Key: "ssh.allowed_keys", Description: "Key filters for proxy mode (comma-separated)", Type: "string", EnvVar: "ADDT_SSH_ALLOWED_KEYS"},
	}
}

// GetSSHValue retrieves an SSH config value
func GetSSHValue(s *cfgtypes.SSHSettings, key string) string {
	if s == nil {
		return ""
	}
	switch key {
	case "ssh.forward_keys":
		if s.ForwardKeys != nil {
			return fmt.Sprintf("%v", *s.ForwardKeys)
		}
	case "ssh.forward_mode":
		return s.ForwardMode
	case "ssh.allowed_keys":
		return strings.Join(s.AllowedKeys, ",")
	}
	return ""
}

// SetSSHValue sets an SSH config value
func SetSSHValue(s *cfgtypes.SSHSettings, key, value string) {
	switch key {
	case "ssh.forward_keys":
		b := value == "true"
		s.ForwardKeys = &b
	case "ssh.forward_mode":
		s.ForwardMode = value
	case "ssh.allowed_keys":
		if value == "" {
			s.AllowedKeys = nil
		} else {
			s.AllowedKeys = strings.Split(value, ",")
		}
	}
}

// UnsetSSHValue clears an SSH config value
func UnsetSSHValue(s *cfgtypes.SSHSettings, key string) {
	switch key {
	case "ssh.forward_keys":
		s.ForwardKeys = nil
	case "ssh.forward_mode":
		s.ForwardMode = ""
	case "ssh.allowed_keys":
		s.AllowedKeys = nil
	}
}
