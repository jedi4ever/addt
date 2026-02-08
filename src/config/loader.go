package config

import (
	"os"
	"strconv"
	"strings"

	"github.com/jedi4ever/addt/config/otel"
	"github.com/jedi4ever/addt/config/security"
	"github.com/jedi4ever/addt/extensions"
)

// LoadConfig loads configuration with precedence: defaults < global config < project config < env vars
func LoadConfig(addtVersion, defaultNodeVersion, defaultGoVersion, defaultUvVersion string, defaultPortRangeStart int) *Config {
	// Load config files (project config overrides global config)
	globalCfg := loadGlobalConfig()
	projectCfg := loadProjectConfig()

	// Start with defaults, then apply global config, then project config, then env vars
	cfg := &Config{
		AddtVersion:           addtVersion,
		ExtensionVersions:     make(map[string]string),
		ExtensionAutomount:    make(map[string]bool),
		ExtensionFlagSettings: make(map[string]map[string]bool),
	}

	// Node version: default -> global -> project -> env
	cfg.NodeVersion = defaultNodeVersion
	if globalCfg.NodeVersion != "" {
		cfg.NodeVersion = globalCfg.NodeVersion
	}
	if projectCfg.NodeVersion != "" {
		cfg.NodeVersion = projectCfg.NodeVersion
	}
	if v := os.Getenv("ADDT_NODE_VERSION"); v != "" {
		cfg.NodeVersion = v
	}

	// Go version: default -> global -> project -> env
	cfg.GoVersion = defaultGoVersion
	if globalCfg.GoVersion != "" {
		cfg.GoVersion = globalCfg.GoVersion
	}
	if projectCfg.GoVersion != "" {
		cfg.GoVersion = projectCfg.GoVersion
	}
	if v := os.Getenv("ADDT_GO_VERSION"); v != "" {
		cfg.GoVersion = v
	}

	// UV version: default -> global -> project -> env
	cfg.UvVersion = defaultUvVersion
	if globalCfg.UvVersion != "" {
		cfg.UvVersion = globalCfg.UvVersion
	}
	if projectCfg.UvVersion != "" {
		cfg.UvVersion = projectCfg.UvVersion
	}
	if v := os.Getenv("ADDT_UV_VERSION"); v != "" {
		cfg.UvVersion = v
	}

	// Ports forward: default (true) -> global -> project -> env
	portsForward := true
	if globalCfg.Ports != nil && globalCfg.Ports.Forward != nil {
		portsForward = *globalCfg.Ports.Forward
	}
	if projectCfg.Ports != nil && projectCfg.Ports.Forward != nil {
		portsForward = *projectCfg.Ports.Forward
	}
	if v := os.Getenv("ADDT_PORTS_FORWARD"); v != "" {
		portsForward = v == "true"
	}

	// Port range start: default -> global -> project -> env
	cfg.PortRangeStart = defaultPortRangeStart
	if globalCfg.Ports != nil && globalCfg.Ports.RangeStart != nil {
		cfg.PortRangeStart = *globalCfg.Ports.RangeStart
	}
	if projectCfg.Ports != nil && projectCfg.Ports.RangeStart != nil {
		cfg.PortRangeStart = *projectCfg.Ports.RangeStart
	}
	if v := os.Getenv("ADDT_PORT_RANGE_START"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			cfg.PortRangeStart = i
		}
	}

	// Ports inject system prompt: default (true) -> global -> project -> env
	cfg.PortsInjectSystemPrompt = true
	if globalCfg.Ports != nil && globalCfg.Ports.InjectSystemPrompt != nil {
		cfg.PortsInjectSystemPrompt = *globalCfg.Ports.InjectSystemPrompt
	}
	if projectCfg.Ports != nil && projectCfg.Ports.InjectSystemPrompt != nil {
		cfg.PortsInjectSystemPrompt = *projectCfg.Ports.InjectSystemPrompt
	}
	if v := os.Getenv("ADDT_PORTS_INJECT_SYSTEM_PROMPT"); v != "" {
		cfg.PortsInjectSystemPrompt = v == "true"
	}

	// SSH forward keys: default (true) -> global -> project -> env
	cfg.SSHForwardKeys = true
	cfg.SSHForwardMode = "proxy"
	if globalCfg.SSH != nil {
		if globalCfg.SSH.ForwardKeys != nil {
			cfg.SSHForwardKeys = *globalCfg.SSH.ForwardKeys
		}
		if globalCfg.SSH.ForwardMode != "" {
			cfg.SSHForwardMode = globalCfg.SSH.ForwardMode
		}
		if len(globalCfg.SSH.AllowedKeys) > 0 {
			cfg.SSHAllowedKeys = globalCfg.SSH.AllowedKeys
		}
	}
	if projectCfg.SSH != nil {
		if projectCfg.SSH.ForwardKeys != nil {
			cfg.SSHForwardKeys = *projectCfg.SSH.ForwardKeys
		}
		if projectCfg.SSH.ForwardMode != "" {
			cfg.SSHForwardMode = projectCfg.SSH.ForwardMode
		}
		if len(projectCfg.SSH.AllowedKeys) > 0 {
			cfg.SSHAllowedKeys = projectCfg.SSH.AllowedKeys
		}
	}
	if v := os.Getenv("ADDT_SSH_FORWARD_KEYS"); v != "" {
		cfg.SSHForwardKeys = v == "true"
	}
	if v := os.Getenv("ADDT_SSH_FORWARD_MODE"); v != "" {
		cfg.SSHForwardMode = v
	}
	if v := os.Getenv("ADDT_SSH_ALLOWED_KEYS"); v != "" {
		cfg.SSHAllowedKeys = strings.Split(v, ",")
	}

	// SSH dir: default ("") -> global -> project -> env
	cfg.SSHDir = ""
	if globalCfg.SSH != nil && globalCfg.SSH.Dir != "" {
		cfg.SSHDir = globalCfg.SSH.Dir
	}
	if projectCfg.SSH != nil && projectCfg.SSH.Dir != "" {
		cfg.SSHDir = projectCfg.SSH.Dir
	}
	if v := os.Getenv("ADDT_SSH_DIR"); v != "" {
		cfg.SSHDir = v
	}

	// Tmux forward: default (false) -> global -> project -> env
	cfg.TmuxForward = false
	if globalCfg.TmuxForward != nil {
		cfg.TmuxForward = *globalCfg.TmuxForward
	}
	if projectCfg.TmuxForward != nil {
		cfg.TmuxForward = *projectCfg.TmuxForward
	}
	if v := os.Getenv("ADDT_TMUX_FORWARD"); v != "" {
		cfg.TmuxForward = v == "true"
	}

	// History persist: default (false) -> global -> project -> env
	cfg.HistoryPersist = false
	if globalCfg.HistoryPersist != nil {
		cfg.HistoryPersist = *globalCfg.HistoryPersist
	}
	if projectCfg.HistoryPersist != nil {
		cfg.HistoryPersist = *projectCfg.HistoryPersist
	}
	if v := os.Getenv("ADDT_HISTORY_PERSIST"); v != "" {
		cfg.HistoryPersist = v == "true"
	}

	// GPG forward: default (off) -> global -> project -> env
	cfg.GPGForward = ""
	if globalCfg.GPG != nil && globalCfg.GPG.Forward != "" {
		cfg.GPGForward = globalCfg.GPG.Forward
	}
	if projectCfg.GPG != nil && projectCfg.GPG.Forward != "" {
		cfg.GPGForward = projectCfg.GPG.Forward
	}
	if v := os.Getenv("ADDT_GPG_FORWARD"); v != "" {
		// Support legacy boolean values
		if v == "true" {
			cfg.GPGForward = "keys"
		} else if v == "false" {
			cfg.GPGForward = ""
		} else {
			cfg.GPGForward = v
		}
	}

	// GPG allowed key IDs: global -> project -> env
	if globalCfg.GPG != nil {
		cfg.GPGAllowedKeyIDs = globalCfg.GPG.AllowedKeyIDs
	}
	if projectCfg.GPG != nil && len(projectCfg.GPG.AllowedKeyIDs) > 0 {
		cfg.GPGAllowedKeyIDs = projectCfg.GPG.AllowedKeyIDs
	}
	if v := os.Getenv("ADDT_GPG_ALLOWED_KEY_IDS"); v != "" {
		cfg.GPGAllowedKeyIDs = strings.Split(v, ",")
	}

	// GPG dir: default ("") -> global -> project -> env
	cfg.GPGDir = ""
	if globalCfg.GPG != nil && globalCfg.GPG.Dir != "" {
		cfg.GPGDir = globalCfg.GPG.Dir
	}
	if projectCfg.GPG != nil && projectCfg.GPG.Dir != "" {
		cfg.GPGDir = projectCfg.GPG.Dir
	}
	if v := os.Getenv("ADDT_GPG_DIR"); v != "" {
		cfg.GPGDir = v
	}

	// DinD mode: default -> global -> project -> env
	if globalCfg.Docker != nil && globalCfg.Docker.Dind != nil {
		cfg.DockerDindMode = globalCfg.Docker.Dind.Mode
	}
	if projectCfg.Docker != nil && projectCfg.Docker.Dind != nil && projectCfg.Docker.Dind.Mode != "" {
		cfg.DockerDindMode = projectCfg.Docker.Dind.Mode
	}
	if v := os.Getenv("ADDT_DOCKER_DIND_MODE"); v != "" {
		cfg.DockerDindMode = v
	}

	// Log output: default (stderr) -> global -> project -> env
	cfg.LogOutput = "stderr"
	if globalCfg.Log != nil && globalCfg.Log.Output != "" {
		cfg.LogOutput = globalCfg.Log.Output
	}
	if projectCfg.Log != nil && projectCfg.Log.Output != "" {
		cfg.LogOutput = projectCfg.Log.Output
	}
	if v := os.Getenv("ADDT_LOG_OUTPUT"); v != "" {
		cfg.LogOutput = v
	}

	// Log file: default -> global -> project -> env
	// Check this first because setting ADDT_LOG_FILE should auto-enable logging
	cfg.LogFile = "addt.log"
	if globalCfg.Log != nil && globalCfg.Log.File != "" {
		cfg.LogFile = globalCfg.Log.File
	}
	if projectCfg.Log != nil && projectCfg.Log.File != "" {
		cfg.LogFile = projectCfg.Log.File
	}
	// Check if ADDT_LOG_FILE is set (even if empty, to allow stderr logging)
	logFileEnvSet := false
	if v, ok := os.LookupEnv("ADDT_LOG_FILE"); ok {
		cfg.LogFile = v // Empty string means stderr, non-empty means file
		logFileEnvSet = true
	}

	// Log enabled: default (false) -> global -> project -> env
	// Auto-enable if ADDT_LOG_FILE is set (even if empty)
	cfg.LogEnabled = logFileEnvSet
	if globalCfg.Log != nil && globalCfg.Log.Enabled != nil {
		cfg.LogEnabled = *globalCfg.Log.Enabled
	}
	if projectCfg.Log != nil && projectCfg.Log.Enabled != nil {
		cfg.LogEnabled = *projectCfg.Log.Enabled
	}
	if v := os.Getenv("ADDT_LOG"); v != "" {
		cfg.LogEnabled = v == "true"
	}

	// Log dir: default (~/.addt/logs) -> global -> project -> env
	cfg.LogDir = ""
	if globalCfg.Log != nil && globalCfg.Log.Dir != "" {
		cfg.LogDir = globalCfg.Log.Dir
	}
	if projectCfg.Log != nil && projectCfg.Log.Dir != "" {
		cfg.LogDir = projectCfg.Log.Dir
	}
	if v := os.Getenv("ADDT_LOG_DIR"); v != "" {
		cfg.LogDir = v
	}

	// Log level: default (INFO) -> global -> project -> env
	cfg.LogLevel = "INFO"
	if globalCfg.Log != nil && globalCfg.Log.Level != "" {
		cfg.LogLevel = globalCfg.Log.Level
	}
	if projectCfg.Log != nil && projectCfg.Log.Level != "" {
		cfg.LogLevel = projectCfg.Log.Level
	}
	if v := os.Getenv("ADDT_LOG_LEVEL"); v != "" {
		cfg.LogLevel = v
	}

	// Log modules: default (*) -> global -> project -> env
	cfg.LogModules = "*"
	if globalCfg.Log != nil && globalCfg.Log.Modules != "" {
		cfg.LogModules = globalCfg.Log.Modules
	}
	if projectCfg.Log != nil && projectCfg.Log.Modules != "" {
		cfg.LogModules = projectCfg.Log.Modules
	}
	if v := os.Getenv("ADDT_LOG_MODULES"); v != "" {
		cfg.LogModules = v
	}

	// Log rotate: default (false) -> global -> project -> env
	cfg.LogRotate = false
	if globalCfg.Log != nil && globalCfg.Log.Rotate != nil {
		cfg.LogRotate = *globalCfg.Log.Rotate
	}
	if projectCfg.Log != nil && projectCfg.Log.Rotate != nil {
		cfg.LogRotate = *projectCfg.Log.Rotate
	}
	if v := os.Getenv("ADDT_LOG_ROTATE"); v != "" {
		cfg.LogRotate = v == "true"
	}

	// Log max size: default (10m) -> global -> project -> env
	cfg.LogMaxSize = "10m"
	if globalCfg.Log != nil && globalCfg.Log.MaxSize != "" {
		cfg.LogMaxSize = globalCfg.Log.MaxSize
	}
	if projectCfg.Log != nil && projectCfg.Log.MaxSize != "" {
		cfg.LogMaxSize = projectCfg.Log.MaxSize
	}
	if v := os.Getenv("ADDT_LOG_MAX_SIZE"); v != "" {
		cfg.LogMaxSize = v
	}

	// Log max files: default (5) -> global -> project -> env
	cfg.LogMaxFiles = 5
	if globalCfg.Log != nil && globalCfg.Log.MaxFiles != nil {
		cfg.LogMaxFiles = *globalCfg.Log.MaxFiles
	}
	if projectCfg.Log != nil && projectCfg.Log.MaxFiles != nil {
		cfg.LogMaxFiles = *projectCfg.Log.MaxFiles
	}
	if v := os.Getenv("ADDT_LOG_MAX_FILES"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			cfg.LogMaxFiles = i
		}
	}

	// Persistent: default (false) -> global -> project -> env
	cfg.Persistent = false
	if globalCfg.Persistent != nil {
		cfg.Persistent = *globalCfg.Persistent
	}
	if projectCfg.Persistent != nil {
		cfg.Persistent = *projectCfg.Persistent
	}
	if v := os.Getenv("ADDT_PERSISTENT"); v != "" {
		cfg.Persistent = v == "true"
	}

	// Workdir automount: default (true) -> global -> project -> env
	cfg.WorkdirAutomount = true
	if globalCfg.Workdir != nil && globalCfg.Workdir.Automount != nil {
		cfg.WorkdirAutomount = *globalCfg.Workdir.Automount
	}
	if projectCfg.Workdir != nil && projectCfg.Workdir.Automount != nil {
		cfg.WorkdirAutomount = *projectCfg.Workdir.Automount
	}
	if v := os.Getenv("ADDT_WORKDIR_AUTOMOUNT"); v != "" {
		cfg.WorkdirAutomount = v != "false"
	}

	// Workdir readonly: default (false) -> global -> project -> env
	cfg.WorkdirReadonly = false
	if globalCfg.Workdir != nil && globalCfg.Workdir.Readonly != nil {
		cfg.WorkdirReadonly = *globalCfg.Workdir.Readonly
	}
	if projectCfg.Workdir != nil && projectCfg.Workdir.Readonly != nil {
		cfg.WorkdirReadonly = *projectCfg.Workdir.Readonly
	}
	if v := os.Getenv("ADDT_WORKDIR_READONLY"); v != "" {
		cfg.WorkdirReadonly = v == "true"
	}

	// Firewall: default (false) -> global -> project -> env
	cfg.FirewallEnabled = false
	if globalCfg.Firewall != nil && globalCfg.Firewall.Enabled != nil {
		cfg.FirewallEnabled = *globalCfg.Firewall.Enabled
	}
	if projectCfg.Firewall != nil && projectCfg.Firewall.Enabled != nil {
		cfg.FirewallEnabled = *projectCfg.Firewall.Enabled
	}
	if v := os.Getenv("ADDT_FIREWALL"); v != "" {
		cfg.FirewallEnabled = v == "true"
	}

	// Firewall mode: default (strict) -> global -> project -> env
	cfg.FirewallMode = "strict"
	if globalCfg.Firewall != nil && globalCfg.Firewall.Mode != "" {
		cfg.FirewallMode = globalCfg.Firewall.Mode
	}
	if projectCfg.Firewall != nil && projectCfg.Firewall.Mode != "" {
		cfg.FirewallMode = projectCfg.Firewall.Mode
	}
	if v := os.Getenv("ADDT_FIREWALL_MODE"); v != "" {
		cfg.FirewallMode = v
	}

	// Firewall rules: keep each layer separate for layered override evaluation
	// Order: Defaults → Extension → Global → Project (project wins)
	if globalCfg.Firewall != nil {
		cfg.GlobalFirewallAllowed = globalCfg.Firewall.Allowed
		cfg.GlobalFirewallDenied = globalCfg.Firewall.Denied
	}
	if projectCfg.Firewall != nil {
		cfg.ProjectFirewallAllowed = projectCfg.Firewall.Allowed
		cfg.ProjectFirewallDenied = projectCfg.Firewall.Denied
	}
	// Extension firewall rules are loaded below after determining the extension

	// GitHub forward token: default (true) -> global -> project -> env
	cfg.GitHubForwardToken = true
	if globalCfg.GitHub != nil && globalCfg.GitHub.ForwardToken != nil {
		cfg.GitHubForwardToken = *globalCfg.GitHub.ForwardToken
	}
	if projectCfg.GitHub != nil && projectCfg.GitHub.ForwardToken != nil {
		cfg.GitHubForwardToken = *projectCfg.GitHub.ForwardToken
	}
	if v := os.Getenv("ADDT_GITHUB_FORWARD_TOKEN"); v != "" {
		cfg.GitHubForwardToken = v == "true"
	}

	// Git forward config: default (true) -> global -> project -> env
	cfg.GitForwardConfig = true
	if globalCfg.Git != nil && globalCfg.Git.ForwardConfig != nil {
		cfg.GitForwardConfig = *globalCfg.Git.ForwardConfig
	}
	if projectCfg.Git != nil && projectCfg.Git.ForwardConfig != nil {
		cfg.GitForwardConfig = *projectCfg.Git.ForwardConfig
	}
	if v := os.Getenv("ADDT_GIT_FORWARD_CONFIG"); v != "" {
		cfg.GitForwardConfig = v == "true"
	}

	// Git config path: default ("") -> global -> project -> env
	cfg.GitConfigPath = ""
	if globalCfg.Git != nil && globalCfg.Git.ConfigPath != "" {
		cfg.GitConfigPath = globalCfg.Git.ConfigPath
	}
	if projectCfg.Git != nil && projectCfg.Git.ConfigPath != "" {
		cfg.GitConfigPath = projectCfg.Git.ConfigPath
	}
	if v := os.Getenv("ADDT_GIT_CONFIG_PATH"); v != "" {
		cfg.GitConfigPath = v
	}

	// GitHub token source: default ("gh_auth") -> global -> project -> env
	cfg.GitHubTokenSource = "gh_auth"
	if globalCfg.GitHub != nil && globalCfg.GitHub.TokenSource != "" {
		cfg.GitHubTokenSource = globalCfg.GitHub.TokenSource
	}
	if projectCfg.GitHub != nil && projectCfg.GitHub.TokenSource != "" {
		cfg.GitHubTokenSource = projectCfg.GitHub.TokenSource
	}
	if v := os.Getenv("ADDT_GITHUB_TOKEN_SOURCE"); v != "" {
		cfg.GitHubTokenSource = v
	}

	// GitHub scope token: default (true) -> global -> project -> env
	cfg.GitHubScopeToken = true
	if globalCfg.GitHub != nil && globalCfg.GitHub.ScopeToken != nil {
		cfg.GitHubScopeToken = *globalCfg.GitHub.ScopeToken
	}
	if projectCfg.GitHub != nil && projectCfg.GitHub.ScopeToken != nil {
		cfg.GitHubScopeToken = *projectCfg.GitHub.ScopeToken
	}
	if v := os.Getenv("ADDT_GITHUB_SCOPE_TOKEN"); v != "" {
		cfg.GitHubScopeToken = v == "true"
	}

	// GitHub scope repos: default ([]) -> global -> project -> env
	cfg.GitHubScopeRepos = nil
	if globalCfg.GitHub != nil && len(globalCfg.GitHub.ScopeRepos) > 0 {
		cfg.GitHubScopeRepos = globalCfg.GitHub.ScopeRepos
	}
	if projectCfg.GitHub != nil && len(projectCfg.GitHub.ScopeRepos) > 0 {
		cfg.GitHubScopeRepos = projectCfg.GitHub.ScopeRepos
	}
	if v := os.Getenv("ADDT_GITHUB_SCOPE_REPOS"); v != "" {
		cfg.GitHubScopeRepos = strings.Split(v, ",")
	}

	// Container CPUs: default (2) -> global -> project -> env
	cfg.ContainerCPUs = "2" // Secure default: limit CPU usage
	if globalCfg.Container != nil && globalCfg.Container.CPUs != "" {
		cfg.ContainerCPUs = globalCfg.Container.CPUs
	}
	if projectCfg.Container != nil && projectCfg.Container.CPUs != "" {
		cfg.ContainerCPUs = projectCfg.Container.CPUs
	}
	if v := os.Getenv("ADDT_CONTAINER_CPUS"); v != "" {
		cfg.ContainerCPUs = v
	}

	// Container Memory: default (4g) -> global -> project -> env
	cfg.ContainerMemory = "4g" // Secure default: limit memory usage
	if globalCfg.Container != nil && globalCfg.Container.Memory != "" {
		cfg.ContainerMemory = globalCfg.Container.Memory
	}
	if projectCfg.Container != nil && projectCfg.Container.Memory != "" {
		cfg.ContainerMemory = projectCfg.Container.Memory
	}
	if v := os.Getenv("ADDT_CONTAINER_MEMORY"); v != "" {
		cfg.ContainerMemory = v
	}

	// Workdir path: default (empty = current dir) -> global -> project -> env
	if globalCfg.Workdir != nil {
		cfg.Workdir = globalCfg.Workdir.Path
	}
	if projectCfg.Workdir != nil && projectCfg.Workdir.Path != "" {
		cfg.Workdir = projectCfg.Workdir.Path
	}
	if v := os.Getenv("ADDT_WORKDIR"); v != "" {
		cfg.Workdir = v
	}

	// Env file load: default (true) -> global -> project -> env
	cfg.EnvFileLoad = true
	if globalCfg.EnvFileLoad != nil {
		cfg.EnvFileLoad = *globalCfg.EnvFileLoad
	}
	if projectCfg.EnvFileLoad != nil {
		cfg.EnvFileLoad = *projectCfg.EnvFileLoad
	}
	if v := os.Getenv("ADDT_ENV_FILE_LOAD"); v != "" {
		cfg.EnvFileLoad = v == "true"
	}

	// Env file path: default ("") -> global -> project -> env
	cfg.EnvFile = globalCfg.EnvFile
	if projectCfg.EnvFile != "" {
		cfg.EnvFile = projectCfg.EnvFile
	}
	if v := os.Getenv("ADDT_ENV_FILE"); v != "" {
		cfg.EnvFile = v
	}

	// These don't have global config equivalents
	cfg.EnvVars = strings.Split(getEnvOrDefault("ADDT_ENV_VARS", "ANTHROPIC_API_KEY,GH_TOKEN"), ",")
	cfg.Mode = getEnvOrDefault("ADDT_MODE", "container")
	// Auto-detect container runtime (Docker > Podman) if not explicitly set
	cfg.Provider = DetectContainerRuntime()
	cfg.Extensions = os.Getenv("ADDT_EXTENSIONS")
	cfg.Command = os.Getenv("ADDT_COMMAND")

	// Load per-extension config from config files
	// Precedence: global config < project config < environment variables
	if globalCfg.Extensions != nil {
		for extName, extCfg := range globalCfg.Extensions {
			if extCfg.Version != "" {
				cfg.ExtensionVersions[extName] = extCfg.Version
			}
			if extCfg.Automount != nil {
				cfg.ExtensionAutomount[extName] = *extCfg.Automount
			}
		}
	}
	if projectCfg.Extensions != nil {
		for extName, extCfg := range projectCfg.Extensions {
			if extCfg.Version != "" {
				cfg.ExtensionVersions[extName] = extCfg.Version
			}
			if extCfg.Automount != nil {
				cfg.ExtensionAutomount[extName] = *extCfg.Automount
			}
		}
	}

	// Load per-extension flag settings from config files
	// Precedence: global config < project config < env vars
	resolveExtensionFlagSettings(cfg, globalCfg, projectCfg)

	// Load extension-specific firewall rules based on ADDT_EXTENSIONS
	// Extension firewall rules are stored in global config under extensions.<name>
	currentExt := os.Getenv("ADDT_EXTENSIONS")
	if currentExt != "" {
		// Use first extension if multiple specified
		extName := strings.Split(currentExt, ",")[0]
		if globalCfg.Extensions != nil && globalCfg.Extensions[extName] != nil {
			extCfg := globalCfg.Extensions[extName]
			cfg.ExtensionFirewallAllowed = extCfg.FirewallAllowed
			cfg.ExtensionFirewallDenied = extCfg.FirewallDenied
		}
	}

	// Load per-extension versions and mount configs from environment (overrides config files)
	// Pattern: ADDT_<EXT>_VERSION and ADDT_<EXT>_AUTOMOUNT
	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key, value := parts[0], parts[1]

		// Check for ADDT_<EXT>_VERSION pattern
		if strings.HasPrefix(key, "ADDT_") && strings.HasSuffix(key, "_VERSION") {
			// Extract extension name (e.g., "ADDT_CLAUDE_VERSION" -> "claude")
			extName := strings.TrimPrefix(key, "ADDT_")
			extName = strings.TrimSuffix(extName, "_VERSION")
			extName = strings.ToLower(extName)
			// Skip non-extension versions (node, go, uv)
			if extName != "node" && extName != "go" && extName != "uv" {
				cfg.ExtensionVersions[extName] = value
			}
		}

		// Check for ADDT_<EXT>_AUTOMOUNT pattern
		if strings.HasPrefix(key, "ADDT_") && strings.HasSuffix(key, "_AUTOMOUNT") {
			// Extract extension name (e.g., "ADDT_CLAUDE_AUTOMOUNT" -> "claude")
			extName := strings.TrimPrefix(key, "ADDT_")
			extName = strings.TrimSuffix(extName, "_AUTOMOUNT")
			extName = strings.ToLower(extName)
			cfg.ExtensionAutomount[extName] = value != "false"
		}
	}

	// Set default version for claude if not specified
	if _, exists := cfg.ExtensionVersions["claude"]; !exists {
		cfg.ExtensionVersions["claude"] = "stable"
	}

	// Ports expose: global -> project -> env
	if globalCfg.Ports != nil && len(globalCfg.Ports.Expose) > 0 {
		cfg.Ports = globalCfg.Ports.Expose
	}
	if projectCfg.Ports != nil && len(projectCfg.Ports.Expose) > 0 {
		cfg.Ports = projectCfg.Ports.Expose
	}
	if ports := os.Getenv("ADDT_PORTS"); ports != "" {
		cfg.Ports = strings.Split(ports, ",")
		for i := range cfg.Ports {
			cfg.Ports[i] = strings.TrimSpace(cfg.Ports[i])
		}
	}

	// If ports.forward is false, clear ports so downstream sees no ports
	if !portsForward {
		cfg.Ports = nil
	}

	// Trim env vars
	for i := range cfg.EnvVars {
		cfg.EnvVars[i] = strings.TrimSpace(cfg.EnvVars[i])
	}

	// Load security configuration using the security package
	cfg.Security = security.LoadConfig(globalCfg.Security, projectCfg.Security)

	// Load OTEL configuration using the otel package
	cfg.Otel = otel.LoadConfig(globalCfg.Otel, projectCfg.Otel)

	return cfg
}

func getEnvOrDefault(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

// resolveExtensionFlagSettings resolves flag settings from config files and env vars
// into cfg.ExtensionFlagSettings. Precedence: global config < project config < env vars
func resolveExtensionFlagSettings(cfg *Config, globalCfg, projectCfg *GlobalConfig) {
	allExts, err := extensions.GetExtensions()
	if err != nil {
		return
	}

	for _, ext := range allExts {
		for _, flag := range ext.Flags {
			if flag.EnvVar == "" {
				continue
			}
			flagKey := strings.TrimPrefix(flag.Flag, "--")

			// Check global config
			if globalCfg.Extensions != nil {
				if extCfg, ok := globalCfg.Extensions[ext.Name]; ok && extCfg.Flags != nil {
					if v, ok := extCfg.Flags[flagKey]; ok && v != nil {
						if cfg.ExtensionFlagSettings[ext.Name] == nil {
							cfg.ExtensionFlagSettings[ext.Name] = make(map[string]bool)
						}
						cfg.ExtensionFlagSettings[ext.Name][flagKey] = *v
					}
				}
			}

			// Check project config (overrides global)
			if projectCfg.Extensions != nil {
				if extCfg, ok := projectCfg.Extensions[ext.Name]; ok && extCfg.Flags != nil {
					if v, ok := extCfg.Flags[flagKey]; ok && v != nil {
						if cfg.ExtensionFlagSettings[ext.Name] == nil {
							cfg.ExtensionFlagSettings[ext.Name] = make(map[string]bool)
						}
						cfg.ExtensionFlagSettings[ext.Name][flagKey] = *v
					}
				}
			}

			// Check env var (overrides config) — pattern: ADDT_EXTENSION_<EXT>_<FLAG>
			envVar := flag.EnvVar
			if v := os.Getenv(envVar); v != "" {
				if cfg.ExtensionFlagSettings[ext.Name] == nil {
					cfg.ExtensionFlagSettings[ext.Name] = make(map[string]bool)
				}
				cfg.ExtensionFlagSettings[ext.Name][flagKey] = v == "true"
			}
		}
	}
}

// mergeStringSlices merges two string slices, removing duplicates
func mergeStringSlices(a, b []string) []string {
	seen := make(map[string]bool)
	var result []string

	for _, s := range a {
		s = strings.TrimSpace(s)
		if s != "" && !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	for _, s := range b {
		s = strings.TrimSpace(s)
		if s != "" && !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	return result
}
