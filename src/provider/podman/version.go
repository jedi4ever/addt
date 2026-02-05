package podman

import (
	"encoding/json"
	"io"
	"net/http"
	"time"
)

const npmRegistryURL = "https://registry.npmjs.org/@anthropic-ai/claude-code"

// getNpmVersionByTag fetches the version for a specific dist-tag (latest, stable, etc.)
func (p *PodmanProvider) getNpmVersionByTag(tag string) string {
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
		if version, ok := distTags[tag].(string); ok {
			return version
		}
	}

	return ""
}

// getNpmLatestVersion fetches the latest stable version from npm registry (kept for compatibility)
func (p *PodmanProvider) getNpmLatestVersion() string {
	return p.getNpmVersionByTag("latest")
}

// validateNpmVersion checks if a specific version exists in npm registry
func (p *PodmanProvider) validateNpmVersion(version string) bool {
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
