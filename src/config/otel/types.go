// Package otel provides OpenTelemetry configuration for addt.
package otel

// Settings represents the OTEL configuration in YAML files.
// Pointer types allow distinguishing between unset and false/empty values.
type Settings struct {
	Enabled     *bool   `yaml:"enabled,omitempty"`      // Enable OTEL (default: false)
	Endpoint    *string `yaml:"endpoint,omitempty"`     // OTLP endpoint (default: http://localhost:4318)
	Protocol    *string `yaml:"protocol,omitempty"`     // Protocol: http/json, http/protobuf, or grpc (default: http/json)
	ServiceName *string `yaml:"service_name,omitempty"` // Service name for traces
	Headers     *string `yaml:"headers,omitempty"`      // Additional headers (key=value,key2=value2)
}

// Config represents the runtime OTEL configuration with defaults applied.
type Config struct {
	Enabled     bool
	Endpoint    string
	Protocol    string
	ServiceName string
	Headers     string
}

// ResourceAttrs holds runtime context injected as OTEL resource attributes.
// These appear on every trace, metric, and log payload.
type ResourceAttrs struct {
	Extension string // e.g. "claude"
	Provider  string // e.g. "podman"
	Version   string // addt version
	Project   string // project directory name
}

// DefaultConfig returns the default OTEL configuration.
// The default endpoint uses host.docker.internal to reach the host from inside the container.
func DefaultConfig() Config {
	return Config{
		Enabled:     false,
		Endpoint:    "http://host.docker.internal:4318",
		Protocol:    "http/json",
		ServiceName: "addt",
		Headers:     "",
	}
}
