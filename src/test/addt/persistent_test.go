//go:build addt

package addt

import (
	"os"
	"strings"
	"testing"

	configcmd "github.com/jedi4ever/addt/cmd/config"
)

// --- Config tests (in-process, no container needed) ---

func TestPersistent_Addt_DefaultValue(t *testing.T) {
	// Scenario: User starts with no persistent config. The default value
	// should be false with source=default in the config list.
	_, cleanup := setupAddtDir(t, "", ``)
	defer cleanup()

	output := captureOutput(t, func() {
		configcmd.HandleCommand([]string{"list"})
	})

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "persistent") {
			if !strings.Contains(line, "false") {
				t.Errorf("Expected persistent default=false, got line: %s", line)
			}
			if !strings.Contains(line, "default") {
				t.Errorf("Expected persistent source=default, got line: %s", line)
			}
			return
		}
	}
	t.Errorf("Expected persistent key in config list, got:\n%s", output)
}

func TestPersistent_Addt_ConfigLoaded(t *testing.T) {
	// Scenario: User sets persistent: true in .addt.yaml project config,
	// then verifies it appears in config list with value=true and source=project.
	_, cleanup := setupAddtDir(t, "", `
persistent: true
`)
	defer cleanup()

	output := captureOutput(t, func() {
		configcmd.HandleCommand([]string{"list"})
	})

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "persistent") {
			if !strings.Contains(line, "true") {
				t.Errorf("Expected persistent=true, got line: %s", line)
			}
			if !strings.Contains(line, "project") {
				t.Errorf("Expected persistent source=project, got line: %s", line)
			}
			return
		}
	}
	t.Errorf("Expected persistent key in config list, got:\n%s", output)
}

func TestPersistent_Addt_ConfigViaSet(t *testing.T) {
	// Scenario: User enables persistent mode via 'config set' command,
	// then verifies it appears in config list with value=true and source=project.
	_, cleanup := setupAddtDir(t, "", ``)
	defer cleanup()

	captureOutput(t, func() {
		configcmd.HandleCommand([]string{"set", "persistent", "true"})
	})

	output := captureOutput(t, func() {
		configcmd.HandleCommand([]string{"list"})
	})

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "persistent") {
			if !strings.Contains(line, "true") {
				t.Errorf("Expected persistent=true after config set, got line: %s", line)
			}
			if !strings.Contains(line, "project") {
				t.Errorf("Expected persistent source=project after config set, got line: %s", line)
			}
			return
		}
	}
	t.Errorf("Expected persistent key in config list, got:\n%s", output)
}

// --- Container tests (subprocess, both providers, with cleanup) ---

func TestPersistent_Addt_StatePreserved(t *testing.T) {
	// Scenario: User enables persistent mode. First command creates a file
	// inside the container. Second command checks the file exists.
	// This proves the container is reused across invocations.
	providers := requireProviders(t)

	for _, prov := range providers {
		t.Run(prov, func(t *testing.T) {
			if prov == "bwrap" {
				requireNsenter(t)
			}
			dir, cleanup := setupAddtDirWithExtensions(t, prov, `
persistent: true
`)
			defer cleanup()
			ensureAddtImage(t, dir, "debug")

			// Set ADDT_PERSISTENT env var for subprocess robustness
			origPersistent := os.Getenv("ADDT_PERSISTENT")
			os.Setenv("ADDT_PERSISTENT", "true")
			defer func() {
				if origPersistent != "" {
					os.Setenv("ADDT_PERSISTENT", origPersistent)
				} else {
					os.Unsetenv("ADDT_PERSISTENT")
				}
			}()

			// Clean up persistent container when done
			defer func() {
				out, err := runContainersSubcommand(t, dir, "clean")
				t.Logf("Cleanup output:\n%s (err=%v)", out, err)
			}()

			// Run 1: Create a marker file inside the container
			output1, err := runRunSubcommand(t, dir, "debug",
				"-c", "touch /tmp/persist_marker && echo CREATE:done")
			t.Logf("Run 1 output:\n%s", output1)
			if err != nil {
				t.Fatalf("run 1 failed: %v\nOutput:\n%s", err, output1)
			}
			result1 := extractMarker(output1, "CREATE:")
			if result1 != "done" {
				t.Fatalf("Expected CREATE:done, got %q", result1)
			}

			// Run 2: Check marker file exists (container should be reused)
			output2, err := runRunSubcommand(t, dir, "debug",
				"-c", "if [ -f /tmp/persist_marker ]; then echo PERSIST:yes; else echo PERSIST:no; fi")
			t.Logf("Run 2 output:\n%s", output2)
			if err != nil {
				t.Fatalf("run 2 failed: %v\nOutput:\n%s", err, output2)
			}

			result2 := extractMarker(output2, "PERSIST:")
			if result2 != "yes" {
				t.Errorf("Expected PERSIST:yes (container reused), got %q\nFull output:\n%s", result2, output2)
			}
		})
	}
}

func TestPersistent_Addt_EphemeralNoState(t *testing.T) {
	// Scenario: Without persistent mode (default), each run gets a fresh
	// container. A file created in one run should NOT exist in the next.
	providers := requireProviders(t)

	for _, prov := range providers {
		t.Run(prov, func(t *testing.T) {
			dir, cleanup := setupAddtDirWithExtensions(t, prov, ``)
			defer cleanup()
			ensureAddtImage(t, dir, "debug")

			// Run 1: Create a marker file
			output1, err := runRunSubcommand(t, dir, "debug",
				"-c", "touch /tmp/persist_marker && echo CREATE:done")
			t.Logf("Run 1 output:\n%s", output1)
			if err != nil {
				t.Fatalf("run 1 failed: %v\nOutput:\n%s", err, output1)
			}

			// Run 2: Check marker file — should NOT exist (fresh container)
			output2, err := runRunSubcommand(t, dir, "debug",
				"-c", "if [ -f /tmp/persist_marker ]; then echo PERSIST:yes; else echo PERSIST:no; fi")
			t.Logf("Run 2 output:\n%s", output2)
			if err != nil {
				t.Fatalf("run 2 failed: %v\nOutput:\n%s", err, output2)
			}

			result2 := extractMarker(output2, "PERSIST:")
			if result2 != "no" {
				t.Errorf("Expected PERSIST:no (ephemeral container), got %q\nFull output:\n%s", result2, output2)
			}
		})
	}
}

func TestPersistent_Addt_ContainerListed(t *testing.T) {
	// Scenario: User enables persistent mode, runs a command, then uses
	// 'addt containers list' to verify the persistent container appears.
	providers := requireProviders(t)

	for _, prov := range providers {
		t.Run(prov, func(t *testing.T) {
			if prov == "bwrap" {
				requireNsenter(t)
			}
			dir, cleanup := setupAddtDirWithExtensions(t, prov, `
persistent: true
`)
			defer cleanup()
			ensureAddtImage(t, dir, "debug")

			origPersistent := os.Getenv("ADDT_PERSISTENT")
			os.Setenv("ADDT_PERSISTENT", "true")
			defer func() {
				if origPersistent != "" {
					os.Setenv("ADDT_PERSISTENT", origPersistent)
				} else {
					os.Unsetenv("ADDT_PERSISTENT")
				}
			}()

			// Clean up persistent container when done
			defer func() {
				out, err := runContainersSubcommand(t, dir, "clean")
				t.Logf("Cleanup output:\n%s (err=%v)", out, err)
			}()

			// Run a command to create a persistent container
			output, err := runRunSubcommand(t, dir, "debug",
				"-c", "echo LIST_TEST:done")
			t.Logf("Run output:\n%s", output)
			if err != nil {
				t.Fatalf("run failed: %v\nOutput:\n%s", err, output)
			}

			// List containers and verify persistent container appears
			listOutput, err := runContainersSubcommand(t, dir, "list")
			t.Logf("List output:\n%s", listOutput)
			if err != nil {
				t.Fatalf("containers list failed: %v\nOutput:\n%s", err, listOutput)
			}

			if !strings.Contains(listOutput, "addt-persistent-") {
				t.Errorf("Expected persistent container in list (addt-persistent-*), got:\n%s", listOutput)
			}
		})
	}
}

func TestPersistent_Addt_ContainerCleaned(t *testing.T) {
	// Scenario: User enables persistent mode, runs a command creating a
	// persistent container, then runs 'addt containers clean' to verify
	// cleanup works. After clean, the container should no longer be listed.
	providers := requireProviders(t)

	for _, prov := range providers {
		t.Run(prov, func(t *testing.T) {
			if prov == "bwrap" {
				requireNsenter(t)
			}
			dir, cleanup := setupAddtDirWithExtensions(t, prov, `
persistent: true
`)
			defer cleanup()
			ensureAddtImage(t, dir, "debug")

			origPersistent := os.Getenv("ADDT_PERSISTENT")
			os.Setenv("ADDT_PERSISTENT", "true")
			defer func() {
				if origPersistent != "" {
					os.Setenv("ADDT_PERSISTENT", origPersistent)
				} else {
					os.Unsetenv("ADDT_PERSISTENT")
				}
			}()

			// Run a command to create a persistent container
			output, err := runRunSubcommand(t, dir, "debug",
				"-c", "echo CLEAN_TEST:done")
			t.Logf("Run output:\n%s", output)
			if err != nil {
				t.Fatalf("run failed: %v\nOutput:\n%s", err, output)
			}

			// Clean all persistent containers
			cleanOutput, err := runContainersSubcommand(t, dir, "clean")
			t.Logf("Clean output:\n%s", cleanOutput)
			if err != nil {
				t.Fatalf("containers clean failed: %v\nOutput:\n%s", err, cleanOutput)
			}

			if !strings.Contains(cleanOutput, "Cleaned") {
				t.Errorf("Expected 'Cleaned' confirmation, got:\n%s", cleanOutput)
			}

			// List containers — persistent container should be gone
			listOutput, err := runContainersSubcommand(t, dir, "list")
			t.Logf("List after clean:\n%s", listOutput)
			if err != nil {
				t.Fatalf("containers list failed: %v\nOutput:\n%s", err, listOutput)
			}

			if strings.Contains(listOutput, "addt-persistent-") {
				t.Errorf("Expected no persistent containers after clean, got:\n%s", listOutput)
			}
		})
	}
}
