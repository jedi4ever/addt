package config

import (
	"fmt"

	cfgtypes "github.com/jedi4ever/addt/config"
)

// GetGitHubKeys returns all valid GitHub config keys
func GetGitHubKeys() []KeyInfo {
	return []KeyInfo{
		{Key: "github.forward_token", Description: "Forward GH_TOKEN to container (default: true)", Type: "bool", EnvVar: "ADDT_GITHUB_FORWARD_TOKEN"},
		{Key: "github.token_source", Description: "Token source: env or gh_auth (default: gh_auth)", Type: "string", EnvVar: "ADDT_GITHUB_TOKEN_SOURCE"},
	}
}

// GetGitHubValue retrieves a GitHub config value
func GetGitHubValue(g *cfgtypes.GitHubSettings, key string) string {
	if g == nil {
		return ""
	}
	switch key {
	case "github.forward_token":
		if g.ForwardToken != nil {
			return fmt.Sprintf("%v", *g.ForwardToken)
		}
	case "github.token_source":
		return g.TokenSource
	}
	return ""
}

// SetGitHubValue sets a GitHub config value
func SetGitHubValue(g *cfgtypes.GitHubSettings, key, value string) {
	switch key {
	case "github.forward_token":
		b := value == "true"
		g.ForwardToken = &b
	case "github.token_source":
		g.TokenSource = value
	}
}

// UnsetGitHubValue clears a GitHub config value
func UnsetGitHubValue(g *cfgtypes.GitHubSettings, key string) {
	switch key {
	case "github.forward_token":
		g.ForwardToken = nil
	case "github.token_source":
		g.TokenSource = ""
	}
}
