package util

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitLogger_CreatesFile(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test.log")

	InitLogger(logFile, true)
	defer func() { defaultLogger.enabled = false }()

	Log("test").Info("hello")

	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		t.Error("Log file was not created")
	}
}

func TestModuleLogger_WritesAllLevels(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test.log")

	InitLogger(logFile, true)
	defer func() { defaultLogger.enabled = false }()

	logger := Log("credentials")
	logger.Info("found token")
	logger.Warning("token expiring")
	logger.Error("token invalid")

	content, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	if len(lines) != 3 {
		t.Fatalf("Expected 3 log entries, got %d", len(lines))
	}

	expected := []struct {
		level   string
		module  string
		message string
	}{
		{"INFO", "credentials", "found token"},
		{"WARN", "credentials", "token expiring"},
		{"ERROR", "credentials", "token invalid"},
	}

	for i, e := range expected {
		if !strings.Contains(lines[i], e.level) {
			t.Errorf("Line %d missing level %q: %s", i, e.level, lines[i])
		}
		if !strings.Contains(lines[i], "["+e.module+"]") {
			t.Errorf("Line %d missing module %q: %s", i, e.module, lines[i])
		}
		if !strings.Contains(lines[i], e.message) {
			t.Errorf("Line %d missing message %q: %s", i, e.message, lines[i])
		}
	}
}

func TestModuleLogger_FormatArgs(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test.log")

	InitLogger(logFile, true)
	defer func() { defaultLogger.enabled = false }()

	Log("env").Info("script for %s set: %s", "claude", "ANTHROPIC_API_KEY")

	content, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	line := string(content)
	if !strings.Contains(line, "script for claude set: ANTHROPIC_API_KEY") {
		t.Errorf("Format args not applied: %s", line)
	}
}

func TestModuleLogger_WritesToStderrByDefault(t *testing.T) {
	// Reset logger state
	defaultLogger.mu.Lock()
	defaultLogger.enabled = true
	defaultLogger.logFile = ""
	if defaultLogger.file != nil {
		defaultLogger.file.Close()
		defaultLogger.file = nil
	}
	defaultLogger.mu.Unlock()

	// Capture stderr
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	// Write a log message
	Log("test").Info("hello world")

	// Close write end and restore stderr
	w.Close()
	os.Stderr = oldStderr

	// Read captured output
	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if !strings.Contains(output, "hello world") {
		t.Errorf("Expected log output to contain 'hello world', got: %s", output)
	}
	if !strings.Contains(output, "[test]") {
		t.Errorf("Expected log output to contain module '[test]', got: %s", output)
	}
	if !strings.Contains(output, "INFO") {
		t.Errorf("Expected log output to contain level 'INFO', got: %s", output)
	}
}

func TestModuleLogger_WritesToStderrWhenEmptyLogFile(t *testing.T) {
	// Reset logger state
	defaultLogger.mu.Lock()
	defaultLogger.enabled = true
	defaultLogger.logFile = ""
	if defaultLogger.file != nil {
		defaultLogger.file.Close()
		defaultLogger.file = nil
	}
	defaultLogger.mu.Unlock()

	// Initialize with empty log file - should use stderr
	InitLogger("", true)

	// Capture stderr
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	// Write a log message
	Log("test").Info("empty logfile test")

	// Close write end and restore stderr
	w.Close()
	os.Stderr = oldStderr

	// Read captured output
	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if !strings.Contains(output, "empty logfile test") {
		t.Errorf("Expected log output to contain 'empty logfile test', got: %s", output)
	}
}

func TestModuleLogger_HasTimestamp(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test.log")

	InitLogger(logFile, true)
	defer func() { defaultLogger.enabled = false }()

	Log("test").Info("hello")

	content, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	line := string(content)
	if !strings.HasPrefix(line, "[") || !strings.Contains(line, "]") {
		t.Errorf("Log entry missing timestamp: %s", line)
	}
}

func TestModuleLogger_LogLevelFiltering_DefaultInfo(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test.log")

	// Clear environment variable to test default
	oldEnv := os.Getenv("ADDT_LOG_LEVEL")
	os.Unsetenv("ADDT_LOG_LEVEL")
	defer func() {
		if oldEnv != "" {
			os.Setenv("ADDT_LOG_LEVEL", oldEnv)
		}
		defaultLogger.enabled = false
	}()

	// Reset logger to default state
	defaultLogger.mu.Lock()
	defaultLogger.logLevel = LogLevelInfo
	defaultLogger.mu.Unlock()

	InitLogger(logFile, true)

	logger := Log("test")
	logger.Debug("debug message")
	logger.Info("info message")
	logger.Warning("warn message")
	logger.Error("error message")

	content, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	output := string(content)
	if strings.Contains(output, "debug message") {
		t.Error("DEBUG message should be filtered at INFO level")
	}
	if !strings.Contains(output, "info message") {
		t.Error("INFO message should be logged at INFO level")
	}
	if !strings.Contains(output, "warn message") {
		t.Error("WARN message should be logged at INFO level")
	}
	if !strings.Contains(output, "error message") {
		t.Error("ERROR message should be logged at INFO level")
	}
}

func TestModuleLogger_LogLevelFiltering_Debug(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test.log")

	oldEnv := os.Getenv("ADDT_LOG_LEVEL")
	os.Setenv("ADDT_LOG_LEVEL", "DEBUG")
	defer func() {
		if oldEnv != "" {
			os.Setenv("ADDT_LOG_LEVEL", oldEnv)
		} else {
			os.Unsetenv("ADDT_LOG_LEVEL")
		}
		defaultLogger.enabled = false
	}()

	InitLogger(logFile, true)

	logger := Log("test")
	logger.Debug("debug message")
	logger.Info("info message")
	logger.Warning("warn message")
	logger.Error("error message")

	content, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	output := string(content)
	if !strings.Contains(output, "debug message") {
		t.Error("DEBUG message should be logged at DEBUG level")
	}
	if !strings.Contains(output, "info message") {
		t.Error("INFO message should be logged at DEBUG level")
	}
	if !strings.Contains(output, "warn message") {
		t.Error("WARN message should be logged at DEBUG level")
	}
	if !strings.Contains(output, "error message") {
		t.Error("ERROR message should be logged at DEBUG level")
	}
}

func TestModuleLogger_LogLevelFiltering_Warn(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test.log")

	oldEnv := os.Getenv("ADDT_LOG_LEVEL")
	os.Setenv("ADDT_LOG_LEVEL", "WARN")
	defer func() {
		if oldEnv != "" {
			os.Setenv("ADDT_LOG_LEVEL", oldEnv)
		} else {
			os.Unsetenv("ADDT_LOG_LEVEL")
		}
		defaultLogger.enabled = false
	}()

	InitLogger(logFile, true)

	logger := Log("test")
	logger.Debug("debug message")
	logger.Info("info message")
	logger.Warning("warn message")
	logger.Error("error message")

	content, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	output := string(content)
	if strings.Contains(output, "debug message") {
		t.Error("DEBUG message should be filtered at WARN level")
	}
	if strings.Contains(output, "info message") {
		t.Error("INFO message should be filtered at WARN level")
	}
	if !strings.Contains(output, "warn message") {
		t.Error("WARN message should be logged at WARN level")
	}
	if !strings.Contains(output, "error message") {
		t.Error("ERROR message should be logged at WARN level")
	}
}

func TestModuleLogger_LogLevelFiltering_Error(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test.log")

	oldEnv := os.Getenv("ADDT_LOG_LEVEL")
	os.Setenv("ADDT_LOG_LEVEL", "ERROR")
	defer func() {
		if oldEnv != "" {
			os.Setenv("ADDT_LOG_LEVEL", oldEnv)
		} else {
			os.Unsetenv("ADDT_LOG_LEVEL")
		}
		defaultLogger.enabled = false
	}()

	InitLogger(logFile, true)

	logger := Log("test")
	logger.Debug("debug message")
	logger.Info("info message")
	logger.Warning("warn message")
	logger.Error("error message")

	content, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	output := string(content)
	if strings.Contains(output, "debug message") {
		t.Error("DEBUG message should be filtered at ERROR level")
	}
	if strings.Contains(output, "info message") {
		t.Error("INFO message should be filtered at ERROR level")
	}
	if strings.Contains(output, "warn message") {
		t.Error("WARN message should be filtered at ERROR level")
	}
	if !strings.Contains(output, "error message") {
		t.Error("ERROR message should be logged at ERROR level")
	}
}

func TestModuleLogger_LogLevelFiltering_InvalidLevel(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test.log")

	oldEnv := os.Getenv("ADDT_LOG_LEVEL")
	os.Setenv("ADDT_LOG_LEVEL", "INVALID")
	defer func() {
		if oldEnv != "" {
			os.Setenv("ADDT_LOG_LEVEL", oldEnv)
		} else {
			os.Unsetenv("ADDT_LOG_LEVEL")
		}
		defaultLogger.enabled = false
	}()

	InitLogger(logFile, true)

	logger := Log("test")
	logger.Debug("debug message")
	logger.Info("info message")
	logger.Warning("warn message")
	logger.Error("error message")

	content, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	output := string(content)
	// Invalid level should default to INFO
	if strings.Contains(output, "debug message") {
		t.Error("DEBUG message should be filtered with invalid level (defaults to INFO)")
	}
	if !strings.Contains(output, "info message") {
		t.Error("INFO message should be logged with invalid level (defaults to INFO)")
	}
	if !strings.Contains(output, "warn message") {
		t.Error("WARN message should be logged with invalid level (defaults to INFO)")
	}
	if !strings.Contains(output, "error message") {
		t.Error("ERROR message should be logged with invalid level (defaults to INFO)")
	}
}
