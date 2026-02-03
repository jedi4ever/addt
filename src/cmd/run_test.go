package cmd

import (
	"os"
	"testing"
)

func TestHandleRunCommand_Help(t *testing.T) {
	testCases := []string{"--help", "-h"}

	for _, flag := range testCases {
		t.Run(flag, func(t *testing.T) {
			result := HandleRunCommand([]string{flag})
			if result != nil {
				t.Errorf("HandleRunCommand(%q) = %v, want nil (help should return nil)", flag, result)
			}
		})
	}
}

func TestHandleRunCommand_NoArgs(t *testing.T) {
	result := HandleRunCommand([]string{})
	if result != nil {
		t.Errorf("HandleRunCommand([]) = %v, want nil (no args should show help)", result)
	}
}

func TestHandleRunCommand_ValidExtension(t *testing.T) {
	// Save original env vars
	origExtensions := os.Getenv("ADDT_EXTENSIONS")
	origCommand := os.Getenv("ADDT_COMMAND")
	defer func() {
		if origExtensions != "" {
			os.Setenv("ADDT_EXTENSIONS", origExtensions)
		} else {
			os.Unsetenv("ADDT_EXTENSIONS")
		}
		if origCommand != "" {
			os.Setenv("ADDT_COMMAND", origCommand)
		} else {
			os.Unsetenv("ADDT_COMMAND")
		}
	}()

	// "claude" is a built-in extension
	result := HandleRunCommand([]string{"claude", "arg1", "arg2"})

	if result == nil {
		t.Fatal("HandleRunCommand(claude, args) = nil, want remaining args")
	}

	if len(result) != 2 {
		t.Errorf("HandleRunCommand returned %d args, want 2", len(result))
	}

	if result[0] != "arg1" || result[1] != "arg2" {
		t.Errorf("HandleRunCommand returned %v, want [arg1 arg2]", result)
	}

	// Check env vars were set
	if os.Getenv("ADDT_EXTENSIONS") != "claude" {
		t.Errorf("ADDT_EXTENSIONS = %q, want %q", os.Getenv("ADDT_EXTENSIONS"), "claude")
	}

	if os.Getenv("ADDT_COMMAND") == "" {
		t.Error("ADDT_COMMAND not set")
	}
}

func TestHandleRunCommand_ValidExtensionNoArgs(t *testing.T) {
	// Save original env vars
	origExtensions := os.Getenv("ADDT_EXTENSIONS")
	origCommand := os.Getenv("ADDT_COMMAND")
	defer func() {
		if origExtensions != "" {
			os.Setenv("ADDT_EXTENSIONS", origExtensions)
		} else {
			os.Unsetenv("ADDT_EXTENSIONS")
		}
		if origCommand != "" {
			os.Setenv("ADDT_COMMAND", origCommand)
		} else {
			os.Unsetenv("ADDT_COMMAND")
		}
	}()

	result := HandleRunCommand([]string{"claude"})

	if result == nil {
		t.Fatal("HandleRunCommand(claude) = nil, want empty slice")
	}

	if len(result) != 0 {
		t.Errorf("HandleRunCommand(claude) returned %d args, want 0", len(result))
	}
}

func TestHandleRunCommand_SetsEntrypoint(t *testing.T) {
	// Save original env vars
	origExtensions := os.Getenv("ADDT_EXTENSIONS")
	origCommand := os.Getenv("ADDT_COMMAND")
	defer func() {
		if origExtensions != "" {
			os.Setenv("ADDT_EXTENSIONS", origExtensions)
		} else {
			os.Unsetenv("ADDT_EXTENSIONS")
		}
		if origCommand != "" {
			os.Setenv("ADDT_COMMAND", origCommand)
		} else {
			os.Unsetenv("ADDT_COMMAND")
		}
	}()

	// Test with an extension that has a different entrypoint
	// "claude" extension has entrypoint "claude"
	HandleRunCommand([]string{"claude"})

	// ADDT_COMMAND should be set to the entrypoint from config.yaml
	command := os.Getenv("ADDT_COMMAND")
	if command != "claude" {
		t.Errorf("ADDT_COMMAND = %q, want %q", command, "claude")
	}
}

// Note: Testing invalid extension would cause os.Exit(1), which is hard to test.
// In production code, you might want to return an error instead of calling os.Exit.
