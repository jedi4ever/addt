//go:build addt

package addt

import (
	"testing"
)

// Scenario: A user does not set any auth config.
// The entrypoint should pass default values: autologin=true, method=auto.
func TestAuth_Addt_GlobalAutologinDefault(t *testing.T) {
	providers := requireProviders(t)

	for _, prov := range providers {
		t.Run(prov, func(t *testing.T) {
			// No auth config — defaults should apply (autologin=true, method=auto)
			dir, cleanup := setupAddtDirWithExtensions(t, prov, ``)
			defer cleanup()
			ensureAddtImage(t, dir, "debug")

			output, err := runRunSubcommand(t, dir, "debug",
				"-c",
				"echo AUTH_AUTOLOGIN:${ADDT_EXT_AUTH_AUTOLOGIN:-UNSET} && echo AUTH_METHOD:${ADDT_EXT_AUTH_METHOD:-UNSET}")
			if err != nil {
				t.Fatalf("shell command failed: %v\nOutput: %s", err, output)
			}

			// Global defaults: autologin=true, method=auto
			autologin := extractMarker(output, "AUTH_AUTOLOGIN:")
			if autologin != "true" {
				t.Errorf("Expected AUTH_AUTOLOGIN:true (default), got AUTH_AUTOLOGIN:%s\nFull output:\n%s",
					autologin, output)
			}

			method := extractMarker(output, "AUTH_METHOD:")
			if method != "auto" {
				t.Errorf("Expected AUTH_METHOD:auto (default), got AUTH_METHOD:%s\nFull output:\n%s",
					method, output)
			}
		})
	}
}

// Scenario: A user overrides auth.autologin to false at the global level.
// All extensions should receive autologin=false.
func TestAuth_Addt_GlobalAutologinDisabled(t *testing.T) {
	providers := requireProviders(t)

	for _, prov := range providers {
		t.Run(prov, func(t *testing.T) {
			dir, cleanup := setupAddtDirWithExtensions(t, prov, `
auth:
  autologin: false
  method: native
`)
			defer cleanup()
			ensureAddtImage(t, dir, "debug")

			output, err := runRunSubcommand(t, dir, "debug",
				"-c",
				"echo AUTH_AUTOLOGIN:${ADDT_EXT_AUTH_AUTOLOGIN:-UNSET} && echo AUTH_METHOD:${ADDT_EXT_AUTH_METHOD:-UNSET}")
			if err != nil {
				t.Fatalf("shell command failed: %v\nOutput: %s", err, output)
			}

			autologin := extractMarker(output, "AUTH_AUTOLOGIN:")
			if autologin != "false" {
				t.Errorf("Expected AUTH_AUTOLOGIN:false (global override), got AUTH_AUTOLOGIN:%s\nFull output:\n%s",
					autologin, output)
			}

			method := extractMarker(output, "AUTH_METHOD:")
			if method != "native" {
				t.Errorf("Expected AUTH_METHOD:native (global override), got AUTH_METHOD:%s\nFull output:\n%s",
					method, output)
			}
		})
	}
}

// Scenario: A user sets per-extension auth override that differs from global.
// The per-extension setting should take precedence.
func TestAuth_Addt_PerExtensionOverridesGlobal(t *testing.T) {
	providers := requireProviders(t)

	for _, prov := range providers {
		t.Run(prov, func(t *testing.T) {
			dir, cleanup := setupAddtDirWithExtensions(t, prov, `
auth:
  autologin: true
  method: auto
extensions:
  debug:
    auth:
      autologin: false
      method: native
`)
			defer cleanup()
			ensureAddtImage(t, dir, "debug")

			output, err := runRunSubcommand(t, dir, "debug",
				"-c",
				"echo AUTH_AUTOLOGIN:${ADDT_EXT_AUTH_AUTOLOGIN:-UNSET} && echo AUTH_METHOD:${ADDT_EXT_AUTH_METHOD:-UNSET}")
			if err != nil {
				t.Fatalf("shell command failed: %v\nOutput: %s", err, output)
			}

			// Per-extension should override global
			autologin := extractMarker(output, "AUTH_AUTOLOGIN:")
			if autologin != "false" {
				t.Errorf("Expected AUTH_AUTOLOGIN:false (per-extension override), got AUTH_AUTOLOGIN:%s\nFull output:\n%s",
					autologin, output)
			}

			method := extractMarker(output, "AUTH_METHOD:")
			if method != "native" {
				t.Errorf("Expected AUTH_METHOD:native (per-extension override), got AUTH_METHOD:%s\nFull output:\n%s",
					method, output)
			}
		})
	}
}

// Scenario: A user sets auth via environment variables.
// Env vars should override config file settings.
func TestAuth_Addt_EnvVarOverridesConfig(t *testing.T) {
	providers := requireProviders(t)

	for _, prov := range providers {
		t.Run(prov, func(t *testing.T) {
			// Config says autologin=true, method=env
			dir, cleanup := setupAddtDirWithExtensions(t, prov, `
auth:
  autologin: true
  method: env
`)
			defer cleanup()
			ensureAddtImage(t, dir, "debug")

			// Per-extension env var overrides config
			t.Setenv("ADDT_DEBUG_AUTH_AUTOLOGIN", "false")
			t.Setenv("ADDT_DEBUG_AUTH_METHOD", "native")

			output, err := runRunSubcommand(t, dir, "debug",
				"-c",
				"echo AUTH_AUTOLOGIN:${ADDT_EXT_AUTH_AUTOLOGIN:-UNSET} && echo AUTH_METHOD:${ADDT_EXT_AUTH_METHOD:-UNSET}")
			if err != nil {
				t.Fatalf("shell command failed: %v\nOutput: %s", err, output)
			}

			autologin := extractMarker(output, "AUTH_AUTOLOGIN:")
			if autologin != "false" {
				t.Errorf("Expected AUTH_AUTOLOGIN:false (env var override), got AUTH_AUTOLOGIN:%s\nFull output:\n%s",
					autologin, output)
			}

			method := extractMarker(output, "AUTH_METHOD:")
			if method != "native" {
				t.Errorf("Expected AUTH_METHOD:native (env var override), got AUTH_METHOD:%s\nFull output:\n%s",
					method, output)
			}
		})
	}
}
