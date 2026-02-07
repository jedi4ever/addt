package util

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// LogLevel represents the minimum log level to output
type LogLevel int

const (
	LogLevelDebug LogLevel = iota
	LogLevelInfo
	LogLevelWarn
	LogLevelError
)

// Logger is the singleton logger that writes to stderr by default,
// or to a file/stdout if configured
type Logger struct {
	mu               sync.Mutex
	output           string // "stderr", "stdout", "file"
	logFile          string
	logDir           string
	enabled          bool
	file             *os.File
	logLevel         LogLevel
	levelInitialized bool // Track if we've initialized from env var
	modules          string
	rotate           bool
	maxSize          int64 // in bytes
	maxFiles         int
}

var defaultLogger = &Logger{
	enabled:  true,         // Enabled by default, logs go to stderr
	output:   "stderr",
	logLevel: LogLevelInfo, // Default to INFO level
	modules:  "*",
	maxSize:  10 * 1024 * 1024, // 10MB
	maxFiles: 5,
}

// parseLogLevel parses the log level from environment variable or string
func parseLogLevel(levelStr string) LogLevel {
	levelStr = strings.ToUpper(strings.TrimSpace(levelStr))
	switch levelStr {
	case "DEBUG":
		return LogLevelDebug
	case "INFO":
		return LogLevelInfo
	case "WARN":
		return LogLevelWarn
	case "ERROR":
		return LogLevelError
	default:
		return LogLevelInfo // Default to INFO if invalid
	}
}

// parseMaxSize parses a human-readable size string (e.g., "10m", "1g", "500k")
func parseMaxSize(s string) int64 {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return 10 * 1024 * 1024 // default 10MB
	}
	multiplier := int64(1)
	if strings.HasSuffix(s, "g") || strings.HasSuffix(s, "gb") {
		multiplier = 1024 * 1024 * 1024
		s = strings.TrimRight(s, "gb")
	} else if strings.HasSuffix(s, "m") || strings.HasSuffix(s, "mb") {
		multiplier = 1024 * 1024
		s = strings.TrimRight(s, "mb")
	} else if strings.HasSuffix(s, "k") || strings.HasSuffix(s, "kb") {
		multiplier = 1024
		s = strings.TrimRight(s, "kb")
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil || n <= 0 {
		return 10 * 1024 * 1024 // default 10MB
	}
	return n * multiplier
}

// initLogLevel initializes the log level from environment variable.
// Must be called with defaultLogger.mu locked.
func initLogLevel() {
	if !defaultLogger.levelInitialized {
		if levelStr := os.Getenv("ADDT_LOG_LEVEL"); levelStr != "" {
			defaultLogger.logLevel = parseLogLevel(levelStr)
		}
		defaultLogger.levelInitialized = true
	}
}

// logFilePath returns the full path to the log file
func (l *Logger) logFilePath() string {
	if l.logFile == "" {
		return ""
	}
	if l.logDir != "" {
		return filepath.Join(l.logDir, l.logFile)
	}
	return l.logFile
}

// InitLogger initializes the singleton logger with the log file path.
// If logFile is empty, logs will go to stderr. If logFile is specified,
// logs will be written to that file.
// The log level is read from ADDT_LOG_LEVEL environment variable (default: INFO).
// If enabled is false, logging is disabled regardless of other settings.
func InitLogger(logFile string, enabled bool) {
	output := "stderr"
	if logFile != "" {
		output = "file"
	}
	InitLoggerFull(logFile, "", output, enabled, "", "*", false, "10m", 5)
}

// InitLoggerFull initializes the logger with all configuration options.
// output can be "stderr", "stdout", or "file".
func InitLoggerFull(logFile, logDir, output string, enabled bool, level, modules string, rotate bool, maxSize string, maxFiles int) {
	defaultLogger.mu.Lock()
	defer defaultLogger.mu.Unlock()

	// Close existing file if open
	if defaultLogger.file != nil {
		defaultLogger.file.Close()
		defaultLogger.file = nil
	}

	defaultLogger.logFile = logFile
	defaultLogger.logDir = logDir
	defaultLogger.output = output
	if defaultLogger.output == "" {
		defaultLogger.output = "stderr"
	}
	defaultLogger.enabled = enabled
	defaultLogger.modules = modules
	defaultLogger.rotate = rotate
	defaultLogger.maxSize = parseMaxSize(maxSize)
	if maxFiles > 0 {
		defaultLogger.maxFiles = maxFiles
	}

	// Ensure log directory exists if specified
	if logDir != "" {
		os.MkdirAll(logDir, 0755)
	}

	// Set level from parameter first, then allow env var override
	if level != "" {
		defaultLogger.logLevel = parseLogLevel(level)
	}
	// Reset level initialization flag so we re-read env var (env overrides config)
	defaultLogger.levelInitialized = false
	// Initialize log level from environment variable
	initLogLevel()
}

// isModuleAllowed checks if a module name passes the module filter.
func (l *Logger) isModuleAllowed(module string) bool {
	if l.modules == "" || l.modules == "*" {
		return true
	}
	for _, m := range strings.Split(l.modules, ",") {
		if strings.TrimSpace(m) == module {
			return true
		}
	}
	return false
}

// rotateIfNeeded checks the current log file size and rotates if necessary.
// Must be called with l.mu locked.
func (l *Logger) rotateIfNeeded() {
	if !l.rotate || l.file == nil {
		return
	}
	info, err := l.file.Stat()
	if err != nil || info.Size() < l.maxSize {
		return
	}

	// Close current file
	l.file.Close()
	l.file = nil

	logPath := l.logFilePath()

	// Rotate: remove oldest, shift others
	oldest := fmt.Sprintf("%s.%d", logPath, l.maxFiles)
	os.Remove(oldest)
	for i := l.maxFiles - 1; i >= 1; i-- {
		src := fmt.Sprintf("%s.%d", logPath, i)
		dst := fmt.Sprintf("%s.%d", logPath, i+1)
		os.Rename(src, dst)
	}
	os.Rename(logPath, logPath+".1")
}

// Log returns a module-scoped handle for the singleton logger
func Log(module string) *ModuleLogger {
	return &ModuleLogger{module: module, logger: defaultLogger}
}

// ModuleLogger provides logging scoped to a module name
type ModuleLogger struct {
	module string
	logger *Logger
}

func (m *ModuleLogger) log(level LogLevel, levelStr, format string, args ...interface{}) {
	m.logger.mu.Lock()
	defer m.logger.mu.Unlock()

	// Initialize log level from environment variable if present (idempotent)
	initLogLevel()

	// Check if this log level should be output
	if level < m.logger.logLevel {
		return
	}

	if !m.logger.enabled {
		return
	}

	// Check module filter
	if !m.logger.isModuleAllowed(m.module) {
		return
	}

	// Rotate if needed before writing
	m.logger.rotateIfNeeded()

	var writer io.Writer

	switch m.logger.output {
	case "stdout":
		writer = os.Stdout
	case "file":
		logPath := m.logger.logFilePath()
		if logPath != "" {
			if m.logger.file == nil {
				f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
				if err != nil {
					writer = os.Stderr // fallback
				} else {
					m.logger.file = f
					writer = f
				}
			} else {
				writer = m.logger.file
			}
		} else {
			writer = os.Stderr // fallback if no file specified
		}
	default: // "stderr"
		writer = os.Stderr
	}

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(writer, "[%s] %s [%s] %s\n", timestamp, levelStr, m.module, msg)
}

// Debug logs a debug message (only if log level is DEBUG)
func (m *ModuleLogger) Debug(format string, args ...interface{}) {
	m.log(LogLevelDebug, "DEBUG", format, args...)
}

// Debugf is an alias for Debug (for API consistency)
func (m *ModuleLogger) Debugf(format string, args ...interface{}) {
	m.Debug(format, args...)
}

// Info logs an informational message
func (m *ModuleLogger) Info(format string, args ...interface{}) {
	m.log(LogLevelInfo, "INFO", format, args...)
}

// Infof is an alias for Info (for API consistency)
func (m *ModuleLogger) Infof(format string, args ...interface{}) {
	m.Info(format, args...)
}

// Warning logs a warning message
func (m *ModuleLogger) Warning(format string, args ...interface{}) {
	m.log(LogLevelWarn, "WARN", format, args...)
}

// Warningf is an alias for Warning (for API consistency)
func (m *ModuleLogger) Warningf(format string, args ...interface{}) {
	m.Warning(format, args...)
}

// Error logs an error message
func (m *ModuleLogger) Error(format string, args ...interface{}) {
	m.log(LogLevelError, "ERROR", format, args...)
}

// Errorf is an alias for Error (for API consistency)
func (m *ModuleLogger) Errorf(format string, args ...interface{}) {
	m.Error(format, args...)
}
