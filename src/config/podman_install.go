package config

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/jedi4ever/addt/util"
)

// PodmanDownloadURLs contains download URLs for Podman static builds
var PodmanDownloadURLs = map[string]string{
	"linux/amd64": "https://github.com/containers/podman/releases/download/v5.3.1/podman-remote-static-linux_amd64.tar.gz",
	"linux/arm64": "https://github.com/containers/podman/releases/download/v5.3.1/podman-remote-static-linux_arm64.tar.gz",
	"darwin/amd64": "https://github.com/containers/podman/releases/download/v5.3.1/podman-remote-release-darwin_amd64.zip",
	"darwin/arm64": "https://github.com/containers/podman/releases/download/v5.3.1/podman-remote-release-darwin_arm64.zip",
}

// GetBundledBinDir returns the path to bundled binaries directory
func GetBundledBinDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(homeDir, ".addt", "bin")
}

// GetBundledPodmanPath returns the path to bundled Podman binary
func GetBundledPodmanPath() string {
	binDir := GetBundledBinDir()
	if binDir == "" {
		return ""
	}
	if runtime.GOOS == "windows" {
		return filepath.Join(binDir, "podman.exe")
	}
	return filepath.Join(binDir, "podman")
}

// IsPodmanBundled checks if Podman is available in the bundled bin directory
func IsPodmanBundled() bool {
	podmanPath := GetBundledPodmanPath()
	if podmanPath == "" {
		return false
	}
	_, err := os.Stat(podmanPath)
	return err == nil
}

// EnsurePodman ensures Podman is available, downloading if necessary
// Returns the path to the Podman binary and any error
func EnsurePodman() (string, error) {
	// First check if system Podman is available
	if path, err := exec.LookPath("podman"); err == nil {
		return path, nil
	}

	// Check if bundled Podman exists
	bundledPath := GetBundledPodmanPath()
	if bundledPath != "" {
		if _, err := os.Stat(bundledPath); err == nil {
			return bundledPath, nil
		}
	}

	// No Podman found, offer to download
	return "", fmt.Errorf("podman not found")
}

// DownloadPodman downloads and installs Podman to the bundled bin directory
func DownloadPodman() error {
	platform := fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)
	url, ok := PodmanDownloadURLs[platform]
	if !ok {
		return fmt.Errorf("no Podman download available for %s", platform)
	}

	binDir := GetBundledBinDir()
	if binDir == "" {
		return fmt.Errorf("could not determine bin directory")
	}

	// Create bin directory
	if err := os.MkdirAll(binDir, 0755); err != nil {
		return fmt.Errorf("failed to create bin directory: %w", err)
	}

	spinner := util.NewSpinner("Downloading Podman...")
	spinner.Start()

	// Download the file
	resp, err := http.Get(url)
	if err != nil {
		spinner.StopWithError("Download failed")
		return fmt.Errorf("failed to download Podman: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		spinner.StopWithError("Download failed")
		return fmt.Errorf("failed to download Podman: HTTP %d", resp.StatusCode)
	}

	// Create temp file
	tmpFile, err := os.CreateTemp("", "podman-download-*")
	if err != nil {
		spinner.StopWithError("Download failed")
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	// Copy to temp file
	_, err = io.Copy(tmpFile, resp.Body)
	tmpFile.Close()
	if err != nil {
		spinner.StopWithError("Download failed")
		return fmt.Errorf("failed to save download: %w", err)
	}

	spinner.UpdateMessage("Extracting Podman...")

	// Extract based on file type
	if strings.HasSuffix(url, ".tar.gz") {
		if err := extractTarGz(tmpFile.Name(), binDir); err != nil {
			spinner.StopWithError("Extraction failed")
			return fmt.Errorf("failed to extract Podman: %w", err)
		}
	} else if strings.HasSuffix(url, ".zip") {
		if err := extractZip(tmpFile.Name(), binDir); err != nil {
			spinner.StopWithError("Extraction failed")
			return fmt.Errorf("failed to extract Podman: %w", err)
		}
	}

	// Find and rename the podman binary
	podmanPath := GetBundledPodmanPath()

	// Make executable
	if err := os.Chmod(podmanPath, 0755); err != nil {
		spinner.StopWithError("Setup failed")
		return fmt.Errorf("failed to make Podman executable: %w", err)
	}

	// Verify it works
	cmd := exec.Command(podmanPath, "--version")
	output, err := cmd.Output()
	if err != nil {
		spinner.StopWithError("Verification failed")
		return fmt.Errorf("Podman verification failed: %w", err)
	}

	version := strings.TrimSpace(string(output))
	spinner.StopWithSuccess(fmt.Sprintf("Podman installed: %s", version))

	return nil
}

// extractTarGz extracts a .tar.gz file
func extractTarGz(src, destDir string) error {
	file, err := os.Open(src)
	if err != nil {
		return err
	}
	defer file.Close()

	gzr, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	extracted := false

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// Look for podman binary (various naming conventions)
		name := filepath.Base(header.Name)
		isPodmanBinary := strings.HasPrefix(name, "podman") &&
			!strings.HasSuffix(name, ".md") &&
			!strings.HasSuffix(name, ".txt") &&
			!strings.HasSuffix(name, ".1") &&
			header.Typeflag == tar.TypeReg

		if !isPodmanBinary {
			continue
		}

		target := filepath.Join(destDir, "podman")

		outFile, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode))
		if err != nil {
			return err
		}
		if _, err := io.Copy(outFile, tr); err != nil {
			outFile.Close()
			return err
		}
		outFile.Close()
		extracted = true
		break // Only need to extract one binary
	}

	if !extracted {
		return fmt.Errorf("podman binary not found in archive")
	}

	return nil
}

// extractZip extracts a .zip file (placeholder - needs archive/zip import)
func extractZip(src, destDir string) error {
	// For macOS, we need to handle zip files
	// Using unzip command as a simple solution
	cmd := exec.Command("unzip", "-o", "-j", src, "-d", destDir)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("unzip failed: %w", err)
	}

	// Rename podman-remote to podman if needed
	remotePath := filepath.Join(destDir, "podman-remote")
	podmanPath := filepath.Join(destDir, "podman")
	if _, err := os.Stat(remotePath); err == nil {
		os.Rename(remotePath, podmanPath)
	}

	return nil
}

// CheckAndOfferPodmanInstall checks if a container runtime is available
// and offers to install Podman if not
func CheckAndOfferPodmanInstall() error {
	// Check Docker first
	if _, err := exec.LookPath("docker"); err == nil {
		return nil // Docker available
	}

	// Check system Podman
	if _, err := exec.LookPath("podman"); err == nil {
		return nil // Podman available
	}

	// Check bundled Podman
	if IsPodmanBundled() {
		return nil // Bundled Podman available
	}

	// No container runtime found
	fmt.Println()
	util.PrintWarning("No container runtime found (Docker or Podman)")
	fmt.Println()
	fmt.Print("Would you like to download Podman? [Y/n] ")

	var response string
	fmt.Scanln(&response)
	response = strings.ToLower(strings.TrimSpace(response))

	if response == "" || response == "y" || response == "yes" {
		return DownloadPodman()
	}

	return fmt.Errorf("no container runtime available")
}
