package otel

import (
	"os"
	"strings"
)

// LoadConfig loads OTEL configuration with precedence:
// defaults < global settings < project settings < environment variables
func LoadConfig(globalSettings, projectSettings *Settings) Config {
	cfg := DefaultConfig()

	// Apply global settings
	if globalSettings != nil {
		applySettings(&cfg, globalSettings)
	}

	// Apply project settings (overrides global)
	if projectSettings != nil {
		applySettings(&cfg, projectSettings)
	}

	// Apply environment variable overrides
	applyEnvOverrides(&cfg)

	return cfg
}

// applySettings applies non-nil settings values to the config.
func applySettings(cfg *Config, settings *Settings) {
	if settings.Enabled != nil {
		cfg.Enabled = *settings.Enabled
	}
	if settings.Endpoint != nil {
		cfg.Endpoint = *settings.Endpoint
	}
	if settings.Protocol != nil {
		cfg.Protocol = *settings.Protocol
	}
	if settings.ServiceName != nil {
		cfg.ServiceName = *settings.ServiceName
	}
	if settings.Headers != nil {
		cfg.Headers = *settings.Headers
	}
}

// applyEnvOverrides applies environment variable overrides to the config.
func applyEnvOverrides(cfg *Config) {
	if val := os.Getenv("ADDT_OTEL_ENABLED"); val != "" {
		cfg.Enabled = strings.ToLower(val) == "true"
	}
	if val := os.Getenv("ADDT_OTEL_ENDPOINT"); val != "" {
		cfg.Endpoint = val
	}
	if val := os.Getenv("ADDT_OTEL_PROTOCOL"); val != "" {
		cfg.Protocol = val
	}
	if val := os.Getenv("ADDT_OTEL_SERVICE_NAME"); val != "" {
		cfg.ServiceName = val
	}
	if val := os.Getenv("ADDT_OTEL_HEADERS"); val != "" {
		cfg.Headers = val
	}
}

// GetEnvVars returns a map of OTEL environment variables to pass to containers.
// These follow the OpenTelemetry specification for SDK configuration.
// The attrs parameter provides runtime context that is injected as resource
// attributes via OTEL_RESOURCE_ATTRIBUTES (appears on all signals).
func GetEnvVars(cfg Config, attrs ResourceAttrs) map[string]string {
	if !cfg.Enabled {
		return nil
	}

	serviceName := cfg.ServiceName
	if serviceName == "addt" && attrs.Extension != "" {
		serviceName = "addt-" + attrs.Extension
	}

	env := map[string]string{
		// Standard OTEL configuration
		"OTEL_EXPORTER_OTLP_ENDPOINT": cfg.Endpoint,
		"OTEL_EXPORTER_OTLP_PROTOCOL": cfg.Protocol,
		"OTEL_SERVICE_NAME":           serviceName,
		// Enable OTLP export for metrics and logs (default SDK is "none")
		"OTEL_METRICS_EXPORTER": "otlp",
		"OTEL_LOGS_EXPORTER":    "otlp",
		// Claude Code specific - enable telemetry export
		"CLAUDE_CODE_ENABLE_TELEMETRY": "1",
	}

	if cfg.Headers != "" {
		env["OTEL_EXPORTER_OTLP_HEADERS"] = cfg.Headers
	}

	// Build OTEL_RESOURCE_ATTRIBUTES for runtime context
	if ra := buildResourceAttrs(attrs); ra != "" {
		env["OTEL_RESOURCE_ATTRIBUTES"] = ra
	}

	return env
}

// buildResourceAttrs builds a comma-separated key=value string for
// OTEL_RESOURCE_ATTRIBUTES from the provided runtime context.
func buildResourceAttrs(attrs ResourceAttrs) string {
	var parts []string
	if attrs.Extension != "" {
		parts = append(parts, "addt.extension="+attrs.Extension)
	}
	if attrs.Provider != "" {
		parts = append(parts, "addt.provider="+attrs.Provider)
	}
	if attrs.Version != "" {
		parts = append(parts, "addt.version="+attrs.Version)
	}
	if attrs.Project != "" {
		parts = append(parts, "addt.project="+attrs.Project)
	}
	return strings.Join(parts, ",")
}
