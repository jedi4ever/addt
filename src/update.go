package main

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// UpdateDClaude checks for and handles updates
func UpdateDClaude() {
	fmt.Println("Checking for updates...")
	fmt.Printf("Current version: %s\n", Version)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get("https://raw.githubusercontent.com/jedi4ever/dclaude/main/VERSION")
	if err != nil {
		fmt.Println("Error: Could not check for updates (network issue or repository unavailable)")
		os.Exit(1)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error: Could not check for updates")
		os.Exit(1)
	}

	latestVersion := strings.TrimSpace(string(body))
	fmt.Printf("Latest version:  %s\n", latestVersion)

	if Version == latestVersion {
		fmt.Println("✓ You are already on the latest version")
		return
	}

	fmt.Println()
	fmt.Println("New version available!")
	fmt.Print("Update now? [Y/n] ")

	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(strings.ToLower(input))

	if input != "" && input != "y" && input != "yes" {
		fmt.Println("Update cancelled")
		return
	}

	// Determine script path
	execPath, err := os.Executable()
	if err != nil {
		fmt.Println("Error: Could not determine executable path")
		os.Exit(1)
	}

	fmt.Printf("Downloading version %s...\n", latestVersion)

	// For Go binary, we'd need to download a different artifact
	// For now, show message about how to update
	fmt.Println()
	fmt.Println("To update the Go binary, please rebuild from source:")
	fmt.Printf("  cd %s && go build -o dclaude\n", filepath.Dir(execPath))
	fmt.Println()
	fmt.Println("Or download the latest release from:")
	fmt.Println("  https://github.com/jedi4ever/dclaude/releases")
}
