//go:build addt

package addt

import (
	"testing"
)

// Scenario: A user enables workdir.autotrust globally. The entrypoint
// should set ADDT_EXT_WORKDIR_AUTOTRUST=true for the extension.
func TestWorkdir_Addt_AutotrustGlobalEnabled(t *testing.T) {
	providers := requireProviders(t)

	for _, prov := range providers {
		t.Run(prov, func(t *testing.T) {
			dir, cleanup := setupAddtDirWithExtensions(t, prov, `
workdir:
  autotrust: true
`)
			defer cleanup()
			ensureAddtImage(t, dir, "debug")

			output, err := runRunSubcommand(t, dir, "debug",
				"-c",
				"echo AUTOTRUST:${ADDT_EXT_WORKDIR_AUTOTRUST:-UNSET}")
			if err != nil {
				t.Fatalf("shell command failed: %v\nOutput: %s", err, output)
			}

			result := extractMarker(output, "AUTOTRUST:")
			if result != "true" {
				t.Errorf("Expected AUTOTRUST:true, got AUTOTRUST:%s\nFull output:\n%s",
					result, output)
			}
		})
	}
}

// Scenario: A user does not set workdir.autotrust. The default should
// be true (trust workspace automatically for convenience).
func TestWorkdir_Addt_AutotrustDefaultEnabled(t *testing.T) {
	providers := requireProviders(t)

	for _, prov := range providers {
		t.Run(prov, func(t *testing.T) {
			dir, cleanup := setupAddtDirWithExtensions(t, prov, ``)
			defer cleanup()
			ensureAddtImage(t, dir, "debug")

			output, err := runRunSubcommand(t, dir, "debug",
				"-c",
				"echo AUTOTRUST:${ADDT_EXT_WORKDIR_AUTOTRUST:-UNSET}")
			if err != nil {
				t.Fatalf("shell command failed: %v\nOutput: %s", err, output)
			}

			result := extractMarker(output, "AUTOTRUST:")
			if result != "true" {
				t.Errorf("Expected AUTOTRUST:true (default), got AUTOTRUST:%s\nFull output:\n%s",
					result, output)
			}
		})
	}
}

// Scenario: A user sets per-extension autotrust override that differs
// from the global setting. Per-extension should take precedence.
func TestWorkdir_Addt_AutotrustPerExtensionOverridesGlobal(t *testing.T) {
	providers := requireProviders(t)

	for _, prov := range providers {
		t.Run(prov, func(t *testing.T) {
			dir, cleanup := setupAddtDirWithExtensions(t, prov, `
workdir:
  autotrust: false
extensions:
  debug:
    workdir:
      autotrust: true
`)
			defer cleanup()
			ensureAddtImage(t, dir, "debug")

			output, err := runRunSubcommand(t, dir, "debug",
				"-c",
				"echo AUTOTRUST:${ADDT_EXT_WORKDIR_AUTOTRUST:-UNSET}")
			if err != nil {
				t.Fatalf("shell command failed: %v\nOutput: %s", err, output)
			}

			result := extractMarker(output, "AUTOTRUST:")
			if result != "true" {
				t.Errorf("Expected AUTOTRUST:true (per-extension override), got AUTOTRUST:%s\nFull output:\n%s",
					result, output)
			}
		})
	}
}

// Scenario: A user sets autotrust via environment variable.
// The env var should override config.
func TestWorkdir_Addt_AutotrustEnvVarOverridesConfig(t *testing.T) {
	providers := requireProviders(t)

	for _, prov := range providers {
		t.Run(prov, func(t *testing.T) {
			// Config says autotrust: false
			dir, cleanup := setupAddtDirWithExtensions(t, prov, `
workdir:
  autotrust: false
`)
			defer cleanup()
			ensureAddtImage(t, dir, "debug")

			// Per-extension env var overrides
			t.Setenv("ADDT_DEBUG_WORKDIR_AUTOTRUST", "true")

			output, err := runRunSubcommand(t, dir, "debug",
				"-c",
				"echo AUTOTRUST:${ADDT_EXT_WORKDIR_AUTOTRUST:-UNSET}")
			if err != nil {
				t.Fatalf("shell command failed: %v\nOutput: %s", err, output)
			}

			result := extractMarker(output, "AUTOTRUST:")
			if result != "true" {
				t.Errorf("Expected AUTOTRUST:true (env var override), got AUTOTRUST:%s\nFull output:\n%s",
					result, output)
			}
		})
	}
}
