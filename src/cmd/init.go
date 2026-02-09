package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	cfgtypes "github.com/jedi4ever/addt/config"
	"gopkg.in/yaml.v3"
)

// ProjectType represents detected project characteristics
type ProjectType struct {
	Language    string
	PackageFile string
	HasGit      bool
	HasGitHub   bool
}

// InitConfig holds the configuration being built during init
type InitConfig struct {
	Extensions  string                     `yaml:"extensions,omitempty"`
	Persistent  *bool                      `yaml:"persistent,omitempty"`
	Firewall    *cfgtypes.FirewallSettings `yaml:"firewall,omitempty"`
	SSH         *cfgtypes.SSHSettings      `yaml:"ssh,omitempty"`
	GPG         *cfgtypes.GPGSettings      `yaml:"gpg,omitempty"`
	Workdir     *cfgtypes.WorkdirSettings  `yaml:"workdir,omitempty"`
	NodeVersion string                     `yaml:"node_version,omitempty"`
	GoVersion   string                     `yaml:"go_version,omitempty"`
	GitHub      *cfgtypes.GitHubSettings   `yaml:"github,omitempty"`
}

// HandleInitCommand handles the init command
func HandleInitCommand(args []string) {
	// Check for flags
	nonInteractive := false
	force := false
	for _, arg := range args {
		switch arg {
		case "-y", "--yes":
			nonInteractive = true
		case "-f", "--force":
			force = true
		case "-h", "--help":
			printInitHelp()
			return
		}
	}

	// Check if .addt.yaml already exists
	if _, err := os.Stat(".addt.yaml"); err == nil && !force {
		fmt.Println("Error: .addt.yaml already exists")
		fmt.Println("Use --force to overwrite")
		os.Exit(1)
	}

	fmt.Println("Initializing addt configuration...")
	fmt.Println()

	// Detect project type
	project := detectProjectType()

	// Show what was detected
	if project.Language != "" {
		fmt.Printf("Detected: %s project", project.Language)
		if project.HasGit {
			fmt.Print(" with Git")
		}
		if project.HasGitHub {
			fmt.Print(" (GitHub)")
		}
		fmt.Println()
		fmt.Println()
	}

	// Build configuration
	config := &InitConfig{}

	if nonInteractive {
		// Use smart defaults
		configureDefaults(config, project)
	} else {
		// Interactive mode
		configureInteractive(config, project)
	}

	// Write configuration
	if err := writeConfig(config); err != nil {
		fmt.Printf("Error writing config: %v\n", err)
		os.Exit(1)
	}

	fmt.Println()
	fmt.Println("Created .addt.yaml")
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Printf("  addt run %s \"Hello!\"\n", config.Extensions)
}

func printInitHelp() {
	fmt.Println("Initialize addt configuration for this project")
	fmt.Println()
	fmt.Println("Usage: addt init [options]")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  -y, --yes     Non-interactive mode (use smart defaults)")
	fmt.Println("  -f, --force   Overwrite existing .addt.yaml")
	fmt.Println("  -h, --help    Show this help")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  addt init           # Interactive setup")
	fmt.Println("  addt init -y        # Quick setup with defaults")
	fmt.Println("  addt init -y -f     # Overwrite with defaults")
}

func detectProjectType() ProjectType {
	project := ProjectType{}

	// Detect language/framework
	if fileExists("package.json") {
		project.Language = "Node.js"
		project.PackageFile = "package.json"
	} else if fileExists("pyproject.toml") || fileExists("requirements.txt") || fileExists("setup.py") {
		project.Language = "Python"
		if fileExists("pyproject.toml") {
			project.PackageFile = "pyproject.toml"
		} else if fileExists("requirements.txt") {
			project.PackageFile = "requirements.txt"
		}
	} else if fileExists("go.mod") {
		project.Language = "Go"
		project.PackageFile = "go.mod"
	} else if fileExists("Cargo.toml") {
		project.Language = "Rust"
		project.PackageFile = "Cargo.toml"
	} else if fileExists("pom.xml") || fileExists("build.gradle") {
		project.Language = "Java"
	} else if fileExists("Gemfile") {
		project.Language = "Ruby"
		project.PackageFile = "Gemfile"
	}

	// Detect Git
	if dirExists(".git") {
		project.HasGit = true

		// Check for GitHub
		if fileExists(".github") || containsGitHubRemote() {
			project.HasGitHub = true
		}
	}

	return project
}

func configureDefaults(config *InitConfig, project ProjectType) {
	// Default extension
	config.Extensions = "claude"

	// Ephemeral by default (safer)
	f := false
	config.Persistent = &f

	// Enable firewall with restricted mode
	t := true
	config.Firewall = &cfgtypes.FirewallSettings{
		Enabled: &t,
		Mode:    "strict",
		Allowed: getDefaultAllowedDomains(project),
	}

	// SSH proxy mode (most secure)
	if project.HasGit {
		t2 := true
		config.SSH = &cfgtypes.SSHSettings{
			ForwardKeys: &t2,
			ForwardMode: "proxy",
		}
	}

	// GitHub token forwarding if GitHub project
	if project.HasGitHub {
		config.GitHub = &cfgtypes.GitHubSettings{
			ForwardToken: &t,
			TokenSource:  "gh_auth",
		}
	}

	// Set tool versions based on project
	switch project.Language {
	case "Node.js":
		config.NodeVersion = "22"
	case "Go":
		config.GoVersion = "1.24"
	}
}

func configureInteractive(config *InitConfig, project ProjectType) {
	reader := bufio.NewReader(os.Stdin)

	// 1. Which AI agent?
	fmt.Println("Which AI agent do you want to use?")
	fmt.Println("  1) claude (Anthropic) [default]")
	fmt.Println("  2) codex (OpenAI)")
	fmt.Println("  3) gemini (Google)")
	fmt.Println("  4) copilot (GitHub)")
	fmt.Println("  5) cursor (Cursor)")
	fmt.Println("  6) tessl (Tessl)")
	fmt.Println("  7) other")
	fmt.Print("Choice [1]: ")
	choice := readLine(reader)
	switch choice {
	case "", "1":
		config.Extensions = "claude"
	case "2":
		config.Extensions = "codex"
	case "3":
		config.Extensions = "gemini"
	case "4":
		config.Extensions = "copilot"
	case "5":
		config.Extensions = "cursor"
	case "6":
		config.Extensions = "tessl"
	case "7":
		fmt.Print("Extension name: ")
		config.Extensions = readLine(reader)
		if config.Extensions == "" {
			config.Extensions = "claude"
		}
	default:
		config.Extensions = "claude"
	}
	fmt.Println()

	// 2. Git operations?
	if project.HasGit {
		fmt.Println("Does this project need Git operations? (clone, push, PRs)")
		fmt.Println("  1) Yes - enable SSH key forwarding [default]")
		fmt.Println("  2) No - disable SSH forwarding")
		fmt.Print("Choice [1]: ")
		choice = readLine(reader)
		if choice == "2" {
			sshOff := false
			config.SSH = &cfgtypes.SSHSettings{ForwardKeys: &sshOff}
		} else {
			sshOn := true
			config.SSH = &cfgtypes.SSHSettings{ForwardKeys: &sshOn, ForwardMode: "proxy"}
		}
		fmt.Println()
	}

	// 3. Network security
	fmt.Println("Network access level?")
	fmt.Println("  1) Restricted - only package registries + APIs [default]")
	fmt.Println("  2) Open - agent can access any URL")
	fmt.Println("  3) Strict - explicit allowlist only")
	fmt.Println("  4) Air-gapped - no network access")
	fmt.Print("Choice [1]: ")
	choice = readLine(reader)
	t := true
	f := false
	switch choice {
	case "", "1":
		config.Firewall = &cfgtypes.FirewallSettings{
			Enabled: &t,
			Mode:    "strict",
			Allowed: getDefaultAllowedDomains(project),
		}
	case "2":
		config.Firewall = &cfgtypes.FirewallSettings{
			Enabled: &f,
		}
	case "3":
		fw := &cfgtypes.FirewallSettings{
			Enabled: &t,
			Mode:    "strict",
		}
		// Ask for allowed domains
		fmt.Print("Allowed domains (comma-separated): ")
		domains := readLine(reader)
		if domains != "" {
			parts := strings.Split(domains, ",")
			for i, d := range parts {
				parts[i] = strings.TrimSpace(d)
			}
			fw.Allowed = parts
		}
		config.Firewall = fw
	case "4":
		config.Firewall = &cfgtypes.FirewallSettings{
			Enabled: &t,
			Mode:    "strict",
		}
	}
	fmt.Println()

	// 4. Workspace permissions
	fmt.Println("Workspace permissions?")
	fmt.Println("  1) Read-write - agent can modify your files [default]")
	fmt.Println("  2) Read-only - agent cannot modify files (safer)")
	fmt.Print("Choice [1]: ")
	choice = readLine(reader)
	if choice == "2" {
		config.Workdir = &cfgtypes.WorkdirSettings{Readonly: &t}
	}
	fmt.Println()

	// 5. Container persistence
	fmt.Println("Container persistence?")
	fmt.Println("  1) Ephemeral - fresh container each run [default]")
	fmt.Println("  2) Persistent - faster startup, keeps state")
	fmt.Print("Choice [1]: ")
	choice = readLine(reader)
	if choice == "2" {
		config.Persistent = &t
	} else {
		config.Persistent = &f
	}
	fmt.Println()

	// GitHub token forwarding if GitHub project
	if project.HasGitHub {
		config.GitHub = &cfgtypes.GitHubSettings{
			ForwardToken: &t,
			TokenSource:  "gh_auth",
		}
	}

	// Set tool versions
	switch project.Language {
	case "Node.js":
		config.NodeVersion = "22"
	case "Go":
		config.GoVersion = "1.24"
	}
}

func getDefaultAllowedDomains(project ProjectType) []string {
	domains := []string{
		"api.anthropic.com",
		"api.openai.com",
		"generativelanguage.googleapis.com",
	}

	switch project.Language {
	case "Node.js":
		domains = append(domains, "registry.npmjs.org", "registry.yarnpkg.com")
	case "Python":
		domains = append(domains, "pypi.org", "files.pythonhosted.org")
	case "Go":
		domains = append(domains, "proxy.golang.org", "sum.golang.org")
	case "Rust":
		domains = append(domains, "crates.io", "static.crates.io")
	case "Ruby":
		domains = append(domains, "rubygems.org")
	case "Java":
		domains = append(domains, "repo.maven.apache.org", "plugins.gradle.org")
	}

	if project.HasGitHub {
		domains = append(domains, "api.github.com", "github.com")
	}

	return domains
}

func writeConfig(config *InitConfig) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return err
	}

	// Add header comment
	header := "# addt configuration\n# Generated by: addt init\n# Docs: https://github.com/jedi4ever/addt\n\n"

	return os.WriteFile(".addt.yaml", []byte(header+string(data)), 0644)
}

// Helper functions

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func containsGitHubRemote() bool {
	// Check .git/config for github.com
	data, err := os.ReadFile(".git/config")
	if err != nil {
		return false
	}
	return strings.Contains(string(data), "github.com")
}

func readLine(reader *bufio.Reader) string {
	line, _ := reader.ReadString('\n')
	return strings.TrimSpace(line)
}

// detectCredentials checks for available credentials
func detectCredentials() map[string]bool {
	creds := make(map[string]bool)

	if os.Getenv("ANTHROPIC_API_KEY") != "" {
		creds["anthropic"] = true
	}
	if os.Getenv("OPENAI_API_KEY") != "" {
		creds["openai"] = true
	}
	if os.Getenv("GEMINI_API_KEY") != "" || os.Getenv("GOOGLE_API_KEY") != "" {
		creds["google"] = true
	}
	if os.Getenv("GH_TOKEN") != "" || os.Getenv("GITHUB_TOKEN") != "" {
		creds["github"] = true
	}

	// Check for Claude login
	homeDir, _ := os.UserHomeDir()
	if fileExists(filepath.Join(homeDir, ".claude.json")) {
		creds["claude_login"] = true
	}

	// Check for gh CLI auth
	if fileExists(filepath.Join(homeDir, ".config", "gh", "hosts.yml")) {
		creds["gh_cli"] = true
	}

	return creds
}
