package security

// Settings holds container security configuration for YAML parsing
type Settings struct {
	PidsLimit       *int     `yaml:"pids_limit,omitempty"`        // Max number of processes (default: 200)
	UlimitNofile    string   `yaml:"ulimit_nofile,omitempty"`     // File descriptor limit "soft:hard" (default: "4096:8192")
	UlimitNproc     string   `yaml:"ulimit_nproc,omitempty"`      // Process limit "soft:hard" (default: "256:512")
	NoNewPrivileges *bool    `yaml:"no_new_privileges,omitempty"` // Prevent privilege escalation (default: true)
	CapDrop         []string `yaml:"cap_drop,omitempty"`          // Capabilities to drop (default: [ALL])
	CapAdd          []string `yaml:"cap_add,omitempty"`           // Capabilities to add back (default: [CHOWN, SETUID, SETGID])
	ReadOnlyRootfs  *bool    `yaml:"read_only_rootfs,omitempty"`  // Read-only root filesystem (default: false)
	SeccompProfile  string   `yaml:"seccomp_profile,omitempty"`   // Seccomp profile: "default", "unconfined", or path
}

// Config holds runtime security configuration with defaults applied
type Config struct {
	PidsLimit       int      // Max number of processes (default: 200)
	UlimitNofile    string   // File descriptor limit "soft:hard" (default: "4096:8192")
	UlimitNproc     string   // Process limit "soft:hard" (default: "256:512")
	NoNewPrivileges bool     // Prevent privilege escalation (default: true)
	CapDrop         []string // Capabilities to drop (default: [ALL])
	CapAdd          []string // Capabilities to add back (default: [CHOWN, SETUID, SETGID])
	ReadOnlyRootfs  bool     // Read-only root filesystem (default: false)
	SeccompProfile  string   // Seccomp profile (default: "")
}

// DefaultConfig returns a Config with secure defaults applied
func DefaultConfig() Config {
	return Config{
		PidsLimit:       200,
		UlimitNofile:    "4096:8192",
		UlimitNproc:     "256:512",
		NoNewPrivileges: true,
		CapDrop:         []string{"ALL"},
		CapAdd:          []string{"CHOWN", "SETUID", "SETGID"},
		ReadOnlyRootfs:  false,
		SeccompProfile:  "",
	}
}
