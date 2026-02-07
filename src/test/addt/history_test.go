//go:build addt

package addt

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	configcmd "github.com/jedi4ever/addt/cmd/config"
)

// --- Config tests (in-process, no container needed) ---

func TestHistory_Addt_DefaultValue(t *testing.T) {
	// Scenario: User starts with no history config. The default value
	// should be false with source=default in the config list.
	_, cleanup := setupAddtDir(t, "", ``)
	defer cleanup()

	output := captureOutput(t, func() {
		configcmd.HandleCommand([]string{"list"})
	})

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "history_persist") {
			if !strings.Contains(line, "false") {
				t.Errorf("Expected history_persist default=false, got line: %s", line)
			}
			if !strings.Contains(line, "default") {
				t.Errorf("Expected history_persist source=default, got line: %s", line)
			}
			return
		}
	}
	t.Errorf("Expected history_persist key in config list, got:\n%s", output)
}

func TestHistory_Addt_ConfigLoaded(t *testing.T) {
	// Scenario: User sets history_persist: true in .addt.yaml project config,
	// then verifies it appears in config list with value=true and source=project.
	_, cleanup := setupAddtDir(t, "", `
history_persist: true
`)
	defer cleanup()

	output := captureOutput(t, func() {
		configcmd.HandleCommand([]string{"list"})
	})

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "history_persist") {
			if !strings.Contains(line, "true") {
				t.Errorf("Expected history_persist=true, got line: %s", line)
			}
			if !strings.Contains(line, "project") {
				t.Errorf("Expected history_persist source=project, got line: %s", line)
			}
			return
		}
	}
	t.Errorf("Expected history_persist key in config list, got:\n%s", output)
}

func TestHistory_Addt_ConfigViaSet(t *testing.T) {
	// Scenario: User enables history persistence via 'config set' command,
	// then verifies it appears in config list with value=true and source=project.
	_, cleanup := setupAddtDir(t, "", ``)
	defer cleanup()

	captureOutput(t, func() {
		configcmd.HandleCommand([]string{"set", "history_persist", "true"})
	})

	output := captureOutput(t, func() {
		configcmd.HandleCommand([]string{"list"})
	})

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "history_persist") {
			if !strings.Contains(line, "true") {
				t.Errorf("Expected history_persist=true after config set, got line: %s", line)
			}
			if !strings.Contains(line, "project") {
				t.Errorf("Expected history_persist source=project after config set, got line: %s", line)
			}
			return
		}
	}
	t.Errorf("Expected history_persist key in config list, got:\n%s", output)
}

// --- Container tests (subprocess, both providers) ---

func TestHistory_Addt_HistoryFileMounted(t *testing.T) {
	// Scenario: User enables history_persist. The bash history file should
	// be mounted inside the container at /home/addt/.bash_history.
	providers := requireProviders(t)

	for _, prov := range providers {
		t.Run(prov, func(t *testing.T) {
			// Use isolated ADDT_HOME so we don't pollute ~/.addt
			addtHome := t.TempDir()
			origAddtHome := os.Getenv("ADDT_HOME")
			os.Setenv("ADDT_HOME", addtHome)
			defer func() {
				if origAddtHome != "" {
					os.Setenv("ADDT_HOME", origAddtHome)
				} else {
					os.Unsetenv("ADDT_HOME")
				}
			}()

			dir, cleanup := setupAddtDirWithExtensions(t, prov, `
history_persist: true
`)
			defer cleanup()
			ensureAddtImage(t, dir, "debug")

			origHistoryPersist := os.Getenv("ADDT_HISTORY_PERSIST")
			os.Setenv("ADDT_HISTORY_PERSIST", "true")
			defer func() {
				if origHistoryPersist != "" {
					os.Setenv("ADDT_HISTORY_PERSIST", origHistoryPersist)
				} else {
					os.Unsetenv("ADDT_HISTORY_PERSIST")
				}
			}()

			output, err := runRunSubcommand(t, dir, "debug",
				"-c", "if [ -f /home/addt/.bash_history ]; then echo HISTFILE:yes; else echo HISTFILE:no; fi")
			t.Logf("Output:\n%s", output)
			if err != nil {
				t.Fatalf("run failed: %v\nOutput:\n%s", err, output)
			}

			result := extractMarker(output, "HISTFILE:")
			if result != "yes" {
				t.Errorf("Expected .bash_history to be mounted, got %q\nFull output:\n%s", result, output)
			}
		})
	}
}

func TestHistory_Addt_HistoryDirCreatedOnHost(t *testing.T) {
	// Scenario: User enables history_persist and runs a command. The history
	// directory should be created on the host at <ADDT_HOME>/history/<hash>/
	// with bash_history and zsh_history files.
	providers := requireProviders(t)

	for _, prov := range providers {
		t.Run(prov, func(t *testing.T) {
			addtHome := t.TempDir()
			origAddtHome := os.Getenv("ADDT_HOME")
			os.Setenv("ADDT_HOME", addtHome)
			defer func() {
				if origAddtHome != "" {
					os.Setenv("ADDT_HOME", origAddtHome)
				} else {
					os.Unsetenv("ADDT_HOME")
				}
			}()

			dir, cleanup := setupAddtDirWithExtensions(t, prov, `
history_persist: true
`)
			defer cleanup()
			ensureAddtImage(t, dir, "debug")

			origHistoryPersist := os.Getenv("ADDT_HISTORY_PERSIST")
			os.Setenv("ADDT_HISTORY_PERSIST", "true")
			defer func() {
				if origHistoryPersist != "" {
					os.Setenv("ADDT_HISTORY_PERSIST", origHistoryPersist)
				} else {
					os.Unsetenv("ADDT_HISTORY_PERSIST")
				}
			}()

			// Run a command to trigger history dir creation
			output, err := runRunSubcommand(t, dir, "debug",
				"-c", "echo HISTORY_DIR_TEST:ok")
			t.Logf("Output:\n%s", output)
			if err != nil {
				t.Fatalf("run failed: %v\nOutput:\n%s", err, output)
			}

			// Compute expected history dir path: <addtHome>/history/<hash>/
			// Resolve symlinks to match os.Getwd() used in the subprocess (macOS: /var -> /private/var)
			resolvedDir, err := filepath.EvalSymlinks(dir)
			if err != nil {
				resolvedDir = dir
			}
			hash := sha256.Sum256([]byte(resolvedDir))
			projectHash := hex.EncodeToString(hash[:8])
			historyDir := filepath.Join(addtHome, "history", projectHash)

			// Verify history directory exists
			if _, err := os.Stat(historyDir); os.IsNotExist(err) {
				t.Errorf("Expected history dir to be created at %s", historyDir)
			}

			// Verify bash_history file exists
			bashHistory := filepath.Join(historyDir, "bash_history")
			if _, err := os.Stat(bashHistory); os.IsNotExist(err) {
				t.Errorf("Expected bash_history file at %s", bashHistory)
			}

			// Verify zsh_history file exists
			zshHistory := filepath.Join(historyDir, "zsh_history")
			if _, err := os.Stat(zshHistory); os.IsNotExist(err) {
				t.Errorf("Expected zsh_history file at %s", zshHistory)
			}
		})
	}
}

func TestHistory_Addt_HistoryPreservedAcrossRuns(t *testing.T) {
	// Scenario: User enables history_persist. First run writes a marker
	// directly to the mounted .bash_history file. Since the file is a
	// host-mounted volume, the content should persist on the host.
	providers := requireProviders(t)

	for _, prov := range providers {
		t.Run(prov, func(t *testing.T) {
			addtHome := t.TempDir()
			origAddtHome := os.Getenv("ADDT_HOME")
			os.Setenv("ADDT_HOME", addtHome)
			defer func() {
				if origAddtHome != "" {
					os.Setenv("ADDT_HOME", origAddtHome)
				} else {
					os.Unsetenv("ADDT_HOME")
				}
			}()

			dir, cleanup := setupAddtDirWithExtensions(t, prov, `
history_persist: true
`)
			defer cleanup()
			ensureAddtImage(t, dir, "debug")

			origHistoryPersist := os.Getenv("ADDT_HISTORY_PERSIST")
			os.Setenv("ADDT_HISTORY_PERSIST", "true")
			defer func() {
				if origHistoryPersist != "" {
					os.Setenv("ADDT_HISTORY_PERSIST", origHistoryPersist)
				} else {
					os.Unsetenv("ADDT_HISTORY_PERSIST")
				}
			}()

			// Run: Write a marker to the mounted .bash_history file
			output, err := runRunSubcommand(t, dir, "debug",
				"-c", "echo 'history_test_marker_12345' >> /home/addt/.bash_history && echo WRITE:done")
			t.Logf("Output:\n%s", output)
			if err != nil {
				t.Fatalf("run failed: %v\nOutput:\n%s", err, output)
			}

			result := extractMarker(output, "WRITE:")
			if result != "done" {
				t.Fatalf("Expected WRITE:done, got %q", result)
			}

			// Check the host-side history file for the marker
			// Resolve symlinks to match os.Getwd() used in the subprocess (macOS: /var -> /private/var)
			resolvedDir, err := filepath.EvalSymlinks(dir)
			if err != nil {
				resolvedDir = dir
			}
			hash := sha256.Sum256([]byte(resolvedDir))
			projectHash := hex.EncodeToString(hash[:8])
			bashHistory := filepath.Join(addtHome, "history", projectHash, "bash_history")

			content, err := os.ReadFile(bashHistory)
			if err != nil {
				t.Fatalf("Failed to read bash_history on host: %v", err)
			}

			if !strings.Contains(string(content), "history_test_marker_12345") {
				t.Errorf("Expected bash_history to contain 'history_test_marker_12345', got:\n%s", string(content))
			}
		})
	}
}

func TestHistory_Addt_DisabledNoHistoryFile(t *testing.T) {
	// Scenario: User does NOT enable history_persist (default). The
	// .bash_history file should NOT be mounted inside the container.
	providers := requireProviders(t)

	for _, prov := range providers {
		t.Run(prov, func(t *testing.T) {
			addtHome := t.TempDir()
			origAddtHome := os.Getenv("ADDT_HOME")
			os.Setenv("ADDT_HOME", addtHome)
			defer func() {
				if origAddtHome != "" {
					os.Setenv("ADDT_HOME", origAddtHome)
				} else {
					os.Unsetenv("ADDT_HOME")
				}
			}()

			dir, cleanup := setupAddtDirWithExtensions(t, prov, `
history_persist: false
`)
			defer cleanup()
			ensureAddtImage(t, dir, "debug")

			origHistoryPersist := os.Getenv("ADDT_HISTORY_PERSIST")
			os.Setenv("ADDT_HISTORY_PERSIST", "false")
			defer func() {
				if origHistoryPersist != "" {
					os.Setenv("ADDT_HISTORY_PERSIST", origHistoryPersist)
				} else {
					os.Unsetenv("ADDT_HISTORY_PERSIST")
				}
			}()

			// The .bash_history should not be a volume mount (just a regular
			// file or non-existent). Check the host history dir was NOT created.
			output, err := runRunSubcommand(t, dir, "debug",
				"-c", "echo DISABLED:ok")
			t.Logf("Output:\n%s", output)
			if err != nil {
				t.Fatalf("run failed: %v\nOutput:\n%s", err, output)
			}

			result := extractMarker(output, "DISABLED:")
			if result != "ok" {
				t.Errorf("Expected DISABLED:ok, got %q", result)
			}

			// Verify no history directory was created on the host
			// Resolve symlinks to match os.Getwd() used in the subprocess (macOS: /var -> /private/var)
			resolvedDir, err := filepath.EvalSymlinks(dir)
			if err != nil {
				resolvedDir = dir
			}
			hash := sha256.Sum256([]byte(resolvedDir))
			projectHash := hex.EncodeToString(hash[:8])
			historyDir := filepath.Join(addtHome, "history", projectHash)

			if _, err := os.Stat(historyDir); err == nil {
				t.Errorf("Expected no history dir when disabled, but found %s", historyDir)
			}
		})
	}
}

func TestHistory_Addt_ZshHistoryMounted(t *testing.T) {
	// Scenario: User enables history_persist. Both bash and zsh history
	// files should be mounted. Verify zsh_history is accessible too.
	providers := requireProviders(t)

	for _, prov := range providers {
		t.Run(prov, func(t *testing.T) {
			addtHome := t.TempDir()
			origAddtHome := os.Getenv("ADDT_HOME")
			os.Setenv("ADDT_HOME", addtHome)
			defer func() {
				if origAddtHome != "" {
					os.Setenv("ADDT_HOME", origAddtHome)
				} else {
					os.Unsetenv("ADDT_HOME")
				}
			}()

			dir, cleanup := setupAddtDirWithExtensions(t, prov, `
history_persist: true
`)
			defer cleanup()
			ensureAddtImage(t, dir, "debug")

			origHistoryPersist := os.Getenv("ADDT_HISTORY_PERSIST")
			os.Setenv("ADDT_HISTORY_PERSIST", "true")
			defer func() {
				if origHistoryPersist != "" {
					os.Setenv("ADDT_HISTORY_PERSIST", origHistoryPersist)
				} else {
					os.Unsetenv("ADDT_HISTORY_PERSIST")
				}
			}()

			output, err := runRunSubcommand(t, dir, "debug",
				"-c", "if [ -f /home/addt/.zsh_history ]; then echo ZSHFILE:yes; else echo ZSHFILE:no; fi")
			t.Logf("Output:\n%s", output)
			if err != nil {
				t.Fatalf("run failed: %v\nOutput:\n%s", err, output)
			}

			result := extractMarker(output, "ZSHFILE:")
			if result != "yes" {
				t.Errorf("Expected .zsh_history to be mounted, got %q\nFull output:\n%s", result, output)
			}
		})
	}
}
