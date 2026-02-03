package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/jedi4ever/addt/config"
	"github.com/jedi4ever/addt/extensions"
)

// configKeyInfo holds metadata about a config key
type configKeyInfo struct {
	Key         string
	Description string
	Type        string // "bool", "string", "int"
	EnvVar      string
}

// getConfigKeys returns all valid config keys with their metadata (sorted alphabetically)
func getConfigKeys() []configKeyInfo {
	keys := []configKeyInfo{
		{Key: "dind", Description: "Enable Docker-in-Docker", Type: "bool", EnvVar: "ADDT_DIND"},
		{Key: "dind_mode", Description: "Docker-in-Docker mode: host or isolated", Type: "string", EnvVar: "ADDT_DIND_MODE"},
		{Key: "docker_cpus", Description: "CPU limit for container (e.g., \"2\", \"0.5\")", Type: "string", EnvVar: "ADDT_DOCKER_CPUS"},
		{Key: "docker_memory", Description: "Memory limit for container (e.g., \"512m\", \"2g\")", Type: "string", EnvVar: "ADDT_DOCKER_MEMORY"},
		{Key: "firewall", Description: "Enable network firewall", Type: "bool", EnvVar: "ADDT_FIREWALL"},
		{Key: "firewall_mode", Description: "Firewall mode: strict, permissive, off", Type: "string", EnvVar: "ADDT_FIREWALL_MODE"},
		{Key: "github_detect", Description: "Auto-detect GitHub token from gh CLI", Type: "bool", EnvVar: "ADDT_GITHUB_DETECT"},
		{Key: "go_version", Description: "Go version", Type: "string", EnvVar: "ADDT_GO_VERSION"},
		{Key: "gpg_forward", Description: "Enable GPG forwarding", Type: "bool", EnvVar: "ADDT_GPG_FORWARD"},
		{Key: "log", Description: "Enable command logging", Type: "bool", EnvVar: "ADDT_LOG"},
		{Key: "log_file", Description: "Log file path", Type: "string", EnvVar: "ADDT_LOG_FILE"},
		{Key: "node_version", Description: "Node.js version", Type: "string", EnvVar: "ADDT_NODE_VERSION"},
		{Key: "persistent", Description: "Enable persistent container mode", Type: "bool", EnvVar: "ADDT_PERSISTENT"},
		{Key: "port_range_start", Description: "Starting port for auto allocation", Type: "int", EnvVar: "ADDT_PORT_RANGE_START"},
		{Key: "ssh_forward", Description: "SSH forwarding mode: agent or keys", Type: "string", EnvVar: "ADDT_SSH_FORWARD"},
		{Key: "uv_version", Description: "UV Python package manager version", Type: "string", EnvVar: "ADDT_UV_VERSION"},
		{Key: "workdir", Description: "Override working directory (default: current directory)", Type: "string", EnvVar: "ADDT_WORKDIR"},
		{Key: "workdir_automount", Description: "Auto-mount working directory to /workspace", Type: "bool", EnvVar: "ADDT_WORKDIR_AUTOMOUNT"},
	}
	return keys
}


// HandleConfigCommand handles the config subcommand
func HandleConfigCommand(args []string) {
	if len(args) == 0 {
		printConfigHelp()
		return
	}

	switch args[0] {
	case "global":
		handleGlobalConfig(args[1:])
	case "project":
		handleProjectConfig(args[1:])
	case "extension":
		handleExtensionConfig(args[1:])
	case "path":
		fmt.Printf("Global config:  %s\n", config.GetGlobalConfigPath())
		fmt.Printf("Project config: %s\n", config.GetProjectConfigPath())
	default:
		fmt.Printf("Unknown config command: %s\n", args[0])
		printConfigHelp()
		os.Exit(1)
	}
}

// handleGlobalConfig handles global config subcommands
func handleGlobalConfig(args []string) {
	if len(args) == 0 {
		printGlobalConfigHelp()
		return
	}

	switch args[0] {
	case "list":
		listConfig()
	case "get":
		if len(args) < 2 {
			fmt.Println("Usage: addt config global get <key>")
			os.Exit(1)
		}
		getConfig(args[1])
	case "set":
		if len(args) < 3 {
			fmt.Println("Usage: addt config global set <key> <value>")
			os.Exit(1)
		}
		setConfig(args[1], args[2])
	case "unset":
		if len(args) < 2 {
			fmt.Println("Usage: addt config global unset <key>")
			os.Exit(1)
		}
		unsetConfig(args[1])
	default:
		fmt.Printf("Unknown global config command: %s\n", args[0])
		printGlobalConfigHelp()
		os.Exit(1)
	}
}

// handleProjectConfig handles project-level config subcommands
func handleProjectConfig(args []string) {
	if len(args) == 0 {
		printProjectConfigHelp()
		return
	}

	switch args[0] {
	case "list":
		listProjectConfig()
	case "get":
		if len(args) < 2 {
			fmt.Println("Usage: addt config project get <key>")
			os.Exit(1)
		}
		getProjectConfig(args[1])
	case "set":
		if len(args) < 3 {
			fmt.Println("Usage: addt config project set <key> <value>")
			os.Exit(1)
		}
		setProjectConfig(args[1], args[2])
	case "unset":
		if len(args) < 2 {
			fmt.Println("Usage: addt config project unset <key>")
			os.Exit(1)
		}
		unsetProjectConfig(args[1])
	default:
		fmt.Printf("Unknown project config command: %s\n", args[0])
		printProjectConfigHelp()
		os.Exit(1)
	}
}

// handleExtensionConfig handles extension-specific config subcommands
func handleExtensionConfig(args []string) {
	if len(args) == 0 {
		printExtensionConfigHelp()
		return
	}

	// Check for --project flag anywhere in args
	useProject := false
	var filteredArgs []string
	for _, arg := range args {
		if arg == "--project" {
			useProject = true
		} else {
			filteredArgs = append(filteredArgs, arg)
		}
	}
	args = filteredArgs

	if len(args) == 0 {
		printExtensionConfigHelp()
		return
	}

	extName := args[0]

	// Check if first arg is a subcommand (user forgot extension name)
	if extName == "list" || extName == "get" || extName == "set" || extName == "unset" {
		fmt.Println("Error: extension name required")
		fmt.Println()
		printExtensionConfigHelp()
		os.Exit(1)
	}

	// Validate that the extension exists
	if !extensionExists(extName) {
		fmt.Printf("Error: extension '%s' does not exist\n", extName)
		fmt.Println("Run 'addt extensions list' to see available extensions")
		os.Exit(1)
	}

	if len(args) < 2 {
		// Default to list for extension
		listExtensionConfig(extName)
		return
	}

	switch args[1] {
	case "list":
		listExtensionConfig(extName)
	case "get":
		if len(args) < 3 {
			fmt.Println("Usage: addt config extension <name> get <key>")
			os.Exit(1)
		}
		getExtensionConfig(extName, args[2])
	case "set":
		if len(args) < 4 {
			fmt.Println("Usage: addt config extension <name> set <key> <value> [--project]")
			os.Exit(1)
		}
		setExtensionConfig(extName, args[2], args[3], useProject)
	case "unset":
		if len(args) < 3 {
			fmt.Println("Usage: addt config extension <name> unset <key> [--project]")
			os.Exit(1)
		}
		unsetExtensionConfig(extName, args[2], useProject)
	default:
		fmt.Printf("Unknown extension config command: %s\n", args[1])
		printExtensionConfigHelp()
		os.Exit(1)
	}
}

// extensionExists checks if an extension with the given name exists
func extensionExists(name string) bool {
	exts, err := getExtensions()
	if err != nil {
		return false
	}
	for _, ext := range exts {
		if ext.Name == name {
			return true
		}
	}
	return false
}

func printConfigHelp() {
	fmt.Println("Usage: addt config <command>")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  global <subcommand>              Manage global configuration (~/.addt/config.yaml)")
	fmt.Println("  project <subcommand>             Manage project configuration (.addt.yaml)")
	fmt.Println("  extension <name> <subcommand>    Manage extension-specific configuration")
	fmt.Println("  path                             Show config file paths")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  addt config global list")
	fmt.Println("  addt config global set docker_cpus 2")
	fmt.Println("  addt config project set persistent true")
	fmt.Println("  addt config extension claude set version 1.0.5")
	fmt.Println()
	fmt.Println("Precedence (highest to lowest):")
	fmt.Println("  1. Environment variables (e.g., ADDT_DOCKER_CPUS)")
	fmt.Println("  2. Project config (.addt.yaml in current directory)")
	fmt.Println("  3. Global config (~/.addt/config.yaml)")
	fmt.Println("  4. Default values")
}

func printGlobalConfigHelp() {
	fmt.Println("Usage: addt config global <command>")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  list              List all global configuration values")
	fmt.Println("  get <key>         Get a configuration value")
	fmt.Println("  set <key> <value> Set a configuration value")
	fmt.Println("  unset <key>       Remove a configuration value (use default)")
	fmt.Println()
	fmt.Println("Available keys:")
	keys := getConfigKeys()
	maxKeyLen := 0
	for _, k := range keys {
		if len(k.Key) > maxKeyLen {
			maxKeyLen = len(k.Key)
		}
	}
	for _, k := range keys {
		fmt.Printf("  %-*s  %s\n", maxKeyLen, k.Key, k.Description)
	}
}

func printProjectConfigHelp() {
	fmt.Println("Usage: addt config project <command>")
	fmt.Println()
	fmt.Println("Manage project-level configuration stored in .addt.yaml")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  list              List all project configuration values")
	fmt.Println("  get <key>         Get a configuration value")
	fmt.Println("  set <key> <value> Set a configuration value")
	fmt.Println("  unset <key>       Remove a configuration value")
	fmt.Println()
	fmt.Println("Project config overrides global config but is overridden by env vars.")
	fmt.Println()
	fmt.Println("Available keys:")
	keys := getConfigKeys()
	maxKeyLen := 0
	for _, k := range keys {
		if len(k.Key) > maxKeyLen {
			maxKeyLen = len(k.Key)
		}
	}
	for _, k := range keys {
		fmt.Printf("  %-*s  %s\n", maxKeyLen, k.Key, k.Description)
	}
}

func printExtensionConfigHelp() {
	fmt.Println("Usage: addt config extension <name> <command> [--project]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  list              List extension configuration")
	fmt.Println("  get <key>         Get a configuration value")
	fmt.Println("  set <key> <value> Set a configuration value")
	fmt.Println("  unset <key>       Remove a configuration value")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("  --project         Save to project config (.addt.yaml) instead of global")
	fmt.Println()
	fmt.Println("Available keys:")
	fmt.Println("  version     Extension version (e.g., \"1.0.5\", \"latest\", \"stable\")")
	fmt.Println("  automount   Auto-mount extension config directories (true/false)")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  addt config extension claude list")
	fmt.Println("  addt config extension claude set version 1.0.5")
	fmt.Println("  addt config extension claude set automount false --project")
}

// getDefaultValue returns the default value for a config key
func getDefaultValue(key string) string {
	switch key {
	case "docker_cpus":
		return ""
	case "dind":
		return "false"
	case "dind_mode":
		return "isolated"
	case "firewall":
		return "false"
	case "firewall_mode":
		return "strict"
	case "github_detect":
		return "false"
	case "go_version":
		return "latest"
	case "gpg_forward":
		return "false"
	case "log":
		return "false"
	case "log_file":
		return "addt.log"
	case "docker_memory":
		return ""
	case "node_version":
		return "22"
	case "persistent":
		return "false"
	case "port_range_start":
		return "30000"
	case "ssh_forward":
		return "agent"
	case "uv_version":
		return "latest"
	case "workdir":
		return "."
	case "workdir_automount":
		return "true"
	}
	return ""
}

func listConfig() {
	globalCfg, err := config.LoadGlobalConfigFile()
	if err != nil {
		fmt.Printf("Error loading global config: %v\n", err)
		os.Exit(1)
	}

	projectCfg, err := config.LoadProjectConfigFile()
	if err != nil {
		fmt.Printf("Error loading project config: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Global config:  %s\n", config.GetGlobalConfigPath())
	fmt.Printf("Project config: %s\n\n", config.GetProjectConfigPath())

	keys := getConfigKeys()

	// Calculate column widths based on content
	maxKeyLen := 3 // "Key"
	maxValLen := 5 // "Value"
	for _, k := range keys {
		if len(k.Key) > maxKeyLen {
			maxKeyLen = len(k.Key)
		}
		envValue := os.Getenv(k.EnvVar)
		projectValue := getConfigValue(projectCfg, k.Key)
		globalValue := getConfigValue(globalCfg, k.Key)
		defaultValue := getDefaultValue(k.Key)
		val := envValue
		if val == "" {
			val = projectValue
		}
		if val == "" {
			val = globalValue
		}
		if val == "" {
			val = defaultValue
		}
		if val == "" {
			val = "-"
		}
		if len(val) > maxValLen {
			maxValLen = len(val)
		}
	}

	// Print header
	fmt.Printf("  %-*s   %-*s   %s\n", maxKeyLen, "Key", maxValLen, "Value", "Source")
	fmt.Printf("  %s   %s   %s\n", strings.Repeat("-", maxKeyLen), strings.Repeat("-", maxValLen), "--------")

	for _, k := range keys {
		envValue := os.Getenv(k.EnvVar)
		projectValue := getConfigValue(projectCfg, k.Key)
		globalValue := getConfigValue(globalCfg, k.Key)
		defaultValue := getDefaultValue(k.Key)

		var displayValue, source string
		if envValue != "" {
			displayValue = envValue
			source = "env"
		} else if projectValue != "" {
			displayValue = projectValue
			source = "project"
		} else if globalValue != "" {
			displayValue = globalValue
			source = "global"
		} else if defaultValue != "" {
			displayValue = defaultValue
			source = "default"
		} else {
			displayValue = "-"
			source = ""
		}

		// Highlight non-default values
		if source == "env" || source == "project" || source == "global" {
			fmt.Printf("* %-*s   %-*s   %s\n", maxKeyLen, k.Key, maxValLen, displayValue, source)
		} else {
			fmt.Printf("  %-*s   %-*s   %s\n", maxKeyLen, k.Key, maxValLen, displayValue, source)
		}
	}
}

func getConfig(key string) {
	// Validate key
	if !isValidConfigKey(key) {
		fmt.Printf("Unknown config key: %s\n", key)
		fmt.Println("Use 'addt config list' to see available keys.")
		os.Exit(1)
	}

	cfg, err := config.LoadGlobalConfigFile()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}

	val := getConfigValue(cfg, key)
	if val == "" {
		fmt.Printf("%s is not set\n", key)
	} else {
		fmt.Println(val)
	}
}

func setConfig(key, value string) {
	// Validate key
	keyInfo := getConfigKeyInfo(key)
	if keyInfo == nil {
		fmt.Printf("Unknown config key: %s\n", key)
		fmt.Println("Use 'addt config --help' to see available keys.")
		os.Exit(1)
	}

	// Validate value based on type
	if keyInfo.Type == "bool" {
		value = strings.ToLower(value)
		if value != "true" && value != "false" {
			fmt.Printf("Invalid value for %s: must be 'true' or 'false'\n", key)
			os.Exit(1)
		}
	}

	cfg, err := config.LoadGlobalConfigFile()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}

	setConfigValue(cfg, key, value)

	if err := config.SaveGlobalConfigFile(cfg); err != nil {
		fmt.Printf("Error saving config: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Set %s = %s\n", key, value)
}

func unsetConfig(key string) {
	// Validate key
	if !isValidConfigKey(key) {
		fmt.Printf("Unknown config key: %s\n", key)
		fmt.Println("Use 'addt config list' to see available keys.")
		os.Exit(1)
	}

	cfg, err := config.LoadGlobalConfigFile()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}

	unsetConfigValue(cfg, key)

	if err := config.SaveGlobalConfigFile(cfg); err != nil {
		fmt.Printf("Error saving config: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Unset %s\n", key)
}

// Project config functions

func listProjectConfig() {
	cfg, err := config.LoadProjectConfigFile()
	if err != nil {
		fmt.Printf("Error loading project config: %v\n", err)
		os.Exit(1)
	}

	configPath := config.GetProjectConfigPath()
	fmt.Printf("Project config: %s\n\n", configPath)

	keys := getConfigKeys()

	// Calculate column widths
	maxKeyLen := 3
	maxValLen := 5
	for _, k := range keys {
		if len(k.Key) > maxKeyLen {
			maxKeyLen = len(k.Key)
		}
		val := getConfigValue(cfg, k.Key)
		if val == "" {
			val = "-"
		}
		if len(val) > maxValLen {
			maxValLen = len(val)
		}
	}

	// Print header
	fmt.Printf("  %-*s   %-*s\n", maxKeyLen, "Key", maxValLen, "Value")
	fmt.Printf("  %s   %s\n", strings.Repeat("-", maxKeyLen), strings.Repeat("-", maxValLen))

	hasValues := false
	for _, k := range keys {
		val := getConfigValue(cfg, k.Key)
		if val != "" {
			hasValues = true
			fmt.Printf("* %-*s   %-*s\n", maxKeyLen, k.Key, maxValLen, val)
		}
	}

	if !hasValues {
		fmt.Println("  (no project config set)")
	}
}

func getProjectConfig(key string) {
	if !isValidConfigKey(key) {
		fmt.Printf("Unknown config key: %s\n", key)
		fmt.Println("Use 'addt config project list' to see available keys.")
		os.Exit(1)
	}

	cfg, err := config.LoadProjectConfigFile()
	if err != nil {
		fmt.Printf("Error loading project config: %v\n", err)
		os.Exit(1)
	}

	val := getConfigValue(cfg, key)
	if val == "" {
		fmt.Printf("%s is not set in project config\n", key)
	} else {
		fmt.Println(val)
	}
}

func setProjectConfig(key, value string) {
	keyInfo := getConfigKeyInfo(key)
	if keyInfo == nil {
		fmt.Printf("Unknown config key: %s\n", key)
		fmt.Println("Use 'addt config project --help' to see available keys.")
		os.Exit(1)
	}

	if keyInfo.Type == "bool" {
		value = strings.ToLower(value)
		if value != "true" && value != "false" {
			fmt.Printf("Invalid value for %s: must be 'true' or 'false'\n", key)
			os.Exit(1)
		}
	}

	cfg, err := config.LoadProjectConfigFile()
	if err != nil {
		fmt.Printf("Error loading project config: %v\n", err)
		os.Exit(1)
	}

	setConfigValue(cfg, key, value)

	if err := config.SaveProjectConfigFile(cfg); err != nil {
		fmt.Printf("Error saving project config: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Set %s = %s (project)\n", key, value)
}

func unsetProjectConfig(key string) {
	if !isValidConfigKey(key) {
		fmt.Printf("Unknown config key: %s\n", key)
		fmt.Println("Use 'addt config project list' to see available keys.")
		os.Exit(1)
	}

	cfg, err := config.LoadProjectConfigFile()
	if err != nil {
		fmt.Printf("Error loading project config: %v\n", err)
		os.Exit(1)
	}

	unsetConfigValue(cfg, key)

	if err := config.SaveProjectConfigFile(cfg); err != nil {
		fmt.Printf("Error saving project config: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Unset %s (project)\n", key)
}

func isValidConfigKey(key string) bool {
	for _, k := range getConfigKeys() {
		if k.Key == key {
			return true
		}
	}
	return false
}

func getConfigKeyInfo(key string) *configKeyInfo {
	for _, k := range getConfigKeys() {
		if k.Key == key {
			return &k
		}
	}
	return nil
}

func getConfigValue(cfg *config.GlobalConfig, key string) string {
	switch key {
	case "docker_cpus":
		return cfg.DockerCPUs
	case "dind":
		if cfg.Dind != nil {
			return fmt.Sprintf("%v", *cfg.Dind)
		}
	case "dind_mode":
		return cfg.DindMode
	case "firewall":
		if cfg.Firewall != nil {
			return fmt.Sprintf("%v", *cfg.Firewall)
		}
	case "firewall_mode":
		return cfg.FirewallMode
	case "github_detect":
		if cfg.GitHubDetect != nil {
			return fmt.Sprintf("%v", *cfg.GitHubDetect)
		}
	case "go_version":
		return cfg.GoVersion
	case "gpg_forward":
		if cfg.GPGForward != nil {
			return fmt.Sprintf("%v", *cfg.GPGForward)
		}
	case "log":
		if cfg.Log != nil {
			return fmt.Sprintf("%v", *cfg.Log)
		}
	case "log_file":
		return cfg.LogFile
	case "docker_memory":
		return cfg.DockerMemory
	case "node_version":
		return cfg.NodeVersion
	case "persistent":
		if cfg.Persistent != nil {
			return fmt.Sprintf("%v", *cfg.Persistent)
		}
	case "port_range_start":
		if cfg.PortRangeStart != nil {
			return fmt.Sprintf("%d", *cfg.PortRangeStart)
		}
	case "ssh_forward":
		return cfg.SSHForward
	case "uv_version":
		return cfg.UvVersion
	case "workdir":
		return cfg.Workdir
	case "workdir_automount":
		if cfg.WorkdirAutomount != nil {
			return fmt.Sprintf("%v", *cfg.WorkdirAutomount)
		}
	}
	return ""
}

func setConfigValue(cfg *config.GlobalConfig, key, value string) {
	switch key {
	case "docker_cpus":
		cfg.DockerCPUs = value
	case "dind":
		b := value == "true"
		cfg.Dind = &b
	case "dind_mode":
		cfg.DindMode = value
	case "firewall":
		b := value == "true"
		cfg.Firewall = &b
	case "firewall_mode":
		cfg.FirewallMode = value
	case "github_detect":
		b := value == "true"
		cfg.GitHubDetect = &b
	case "go_version":
		cfg.GoVersion = value
	case "gpg_forward":
		b := value == "true"
		cfg.GPGForward = &b
	case "log":
		b := value == "true"
		cfg.Log = &b
	case "log_file":
		cfg.LogFile = value
	case "docker_memory":
		cfg.DockerMemory = value
	case "node_version":
		cfg.NodeVersion = value
	case "persistent":
		b := value == "true"
		cfg.Persistent = &b
	case "port_range_start":
		var i int
		fmt.Sscanf(value, "%d", &i)
		cfg.PortRangeStart = &i
	case "ssh_forward":
		cfg.SSHForward = value
	case "uv_version":
		cfg.UvVersion = value
	case "workdir":
		cfg.Workdir = value
	case "workdir_automount":
		b := value == "true"
		cfg.WorkdirAutomount = &b
	}
}

func unsetConfigValue(cfg *config.GlobalConfig, key string) {
	switch key {
	case "docker_cpus":
		cfg.DockerCPUs = ""
	case "dind":
		cfg.Dind = nil
	case "dind_mode":
		cfg.DindMode = ""
	case "firewall":
		cfg.Firewall = nil
	case "firewall_mode":
		cfg.FirewallMode = ""
	case "github_detect":
		cfg.GitHubDetect = nil
	case "go_version":
		cfg.GoVersion = ""
	case "gpg_forward":
		cfg.GPGForward = nil
	case "log":
		cfg.Log = nil
	case "log_file":
		cfg.LogFile = ""
	case "docker_memory":
		cfg.DockerMemory = ""
	case "node_version":
		cfg.NodeVersion = ""
	case "persistent":
		cfg.Persistent = nil
	case "port_range_start":
		cfg.PortRangeStart = nil
	case "ssh_forward":
		cfg.SSHForward = ""
	case "uv_version":
		cfg.UvVersion = ""
	case "workdir":
		cfg.Workdir = ""
	case "workdir_automount":
		cfg.WorkdirAutomount = nil
	}
}

// Extension config keys
func getExtensionConfigKeys() []configKeyInfo {
	return []configKeyInfo{
		{Key: "version", Description: "Extension version", Type: "string", EnvVar: "ADDT_%s_VERSION"},
		{Key: "automount", Description: "Auto-mount extension config directories", Type: "bool", EnvVar: "ADDT_%s_AUTOMOUNT"},
	}
}

func isValidExtensionConfigKey(key string) bool {
	for _, k := range getExtensionConfigKeys() {
		if k.Key == key {
			return true
		}
	}
	return false
}

func listExtensionConfig(extName string) {
	globalCfg, err := config.LoadGlobalConfigFile()
	if err != nil {
		fmt.Printf("Error loading global config: %v\n", err)
		os.Exit(1)
	}

	projectCfg, err := config.LoadProjectConfigFile()
	if err != nil {
		fmt.Printf("Error loading project config: %v\n", err)
		os.Exit(1)
	}

	// Get extension defaults from extension's config.yaml
	var extDefaults *extensions.ExtensionConfig
	exts, err := getExtensions()
	if err == nil {
		for _, ext := range exts {
			if ext.Name == extName {
				extDefaults = &ext
				break
			}
		}
	}

	extNameUpper := strings.ToUpper(extName)
	fmt.Printf("Extension: %s\n\n", extName)

	keys := getExtensionConfigKeys()

	// Get extension config from global and project config files
	var globalExtCfg, projectExtCfg *config.ExtensionSettings
	if globalCfg.Extensions != nil {
		globalExtCfg = globalCfg.Extensions[extName]
	}
	if projectCfg.Extensions != nil {
		projectExtCfg = projectCfg.Extensions[extName]
	}

	// Print header
	fmt.Printf("  %-10s   %-15s   %s\n", "Key", "Value", "Source")
	fmt.Printf("  %s   %s   %s\n", strings.Repeat("-", 10), strings.Repeat("-", 15), "--------")

	for _, k := range keys {
		envVar := fmt.Sprintf(k.EnvVar, extNameUpper)
		envValue := os.Getenv(envVar)

		var projectValue, globalValue, defaultValue string

		// Get project config value
		if projectExtCfg != nil {
			switch k.Key {
			case "version":
				projectValue = projectExtCfg.Version
			case "automount":
				if projectExtCfg.Automount != nil {
					projectValue = fmt.Sprintf("%v", *projectExtCfg.Automount)
				}
			}
		}

		// Get global config value
		if globalExtCfg != nil {
			switch k.Key {
			case "version":
				globalValue = globalExtCfg.Version
			case "automount":
				if globalExtCfg.Automount != nil {
					globalValue = fmt.Sprintf("%v", *globalExtCfg.Automount)
				}
			}
		}

		// Get extension default value
		if extDefaults != nil {
			switch k.Key {
			case "version":
				defaultValue = extDefaults.DefaultVersion
			case "automount":
				defaultValue = fmt.Sprintf("%v", extDefaults.AutoMount)
			}
		}

		// Determine effective value and source (env > project > global > default)
		var displayValue, source string
		if envValue != "" {
			displayValue = envValue
			source = "env"
		} else if projectValue != "" {
			displayValue = projectValue
			source = "project"
		} else if globalValue != "" {
			displayValue = globalValue
			source = "global"
		} else if defaultValue != "" {
			displayValue = defaultValue
			source = "default"
		} else {
			displayValue = "-"
			source = ""
		}

		if source == "env" || source == "project" || source == "global" {
			fmt.Printf("* %-10s   %-15s   %s\n", k.Key, displayValue, source)
		} else {
			fmt.Printf("  %-10s   %-15s   %s\n", k.Key, displayValue, source)
		}
	}
}

func getExtensionConfig(extName, key string) {
	if !isValidExtensionConfigKey(key) {
		fmt.Printf("Unknown extension config key: %s\n", key)
		fmt.Println("Available keys: version, automount")
		os.Exit(1)
	}

	cfg, err := config.LoadGlobalConfigFile()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}

	var extCfg *config.ExtensionSettings
	if cfg.Extensions != nil {
		extCfg = cfg.Extensions[extName]
	}

	if extCfg == nil {
		fmt.Printf("%s is not set\n", key)
		return
	}

	var val string
	switch key {
	case "version":
		val = extCfg.Version
	case "automount":
		if extCfg.Automount != nil {
			val = fmt.Sprintf("%v", *extCfg.Automount)
		}
	}

	if val == "" {
		fmt.Printf("%s is not set\n", key)
	} else {
		fmt.Println(val)
	}
}

func setExtensionConfig(extName, key, value string, useProject bool) {
	if !isValidExtensionConfigKey(key) {
		fmt.Printf("Unknown extension config key: %s\n", key)
		fmt.Println("Available keys: version, automount")
		os.Exit(1)
	}

	// Validate bool values
	if key == "automount" {
		value = strings.ToLower(value)
		if value != "true" && value != "false" {
			fmt.Printf("Invalid value for %s: must be 'true' or 'false'\n", key)
			os.Exit(1)
		}
	}

	var cfg *config.GlobalConfig
	var err error
	if useProject {
		cfg, err = config.LoadProjectConfigFile()
	} else {
		cfg, err = config.LoadGlobalConfigFile()
	}
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Initialize extensions map if needed
	if cfg.Extensions == nil {
		cfg.Extensions = make(map[string]*config.ExtensionSettings)
	}

	// Initialize extension config if needed
	if cfg.Extensions[extName] == nil {
		cfg.Extensions[extName] = &config.ExtensionSettings{}
	}

	extCfg := cfg.Extensions[extName]
	switch key {
	case "version":
		extCfg.Version = value
	case "automount":
		b := value == "true"
		extCfg.Automount = &b
	}

	if useProject {
		if err := config.SaveProjectConfigFile(cfg); err != nil {
			fmt.Printf("Error saving project config: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Set %s.%s = %s (project)\n", extName, key, value)
	} else {
		if err := config.SaveGlobalConfigFile(cfg); err != nil {
			fmt.Printf("Error saving config: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Set %s.%s = %s\n", extName, key, value)
	}
}

func unsetExtensionConfig(extName, key string, useProject bool) {
	if !isValidExtensionConfigKey(key) {
		fmt.Printf("Unknown extension config key: %s\n", key)
		fmt.Println("Available keys: version, automount")
		os.Exit(1)
	}

	var cfg *config.GlobalConfig
	var err error
	if useProject {
		cfg, err = config.LoadProjectConfigFile()
	} else {
		cfg, err = config.LoadGlobalConfigFile()
	}
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}

	configType := "global"
	if useProject {
		configType = "project"
	}

	if cfg.Extensions == nil || cfg.Extensions[extName] == nil {
		fmt.Printf("%s.%s is not set in %s config\n", extName, key, configType)
		return
	}

	extCfg := cfg.Extensions[extName]
	switch key {
	case "version":
		extCfg.Version = ""
	case "automount":
		extCfg.Automount = nil
	}

	// Clean up empty extension config
	if extCfg.Version == "" && extCfg.Automount == nil {
		delete(cfg.Extensions, extName)
	}

	// Clean up empty extensions map
	if len(cfg.Extensions) == 0 {
		cfg.Extensions = nil
	}

	if useProject {
		if err := config.SaveProjectConfigFile(cfg); err != nil {
			fmt.Printf("Error saving project config: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Unset %s.%s (project)\n", extName, key)
	} else {
		if err := config.SaveGlobalConfigFile(cfg); err != nil {
			fmt.Printf("Error saving config: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Unset %s.%s\n", extName, key)
	}
}
