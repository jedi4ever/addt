package config

import (
	"fmt"
	"os"
	"strings"

	cfgtypes "github.com/jedi4ever/addt/config"
)

func listGlobal(verbose bool) {
	globalCfg, err := cfgtypes.LoadGlobalConfigFile()
	if err != nil {
		fmt.Printf("Error loading global config: %v\n", err)
		os.Exit(1)
	}

	projectCfg, err := cfgtypes.LoadProjectConfigFile()
	if err != nil {
		fmt.Printf("Error loading project config: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Global config:  %s\n", cfgtypes.GetGlobalConfigPath())
	fmt.Printf("Project config: %s\n\n", cfgtypes.GetProjectConfigPath())

	printConfigTable(projectCfg, globalCfg, verbose)
}

func getGlobal(key string) {
	// Validate key
	if !IsValidKey(key) {
		fmt.Printf("Unknown config key: %s\n", key)
		fmt.Println("Use 'addt config list' to see available keys.")
		os.Exit(1)
	}

	cfg, err := cfgtypes.LoadGlobalConfigFile()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}

	val := GetValue(cfg, key)
	if val == "" {
		fmt.Printf("%s is not set\n", key)
	} else {
		fmt.Println(val)
	}
}

func setGlobal(key, value string) {
	// Validate key
	keyInfo := GetKeyInfo(key)
	if keyInfo == nil {
		fmt.Printf("Unknown config key: %s\n", key)
		fmt.Println("Use 'addt config --help' to see available keys.")
		os.Exit(1)
	}

	// Validate value based on type
	if keyInfo.Type == "bool" {
		value = strings.ToLower(value)
		if value != "true" && value != "false" {
			fmt.Printf("Invalid value for %s: must be 'true' or 'false'\n", key)
			os.Exit(1)
		}
	}

	cfg, err := cfgtypes.LoadGlobalConfigFile()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}

	SetValue(cfg, key, value)

	if err := cfgtypes.SaveGlobalConfigFile(cfg); err != nil {
		fmt.Printf("Error saving config: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Set %s = %s\n", key, value)
}

func unsetGlobal(key string) {
	// Validate key
	if !IsValidKey(key) {
		fmt.Printf("Unknown config key: %s\n", key)
		fmt.Println("Use 'addt config list' to see available keys.")
		os.Exit(1)
	}

	cfg, err := cfgtypes.LoadGlobalConfigFile()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}

	UnsetValue(cfg, key)

	if err := cfgtypes.SaveGlobalConfigFile(cfg); err != nil {
		fmt.Printf("Error saving config: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Unset %s\n", key)
}
