package config

import (
	"fmt"

	"github.com/jedi4ever/addt/config/otel"
)

// GetOtelKeys returns all valid OpenTelemetry config keys
func GetOtelKeys() []KeyInfo {
	return []KeyInfo{
		{Key: "otel.enabled", Description: "Enable OpenTelemetry", Type: "bool", EnvVar: "ADDT_OTEL_ENABLED"},
		{Key: "otel.endpoint", Description: "OTLP endpoint URL", Type: "string", EnvVar: "ADDT_OTEL_ENDPOINT"},
		{Key: "otel.protocol", Description: "OTLP protocol: http/json, http/protobuf, or grpc", Type: "string", EnvVar: "ADDT_OTEL_PROTOCOL"},
		{Key: "otel.service_name", Description: "Service name for telemetry", Type: "string", EnvVar: "ADDT_OTEL_SERVICE_NAME"},
		{Key: "otel.headers", Description: "OTLP headers (key=value,key2=value2)", Type: "string", EnvVar: "ADDT_OTEL_HEADERS"},
	}
}

// GetOtelValue retrieves an OTEL config value
func GetOtelValue(o *otel.Settings, key string) string {
	if o == nil {
		return ""
	}
	switch key {
	case "otel.enabled":
		if o.Enabled != nil {
			return fmt.Sprintf("%v", *o.Enabled)
		}
	case "otel.endpoint":
		if o.Endpoint != nil {
			return *o.Endpoint
		}
	case "otel.protocol":
		if o.Protocol != nil {
			return *o.Protocol
		}
	case "otel.service_name":
		if o.ServiceName != nil {
			return *o.ServiceName
		}
	case "otel.headers":
		if o.Headers != nil {
			return *o.Headers
		}
	}
	return ""
}

// SetOtelValue sets an OTEL config value
func SetOtelValue(o *otel.Settings, key, value string) {
	switch key {
	case "otel.enabled":
		b := value == "true"
		o.Enabled = &b
	case "otel.endpoint":
		o.Endpoint = &value
	case "otel.protocol":
		o.Protocol = &value
	case "otel.service_name":
		o.ServiceName = &value
	case "otel.headers":
		o.Headers = &value
	}
}

// UnsetOtelValue clears an OTEL config value
func UnsetOtelValue(o *otel.Settings, key string) {
	switch key {
	case "otel.enabled":
		o.Enabled = nil
	case "otel.endpoint":
		o.Endpoint = nil
	case "otel.protocol":
		o.Protocol = nil
	case "otel.service_name":
		o.ServiceName = nil
	case "otel.headers":
		o.Headers = nil
	}
}
