// addt-otel is a simple OpenTelemetry collector that logs received telemetry data.
// It listens for OTLP HTTP requests and outputs them to stdout/file for debugging.
package main

import (
	"compress/gzip"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

var (
	port    = flag.Int("port", 4318, "Port to listen on")
	logFile = flag.String("log", "", "Log file path (default: stdout)")
	verbose = flag.Bool("verbose", false, "Verbose output (show full payloads)")
	jsonOut = flag.Bool("json", false, "Output as JSON lines")
)

// Logger handles output formatting
type Logger struct {
	out     io.Writer
	jsonOut bool
	verbose bool
}

// LogEntry represents a log entry for JSON output
type LogEntry struct {
	Timestamp string                 `json:"timestamp"`
	Type      string                 `json:"type"`
	Count     int                    `json:"count,omitempty"`
	Summary   string                 `json:"summary,omitempty"`
	Data      map[string]interface{} `json:"data,omitempty"`
}

func (l *Logger) log(telemetryType string, data map[string]interface{}, count int) {
	timestamp := time.Now().Format("15:04:05")

	if l.jsonOut {
		entry := LogEntry{
			Timestamp: timestamp,
			Type:      telemetryType,
			Count:     count,
		}
		if l.verbose {
			entry.Data = data
		} else {
			entry.Summary = compactSummary(telemetryType, data)
		}
		jsonBytes, _ := json.Marshal(entry)
		fmt.Fprintln(l.out, string(jsonBytes))
	} else {
		if l.verbose && data != nil {
			jsonBytes, _ := json.MarshalIndent(data, "", "  ")
			fmt.Fprintf(l.out, "[%s] %s (%d items):\n%s\n", timestamp, telemetryType, count, string(jsonBytes))
		} else {
			summary := compactSummary(telemetryType, data)
			fmt.Fprintf(l.out, "[%s] %s (%d) %s\n", timestamp, telemetryType, count, summary)
		}
	}
}

// compactSummary extracts key details from OTEL data into a short string.
func compactSummary(telemetryType string, data map[string]interface{}) string {
	if data == nil {
		return ""
	}

	switch telemetryType {
	case "traces":
		return summarizeTraces(data)
	case "metrics":
		return summarizeMetrics(data)
	case "logs":
		return summarizeLogs(data)
	}
	return ""
}

func summarizeTraces(data map[string]interface{}) string {
	var parts []string
	svc := extractServiceName(data, "resourceSpans")
	if svc != "" {
		parts = append(parts, "svc="+svc)
	}
	names := extractSpanNames(data)
	if len(names) > 0 {
		parts = append(parts, "spans=["+strings.Join(names, ", ")+"]")
	}
	return strings.Join(parts, " ")
}

func summarizeMetrics(data map[string]interface{}) string {
	var parts []string
	svc := extractServiceName(data, "resourceMetrics")
	if svc != "" {
		parts = append(parts, "svc="+svc)
	}
	names := extractMetricNames(data)
	if len(names) > 0 {
		parts = append(parts, "metrics=["+strings.Join(names, ", ")+"]")
	}
	return strings.Join(parts, " ")
}

func summarizeLogs(data map[string]interface{}) string {
	var parts []string
	svc := extractServiceName(data, "resourceLogs")
	if svc != "" {
		parts = append(parts, "svc="+svc)
	}
	bodies := extractLogBodies(data)
	if len(bodies) > 0 {
		parts = append(parts, "logs=["+strings.Join(bodies, ", ")+"]")
	}
	return strings.Join(parts, " ")
}

// extractServiceName gets service.name from the first resource's attributes.
func extractServiceName(data map[string]interface{}, resourceKey string) string {
	resources, ok := data[resourceKey].([]interface{})
	if !ok || len(resources) == 0 {
		return ""
	}
	resMap, ok := resources[0].(map[string]interface{})
	if !ok {
		return ""
	}
	resource, ok := resMap["resource"].(map[string]interface{})
	if !ok {
		return ""
	}
	attrs, ok := resource["attributes"].([]interface{})
	if !ok {
		return ""
	}
	for _, attr := range attrs {
		a, ok := attr.(map[string]interface{})
		if !ok {
			continue
		}
		if a["key"] == "service.name" {
			if val, ok := a["value"].(map[string]interface{}); ok {
				if s, ok := val["stringValue"].(string); ok {
					return s
				}
			}
		}
	}
	return ""
}

func extractSpanNames(data map[string]interface{}) []string {
	var names []string
	resources, ok := data["resourceSpans"].([]interface{})
	if !ok {
		return nil
	}
	for _, res := range resources {
		resMap, ok := res.(map[string]interface{})
		if !ok {
			continue
		}
		scopes, ok := resMap["scopeSpans"].([]interface{})
		if !ok {
			continue
		}
		for _, scope := range scopes {
			scopeMap, ok := scope.(map[string]interface{})
			if !ok {
				continue
			}
			spans, ok := scopeMap["spans"].([]interface{})
			if !ok {
				continue
			}
			for _, span := range spans {
				spanMap, ok := span.(map[string]interface{})
				if !ok {
					continue
				}
				if name, ok := spanMap["name"].(string); ok {
					names = append(names, name)
				}
			}
		}
	}
	return truncateList(names, 5)
}

func extractMetricNames(data map[string]interface{}) []string {
	var names []string
	resources, ok := data["resourceMetrics"].([]interface{})
	if !ok {
		return nil
	}
	for _, res := range resources {
		resMap, ok := res.(map[string]interface{})
		if !ok {
			continue
		}
		scopes, ok := resMap["scopeMetrics"].([]interface{})
		if !ok {
			continue
		}
		for _, scope := range scopes {
			scopeMap, ok := scope.(map[string]interface{})
			if !ok {
				continue
			}
			metrics, ok := scopeMap["metrics"].([]interface{})
			if !ok {
				continue
			}
			for _, metric := range metrics {
				metricMap, ok := metric.(map[string]interface{})
				if !ok {
					continue
				}
				if name, ok := metricMap["name"].(string); ok {
					names = append(names, name)
				}
			}
		}
	}
	return truncateList(names, 5)
}

func extractLogBodies(data map[string]interface{}) []string {
	var bodies []string
	resources, ok := data["resourceLogs"].([]interface{})
	if !ok {
		return nil
	}
	for _, res := range resources {
		resMap, ok := res.(map[string]interface{})
		if !ok {
			continue
		}
		scopes, ok := resMap["scopeLogs"].([]interface{})
		if !ok {
			continue
		}
		for _, scope := range scopes {
			scopeMap, ok := scope.(map[string]interface{})
			if !ok {
				continue
			}
			records, ok := scopeMap["logRecords"].([]interface{})
			if !ok {
				continue
			}
			for _, rec := range records {
				recMap, ok := rec.(map[string]interface{})
				if !ok {
					continue
				}
				body := extractStringValue(recMap["body"])
				if body != "" {
					if len(body) > 60 {
						body = body[:57] + "..."
					}
					bodies = append(bodies, body)
				}
			}
		}
	}
	return truncateList(bodies, 3)
}

// extractStringValue gets a string from an OTLP AnyValue.
func extractStringValue(v interface{}) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	if m, ok := v.(map[string]interface{}); ok {
		if s, ok := m["stringValue"].(string); ok {
			return s
		}
	}
	return ""
}

// truncateList limits a list and adds "+N more" if needed.
func truncateList(items []string, max int) []string {
	if len(items) <= max {
		return items
	}
	result := items[:max]
	result = append(result, fmt.Sprintf("+%d more", len(items)-max))
	return result
}

func main() {
	flag.Parse()

	// Setup output
	var out io.Writer = os.Stdout
	if *logFile != "" {
		f, err := os.OpenFile(*logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			log.Fatalf("Failed to open log file: %v", err)
		}
		defer f.Close()
		out = f
	}

	logger := &Logger{
		out:     out,
		jsonOut: *jsonOut,
		verbose: *verbose,
	}

	// Setup HTTP handlers
	http.HandleFunc("/v1/traces", makeHandler(logger, "traces"))
	http.HandleFunc("/v1/metrics", makeHandler(logger, "metrics"))
	http.HandleFunc("/v1/logs", makeHandler(logger, "logs"))
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/", rootHandler)

	addr := fmt.Sprintf(":%d", *port)
	fmt.Fprintf(os.Stderr, "addt-otel listening on %s\n", addr)
	fmt.Fprintf(os.Stderr, "Endpoints:\n")
	fmt.Fprintf(os.Stderr, "  POST /v1/traces  - Receive trace data\n")
	fmt.Fprintf(os.Stderr, "  POST /v1/metrics - Receive metrics data\n")
	fmt.Fprintf(os.Stderr, "  POST /v1/logs    - Receive log data\n")
	fmt.Fprintf(os.Stderr, "  GET  /health     - Health check\n")

	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func makeHandler(logger *Logger, telemetryType string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Read body, handling gzip compression
		var reader io.Reader = r.Body
		if r.Header.Get("Content-Encoding") == "gzip" {
			gzReader, err := gzip.NewReader(r.Body)
			if err != nil {
				http.Error(w, "Failed to decompress", http.StatusBadRequest)
				return
			}
			defer gzReader.Close()
			reader = gzReader
		}

		body, err := io.ReadAll(reader)
		if err != nil {
			http.Error(w, "Failed to read body", http.StatusBadRequest)
			return
		}

		// Parse based on content type
		contentType := r.Header.Get("Content-Type")
		var data map[string]interface{}
		count := 0

		if strings.Contains(contentType, "application/json") {
			if err := json.Unmarshal(body, &data); err == nil {
				count = countItems(data, telemetryType)
			}
		} else if strings.Contains(contentType, "application/x-protobuf") {
			// For protobuf, we just count bytes and note it's binary
			data = map[string]interface{}{
				"format": "protobuf",
				"bytes":  len(body),
			}
			count = 1 // We can't easily count items in protobuf without full parsing
		} else {
			// Try JSON anyway
			if err := json.Unmarshal(body, &data); err == nil {
				count = countItems(data, telemetryType)
			} else {
				data = map[string]interface{}{
					"format": "unknown",
					"bytes":  len(body),
				}
				count = 1
			}
		}

		logger.log(telemetryType, data, count)

		// Return success response (OTLP expects empty JSON object)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("{}"))
	}
}

func countItems(data map[string]interface{}, telemetryType string) int {
	count := 0
	var resourceKey, itemKey string

	switch telemetryType {
	case "traces":
		resourceKey = "resourceSpans"
		itemKey = "scopeSpans"
	case "metrics":
		resourceKey = "resourceMetrics"
		itemKey = "scopeMetrics"
	case "logs":
		resourceKey = "resourceLogs"
		itemKey = "scopeLogs"
	}

	if resources, ok := data[resourceKey].([]interface{}); ok {
		for _, res := range resources {
			if resMap, ok := res.(map[string]interface{}); ok {
				if items, ok := resMap[itemKey].([]interface{}); ok {
					count += len(items)
				}
			}
		}
	}

	if count == 0 {
		count = 1 // At least we received something
	}
	return count
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "addt-otel collector\n\n")
	fmt.Fprintf(w, "Endpoints:\n")
	fmt.Fprintf(w, "  POST /v1/traces  - Receive trace data\n")
	fmt.Fprintf(w, "  POST /v1/metrics - Receive metrics data\n")
	fmt.Fprintf(w, "  POST /v1/logs    - Receive log data\n")
	fmt.Fprintf(w, "  GET  /health     - Health check\n")
}
