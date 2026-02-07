package config

import (
	"fmt"
	"os"
	"strings"

	cfgtypes "github.com/jedi4ever/addt/config"
)

// configRow represents a single row in the config table output.
type configRow struct {
	Key          string
	Value        string
	Default      string
	Source       string // "env", "project", "global", "default", or ""
	IsOverridden bool   // true when source is env, project, or global
	Description  string
}

// printRows prints a formatted table of config rows with Key, Value, Default, Source columns.
// When verbose is true, an additional Description column is shown.
func printRows(rows []configRow, verbose bool) {
	// Calculate column widths
	maxKeyLen := 3   // "Key"
	maxValLen := 5   // "Value"
	maxDefLen := 7   // "Default"
	maxDescLen := 11 // "Description"
	for _, r := range rows {
		if len(r.Key) > maxKeyLen {
			maxKeyLen = len(r.Key)
		}
		if len(r.Value) > maxValLen {
			maxValLen = len(r.Value)
		}
		if len(r.Default) > maxDefLen {
			maxDefLen = len(r.Default)
		}
		if verbose && len(r.Description) > maxDescLen {
			maxDescLen = len(r.Description)
		}
	}

	// Print header
	if verbose {
		fmt.Printf("  %-*s   %-*s   %-*s   %-8s   %s\n", maxKeyLen, "Key", maxValLen, "Value", maxDefLen, "Default", "Source", "Description")
		fmt.Printf("  %s   %s   %s   %s   %s\n", strings.Repeat("-", maxKeyLen), strings.Repeat("-", maxValLen), strings.Repeat("-", maxDefLen), "--------", strings.Repeat("-", maxDescLen))
	} else {
		fmt.Printf("  %-*s   %-*s   %-*s   %s\n", maxKeyLen, "Key", maxValLen, "Value", maxDefLen, "Default", "Source")
		fmt.Printf("  %s   %s   %s   %s\n", strings.Repeat("-", maxKeyLen), strings.Repeat("-", maxValLen), strings.Repeat("-", maxDefLen), "--------")
	}

	// Print rows
	for _, r := range rows {
		prefix := " "
		if r.IsOverridden {
			prefix = "*"
		}
		if verbose {
			desc := r.Description
			if desc == "" {
				desc = "-"
			}
			fmt.Printf("%s %-*s   %-*s   %-*s   %-8s   %s\n", prefix, maxKeyLen, r.Key, maxValLen, r.Value, maxDefLen, r.Default, r.Source, desc)
		} else {
			fmt.Printf("%s %-*s   %-*s   %-*s   %s\n", prefix, maxKeyLen, r.Key, maxValLen, r.Value, maxDefLen, r.Default, r.Source)
		}
	}
}

// printConfigTable prints a formatted table of all config keys with their
// effective values, defaults, and source (env > project > global > default).
func printConfigTable(projectCfg, globalCfg *cfgtypes.GlobalConfig, verbose bool) {
	keys := GetKeys()
	rows := make([]configRow, 0, len(keys))

	for _, k := range keys {
		value, source := resolveValueAndSource(k, projectCfg, globalCfg)
		def := GetDefaultValue(k.Key)
		if def == "" {
			def = "-"
		}

		rows = append(rows, configRow{
			Key:          k.Key,
			Value:        value,
			Default:      def,
			Source:       source,
			IsOverridden: source == "env" || source == "project" || source == "global",
			Description:  k.Description,
		})
	}

	printRows(rows, verbose)
}

// resolveValueAndSource returns the effective value and its source label.
func resolveValueAndSource(k KeyInfo, projectCfg, globalCfg *cfgtypes.GlobalConfig) (string, string) {
	if v := os.Getenv(k.EnvVar); v != "" {
		return v, "env"
	}
	if v := GetValue(projectCfg, k.Key); v != "" {
		return v, "project"
	}
	if v := GetValue(globalCfg, k.Key); v != "" {
		return v, "global"
	}
	if v := GetDefaultValue(k.Key); v != "" {
		return v, "default"
	}
	return "-", ""
}
