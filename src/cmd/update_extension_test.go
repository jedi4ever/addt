package cmd

import (
	"testing"
)

func TestHandleUpdateCommand_Help(t *testing.T) {
	testCases := []string{"--help", "-h"}

	for _, flag := range testCases {
		t.Run(flag, func(t *testing.T) {
			// Should not panic; prints help and returns
			HandleUpdateCommand([]string{flag}, "0.0.0-test", "20", "1.21", "0.1.0", 30000)
		})
	}
}

func TestHandleUpdateCommand_NoArgs(t *testing.T) {
	// Should not panic; prints help and returns
	HandleUpdateCommand([]string{}, "0.0.0-test", "20", "1.21", "0.1.0", 30000)
}

// Note: Testing invalid extension or valid extension would cause os.Exit(1) or
// trigger a real build, which are not suitable for unit tests.
