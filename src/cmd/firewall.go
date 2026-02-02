package cmd

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"
)

// HandleFirewallCommand handles the firewall subcommand
func HandleFirewallCommand(args []string) {
	if len(args) == 0 {
		printFirewallHelp()
		return
	}

	command := args[0]

	switch command {
	case "list", "ls":
		listAllowedDomains()
	case "add":
		if len(args) < 2 {
			fmt.Println("Error: domain required")
			fmt.Println("Usage: dclaude firewall add <domain>")
			os.Exit(1)
		}
		addAllowedDomain(args[1])
	case "remove", "rm":
		if len(args) < 2 {
			fmt.Println("Error: domain required")
			fmt.Println("Usage: dclaude firewall remove <domain>")
			os.Exit(1)
		}
		removeAllowedDomain(args[1])
	case "reset":
		resetToDefaults()
	case "help", "--help":
		printFirewallHelp()
	default:
		fmt.Printf("Unknown firewall command: %s\n", command)
		printFirewallHelp()
		os.Exit(1)
	}
}

func getFirewallConfigPath() string {
	currentUser, err := user.Current()
	if err != nil {
		return filepath.Join(os.Getenv("HOME"), ".dclaude", "firewall", "allowed-domains.txt")
	}
	return filepath.Join(currentUser.HomeDir, ".dclaude", "firewall", "allowed-domains.txt")
}

func ensureFirewallConfigExists() error {
	configPath := getFirewallConfigPath()
	configDir := filepath.Dir(configPath)

	// Create directory if it doesn't exist
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Create default config if it doesn't exist
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		defaultConfig := `# Default allowed domains for DClaude firewall
# Lines starting with # are comments

# Anthropic API
api.anthropic.com

# GitHub
github.com
api.github.com
raw.githubusercontent.com
objects.githubusercontent.com

# npm registry
registry.npmjs.org

# PyPI
pypi.org
files.pythonhosted.org

# Go modules
proxy.golang.org
sum.golang.org

# Docker Hub (if needed)
registry-1.docker.io
auth.docker.io
production.cloudflare.docker.com

# Common CDNs
cdn.jsdelivr.net
unpkg.com
`
		if err := os.WriteFile(configPath, []byte(defaultConfig), 0644); err != nil {
			return fmt.Errorf("failed to create default config: %w", err)
		}
	}

	return nil
}

func listAllowedDomains() {
	if err := ensureFirewallConfigExists(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	configPath := getFirewallConfigPath()
	content, err := os.ReadFile(configPath)
	if err != nil {
		fmt.Printf("Error reading config: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Allowed domains configuration: %s\n\n", configPath)
	fmt.Println(string(content))
}

func addAllowedDomain(domain string) {
	if err := ensureFirewallConfigExists(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	configPath := getFirewallConfigPath()

	// Read existing content
	content, err := os.ReadFile(configPath)
	if err != nil {
		fmt.Printf("Error reading config: %v\n", err)
		os.Exit(1)
	}

	// Check if domain already exists
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == domain {
			fmt.Printf("Domain '%s' already exists in allowed list\n", domain)
			return
		}
	}

	// Append domain
	newContent := string(content)
	if !strings.HasSuffix(newContent, "\n") {
		newContent += "\n"
	}
	newContent += domain + "\n"

	if err := os.WriteFile(configPath, []byte(newContent), 0644); err != nil {
		fmt.Printf("Error writing config: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Added '%s' to allowed domains\n", domain)
	fmt.Printf("Config: %s\n", configPath)
}

func removeAllowedDomain(domain string) {
	if err := ensureFirewallConfigExists(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	configPath := getFirewallConfigPath()

	// Read existing content
	content, err := os.ReadFile(configPath)
	if err != nil {
		fmt.Printf("Error reading config: %v\n", err)
		os.Exit(1)
	}

	// Filter out the domain
	lines := strings.Split(string(content), "\n")
	var newLines []string
	found := false

	for _, line := range lines {
		if strings.TrimSpace(line) == domain {
			found = true
			continue
		}
		newLines = append(newLines, line)
	}

	if !found {
		fmt.Printf("Domain '%s' not found in allowed list\n", domain)
		return
	}

	// Write back
	newContent := strings.Join(newLines, "\n")
	if err := os.WriteFile(configPath, []byte(newContent), 0644); err != nil {
		fmt.Printf("Error writing config: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Removed '%s' from allowed domains\n", domain)
	fmt.Printf("Config: %s\n", configPath)
}

func resetToDefaults() {
	configPath := getFirewallConfigPath()

	defaultConfig := `# Default allowed domains for DClaude firewall
# Lines starting with # are comments

# Anthropic API
api.anthropic.com

# GitHub
github.com
api.github.com
raw.githubusercontent.com
objects.githubusercontent.com

# npm registry
registry.npmjs.org

# PyPI
pypi.org
files.pythonhosted.org

# Go modules
proxy.golang.org
sum.golang.org

# Docker Hub (if needed)
registry-1.docker.io
auth.docker.io
production.cloudflare.docker.com

# Common CDNs
cdn.jsdelivr.net
unpkg.com
`

	// Ensure directory exists
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		fmt.Printf("Error creating directory: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(configPath, []byte(defaultConfig), 0644); err != nil {
		fmt.Printf("Error writing config: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Reset to default allowed domains\n")
	fmt.Printf("Config: %s\n", configPath)
}

func printFirewallHelp() {
	fmt.Println(`dclaude firewall - Manage network firewall allowed domains

Usage: dclaude firewall <command> [args]

Commands:
  list, ls               List all allowed domains
  add <domain>           Add a domain to the allowed list
  remove, rm <domain>    Remove a domain from the allowed list
  reset                  Reset to default allowed domains
  help                   Show this help

Examples:
  dclaude firewall list
  dclaude firewall add example.com
  dclaude firewall rm example.com
  dclaude firewall reset

Configuration file: ~/.dclaude/firewall/allowed-domains.txt

To enable the firewall:
  export DCLAUDE_FIREWALL=true
  export DCLAUDE_FIREWALL_MODE=strict  # or 'permissive' or 'off'
  dclaude

Note: The firewall works particularly well in CI/CD environments where you want
to restrict network access to only approved domains.

Firewall modes:
  strict      - Block all non-whitelisted traffic (default)
  permissive  - Log but allow all traffic (for testing)
  off         - Disable firewall`)
}
