package config

import (
	"fmt"
	"strconv"

	cfgtypes "github.com/jedi4ever/addt/config"
)

// GetLogKeys returns all valid log config keys
func GetLogKeys() []KeyInfo {
	return []KeyInfo{
		{Key: "log.enabled", Description: "Enable command logging", Type: "bool", EnvVar: "ADDT_LOG"},
		{Key: "log.output", Description: "Output target: stderr, stdout, file (default: stderr)", Type: "string", EnvVar: "ADDT_LOG_OUTPUT"},
		{Key: "log.file", Description: "Log file name (default: addt.log)", Type: "string", EnvVar: "ADDT_LOG_FILE"},
		{Key: "log.dir", Description: "Log directory (default: ~/.addt/logs)", Type: "string", EnvVar: "ADDT_LOG_DIR"},
		{Key: "log.level", Description: "Log level: DEBUG, INFO, WARN, ERROR (default: INFO)", Type: "string", EnvVar: "ADDT_LOG_LEVEL"},
		{Key: "log.modules", Description: "Comma-separated module filter (default: * for all)", Type: "string", EnvVar: "ADDT_LOG_MODULES"},
		{Key: "log.rotate", Description: "Enable log rotation (default: false)", Type: "bool", EnvVar: "ADDT_LOG_ROTATE"},
		{Key: "log.max_size", Description: "Max file size before rotating (e.g. 10m)", Type: "string", EnvVar: "ADDT_LOG_MAX_SIZE"},
		{Key: "log.max_files", Description: "Number of rotated files to keep (default: 5)", Type: "int", EnvVar: "ADDT_LOG_MAX_FILES"},
	}
}

// GetLogValue retrieves a log config value
func GetLogValue(l *cfgtypes.LogSettings, key string) string {
	if l == nil {
		return ""
	}
	switch key {
	case "log.enabled":
		if l.Enabled != nil {
			return fmt.Sprintf("%v", *l.Enabled)
		}
	case "log.output":
		return l.Output
	case "log.file":
		return l.File
	case "log.dir":
		return l.Dir
	case "log.level":
		return l.Level
	case "log.modules":
		return l.Modules
	case "log.rotate":
		if l.Rotate != nil {
			return fmt.Sprintf("%v", *l.Rotate)
		}
	case "log.max_size":
		return l.MaxSize
	case "log.max_files":
		if l.MaxFiles != nil {
			return fmt.Sprintf("%d", *l.MaxFiles)
		}
	}
	return ""
}

// SetLogValue sets a log config value
func SetLogValue(l *cfgtypes.LogSettings, key, value string) {
	switch key {
	case "log.enabled":
		b := value == "true"
		l.Enabled = &b
	case "log.output":
		l.Output = value
	case "log.file":
		l.File = value
	case "log.dir":
		l.Dir = value
	case "log.level":
		l.Level = value
	case "log.modules":
		l.Modules = value
	case "log.rotate":
		b := value == "true"
		l.Rotate = &b
	case "log.max_size":
		l.MaxSize = value
	case "log.max_files":
		if i, err := strconv.Atoi(value); err == nil {
			l.MaxFiles = &i
		}
	}
}

// UnsetLogValue clears a log config value
func UnsetLogValue(l *cfgtypes.LogSettings, key string) {
	switch key {
	case "log.enabled":
		l.Enabled = nil
	case "log.output":
		l.Output = ""
	case "log.file":
		l.File = ""
	case "log.dir":
		l.Dir = ""
	case "log.level":
		l.Level = ""
	case "log.modules":
		l.Modules = ""
	case "log.rotate":
		l.Rotate = nil
	case "log.max_size":
		l.MaxSize = ""
	case "log.max_files":
		l.MaxFiles = nil
	}
}
