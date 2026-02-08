package config

import (
	"fmt"
	"strings"

	cfgtypes "github.com/jedi4ever/addt/config"
)

// GetGitHubKeys returns all valid GitHub config keys
func GetGitHubKeys() []KeyInfo {
	return []KeyInfo{
		{Key: "github.forward_token", Description: "Forward GH_TOKEN to container (default: true)", Type: "bool", EnvVar: "ADDT_GITHUB_FORWARD_TOKEN"},
		{Key: "github.token_source", Description: "Token source: env or gh_auth (default: gh_auth)", Type: "string", EnvVar: "ADDT_GITHUB_TOKEN_SOURCE"},
		{Key: "github.scope_token", Description: "Scope GH_TOKEN to workspace repo via git credential-cache (default: true)", Type: "bool", EnvVar: "ADDT_GITHUB_SCOPE_TOKEN"},
		{Key: "github.scope_repos", Description: "Additional repos to allow when scoping (comma-separated owner/repo)", Type: "string", EnvVar: "ADDT_GITHUB_SCOPE_REPOS"},
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
	case "github.scope_token":
		if g.ScopeToken != nil {
			return fmt.Sprintf("%v", *g.ScopeToken)
		}
	case "github.scope_repos":
		if len(g.ScopeRepos) > 0 {
			return strings.Join(g.ScopeRepos, ",")
		}
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
	case "github.scope_token":
		b := value == "true"
		g.ScopeToken = &b
	case "github.scope_repos":
		if value == "" {
			g.ScopeRepos = nil
		} else {
			g.ScopeRepos = strings.Split(value, ",")
		}
	}
}

// UnsetGitHubValue clears a GitHub config value
func UnsetGitHubValue(g *cfgtypes.GitHubSettings, key string) {
	switch key {
	case "github.forward_token":
		g.ForwardToken = nil
	case "github.token_source":
		g.TokenSource = ""
	case "github.scope_token":
		g.ScopeToken = nil
	case "github.scope_repos":
		g.ScopeRepos = nil
	}
}
