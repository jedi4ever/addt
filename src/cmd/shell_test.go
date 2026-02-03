package cmd

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestPrintShellHelp(t *testing.T) {
	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	printShellHelp()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// Check for expected content
	expectedPhrases := []string{
		"Usage: addt shell",
		"extension",
		"Examples:",
		"addt shell claude",
		"addt extensions list",
	}

	for _, phrase := range expectedPhrases {
		if !strings.Contains(output, phrase) {
			t.Errorf("printShellHelp() output missing %q", phrase)
		}
	}
}

// Note: Testing HandleShellCommand directly is difficult because it:
// 1. Creates actual providers
// 2. Calls os.Exit on errors
// 3. Runs the orchestrator which needs Docker
//
// For full integration testing, use the built binary:
//   ./dist/addt shell --help
//   ./dist/addt shell nonexistent (should error)
//   ./dist/addt shell claude (requires Docker)
