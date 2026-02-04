package security

import (
	"os"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.PidsLimit != 200 {
		t.Errorf("PidsLimit = %d, want 200", cfg.PidsLimit)
	}
	if cfg.UlimitNofile != "4096:8192" {
		t.Errorf("UlimitNofile = %q, want \"4096:8192\"", cfg.UlimitNofile)
	}
	if cfg.UlimitNproc != "256:512" {
		t.Errorf("UlimitNproc = %q, want \"256:512\"", cfg.UlimitNproc)
	}
	if !cfg.NoNewPrivileges {
		t.Error("NoNewPrivileges = false, want true")
	}
	if len(cfg.CapDrop) != 1 || cfg.CapDrop[0] != "ALL" {
		t.Errorf("CapDrop = %v, want [ALL]", cfg.CapDrop)
	}
	if len(cfg.CapAdd) != 3 {
		t.Errorf("CapAdd = %v, want [CHOWN, SETUID, SETGID]", cfg.CapAdd)
	}
	if cfg.ReadOnlyRootfs {
		t.Error("ReadOnlyRootfs = true, want false")
	}
}

func TestApplySettings(t *testing.T) {
	cfg := DefaultConfig()

	pidsLimit := 500
	noNewPriv := false
	settings := &Settings{
		PidsLimit:       &pidsLimit,
		NoNewPrivileges: &noNewPriv,
		CapDrop:         []string{"NET_RAW"},
		CapAdd:          []string{"MKNOD"},
	}

	ApplySettings(&cfg, settings)

	if cfg.PidsLimit != 500 {
		t.Errorf("PidsLimit = %d, want 500", cfg.PidsLimit)
	}
	if cfg.NoNewPrivileges {
		t.Error("NoNewPrivileges = true, want false")
	}
	if len(cfg.CapDrop) != 1 || cfg.CapDrop[0] != "NET_RAW" {
		t.Errorf("CapDrop = %v, want [NET_RAW]", cfg.CapDrop)
	}
	if len(cfg.CapAdd) != 1 || cfg.CapAdd[0] != "MKNOD" {
		t.Errorf("CapAdd = %v, want [MKNOD]", cfg.CapAdd)
	}
	// Unchanged values should remain at defaults
	if cfg.UlimitNofile != "4096:8192" {
		t.Errorf("UlimitNofile = %q, want \"4096:8192\" (unchanged)", cfg.UlimitNofile)
	}
}

func TestApplyEnvOverrides(t *testing.T) {
	cfg := DefaultConfig()

	// Set env overrides
	os.Setenv("ADDT_SECURITY_PIDS_LIMIT", "300")
	os.Setenv("ADDT_SECURITY_NO_NEW_PRIVILEGES", "false")
	os.Setenv("ADDT_SECURITY_CAP_DROP", "NET_RAW,SYS_ADMIN")
	os.Setenv("ADDT_SECURITY_CAP_ADD", "MKNOD")
	defer func() {
		os.Unsetenv("ADDT_SECURITY_PIDS_LIMIT")
		os.Unsetenv("ADDT_SECURITY_NO_NEW_PRIVILEGES")
		os.Unsetenv("ADDT_SECURITY_CAP_DROP")
		os.Unsetenv("ADDT_SECURITY_CAP_ADD")
	}()

	ApplyEnvOverrides(&cfg)

	if cfg.PidsLimit != 300 {
		t.Errorf("PidsLimit = %d, want 300 (from env)", cfg.PidsLimit)
	}
	if cfg.NoNewPrivileges {
		t.Error("NoNewPrivileges = true, want false (from env)")
	}
	if len(cfg.CapDrop) != 2 || cfg.CapDrop[0] != "NET_RAW" || cfg.CapDrop[1] != "SYS_ADMIN" {
		t.Errorf("CapDrop = %v, want [NET_RAW, SYS_ADMIN] (from env)", cfg.CapDrop)
	}
	if len(cfg.CapAdd) != 1 || cfg.CapAdd[0] != "MKNOD" {
		t.Errorf("CapAdd = %v, want [MKNOD] (from env)", cfg.CapAdd)
	}
}

func TestLoadConfig(t *testing.T) {
	// Clear env vars
	os.Unsetenv("ADDT_SECURITY_PIDS_LIMIT")
	os.Unsetenv("ADDT_SECURITY_CAP_DROP")

	pidsLimit := 400
	globalSettings := &Settings{
		PidsLimit: &pidsLimit,
	}

	projectPids := 600
	projectSettings := &Settings{
		PidsLimit: &projectPids,
		CapDrop:   []string{"NET_RAW"},
	}

	cfg := LoadConfig(globalSettings, projectSettings)

	// Project should override global
	if cfg.PidsLimit != 600 {
		t.Errorf("PidsLimit = %d, want 600 (project overrides global)", cfg.PidsLimit)
	}
	if len(cfg.CapDrop) != 1 || cfg.CapDrop[0] != "NET_RAW" {
		t.Errorf("CapDrop = %v, want [NET_RAW] (from project)", cfg.CapDrop)
	}
	// CapAdd should remain at default since neither global nor project set it
	if len(cfg.CapAdd) != 3 {
		t.Errorf("CapAdd = %v, want [CHOWN, SETUID, SETGID] (default)", cfg.CapAdd)
	}
}
