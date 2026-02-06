package core

import (
	"os"
	"path/filepath"

	"github.com/jedi4ever/addt/provider"
	"github.com/jedi4ever/addt/util"
	"github.com/jedi4ever/addt/util/terminal"
)

var optionsLogger = util.Log("options")

// BuildRunOptions creates a RunSpec from the configuration
func BuildRunOptions(p provider.Provider, cfg *provider.Config, name string, args []string, openShell bool) *provider.RunSpec {
	optionsLogger.Debugf("BuildRunOptions called: name=%s, openShell=%v, args=%v", name, openShell, args)

	// Use configured workdir or fall back to current directory
	cwd := cfg.Workdir
	if cwd == "" {
		cwd, _ = os.Getwd()
	}
	optionsLogger.Debugf("Working directory: %s", cwd)

	// Interactive mode: if running in a terminal, allow interactive input
	// Both shell mode and run mode can be interactive when in a terminal
	isTerminal := terminal.IsTerminal()
	isInteractive := isTerminal // Both shell and run can be interactive in a terminal
	optionsLogger.Debugf("Terminal check: isTerminal=%v, openShell=%v, isInteractive=%v", isTerminal, openShell, isInteractive)

	// Build the run spec
	spec := &provider.RunSpec{
		Name:             name,
		ImageName:        cfg.ImageName,
		Args:             args,
		WorkDir:          cwd,
		Interactive:      isInteractive,
		Persistent:       cfg.Persistent,
		Volumes:          BuildVolumes(cfg, cwd),
		Ports:            BuildPorts(cfg),
		Env:              BuildEnvironment(p, cfg),
		SSHForward:       cfg.SSHForward,
		SSHAllowedKeys:   cfg.SSHAllowedKeys,
		TmuxForward:      cfg.TmuxForward,
		HistoryPersist:   cfg.HistoryPersist,
		GPGForward:       cfg.GPGForward,
		GPGAllowedKeyIDs: cfg.GPGAllowedKeyIDs,
		DockerDindMode:   cfg.DockerDindMode,
		DockerCPUs:       cfg.DockerCPUs,
		DockerMemory:     cfg.DockerMemory,
	}
	// Resolve flag → env var mappings (e.g., --yolo → ADDT_EXTENSION_CLAUDE_YOLO=true)
	addFlagEnvVars(spec.Env, cfg, args)

	optionsLogger.Debugf("RunSpec created: Name=%s, ImageName=%s, Interactive=%v, Persistent=%v, DockerDindMode=%s",
		spec.Name, spec.ImageName, spec.Interactive, spec.Persistent, spec.DockerDindMode)

	// Handle args based on mode
	if openShell && len(args) > 0 {
		spec.Args = args
		optionsLogger.Debugf("Shell mode with args: %v", args)
	} else if openShell {
		spec.Args = []string{}
		optionsLogger.Debug("Shell mode without args")
	} else {
		spec.Args = args
		optionsLogger.Debugf("Run mode with args: %v", args)
	}

	// Log command if enabled
	if cfg.LogEnabled {
		optionsLogger.Debug("Command logging enabled, logging command")
		LogCommand(cfg.LogFile, cwd, name, args)
	} else {
		optionsLogger.Debug("Command logging disabled")
	}

	// Add env file path if exists
	addEnvFilePath(spec, cfg, cwd)
	if spec.Env["ADDT_ENV_FILE"] != "" {
		optionsLogger.Debugf("Env file found: %s", spec.Env["ADDT_ENV_FILE"])
	}

	optionsLogger.Debugf("BuildRunOptions completed: spec.Args=%v, spec.Env count=%d", spec.Args, len(spec.Env))
	return spec
}

// addEnvFilePath adds the env file path to the spec if it exists
func addEnvFilePath(spec *provider.RunSpec, cfg *provider.Config, cwd string) {
	envFilePath := cfg.EnvFile
	if envFilePath == "" {
		envFilePath = ".env"
	}
	if !filepath.IsAbs(envFilePath) {
		envFilePath = filepath.Join(cwd, envFilePath)
	}
	if info, err := os.Stat(envFilePath); err == nil && !info.IsDir() {
		spec.Env["ADDT_ENV_FILE"] = envFilePath
	}
}
