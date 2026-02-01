package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// LoadEnvFile loads environment variables from a .env file
func LoadEnvFile(envFile string) {
	specifiedByUser := envFile != ""
	if envFile == "" {
		envFile = ".env"
	}

	file, err := os.Open(envFile)
	if err != nil {
		// Only warn if user explicitly specified the env file
		if specifiedByUser {
			fmt.Printf("Warning: Specified env file not found: %s\n", envFile)
		}
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if idx := strings.Index(line, "="); idx > 0 {
			key := strings.TrimSpace(line[:idx])
			value := strings.TrimSpace(line[idx+1:])
			// Remove quotes if present
			value = strings.Trim(value, "\"'")
			os.Setenv(key, value)
		}
	}
}

// DetectGitHubToken attempts to get the GitHub token from the gh CLI
func DetectGitHubToken() string {
	if _, err := exec.LookPath("gh"); err != nil {
		return ""
	}
	cmd := exec.Command("gh", "auth", "token")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}
