package main

import (
	"os"
	"strconv"
	"strings"
)

// Config holds all configuration options
type Config struct {
	ClaudeVersion  string
	NodeVersion    string
	EnvVars        []string
	GitHubDetect   bool
	Ports          []string
	PortRangeStart int
	SSHForward     string
	GPGForward     bool
	DockerForward  string
	EnvFile        string
	LogEnabled     bool
	LogFile        string
	ImageName      string
}

// LoadConfig loads configuration from environment variables
func LoadConfig() *Config {
	cfg := &Config{
		ClaudeVersion:  getEnvOrDefault("DCLAUDE_CLAUDE_VERSION", "latest"),
		NodeVersion:    getEnvOrDefault("DCLAUDE_NODE_VERSION", DefaultNodeVersion),
		EnvVars:        strings.Split(getEnvOrDefault("DCLAUDE_ENV_VARS", "ANTHROPIC_API_KEY,GH_TOKEN"), ","),
		GitHubDetect:   getEnvOrDefault("DCLAUDE_GITHUB_DETECT", "false") == "true",
		PortRangeStart: getEnvInt("DCLAUDE_PORT_RANGE_START", DefaultPortRangeStart),
		SSHForward:     os.Getenv("DCLAUDE_SSH_FORWARD"),
		GPGForward:     os.Getenv("DCLAUDE_GPG_FORWARD") == "true",
		DockerForward:  os.Getenv("DCLAUDE_DOCKER_FORWARD"),
		EnvFile:        os.Getenv("DCLAUDE_ENV_FILE"), // Empty means use default .env
		LogEnabled:     os.Getenv("DCLAUDE_LOG") == "true",
		LogFile:        getEnvOrDefault("DCLAUDE_LOG_FILE", "dclaude.log"),
	}

	// Parse ports
	if ports := os.Getenv("DCLAUDE_PORTS"); ports != "" {
		cfg.Ports = strings.Split(ports, ",")
		for i := range cfg.Ports {
			cfg.Ports[i] = strings.TrimSpace(cfg.Ports[i])
		}
	}

	// Trim env vars
	for i := range cfg.EnvVars {
		cfg.EnvVars[i] = strings.TrimSpace(cfg.EnvVars[i])
	}

	return cfg
}

func getEnvOrDefault(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return defaultVal
}
