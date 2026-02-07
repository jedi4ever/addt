//go:build addt

package addt

import (
	"strings"
	"testing"

	"github.com/jedi4ever/addt/cmd"
)

// --- Shell completion tests (in-process, no container needed) ---

func TestCompletion_Addt_BashOutput(t *testing.T) {
	// Scenario: User runs 'addt completion bash' to generate a bash completion script.
	// The output should be a valid bash completion with function definition,
	// registration, main commands, config subcommands, and dynamic data.
	_, cleanup := setupAddtDir(t, "", "")
	defer cleanup()

	output := captureOutput(t, func() {
		cmd.HandleCompletionCommand([]string{"bash"})
	})

	// Verify bash completion function definition
	if !strings.Contains(output, "_addt_completions") {
		t.Errorf("Expected bash completion to contain _addt_completions function, got:\n%s", output)
	}

	// Verify completion registration
	if !strings.Contains(output, "complete -F _addt_completions addt") {
		t.Errorf("Expected bash completion to register with 'complete -F _addt_completions addt', got:\n%s", output)
	}

	// Verify main commands are present
	for _, cmd := range []string{"run", "build", "shell", "config", "firewall", "completion"} {
		if !strings.Contains(output, cmd) {
			t.Errorf("Expected bash completion to contain command %q, got:\n%s", cmd, output)
		}
	}

	// Verify config subcommands
	for _, sub := range []string{"list", "get", "set", "unset", "extension"} {
		if !strings.Contains(output, sub) {
			t.Errorf("Expected bash completion to contain config subcommand %q, got:\n%s", sub, output)
		}
	}

	// Verify dynamic extension name is injected
	if !strings.Contains(output, "claude") {
		t.Errorf("Expected bash completion to contain extension 'claude', got:\n%s", output)
	}

	// Verify dynamic config key is injected
	if !strings.Contains(output, "ssh.forward_mode") {
		t.Errorf("Expected bash completion to contain config key 'ssh.forward_mode', got:\n%s", output)
	}
}

func TestCompletion_Addt_ZshOutput(t *testing.T) {
	// Scenario: User runs 'addt completion zsh' to generate a zsh completion script.
	// The output should contain zsh-specific completion directives and dynamic data.
	_, cleanup := setupAddtDir(t, "", "")
	defer cleanup()

	output := captureOutput(t, func() {
		cmd.HandleCompletionCommand([]string{"zsh"})
	})

	// Verify zsh compdef header
	if !strings.Contains(output, "#compdef addt") {
		t.Errorf("Expected zsh completion to start with '#compdef addt', got:\n%s", output)
	}

	// Verify zsh function name
	if !strings.Contains(output, "_addt") {
		t.Errorf("Expected zsh completion to contain _addt function, got:\n%s", output)
	}

	// Verify zsh completion API usage
	if !strings.Contains(output, "_describe") {
		t.Errorf("Expected zsh completion to use _describe API, got:\n%s", output)
	}

	// Verify main command with description
	if !strings.Contains(output, "run:Run an agent") {
		t.Errorf("Expected zsh completion to contain 'run:Run an agent' description, got:\n%s", output)
	}

	// Verify dynamic extension name is injected
	if !strings.Contains(output, "claude") {
		t.Errorf("Expected zsh completion to contain extension 'claude', got:\n%s", output)
	}

	// Verify dynamic config key is injected
	if !strings.Contains(output, "ssh.forward_mode") {
		t.Errorf("Expected zsh completion to contain config key 'ssh.forward_mode', got:\n%s", output)
	}
}

func TestCompletion_Addt_FishOutput(t *testing.T) {
	// Scenario: User runs 'addt completion fish' to generate a fish completion script.
	// The output should contain fish-specific completion directives and dynamic data.
	_, cleanup := setupAddtDir(t, "", "")
	defer cleanup()

	output := captureOutput(t, func() {
		cmd.HandleCompletionCommand([]string{"fish"})
	})

	// Verify fish completion registration
	if !strings.Contains(output, "complete -c addt") {
		t.Errorf("Expected fish completion to contain 'complete -c addt', got:\n%s", output)
	}

	// Verify fish completion helper usage
	if !strings.Contains(output, "__fish_use_subcommand") {
		t.Errorf("Expected fish completion to use __fish_use_subcommand, got:\n%s", output)
	}

	// Verify main commands are present
	for _, cmd := range []string{"run", "build", "config"} {
		if !strings.Contains(output, cmd) {
			t.Errorf("Expected fish completion to contain command %q, got:\n%s", cmd, output)
		}
	}

	// Verify dynamic extension name is injected
	if !strings.Contains(output, "claude") {
		t.Errorf("Expected fish completion to contain extension 'claude', got:\n%s", output)
	}

	// Verify dynamic config key is injected
	if !strings.Contains(output, "ssh.forward_mode") {
		t.Errorf("Expected fish completion to contain config key 'ssh.forward_mode', got:\n%s", output)
	}
}

func TestCompletion_Addt_HelpOutput(t *testing.T) {
	// Scenario: User runs 'addt completion' with no shell argument.
	// Should display help text listing available shells.
	_, cleanup := setupAddtDir(t, "", "")
	defer cleanup()

	output := captureOutput(t, func() {
		cmd.HandleCompletionCommand([]string{})
	})

	// Verify help text mentions supported shells
	for _, shell := range []string{"bash", "zsh", "fish"} {
		if !strings.Contains(output, shell) {
			t.Errorf("Expected help output to mention %q shell, got:\n%s", shell, output)
		}
	}

	// Verify help text mentions completion command
	if !strings.Contains(output, "completion") {
		t.Errorf("Expected help output to mention 'completion', got:\n%s", output)
	}
}

func TestCompletion_Addt_ExtensionsIncluded(t *testing.T) {
	// Scenario: User generates bash completion and verifies that known extensions
	// (claude, codex) are dynamically injected into the completion script.
	_, cleanup := setupAddtDir(t, "", "")
	defer cleanup()

	output := captureOutput(t, func() {
		cmd.HandleCompletionCommand([]string{"bash"})
	})

	// Verify known extensions appear in the completion output
	for _, ext := range []string{"claude", "codex"} {
		if !strings.Contains(output, ext) {
			t.Errorf("Expected bash completion to include extension %q, got:\n%s", ext, output)
		}
	}
}

func TestCompletion_Addt_ConfigKeysIncluded(t *testing.T) {
	// Scenario: User generates bash completion and verifies that known config keys
	// are dynamically injected into the completion script.
	_, cleanup := setupAddtDir(t, "", "")
	defer cleanup()

	output := captureOutput(t, func() {
		cmd.HandleCompletionCommand([]string{"bash"})
	})

	// Verify representative config keys appear in the completion output
	for _, key := range []string{"ssh.forward_mode", "github.forward_token", "ports.expose"} {
		if !strings.Contains(output, key) {
			t.Errorf("Expected bash completion to include config key %q, got:\n%s", key, output)
		}
	}
}
