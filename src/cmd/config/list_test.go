package config

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func TestParseVerboseFlag(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantArgs []string
		wantFlag bool
	}{
		{
			name:     "no verbose flag",
			args:     []string{"list"},
			wantArgs: []string{"list"},
			wantFlag: false,
		},
		{
			name:     "short verbose flag",
			args:     []string{"list", "-v"},
			wantArgs: []string{"list"},
			wantFlag: true,
		},
		{
			name:     "long verbose flag",
			args:     []string{"list", "--verbose"},
			wantArgs: []string{"list"},
			wantFlag: true,
		},
		{
			name:     "verbose flag before command",
			args:     []string{"-v", "list"},
			wantArgs: []string{"list"},
			wantFlag: true,
		},
		{
			name:     "verbose with other flags",
			args:     []string{"list", "-v", "-g"},
			wantArgs: []string{"list", "-g"},
			wantFlag: true,
		},
		{
			name:     "empty args",
			args:     []string{},
			wantArgs: nil,
			wantFlag: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotArgs, gotFlag := parseVerboseFlag(tt.args)
			if gotFlag != tt.wantFlag {
				t.Errorf("parseVerboseFlag() flag = %v, want %v", gotFlag, tt.wantFlag)
			}
			if len(gotArgs) != len(tt.wantArgs) {
				t.Errorf("parseVerboseFlag() args = %v, want %v", gotArgs, tt.wantArgs)
				return
			}
			for i, arg := range gotArgs {
				if arg != tt.wantArgs[i] {
					t.Errorf("parseVerboseFlag() args[%d] = %q, want %q", i, arg, tt.wantArgs[i])
				}
			}
		})
	}
}

// captureStdout captures stdout output from a function call.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	fn()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

func TestPrintRowsWithoutVerbose(t *testing.T) {
	rows := []configRow{
		{Key: "foo.bar", Value: "baz", Default: "qux", Source: "project", IsOverridden: true, Description: "A test key"},
		{Key: "abc", Value: "-", Default: "def", Source: "default", IsOverridden: false, Description: "Another key"},
	}

	output := captureStdout(t, func() {
		printRows(rows, false)
	})

	// Should have Key, Value, Default, Source headers
	if !strings.Contains(output, "Key") {
		t.Error("output should contain 'Key' header")
	}
	if !strings.Contains(output, "Value") {
		t.Error("output should contain 'Value' header")
	}
	if !strings.Contains(output, "Source") {
		t.Error("output should contain 'Source' header")
	}
	// Should NOT have Description header
	if strings.Contains(output, "Description") {
		t.Error("output should NOT contain 'Description' header when verbose is false")
	}
	// Should contain row data
	if !strings.Contains(output, "foo.bar") {
		t.Error("output should contain 'foo.bar'")
	}
}

func TestPrintRowsWithVerbose(t *testing.T) {
	rows := []configRow{
		{Key: "foo.bar", Value: "baz", Default: "qux", Source: "project", IsOverridden: true, Description: "A test key"},
		{Key: "abc", Value: "-", Default: "def", Source: "default", IsOverridden: false, Description: ""},
	}

	output := captureStdout(t, func() {
		printRows(rows, true)
	})

	// Should have Description header
	if !strings.Contains(output, "Description") {
		t.Error("output should contain 'Description' header when verbose is true")
	}
	// Should contain the description text
	if !strings.Contains(output, "A test key") {
		t.Error("output should contain description 'A test key'")
	}
	// Empty description should show as "-"
	if !strings.Contains(output, "abc") {
		t.Error("output should contain key 'abc'")
	}
}

func TestPrintRowsOverriddenPrefix(t *testing.T) {
	rows := []configRow{
		{Key: "overridden", Value: "val", Default: "def", Source: "project", IsOverridden: true},
		{Key: "default", Value: "val", Default: "def", Source: "default", IsOverridden: false},
	}

	output := captureStdout(t, func() {
		printRows(rows, false)
	})

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "overridden") && !strings.HasPrefix(line, "*") {
			t.Error("overridden row should have '*' prefix")
		}
		if strings.Contains(line, "default") && !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "*") && line != "" && !strings.Contains(line, "---") && !strings.Contains(line, "Key") && !strings.Contains(line, "Default") {
			t.Error("non-overridden row should have ' ' prefix")
		}
	}
}
