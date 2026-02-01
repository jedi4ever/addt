package main

import (
	"encoding/json"
	"io"
	"net/http"
	"time"
)

const npmRegistryURL = "https://registry.npmjs.org/@anthropic-ai/claude-code"

// GetNpmLatestVersion fetches the latest stable version from npm registry
func GetNpmLatestVersion() string {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(npmRegistryURL)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ""
	}

	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return ""
	}

	if distTags, ok := data["dist-tags"].(map[string]interface{}); ok {
		if stable, ok := distTags["stable"].(string); ok {
			return stable
		}
		if latest, ok := distTags["latest"].(string); ok {
			return latest
		}
	}

	return ""
}

// ValidateNpmVersion checks if a specific version exists in npm registry
func ValidateNpmVersion(version string) bool {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(npmRegistryURL)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false
	}

	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return false
	}

	if versions, ok := data["versions"].(map[string]interface{}); ok {
		_, exists := versions[version]
		return exists
	}

	return false
}
