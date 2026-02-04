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
	TmpfsTmpSize    string   `yaml:"tmpfs_tmp_size,omitempty"`    // Size of /tmp tmpfs (default: "256m")
	TmpfsHomeSize   string   `yaml:"tmpfs_home_size,omitempty"`   // Size of /home/addt tmpfs (default: "512m")
	SeccompProfile  string   `yaml:"seccomp_profile,omitempty"`   // Seccomp profile: "default", "unconfined", or path
	NetworkMode     string   `yaml:"network_mode,omitempty"`      // Network mode: "bridge", "none", "host" (default: "bridge")
	DisableIPC      *bool    `yaml:"disable_ipc,omitempty"`       // Disable IPC namespace sharing (default: false)
	TimeLimit       int      `yaml:"time_limit,omitempty"`        // Auto-kill container after N minutes (default: 0 = disabled)
	UserNamespace   string   `yaml:"user_namespace,omitempty"`    // User namespace: "host", "private", or "" (default: "")
	DisableDevices  *bool    `yaml:"disable_devices,omitempty"`   // Drop MKNOD capability (default: false)
	MemorySwap      string   `yaml:"memory_swap,omitempty"`       // Memory swap limit: "-1" to disable, or size (default: "")
	SecretsToFiles  *bool    `yaml:"secrets_to_files,omitempty"`  // Write secrets to files instead of env vars (default: false)
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
	TmpfsTmpSize    string   // Size of /tmp tmpfs (default: "256m")
	TmpfsHomeSize   string   // Size of /home/addt tmpfs (default: "512m")
	SeccompProfile  string   // Seccomp profile (default: "")
	NetworkMode     string   // Network mode: "bridge", "none", "host" (default: "bridge")
	DisableIPC      bool     // Disable IPC namespace sharing (default: false)
	TimeLimit       int      // Auto-kill container after N minutes (default: 0 = disabled)
	UserNamespace   string   // User namespace: "host", "private", or "" (default: "")
	DisableDevices  bool     // Drop MKNOD capability (default: false)
	MemorySwap      string   // Memory swap limit: "-1" to disable, or size (default: "")
	SecretsToFiles  bool     // Write secrets to files instead of env vars (default: false)
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
		TmpfsTmpSize:    "256m",
		TmpfsHomeSize:   "512m",
		SeccompProfile:  "",
		NetworkMode:     "", // Empty means use Docker default (bridge)
		DisableIPC:      false,
		TimeLimit:       0,  // 0 = disabled
		UserNamespace:   "", // Empty = Docker default
		DisableDevices:  false,
		MemorySwap:      "", // Empty = Docker default
		SecretsToFiles:  false,
	}
}
