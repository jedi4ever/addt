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

// PodmanDownloadURLs contains download URLs for Podman static builds (Linux only)
var PodmanDownloadURLs = map[string]string{
	"linux/amd64": "https://github.com/containers/podman/releases/download/v5.3.1/podman-remote-static-linux_amd64.tar.gz",
	"linux/arm64": "https://github.com/containers/podman/releases/download/v5.3.1/podman-remote-static-linux_arm64.tar.gz",
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

// DownloadPodman downloads and installs Podman
// On macOS, uses Homebrew. On Linux, downloads static binary.
func DownloadPodman() error {
	// On macOS, use Homebrew
	if runtime.GOOS == "darwin" {
		return installPodmanWithHomebrew()
	}

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

	// Download the file
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download Podman: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download Podman: HTTP %d", resp.StatusCode)
	}

	// Create temp file
	tmpFile, err := os.CreateTemp("", "podman-download-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	// Copy to temp file with progress tracking
	progressReader := util.NewProgressReader(resp.Body, resp.ContentLength, "Downloading Podman")
	_, err = io.Copy(tmpFile, progressReader)
	tmpFile.Close()
	if err != nil {
		progressReader.Fail("Download failed")
		return fmt.Errorf("failed to save download: %w", err)
	}
	progressReader.Complete()

	spinner := util.NewSpinner("Extracting Podman...")
	spinner.Start()

	// Extract tar.gz (Linux only - macOS uses Homebrew)
	if err := extractTarGz(tmpFile.Name(), binDir); err != nil {
		spinner.StopWithError("Extraction failed")
		return fmt.Errorf("failed to extract Podman: %w", err)
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

// installPodmanWithHomebrew installs Podman using Homebrew (macOS)
func installPodmanWithHomebrew() error {
	// Check if Homebrew is available
	brewPath, err := exec.LookPath("brew")
	if err != nil {
		util.PrintError("Homebrew not found")
		fmt.Println()
		fmt.Println("To install Podman on macOS, first install Homebrew:")
		fmt.Println("  /bin/bash -c \"$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)\"")
		fmt.Println()
		fmt.Println("Then run:")
		fmt.Println("  brew install podman")
		return fmt.Errorf("homebrew not found - install it first")
	}

	spinner := util.NewSpinner("Installing Podman via Homebrew...")
	spinner.Start()

	cmd := exec.Command(brewPath, "install", "podman")
	output, err := cmd.CombinedOutput()
	if err != nil {
		spinner.StopWithError("Installation failed")
		fmt.Printf("\n%s\n", string(output))
		return fmt.Errorf("brew install podman failed: %w", err)
	}
	spinner.StopWithSuccess("Podman installed via Homebrew")

	// Initialize and start the machine
	podmanPath, err := exec.LookPath("podman")
	if err != nil {
		return fmt.Errorf("podman not found after installation: %w", err)
	}

	return ensurePodmanMachine(podmanPath)
}

// ensurePodmanMachine ensures a Podman machine exists and is running (macOS only)
func ensurePodmanMachine(podmanPath string) error {
	// Check if any machine exists
	cmd := exec.Command(podmanPath, "machine", "list", "--format", "{{.Name}}")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to list machines: %w", err)
	}

	machines := strings.TrimSpace(string(output))
	if machines == "" {
		// No machine exists, initialize one
		// Read VM resource settings: env var -> global config -> defaults
		globalCfg := loadGlobalConfig()

		vmMemory := os.Getenv("ADDT_VM_MEMORY")
		if vmMemory == "" && globalCfg.VmMemory != "" {
			vmMemory = globalCfg.VmMemory
		}
		if vmMemory == "" {
			vmMemory = "8192"
		}

		vmCpus := os.Getenv("ADDT_VM_CPUS")
		if vmCpus == "" && globalCfg.VmCpus != "" {
			vmCpus = globalCfg.VmCpus
		}
		if vmCpus == "" {
			vmCpus = "4"
		}

		spinner := util.NewSpinner(fmt.Sprintf("Initializing Podman machine (memory: %sMB, cpus: %s)...", vmMemory, vmCpus))
		spinner.Start()

		initCmd := exec.Command(podmanPath, "machine", "init",
			"--memory", vmMemory,
			"--cpus", vmCpus,
		)
		initOutput, err := initCmd.CombinedOutput()
		if err != nil {
			spinner.StopWithError("Machine init failed")
			return fmt.Errorf("failed to initialize machine: %w\n%s", err, string(initOutput))
		}
		spinner.StopWithSuccess("Podman machine initialized")
	}

	// Check if machine is running
	cmd = exec.Command(podmanPath, "machine", "list", "--format", "{{.Running}}")
	output, err = cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to check machine status: %w", err)
	}

	running := strings.TrimSpace(string(output))
	if running != "true" {
		spinner := util.NewSpinner("Starting Podman machine...")
		spinner.Start()

		startCmd := exec.Command(podmanPath, "machine", "start")
		startOutput, err := startCmd.CombinedOutput()
		if err != nil {
			spinner.StopWithError("Machine start failed")
			return fmt.Errorf("failed to start machine: %w\n%s", err, string(startOutput))
		}
		spinner.StopWithSuccess("Podman machine started")
	}

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
