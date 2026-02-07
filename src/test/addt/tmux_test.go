//go:build addt

package addt

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	configcmd "github.com/jedi4ever/addt/cmd/config"
)

// --- Helpers for tmux test isolation ---

// tmuxSession holds info about a dedicated tmux server started for testing.
// Uses a separate socket so the user's real tmux is not affected.
type tmuxSession struct {
	socketPath string
	sessionName string
	tmuxEnv    string // value for TMUX env var
}

// startTestTmuxServer starts an isolated tmux server with its own socket.
// Returns the session info and a cleanup function that kills the server.
func startTestTmuxServer(t *testing.T) (*tmuxSession, func()) {
	t.Helper()

	// Require tmux binary
	tmuxBin, err := exec.LookPath("tmux")
	if err != nil {
		t.Skip("tmux not installed, skipping")
	}

	// Create a temp dir for the tmux socket.
	// Use os.MkdirTemp in /tmp to keep the path short — tmux has a ~107 char
	// socket path limit, and Go's t.TempDir() paths are too long.
	socketDir, err2 := os.MkdirTemp("", "addt-tmux-")
	if err2 != nil {
		t.Fatalf("Failed to create tmux socket dir: %v", err2)
	}
	socketPath := filepath.Join(socketDir, "tmux.sock")
	sessionName := "addt-test"

	// Start a new tmux server with a detached session using our custom socket
	cmd := exec.Command(tmuxBin, "-S", socketPath, "-f", "/dev/null",
		"new-session", "-d", "-s", sessionName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Skipf("Failed to start tmux server: %v\n%s", err, string(output))
	}

	// Get the server PID by querying tmux
	pidCmd := exec.Command(tmuxBin, "-S", socketPath, "display-message", "-p", "#{pid}")
	pidOut, err := pidCmd.Output()
	if err != nil {
		// Clean up on failure
		killCmd := exec.Command(tmuxBin, "-S", socketPath, "kill-server")
		killCmd.Run()
		t.Fatalf("Failed to get tmux pid: %v", err)
	}
	pid := strings.TrimSpace(string(pidOut))

	// Build the TMUX env var: <socket>,<pid>,0
	tmuxEnv := fmt.Sprintf("%s,%s,0", socketPath, pid)

	cleanup := func() {
		killCmd := exec.Command(tmuxBin, "-S", socketPath, "kill-server")
		killCmd.Run()
		os.RemoveAll(socketDir)
	}

	return &tmuxSession{
		socketPath:  socketPath,
		sessionName: sessionName,
		tmuxEnv:     tmuxEnv,
	}, cleanup
}

// --- Config tests (in-process, no container needed) ---

func TestTmux_Addt_DefaultValue(t *testing.T) {
	// Scenario: User starts with no tmux config. The default value
	// should be false with source=default in the config list.
	_, cleanup := setupAddtDir(t, "", ``)
	defer cleanup()

	output := captureOutput(t, func() {
		configcmd.HandleCommand([]string{"list"})
	})

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "tmux_forward") {
			if !strings.Contains(line, "false") {
				t.Errorf("Expected tmux_forward default=false, got line: %s", line)
			}
			if !strings.Contains(line, "default") {
				t.Errorf("Expected tmux_forward source=default, got line: %s", line)
			}
			return
		}
	}
	t.Errorf("Expected tmux_forward key in config list, got:\n%s", output)
}

func TestTmux_Addt_ConfigLoaded(t *testing.T) {
	// Scenario: User sets tmux_forward: true in .addt.yaml project config,
	// then verifies it appears in config list with value=true and source=project.
	_, cleanup := setupAddtDir(t, "", `
tmux_forward: true
`)
	defer cleanup()

	output := captureOutput(t, func() {
		configcmd.HandleCommand([]string{"list"})
	})

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "tmux_forward") {
			if !strings.Contains(line, "true") {
				t.Errorf("Expected tmux_forward=true, got line: %s", line)
			}
			if !strings.Contains(line, "project") {
				t.Errorf("Expected tmux_forward source=project, got line: %s", line)
			}
			return
		}
	}
	t.Errorf("Expected tmux_forward key in config list, got:\n%s", output)
}

func TestTmux_Addt_ConfigViaSet(t *testing.T) {
	// Scenario: User enables tmux forwarding via 'config set' command,
	// then verifies it appears in config list with value=true and source=project.
	_, cleanup := setupAddtDir(t, "", ``)
	defer cleanup()

	captureOutput(t, func() {
		configcmd.HandleCommand([]string{"set", "tmux_forward", "true"})
	})

	output := captureOutput(t, func() {
		configcmd.HandleCommand([]string{"list"})
	})

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "tmux_forward") {
			if !strings.Contains(line, "true") {
				t.Errorf("Expected tmux_forward=true after config set, got line: %s", line)
			}
			if !strings.Contains(line, "project") {
				t.Errorf("Expected tmux_forward source=project after config set, got line: %s", line)
			}
			return
		}
	}
	t.Errorf("Expected tmux_forward key in config list, got:\n%s", output)
}

// --- Container tests (subprocess, both providers) ---

func TestTmux_Addt_EnvVarForwarded(t *testing.T) {
	// Scenario: User is inside a tmux session and enables tmux forwarding.
	// The TMUX env var should be available inside the container, proving
	// that the forwarding plumbing passes tmux context to the container.
	providers := requireProviders(t)

	for _, prov := range providers {
		t.Run(prov, func(t *testing.T) {
			// Start our own isolated tmux server
			tmuxSess, tmuxCleanup := startTestTmuxServer(t)
			defer tmuxCleanup()

			dir, cleanup := setupAddtDirWithExtensions(t, prov, `
tmux_forward: true
`)
			defer cleanup()
			ensureAddtImage(t, dir, "debug")

			// Set TMUX env var so the forwarding code detects a session
			origTmux := os.Getenv("TMUX")
			origTmuxForward := os.Getenv("ADDT_TMUX_FORWARD")
			os.Setenv("TMUX", tmuxSess.tmuxEnv)
			os.Setenv("ADDT_TMUX_FORWARD", "true")
			defer func() {
				if origTmux != "" {
					os.Setenv("TMUX", origTmux)
				} else {
					os.Unsetenv("TMUX")
				}
				if origTmuxForward != "" {
					os.Setenv("ADDT_TMUX_FORWARD", origTmuxForward)
				} else {
					os.Unsetenv("ADDT_TMUX_FORWARD")
				}
			}()

			output, err := runRunSubcommand(t, dir, "debug",
				"-c", "echo TMUX_SET:${TMUX:-NOTSET}")
			t.Logf("Output:\n%s", output)
			if err != nil {
				t.Fatalf("run failed: %v\nOutput:\n%s", err, output)
			}

			result := extractMarker(output, "TMUX_SET:")
			if result == "NOTSET" || result == "" {
				t.Errorf("Expected TMUX env var to be set inside container, got %q\nFull output:\n%s", result, output)
			}
		})
	}
}

func TestTmux_Addt_SocketAccessible(t *testing.T) {
	// Scenario: User enables tmux forwarding. The tmux socket should be
	// accessible inside the container. Docker mounts the socket directory
	// directly; Podman on macOS uses a TCP bridge with proxy env vars.
	providers := requireProviders(t)

	for _, prov := range providers {
		t.Run(prov, func(t *testing.T) {
			tmuxSess, tmuxCleanup := startTestTmuxServer(t)
			defer tmuxCleanup()

			dir, cleanup := setupAddtDirWithExtensions(t, prov, `
tmux_forward: true
`)
			defer cleanup()
			ensureAddtImage(t, dir, "debug")

			origTmux := os.Getenv("TMUX")
			origTmuxForward := os.Getenv("ADDT_TMUX_FORWARD")
			os.Setenv("TMUX", tmuxSess.tmuxEnv)
			os.Setenv("ADDT_TMUX_FORWARD", "true")
			defer func() {
				if origTmux != "" {
					os.Setenv("TMUX", origTmux)
				} else {
					os.Unsetenv("TMUX")
				}
				if origTmuxForward != "" {
					os.Setenv("ADDT_TMUX_FORWARD", origTmuxForward)
				} else {
					os.Unsetenv("ADDT_TMUX_FORWARD")
				}
			}()

			if prov == "podman" {
				// Podman on macOS uses a TCP bridge — verify the proxy
				// env vars are set inside the container
				output, err := runRunSubcommand(t, dir, "debug",
					"-c", "echo HOST:${ADDT_TMUX_PROXY_HOST:-NOTSET} && echo PORT:${ADDT_TMUX_PROXY_PORT:-NOTSET}")
				t.Logf("Output:\n%s", output)
				if err != nil {
					t.Fatalf("run failed: %v\nOutput:\n%s", err, output)
				}

				host := extractMarker(output, "HOST:")
				if host == "NOTSET" || host == "" {
					t.Errorf("Expected ADDT_TMUX_PROXY_HOST to be set, got %q\nFull output:\n%s", host, output)
				}

				port := extractMarker(output, "PORT:")
				if port == "NOTSET" || port == "" {
					t.Errorf("Expected ADDT_TMUX_PROXY_PORT to be set, got %q\nFull output:\n%s", port, output)
				}
			} else {
				// Docker mounts the socket directory directly — verify
				// the socket dir exists inside the container
				socketDir := filepath.Dir(tmuxSess.socketPath)
				output, err := runRunSubcommand(t, dir, "debug",
					"-c", fmt.Sprintf("if [ -d %s ]; then echo SOCKET_DIR:yes; else echo SOCKET_DIR:no; fi", socketDir))
				t.Logf("Output:\n%s", output)
				if err != nil {
					t.Fatalf("run failed: %v\nOutput:\n%s", err, output)
				}

				result := extractMarker(output, "SOCKET_DIR:")
				if result != "yes" {
					t.Errorf("Expected tmux socket dir to be mounted, got %q\nFull output:\n%s", result, output)
				}
			}
		})
	}
}

func TestTmux_Addt_DisabledNoForwarding(t *testing.T) {
	// Scenario: User does NOT enable tmux forwarding (default). Even if
	// the host has a TMUX session, the TMUX env var should NOT be inside
	// the container.
	providers := requireProviders(t)

	for _, prov := range providers {
		t.Run(prov, func(t *testing.T) {
			tmuxSess, tmuxCleanup := startTestTmuxServer(t)
			defer tmuxCleanup()

			dir, cleanup := setupAddtDirWithExtensions(t, prov, `
tmux_forward: false
`)
			defer cleanup()
			ensureAddtImage(t, dir, "debug")

			// Set TMUX on host but leave forwarding disabled
			origTmux := os.Getenv("TMUX")
			origTmuxForward := os.Getenv("ADDT_TMUX_FORWARD")
			os.Setenv("TMUX", tmuxSess.tmuxEnv)
			os.Setenv("ADDT_TMUX_FORWARD", "false")
			defer func() {
				if origTmux != "" {
					os.Setenv("TMUX", origTmux)
				} else {
					os.Unsetenv("TMUX")
				}
				if origTmuxForward != "" {
					os.Setenv("ADDT_TMUX_FORWARD", origTmuxForward)
				} else {
					os.Unsetenv("ADDT_TMUX_FORWARD")
				}
			}()

			output, err := runRunSubcommand(t, dir, "debug",
				"-c", "echo TMUX_SET:${TMUX:-NOTSET}")
			t.Logf("Output:\n%s", output)
			if err != nil {
				t.Fatalf("run failed: %v\nOutput:\n%s", err, output)
			}

			result := extractMarker(output, "TMUX_SET:")
			if result != "NOTSET" {
				t.Errorf("Expected TMUX to be NOTSET when forwarding disabled, got %q\nFull output:\n%s", result, output)
			}
		})
	}
}

func TestTmux_Addt_NoSessionNoError(t *testing.T) {
	// Scenario: User enables tmux forwarding but is NOT inside a tmux
	// session (no TMUX env var). The container should start normally
	// without errors — forwarding is silently skipped.
	providers := requireProviders(t)

	for _, prov := range providers {
		t.Run(prov, func(t *testing.T) {
			dir, cleanup := setupAddtDirWithExtensions(t, prov, `
tmux_forward: true
`)
			defer cleanup()
			ensureAddtImage(t, dir, "debug")

			// Ensure TMUX is NOT set on the host
			origTmux := os.Getenv("TMUX")
			origTmuxForward := os.Getenv("ADDT_TMUX_FORWARD")
			os.Unsetenv("TMUX")
			os.Setenv("ADDT_TMUX_FORWARD", "true")
			defer func() {
				if origTmux != "" {
					os.Setenv("TMUX", origTmux)
				} else {
					os.Unsetenv("TMUX")
				}
				if origTmuxForward != "" {
					os.Setenv("ADDT_TMUX_FORWARD", origTmuxForward)
				} else {
					os.Unsetenv("ADDT_TMUX_FORWARD")
				}
			}()

			output, err := runRunSubcommand(t, dir, "debug",
				"-c", "echo NO_SESSION:ok")
			t.Logf("Output:\n%s", output)
			if err != nil {
				t.Fatalf("run failed: %v\nOutput:\n%s", err, output)
			}

			result := extractMarker(output, "NO_SESSION:")
			if result != "ok" {
				t.Errorf("Expected container to start without tmux session, got %q\nFull output:\n%s", result, output)
			}
		})
	}
}
