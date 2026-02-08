//go:build addt

package addt

import (
	"testing"
)

// Scenario: A user enables config.automount globally. The env var
// ADDT_CONFIG_AUTOMOUNT should be set to true inside the container.
func TestConfigMount_Addt_GlobalAutomountEnabled(t *testing.T) {
	providers := requireProviders(t)

	for _, prov := range providers {
		t.Run(prov, func(t *testing.T) {
			addtHome := t.TempDir()
			defer saveRestoreEnv(t, "ADDT_HOME", addtHome)()

			dir, cleanup := setupAddtDirWithExtensions(t, prov, `
config:
  automount: true
`)
			defer cleanup()
			ensureAddtImage(t, dir, "debug")

			output, err := runRunSubcommand(t, dir, "debug",
				"-c",
				"echo AUTOMOUNT:${ADDT_CONFIG_AUTOMOUNT:-UNSET} && echo READONLY:${ADDT_CONFIG_READONLY:-UNSET}")
			if err != nil {
				t.Fatalf("run failed: %v\nOutput: %s", err, output)
			}

			automount := extractMarker(output, "AUTOMOUNT:")
			if automount != "true" {
				t.Errorf("Expected AUTOMOUNT:true, got AUTOMOUNT:%s\nFull output:\n%s",
					automount, output)
			}
		})
	}
}

// Scenario: Config automount defaults to false when not configured.
func TestConfigMount_Addt_AutomountDefaultDisabled(t *testing.T) {
	providers := requireProviders(t)

	for _, prov := range providers {
		t.Run(prov, func(t *testing.T) {
			addtHome := t.TempDir()
			defer saveRestoreEnv(t, "ADDT_HOME", addtHome)()

			dir, cleanup := setupAddtDirWithExtensions(t, prov, ``)
			defer cleanup()
			ensureAddtImage(t, dir, "debug")

			output, err := runRunSubcommand(t, dir, "debug",
				"-c",
				"echo AUTOMOUNT:${ADDT_CONFIG_AUTOMOUNT:-UNSET}")
			if err != nil {
				t.Fatalf("run failed: %v\nOutput: %s", err, output)
			}

			automount := extractMarker(output, "AUTOMOUNT:")
			if automount != "false" {
				t.Errorf("Expected AUTOMOUNT:false (default), got AUTOMOUNT:%s\nFull output:\n%s",
					automount, output)
			}
		})
	}
}

// Scenario: A user enables config.readonly globally. The env var
// ADDT_CONFIG_READONLY should be set to true inside the container.
func TestConfigMount_Addt_GlobalReadonlyEnabled(t *testing.T) {
	providers := requireProviders(t)

	for _, prov := range providers {
		t.Run(prov, func(t *testing.T) {
			addtHome := t.TempDir()
			defer saveRestoreEnv(t, "ADDT_HOME", addtHome)()

			dir, cleanup := setupAddtDirWithExtensions(t, prov, `
config:
  automount: true
  readonly: true
`)
			defer cleanup()
			ensureAddtImage(t, dir, "debug")

			output, err := runRunSubcommand(t, dir, "debug",
				"-c",
				"echo READONLY:${ADDT_CONFIG_READONLY:-UNSET}")
			if err != nil {
				t.Fatalf("run failed: %v\nOutput: %s", err, output)
			}

			readonly := extractMarker(output, "READONLY:")
			if readonly != "true" {
				t.Errorf("Expected READONLY:true, got READONLY:%s\nFull output:\n%s",
					readonly, output)
			}
		})
	}
}

// Scenario: Config readonly defaults to false when not configured.
func TestConfigMount_Addt_ReadonlyDefaultDisabled(t *testing.T) {
	providers := requireProviders(t)

	for _, prov := range providers {
		t.Run(prov, func(t *testing.T) {
			addtHome := t.TempDir()
			defer saveRestoreEnv(t, "ADDT_HOME", addtHome)()

			dir, cleanup := setupAddtDirWithExtensions(t, prov, ``)
			defer cleanup()
			ensureAddtImage(t, dir, "debug")

			output, err := runRunSubcommand(t, dir, "debug",
				"-c",
				"echo READONLY:${ADDT_CONFIG_READONLY:-UNSET}")
			if err != nil {
				t.Fatalf("run failed: %v\nOutput: %s", err, output)
			}

			readonly := extractMarker(output, "READONLY:")
			if readonly != "false" {
				t.Errorf("Expected READONLY:false (default), got READONLY:%s\nFull output:\n%s",
					readonly, output)
			}
		})
	}
}

// Scenario: A user sets per-extension config automount override via env var.
func TestConfigMount_Addt_PerExtensionAutomountEnvVar(t *testing.T) {
	providers := requireProviders(t)

	for _, prov := range providers {
		t.Run(prov, func(t *testing.T) {
			addtHome := t.TempDir()
			defer saveRestoreEnv(t, "ADDT_HOME", addtHome)()

			// Global config automount is off
			dir, cleanup := setupAddtDirWithExtensions(t, prov, ``)
			defer cleanup()
			ensureAddtImage(t, dir, "debug")

			// Per-extension env var turns it on
			defer saveRestoreEnv(t, "ADDT_DEBUG_CONFIG_AUTOMOUNT", "true")()

			output, err := runRunSubcommand(t, dir, "debug",
				"-c",
				"echo AUTOMOUNT_OVERRIDE:${ADDT_DEBUG_CONFIG_AUTOMOUNT:-UNSET}")
			if err != nil {
				t.Fatalf("run failed: %v\nOutput: %s", err, output)
			}

			result := extractMarker(output, "AUTOMOUNT_OVERRIDE:")
			if result != "true" {
				t.Errorf("Expected AUTOMOUNT_OVERRIDE:true (env var), got AUTOMOUNT_OVERRIDE:%s\nFull output:\n%s",
					result, output)
			}
		})
	}
}

// Scenario: When config.automount is enabled but the extension defines
// no config mounts, the container should still start successfully.
func TestConfigMount_Addt_AutomountNoMountsStillRuns(t *testing.T) {
	providers := requireProviders(t)

	for _, prov := range providers {
		t.Run(prov, func(t *testing.T) {
			addtHome := t.TempDir()
			defer saveRestoreEnv(t, "ADDT_HOME", addtHome)()

			dir, cleanup := setupAddtDirWithExtensions(t, prov, `
config:
  automount: true
`)
			defer cleanup()
			ensureAddtImage(t, dir, "debug")

			output, err := runRunSubcommand(t, dir, "debug",
				"-c", "echo MOUNT_RESULT:OK")
			if err != nil {
				t.Fatalf("run failed: %v\nOutput: %s", err, output)
			}

			result := extractMarker(output, "MOUNT_RESULT:")
			if result != "OK" {
				t.Errorf("Expected container to run successfully with automount=true, got MOUNT_RESULT:%s\nFull output:\n%s",
					result, output)
			}
		})
	}
}
