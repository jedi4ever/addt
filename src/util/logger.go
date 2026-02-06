package util

import (
	"fmt"
	"io"
	"os"
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
// or to a file if one is specified
type Logger struct {
	mu               sync.Mutex
	logFile          string
	enabled          bool
	file             *os.File
	logLevel         LogLevel
	levelInitialized bool // Track if we've initialized from env var
}

var defaultLogger = &Logger{
	enabled:  true,         // Enabled by default, logs go to stderr
	logLevel: LogLevelInfo, // Default to INFO level
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

// InitLogger initializes the singleton logger with the log file path.
// If logFile is empty, logs will go to stderr. If logFile is specified,
// logs will be written to that file.
// The log level is read from ADDT_LOG_LEVEL environment variable (default: INFO).
// If enabled is false, logging is disabled regardless of other settings.
func InitLogger(logFile string, enabled bool) {
	defaultLogger.mu.Lock()
	defer defaultLogger.mu.Unlock()

	// Close existing file if open
	if defaultLogger.file != nil {
		defaultLogger.file.Close()
		defaultLogger.file = nil
	}

	defaultLogger.logFile = logFile
	defaultLogger.enabled = enabled

	// Reset level initialization flag so we re-read env var
	defaultLogger.levelInitialized = false
	// Initialize log level from environment variable
	initLogLevel()
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

	var writer io.Writer

	// If logFile is specified, write to file
	if m.logger.logFile != "" {
		// Open file if not already open
		if m.logger.file == nil {
			f, err := os.OpenFile(m.logger.logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				// Fallback to stderr if file can't be opened
				writer = os.Stderr
			} else {
				m.logger.file = f
				writer = f
			}
		} else {
			writer = m.logger.file
		}
	} else {
		// Default to stderr
		writer = os.Stderr
	}

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(writer, "[%s] %s [%s] %s\n", timestamp, levelStr, m.module, msg)
	// Note: fmt.Fprintf flushes automatically on newlines for terminal output
	// For file output, buffering is acceptable for performance
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
