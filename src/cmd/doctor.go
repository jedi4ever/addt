package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/jedi4ever/addt/config"
)

// DoctorCheck represents a single health check result
type DoctorCheck struct {
	Name    string
	Status  string // "ok", "warn", "fail"
	Message string
	Fix     string // Suggested fix for failures
}

// HandleDoctorCommand runs system health checks
func HandleDoctorCommand(args []string) {
	fmt.Println("addt doctor - System Health Check")
	fmt.Println("==================================")
	fmt.Println()

	checks := runAllChecks()

	// Print results
	okCount := 0
	warnCount := 0
	failCount := 0

	for _, check := range checks {
		icon := getStatusIcon(check.Status)
		fmt.Printf("%s %s: %s\n", icon, check.Name, check.Message)
		if check.Fix != "" && check.Status != "ok" {
			fmt.Printf("   Fix: %s\n", check.Fix)
		}

		switch check.Status {
		case "ok":
			okCount++
		case "warn":
			warnCount++
		case "fail":
			failCount++
		}
	}

	fmt.Println()
	fmt.Println("----------------------------------")
	fmt.Printf("Summary: %d passed, %d warnings, %d failed\n", okCount, warnCount, failCount)

	if failCount > 0 {
		fmt.Println()
		fmt.Println("Some checks failed. Please address the issues above.")
		os.Exit(1)
	} else if warnCount > 0 {
		fmt.Println()
		fmt.Println("All critical checks passed with some warnings.")
	} else {
		fmt.Println()
		fmt.Println("All checks passed! Your system is ready to use addt.")
	}
}

func getStatusIcon(status string) string {
	switch status {
	case "ok":
		return "✓"
	case "warn":
		return "!"
	case "fail":
		return "✗"
	default:
		return "?"
	}
}

func runAllChecks() []DoctorCheck {
	var checks []DoctorCheck

	// Container runtime checks
	checks = append(checks, checkDocker())
	checks = append(checks, checkPodman())

	// Git check
	checks = append(checks, checkGit())

	// API keys
	checks = append(checks, checkAnthropicKey())
	checks = append(checks, checkGitHubToken())

	// Disk space
	checks = append(checks, checkDiskSpace())

	// Config files
	checks = append(checks, checkGlobalConfig())
	checks = append(checks, checkProjectConfig())

	// Network connectivity (optional)
	checks = append(checks, checkNetworkConnectivity())

	return checks
}

func checkDocker() DoctorCheck {
	check := DoctorCheck{Name: "Docker"}

	// Check if docker is installed
	dockerPath, err := exec.LookPath("docker")
	if err != nil {
		check.Status = "warn"
		check.Message = "not installed"
		check.Fix = "Install Docker from https://docs.docker.com/get-docker/"
		return check
	}

	// Get docker version
	cmd := exec.Command(dockerPath, "version", "--format", "{{.Server.Version}}")
	output, err := cmd.Output()
	if err != nil {
		// Docker might be installed but daemon not running
		check.Status = "warn"
		check.Message = "installed but daemon not running"
		if runtime.GOOS == "darwin" || runtime.GOOS == "windows" {
			check.Fix = "Start Docker Desktop"
		} else {
			check.Fix = "Run: sudo systemctl start docker"
		}
		return check
	}

	version := strings.TrimSpace(string(output))
	check.Status = "ok"
	check.Message = fmt.Sprintf("running (v%s)", version)
	return check
}

func checkPodman() DoctorCheck {
	check := DoctorCheck{Name: "Podman"}

	// Check for system Podman first, then bundled
	podmanPath := config.GetPodmanPath()
	if podmanPath == "" {
		check.Status = "warn"
		check.Message = "not installed (optional)"
		check.Fix = "Run: addt cli install-podman"
		return check
	}

	// Get podman version (use --version flag which works without daemon)
	cmd := exec.Command(podmanPath, "--version")
	output, err := cmd.Output()
	if err != nil {
		check.Status = "warn"
		check.Message = "installed but not working"
		check.Fix = "Check podman installation: podman info"
		return check
	}

	// Parse "podman version X.Y.Z" -> "X.Y.Z"
	version := strings.TrimSpace(string(output))
	version = strings.TrimPrefix(version, "podman version ")
	source := "system"
	if config.IsPodmanBundled() && podmanPath == config.GetBundledPodmanPath() {
		source = "bundled"
	}
	check.Status = "ok"
	check.Message = fmt.Sprintf("available (v%s, %s)", version, source)

	// Check if pasta is available for podman
	if _, err := exec.LookPath("pasta"); err == nil {
		check.Message += " + pasta"
	}

	return check
}

func checkGit() DoctorCheck {
	check := DoctorCheck{Name: "Git"}

	gitPath, err := exec.LookPath("git")
	if err != nil {
		check.Status = "fail"
		check.Message = "not installed"
		check.Fix = "Install Git from https://git-scm.com/"
		return check
	}

	cmd := exec.Command(gitPath, "version")
	output, err := cmd.Output()
	if err != nil {
		check.Status = "fail"
		check.Message = "not working"
		return check
	}

	// Parse "git version 2.39.0" -> "2.39.0"
	version := strings.TrimPrefix(strings.TrimSpace(string(output)), "git version ")
	check.Status = "ok"
	check.Message = fmt.Sprintf("installed (v%s)", version)
	return check
}

func checkAnthropicKey() DoctorCheck {
	check := DoctorCheck{Name: "ANTHROPIC_API_KEY"}

	key := os.Getenv("ANTHROPIC_API_KEY")
	if key == "" {
		check.Status = "warn"
		check.Message = "not set"
		check.Fix = "Set ANTHROPIC_API_KEY or run 'claude login' locally"
		return check
	}

	// Mask the key for display
	if len(key) > 10 {
		check.Message = fmt.Sprintf("set (%s...)", key[:10])
	} else {
		check.Message = "set"
	}
	check.Status = "ok"
	return check
}

func checkGitHubToken() DoctorCheck {
	check := DoctorCheck{Name: "GitHub Token"}

	// Check GH_TOKEN first
	token := os.Getenv("GH_TOKEN")
	if token == "" {
		token = os.Getenv("GITHUB_TOKEN")
	}

	if token != "" {
		if len(token) > 10 {
			check.Message = fmt.Sprintf("set via env (%s...)", token[:10])
		} else {
			check.Message = "set via env"
		}
		check.Status = "ok"
		return check
	}

	// Try to detect from gh CLI
	ghPath, err := exec.LookPath("gh")
	if err == nil {
		cmd := exec.Command(ghPath, "auth", "status")
		if err := cmd.Run(); err == nil {
			check.Status = "ok"
			check.Message = "available via gh CLI"
			return check
		}
	}

	check.Status = "warn"
	check.Message = "not configured"
	check.Fix = "Set GH_TOKEN or run 'gh auth login'"
	return check
}

func checkDiskSpace() DoctorCheck {
	check := DoctorCheck{Name: "Disk Space"}

	// Get home directory for checking
	homeDir, err := os.UserHomeDir()
	if err != nil {
		check.Status = "warn"
		check.Message = "could not determine home directory"
		return check
	}

	// Use df to check available space (works on macOS and Linux)
	cmd := exec.Command("df", "-h", homeDir)
	output, err := cmd.Output()
	if err != nil {
		check.Status = "warn"
		check.Message = "could not check disk space"
		return check
	}

	// Parse df output to get available space
	lines := strings.Split(string(output), "\n")
	if len(lines) < 2 {
		check.Status = "warn"
		check.Message = "could not parse disk space"
		return check
	}

	fields := strings.Fields(lines[1])
	if len(fields) < 4 {
		check.Status = "warn"
		check.Message = "could not parse disk space"
		return check
	}

	available := fields[3] // Available space column

	// Check if it's a warning level (less than 5GB)
	check.Status = "ok"
	check.Message = fmt.Sprintf("%s available", available)

	// Simple check: if it ends in M or K, it's probably low
	if strings.HasSuffix(available, "M") || strings.HasSuffix(available, "K") {
		check.Status = "warn"
		check.Message = fmt.Sprintf("only %s available", available)
		check.Fix = "Free up disk space for container images"
	}

	return check
}

func checkGlobalConfig() DoctorCheck {
	check := DoctorCheck{Name: "Global Config"}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		check.Status = "warn"
		check.Message = "could not determine home directory"
		return check
	}

	configPath := filepath.Join(homeDir, ".addt", "config.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		check.Status = "ok"
		check.Message = "not created (using defaults)"
		return check
	}

	check.Status = "ok"
	check.Message = fmt.Sprintf("found at %s", configPath)
	return check
}

func checkProjectConfig() DoctorCheck {
	check := DoctorCheck{Name: "Project Config"}

	cwd, err := os.Getwd()
	if err != nil {
		check.Status = "warn"
		check.Message = "could not determine current directory"
		return check
	}

	configPath := filepath.Join(cwd, ".addt.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		check.Status = "ok"
		check.Message = "not found in current directory"
		return check
	}

	check.Status = "ok"
	check.Message = "found .addt.yaml"
	return check
}

func checkNetworkConnectivity() DoctorCheck {
	check := DoctorCheck{Name: "Network"}

	// Try to reach a common endpoint
	cmd := exec.Command("curl", "-s", "-o", "/dev/null", "-w", "%{http_code}", "--max-time", "5", "https://api.anthropic.com")
	output, err := cmd.Output()
	if err != nil {
		check.Status = "warn"
		check.Message = "could not reach api.anthropic.com"
		check.Fix = "Check your internet connection or firewall settings"
		return check
	}

	statusCode := strings.TrimSpace(string(output))
	if statusCode == "401" || statusCode == "200" || statusCode == "403" {
		// Any response means network is working
		check.Status = "ok"
		check.Message = "can reach api.anthropic.com"
		return check
	}

	check.Status = "warn"
	check.Message = fmt.Sprintf("unexpected response from api.anthropic.com (%s)", statusCode)
	return check
}
