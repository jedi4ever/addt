package config

import (
	"fmt"
	"os"
	"strings"

	cfgtypes "github.com/jedi4ever/addt/config"
)

func listProject(verbose bool) {
	projectCfg, err := cfgtypes.LoadProjectConfigFile()
	if err != nil {
		fmt.Printf("Error loading project config: %v\n", err)
		os.Exit(1)
	}

	globalCfg, err := cfgtypes.LoadGlobalConfigFile()
	if err != nil {
		fmt.Printf("Error loading global config: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Project config: %s\n", cfgtypes.GetProjectConfigPath())
	fmt.Printf("Global config:  %s\n\n", cfgtypes.GetGlobalConfigPath())

	printConfigTable(projectCfg, globalCfg, verbose)
}

func getProject(key string) {
	if !IsValidKey(key) {
		fmt.Printf("Unknown config key: %s\n", key)
		fmt.Println("Use 'addt config project list' to see available keys.")
		os.Exit(1)
	}

	cfg, err := cfgtypes.LoadProjectConfigFile()
	if err != nil {
		fmt.Printf("Error loading project config: %v\n", err)
		os.Exit(1)
	}

	val := GetValue(cfg, key)
	if val == "" {
		fmt.Printf("%s is not set in project config\n", key)
	} else {
		fmt.Println(val)
	}
}

func setProject(key, value string) {
	keyInfo := GetKeyInfo(key)
	if keyInfo == nil {
		fmt.Printf("Unknown config key: %s\n", key)
		fmt.Println("Use 'addt config project --help' to see available keys.")
		os.Exit(1)
	}

	if keyInfo.Type == "bool" {
		value = strings.ToLower(value)
		if value != "true" && value != "false" {
			fmt.Printf("Invalid value for %s: must be 'true' or 'false'\n", key)
			os.Exit(1)
		}
	}

	cfg, err := cfgtypes.LoadProjectConfigFile()
	if err != nil {
		fmt.Printf("Error loading project config: %v\n", err)
		os.Exit(1)
	}

	SetValue(cfg, key, value)

	if err := cfgtypes.SaveProjectConfigFile(cfg); err != nil {
		fmt.Printf("Error saving project config: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Set %s = %s (project)\n", key, value)
}

func unsetProject(key string) {
	if !IsValidKey(key) {
		fmt.Printf("Unknown config key: %s\n", key)
		fmt.Println("Use 'addt config project list' to see available keys.")
		os.Exit(1)
	}

	cfg, err := cfgtypes.LoadProjectConfigFile()
	if err != nil {
		fmt.Printf("Error loading project config: %v\n", err)
		os.Exit(1)
	}

	UnsetValue(cfg, key)

	if err := cfgtypes.SaveProjectConfigFile(cfg); err != nil {
		fmt.Printf("Error saving project config: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Unset %s (project)\n", key)
}
