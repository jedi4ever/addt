package core

import (
	"os"
	"path/filepath"

	"github.com/jedi4ever/addt/provider"
	"github.com/jedi4ever/addt/util/terminal"
)

// BuildRunOptions creates a RunSpec from the configuration
func BuildRunOptions(p provider.Provider, cfg *provider.Config, name string, args []string, openShell bool) *provider.RunSpec {
	// Use configured workdir or fall back to current directory
	cwd := cfg.Workdir
	if cwd == "" {
		cwd, _ = os.Getwd()
	}

	// Build the run spec
	spec := &provider.RunSpec{
		Name:             name,
		ImageName:        cfg.ImageName,
		Args:             args,
		WorkDir:          cwd,
		Interactive:      terminal.IsTerminal(),
		Persistent:       cfg.Persistent,
		Volumes:          BuildVolumes(cfg, cwd),
		Ports:            BuildPorts(cfg),
		Env:              BuildEnvironment(p, cfg),
		SSHForward:       cfg.SSHForward,
		SSHAllowedKeys:   cfg.SSHAllowedKeys,
		GPGForward:       cfg.GPGForward,
		GPGAllowedKeyIDs: cfg.GPGAllowedKeyIDs,
		DindMode:         cfg.DindMode,
		CPUs:             cfg.CPUs,
		Memory:           cfg.Memory,
	}

	// Handle args based on mode
	if openShell && len(args) > 0 {
		spec.Args = args
	} else if openShell {
		spec.Args = []string{}
	} else {
		spec.Args = args
	}

	// Log command if enabled
	if cfg.LogEnabled {
		LogCommand(cfg.LogFile, cwd, name, args)
	}

	// Add env file path if exists
	addEnvFilePath(spec, cfg, cwd)

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
