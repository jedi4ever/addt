package podman

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	podmanVersion   = "5.7.1"
	podmanGitHubURL = "https://github.com/containers/podman/releases/download"
)

// InstallPodman attempts to install Podman on the current system
func InstallPodman() error {
	switch runtime.GOOS {
	case "linux":
		return installPodmanLinux()
	case "darwin":
		return installPodmanMacOS()
	default:
		return fmt.Errorf("automatic Podman installation not supported on %s", runtime.GOOS)
	}
}

// installPodmanLinux installs Podman on Linux using the appropriate package manager
func installPodmanLinux() error {
	// Detect package manager and install
	if commandExists("apt-get") {
		// Debian/Ubuntu
		fmt.Println("Detected Debian/Ubuntu, installing Podman via apt...")
		return runInstallCommands([][]string{
			{"sudo", "apt-get", "update"},
			{"sudo", "apt-get", "install", "-y", "podman"},
		})
	} else if commandExists("dnf") {
		// Fedora/RHEL/CentOS
		fmt.Println("Detected Fedora/RHEL, installing Podman via dnf...")
		return runInstallCommands([][]string{
			{"sudo", "dnf", "install", "-y", "podman"},
		})
	} else if commandExists("yum") {
		// Older RHEL/CentOS
		fmt.Println("Detected RHEL/CentOS, installing Podman via yum...")
		return runInstallCommands([][]string{
			{"sudo", "yum", "install", "-y", "podman"},
		})
	} else if commandExists("pacman") {
		// Arch Linux
		fmt.Println("Detected Arch Linux, installing Podman via pacman...")
		return runInstallCommands([][]string{
			{"sudo", "pacman", "-S", "--noconfirm", "podman"},
		})
	} else if commandExists("zypper") {
		// openSUSE
		fmt.Println("Detected openSUSE, installing Podman via zypper...")
		return runInstallCommands([][]string{
			{"sudo", "zypper", "install", "-y", "podman"},
		})
	} else if commandExists("apk") {
		// Alpine
		fmt.Println("Detected Alpine Linux, installing Podman via apk...")
		return runInstallCommands([][]string{
			{"sudo", "apk", "add", "podman"},
		})
	}

	// Fallback: download static binary
	fmt.Println("No package manager detected, downloading Podman static binary...")
	return downloadPodmanBinary()
}

// installPodmanMacOS installs Podman on macOS using Homebrew
func installPodmanMacOS() error {
	if !commandExists("brew") {
		return fmt.Errorf("Homebrew is required to install Podman on macOS. Install it from: https://brew.sh")
	}

	fmt.Println("Installing Podman via Homebrew...")
	if err := runInstallCommands([][]string{
		{"brew", "install", "podman"},
	}); err != nil {
		return err
	}

	// Initialize podman machine on macOS
	fmt.Println("\nInitializing Podman machine (required on macOS)...")
	if err := runInstallCommands([][]string{
		{"podman", "machine", "init"},
		{"podman", "machine", "start"},
	}); err != nil {
		fmt.Println("Note: If machine already exists, you may need to run 'podman machine start' manually")
	}

	return nil
}

// commandExists checks if a command is available in PATH
func commandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

// runInstallCommands runs a series of commands, stopping on first error
func runInstallCommands(commands [][]string) error {
	for _, cmdArgs := range commands {
		fmt.Printf("Running: %s\n", strings.Join(cmdArgs, " "))
		cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("command failed: %s: %w", strings.Join(cmdArgs, " "), err)
		}
	}
	return nil
}

// PromptInstallPodman asks the user if they want to install Podman
func PromptInstallPodman() bool {
	fmt.Print("\nWould you like to install Podman now? [y/N]: ")
	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}
	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes"
}

// GetInstallInstructions returns platform-specific installation instructions
func GetInstallInstructions() string {
	switch runtime.GOOS {
	case "linux":
		return `Install Podman using your package manager:
  Debian/Ubuntu: sudo apt-get install podman
  Fedora/RHEL:   sudo dnf install podman
  Arch Linux:    sudo pacman -S podman

Or visit: https://podman.io/docs/installation`
	case "darwin":
		return `Install Podman using Homebrew:
  brew install podman
  podman machine init
  podman machine start

Or visit: https://podman.io/docs/installation`
	default:
		return "Visit: https://podman.io/docs/installation"
	}
}

// downloadPodmanBinary downloads the static Podman binary for the current platform
func downloadPodmanBinary() error {
	// Determine architecture
	arch := runtime.GOARCH
	if arch == "amd64" {
		arch = "amd64"
	} else if arch == "arm64" {
		arch = "arm64"
	} else {
		return fmt.Errorf("unsupported architecture: %s", arch)
	}

	// Construct download URL
	var url string
	var filename string
	switch runtime.GOOS {
	case "linux":
		filename = fmt.Sprintf("podman-remote-static-linux_%s.tar.gz", arch)
		url = fmt.Sprintf("%s/v%s/%s", podmanGitHubURL, podmanVersion, filename)
	case "darwin":
		filename = fmt.Sprintf("podman-remote-release-darwin_%s.zip", arch)
		url = fmt.Sprintf("%s/v%s/%s", podmanGitHubURL, podmanVersion, filename)
	default:
		return fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}

	// Create addt bin directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}
	binDir := filepath.Join(homeDir, ".addt", "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		return fmt.Errorf("failed to create bin directory: %w", err)
	}

	podmanPath := filepath.Join(binDir, "podman")

	// Download the file
	fmt.Printf("Downloading Podman v%s from %s...\n", podmanVersion, url)
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download Podman: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download Podman: HTTP %d", resp.StatusCode)
	}

	// Create temp file for download
	tmpFile, err := os.CreateTemp("", "podman-download-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	// Copy download to temp file
	_, err = io.Copy(tmpFile, resp.Body)
	if err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to download Podman: %w", err)
	}
	tmpFile.Close()

	// Extract the binary
	fmt.Println("Extracting Podman binary...")
	if runtime.GOOS == "linux" {
		if err := extractTarGz(tmpFile.Name(), binDir); err != nil {
			return fmt.Errorf("failed to extract Podman: %w", err)
		}
		// Rename podman-remote to podman
		remotePath := filepath.Join(binDir, "podman-remote-static-linux_"+arch, "podman-remote")
		if err := os.Rename(remotePath, podmanPath); err != nil {
			// Try alternate location
			remotePath = filepath.Join(binDir, "bin", "podman-remote")
			if err := os.Rename(remotePath, podmanPath); err != nil {
				return fmt.Errorf("failed to move podman binary: %w", err)
			}
		}
	} else {
		// For macOS, would need unzip handling
		return fmt.Errorf("macOS download not yet implemented - please use: brew install podman")
	}

	// Make executable
	if err := os.Chmod(podmanPath, 0755); err != nil {
		return fmt.Errorf("failed to make podman executable: %w", err)
	}

	fmt.Printf("\n✓ Podman installed to: %s\n", podmanPath)
	fmt.Printf("\nAdd to your PATH:\n  export PATH=\"%s:$PATH\"\n", binDir)
	fmt.Println("\nOr run with full path:")
	fmt.Printf("  ADDT_PODMAN_PATH=%s addt run claude\n", podmanPath)

	return nil
}

// extractTarGz extracts a .tar.gz file to the destination directory
func extractTarGz(src, dst string) error {
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

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		target := filepath.Join(dst, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			// Ensure parent directory exists
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			outFile, err := os.Create(target)
			if err != nil {
				return err
			}
			if _, err := io.Copy(outFile, tr); err != nil {
				outFile.Close()
				return err
			}
			outFile.Close()
			// Preserve executable permissions
			if header.Mode&0111 != 0 {
				os.Chmod(target, 0755)
			}
		}
	}

	return nil
}

// GetPodmanPath returns the path to the podman binary, checking addt's bin dir first
func GetPodmanPath() string {
	// Check environment variable override
	if path := os.Getenv("ADDT_PODMAN_PATH"); path != "" {
		return path
	}

	// Check addt's bin directory first
	homeDir, err := os.UserHomeDir()
	if err == nil {
		addtPodman := filepath.Join(homeDir, ".addt", "bin", "podman")
		if _, err := os.Stat(addtPodman); err == nil {
			return addtPodman
		}
	}

	// Fall back to system podman
	return "podman"
}
