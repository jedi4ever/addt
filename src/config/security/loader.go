package security

import (
	"os"
	"strconv"
	"strings"
)

// ApplySettings applies Settings overrides to a Config
func ApplySettings(cfg *Config, settings *Settings) {
	if settings == nil {
		return
	}
	if settings.PidsLimit != nil {
		cfg.PidsLimit = *settings.PidsLimit
	}
	if settings.UlimitNofile != "" {
		cfg.UlimitNofile = settings.UlimitNofile
	}
	if settings.UlimitNproc != "" {
		cfg.UlimitNproc = settings.UlimitNproc
	}
	if settings.NoNewPrivileges != nil {
		cfg.NoNewPrivileges = *settings.NoNewPrivileges
	}
	if len(settings.CapDrop) > 0 {
		cfg.CapDrop = settings.CapDrop
	}
	if len(settings.CapAdd) > 0 {
		cfg.CapAdd = settings.CapAdd
	}
	if settings.ReadOnlyRootfs != nil {
		cfg.ReadOnlyRootfs = *settings.ReadOnlyRootfs
	}
	if settings.SeccompProfile != "" {
		cfg.SeccompProfile = settings.SeccompProfile
	}
}

// ApplyEnvOverrides applies environment variable overrides to a Config
func ApplyEnvOverrides(cfg *Config) {
	if v := os.Getenv("ADDT_SECURITY_PIDS_LIMIT"); v != "" {
		if pids, err := strconv.Atoi(v); err == nil {
			cfg.PidsLimit = pids
		}
	}
	if v := os.Getenv("ADDT_SECURITY_ULIMIT_NOFILE"); v != "" {
		cfg.UlimitNofile = v
	}
	if v := os.Getenv("ADDT_SECURITY_ULIMIT_NPROC"); v != "" {
		cfg.UlimitNproc = v
	}
	if v := os.Getenv("ADDT_SECURITY_NO_NEW_PRIVILEGES"); v != "" {
		cfg.NoNewPrivileges = v != "false"
	}
	if v := os.Getenv("ADDT_SECURITY_CAP_DROP"); v != "" {
		cfg.CapDrop = strings.Split(v, ",")
	}
	if v := os.Getenv("ADDT_SECURITY_CAP_ADD"); v != "" {
		cfg.CapAdd = strings.Split(v, ",")
	}
	if v := os.Getenv("ADDT_SECURITY_READ_ONLY_ROOTFS"); v != "" {
		cfg.ReadOnlyRootfs = v == "true"
	}
	if v := os.Getenv("ADDT_SECURITY_SECCOMP_PROFILE"); v != "" {
		cfg.SeccompProfile = v
	}
}

// LoadConfig loads security configuration with full precedence chain
func LoadConfig(globalSettings, projectSettings *Settings) Config {
	cfg := DefaultConfig()
	ApplySettings(&cfg, globalSettings)
	ApplySettings(&cfg, projectSettings)
	ApplyEnvOverrides(&cfg)
	return cfg
}
