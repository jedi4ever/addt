//go:build addt

package addt

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"time"
)

// extensionTrustCase holds per-extension test configuration.
type extensionTrustCase struct {
	name   string // extension name (e.g., "claude")
	envKey string // env var holding the API key
}

// trustDialogPatterns are strings that indicate a trust/approval prompt.
// If any of these appear in the captured terminal output (case-insensitive),
// the trust auto-configuration failed.
var trustDialogPatterns = []string{
	"do you trust",
	"trust the files",
	"trust this folder",
	"trust this project",
	"trust this directory",
	"would you like to trust",
	"approve this workspace",
	"--help",
}

// ansiRegex matches ANSI escape sequences (colors, cursor movement, etc.)
var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]|\x1b\][^\x1b]*\x1b\\|\x1b\[[0-9;]*[mGKHJ]`)

// stripAnsiCodes removes ANSI escape sequences from terminal output.
func stripAnsiCodes(s string) string {
	return ansiRegex.ReplaceAllString(s, "")
}

// startTmuxWithCommand creates an isolated tmux server and starts a session
// running the given command in the specified directory.
// Returns socketPath, sessionName, and a cleanup function.
func startTmuxWithCommand(t *testing.T, tmuxBin, dir, command string) (string, string, func()) {
	t.Helper()

	// Create short-path socket dir in /tmp (tmux has ~107 char socket limit)
	socketDir, err := os.MkdirTemp("", "addt-trust-")
	if err != nil {
		t.Fatalf("Failed to create tmux socket dir: %v", err)
	}
	socketPath := filepath.Join(socketDir, "tmux.sock")
	sessionName := "addt-trust"

	// Start tmux server with a session running our command
	fullCmd := fmt.Sprintf("cd %s && %s", dir, command)
	cmd := exec.Command(tmuxBin, "-S", socketPath, "-f", "/dev/null",
		"new-session", "-d", "-s", sessionName, "-x", "200", "-y", "50",
		fullCmd)
	output, err := cmd.CombinedOutput()
	if err != nil {
		os.RemoveAll(socketDir)
		t.Fatalf("Failed to start tmux session: %v\n%s", err, string(output))
	}

	cleanup := func() {
		killCmd := exec.Command(tmuxBin, "-S", socketPath, "kill-server")
		killCmd.Run()
		os.RemoveAll(socketDir)
	}

	return socketPath, sessionName, cleanup
}

// captureTmuxPane captures the full content of a tmux pane.
func captureTmuxPane(t *testing.T, tmuxBin, socketPath, sessionName string) string {
	t.Helper()

	cmd := exec.Command(tmuxBin, "-S", socketPath,
		"capture-pane", "-t", sessionName, "-p", "-S", "-")
	output, err := cmd.Output()
	if err != nil {
		t.Logf("Warning: capture-pane failed: %v", err)
		return ""
	}
	return string(output)
}

// waitForTmuxContent polls the tmux pane until content appears and stabilizes
// (same content on consecutive captures), or the timeout is reached.
func waitForTmuxContent(t *testing.T, tmuxBin, socketPath, sessionName string, timeout time.Duration) string {
	t.Helper()

	deadline := time.Now().Add(timeout)
	var lastContent string
	stableCount := 0

	for time.Now().Before(deadline) {
		content := captureTmuxPane(t, tmuxBin, socketPath, sessionName)
		stripped := strings.TrimSpace(content)

		if stripped == "" {
			// No content yet, keep waiting
			time.Sleep(2 * time.Second)
			continue
		}

		// Content appeared — check if it has stabilized
		if stripped == strings.TrimSpace(lastContent) {
			stableCount++
			if stableCount >= 2 {
				// Content unchanged for 2 consecutive polls — stable
				return content
			}
		} else {
			stableCount = 0
		}

		lastContent = content
		time.Sleep(3 * time.Second)
	}

	return lastContent
}

func TestTrust_Addt_NoTrustPrompt(t *testing.T) {
	// Scenario: Each extension (claude, codex, gemini, copilot) should auto-trust
	// the /workspace directory when workdir.autotrust is enabled. We start the
	// extension inside a tmux pane, wait for it to render, capture the screen,
	// and verify no trust dialog appears.

	tmuxBin := requireTmux(t)
	providers := requireProviders(t)
	binary := getAddtBinary(t)

	extensions := []extensionTrustCase{
		{"claude", "ANTHROPIC_API_KEY"},
		{"codex", "OPENAI_API_KEY"},
		{"gemini", "GEMINI_API_KEY"},
		{"copilot", "COPILOT_GITHUB_TOKEN"},
	}

	for _, ext := range extensions {
		for _, prov := range providers {
			t.Run(ext.name+"/"+prov, func(t *testing.T) {
				// 1. Require the API key for this extension
				apiKey := requireEnvKey(t, ext.envKey)

				// 2. Setup isolated project dir with autotrust config
				// Use setupAddtDir (NOT WithExtensions) to use real embedded extensions
				dir, cleanup := setupAddtDir(t, prov, `
workdir:
  autotrust: true
auth:
  autologin: true
  method: env
`)
				defer cleanup()

				// 3. Set the API key env var
				restoreKey := saveRestoreEnv(t, ext.envKey, apiKey)
				defer restoreKey()

				// 4. Build the extension image using the same binary that will run it
				buildCmd := exec.Command(binary, "build", ext.name)
				buildCmd.Dir = dir
				buildOutput, buildErr := buildCmd.CombinedOutput()
				if buildErr != nil {
					t.Fatalf("Failed to build %s image: %v\n%s", ext.name, buildErr, string(buildOutput))
				}
				t.Logf("Build output: %s", string(buildOutput))

				// 5. Start tmux session with "addt run <extension>"
				command := fmt.Sprintf("%s run %s", binary, ext.name)
				socketPath, sessionName, tmuxCleanup := startTmuxWithCommand(t, tmuxBin, dir, command)
				defer tmuxCleanup()

				// 6. Wait for content to appear (up to 30s)
				content := waitForTmuxContent(t, tmuxBin, socketPath, sessionName, 30*time.Second)

				// 7. Strip ANSI codes for clean text matching
				cleanContent := stripAnsiCodes(content)

				// 8. Save screenshot to testdata/screenshots and log it
				_, thisFile, _, _ := runtime.Caller(0)
				screenshotDir := filepath.Join(filepath.Dir(filepath.Dir(thisFile)), "testdata", "screenshots")
				os.MkdirAll(screenshotDir, 0o755)
				screenshotFile := filepath.Join(screenshotDir, fmt.Sprintf("%s_%s.txt", ext.name, prov))
				if err := os.WriteFile(screenshotFile, []byte(cleanContent), 0o644); err != nil {
					t.Logf("Warning: failed to save screenshot: %v", err)
				} else {
					t.Logf("Screenshot saved to %s", screenshotFile)
				}
				t.Logf("=== tmux screenshot for %s/%s ===\n%s\n=== end screenshot ===", ext.name, prov, cleanContent)

				// 9. Kill the tmux session (and its container) now that we have our screenshot.
				//    The interactive command won't exit on its own, so we kill it
				//    to avoid hanging the test.
				time.Sleep(5 * time.Second)
				tmuxCleanup()

				// 10. Check for absence of trust dialog patterns
				lowerContent := strings.ToLower(cleanContent)
				for _, pattern := range trustDialogPatterns {
					if strings.Contains(lowerContent, pattern) {
						t.Errorf("Trust dialog detected! Found %q in terminal output for %s/%s",
							pattern, ext.name, prov)
					}
				}
			})
		}
	}
}
