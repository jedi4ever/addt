package daytona

import (
	"bufio"
	"context"
	"crypto/md5"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	apiclient "github.com/daytonaio/daytona/libs/api-client-go"
	"github.com/jedi4ever/addt/provider"
)

// DaytonaProvider implements the Provider interface for Daytona
type DaytonaProvider struct {
	config             *provider.Config
	embeddedDockerfile []byte
	embeddedEntrypoint []byte
}

// NewDaytonaProvider creates a new Daytona provider
func NewDaytonaProvider(cfg *provider.Config, dockerfile, entrypoint []byte) (provider.Provider, error) {
	return &DaytonaProvider{
		config:             cfg,
		embeddedDockerfile: dockerfile,
		embeddedEntrypoint: entrypoint,
	}, nil
}

// Initialize initializes the Daytona provider
func (p *DaytonaProvider) Initialize(cfg *provider.Config) error {
	p.config = cfg
	return p.CheckPrerequisites()
}

// GetName returns the provider name
func (p *DaytonaProvider) GetName() string {
	return "daytona"
}

// CheckPrerequisites verifies Daytona is installed and authenticated
func (p *DaytonaProvider) CheckPrerequisites() error {
	// Check Daytona is installed
	if _, err := exec.LookPath("daytona"); err != nil {
		return fmt.Errorf("Daytona is not installed. Please install Daytona from: https://github.com/daytonaio/daytona")
	}

	// Check if user is logged in (Daytona v0.138+ uses cloud-based authentication)
	cmd := exec.Command("daytona", "list")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Not logged in to Daytona. Please run: daytona login")
	}

	return nil
}

// Exists checks if a workspace exists
func (p *DaytonaProvider) Exists(name string) bool {
	cmd := exec.Command("daytona", "list", "--format", "json")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	// Simple check - just see if the name appears in the output
	// In production, parse JSON properly
	return strings.Contains(string(output), name)
}

// IsRunning checks if a workspace is currently running
func (p *DaytonaProvider) IsRunning(name string) bool {
	// Daytona workspaces are always "running" once created
	return p.Exists(name)
}

// Start starts a stopped workspace (no-op for Daytona)
func (p *DaytonaProvider) Start(name string) error {
	// Daytona workspaces don't need explicit start
	return nil
}

// Stop stops a running workspace
func (p *DaytonaProvider) Stop(name string) error {
	cmd := exec.Command("daytona", "stop", name)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Remove removes a workspace
func (p *DaytonaProvider) Remove(name string) error {
	cmd := exec.Command("daytona", "delete", name, "-y")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// List lists all addt workspaces
func (p *DaytonaProvider) List() ([]provider.Environment, error) {
	cmd := exec.Command("daytona", "list", "--format", "table")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var envs []provider.Environment
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for i, line := range lines {
		if i == 0 || line == "" {
			// Skip header
			continue
		}
		// Parse table output - this is a simple implementation
		// In production, use JSON format and proper parsing
		parts := strings.Fields(line)
		if len(parts) > 0 && strings.HasPrefix(parts[0], "addt-") {
			envs = append(envs, provider.Environment{
				Name:      parts[0],
				Status:    "running",
				CreatedAt: "",
			})
		}
	}
	return envs, nil
}

// Run runs a command in a workspace
func (p *DaytonaProvider) Run(spec *provider.RunSpec) error {
	workspaceName := spec.Name

	// Check if workspace exists
	if !p.Exists(workspaceName) {
		// Create sandbox (Daytona v0.138+ terminology)
		fmt.Printf("Creating Daytona sandbox: %s\n", workspaceName)
		createArgs := []string{"create", "--name", workspaceName}

		// Use embedded Dockerfile and entrypoint to build custom snapshot with Claude Code
		if len(p.embeddedDockerfile) > 0 && len(p.embeddedEntrypoint) > 0 {
			fmt.Println("Building custom Daytona sandbox with Claude Code installed...")
			fmt.Println("This will take a few minutes on first build...")

			// Create temp directory for build context
			tmpDir, err := os.MkdirTemp("", "daytona-build-*")
			if err != nil {
				return fmt.Errorf("failed to create temp directory: %w", err)
			}
			defer os.RemoveAll(tmpDir)

			// Write Dockerfile to temp directory
			dockerfilePath := filepath.Join(tmpDir, "Dockerfile")
			if err := os.WriteFile(dockerfilePath, p.embeddedDockerfile, 0644); err != nil {
				return fmt.Errorf("failed to write Dockerfile: %w", err)
			}

			// Write entrypoint script to temp directory
			entrypointPath := filepath.Join(tmpDir, "daytona-entrypoint.sh")
			if err := os.WriteFile(entrypointPath, p.embeddedEntrypoint, 0755); err != nil {
				return fmt.Errorf("failed to write entrypoint: %w", err)
			}

			// Add Dockerfile and context flags
			createArgs = append(createArgs, "--dockerfile", dockerfilePath)
			createArgs = append(createArgs, "--context", tmpDir)
		} else if spec.ImageName != "" {
			// Fall back to using a snapshot if specified
			createArgs = append(createArgs, "--snapshot", spec.ImageName)
		} else {
			// Use default Daytona snapshot
			fmt.Println("Note: Using default snapshot. Claude Code not pre-installed.")
		}

		// Note: Daytona cloud sandboxes don't support mounting local filesystem paths
		// The --volume flag is for Daytona-managed volumes only
		// We skip volume mounting for cloud sandboxes
		// TODO: Consider uploading files via other means or using Daytona volumes

		// Add environment variables from env file if specified
		if envFile := spec.Env["ADDT_ENV_FILE"]; envFile != "" {
			createArgs = append(createArgs, p.loadEnvFile(envFile)...)
		}

		// Add environment variables
		for k, v := range spec.Env {
			if k != "ADDT_ENV_FILE" { // Skip the env file path itself
				createArgs = append(createArgs, "--env", fmt.Sprintf("%s=%s", k, v))
			}
		}

		// Add GPG_TTY if GPG forwarding is enabled
		if spec.GPGForward != "" && spec.GPGForward != "off" && spec.GPGForward != "false" {
			createArgs = append(createArgs, "--env", "GPG_TTY=/dev/console")
		}

		cmd := exec.Command("daytona", createArgs...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to create Daytona workspace: %w", err)
		}

		// Wait for sandbox to be fully started (max 60 seconds)
		fmt.Println("Waiting for sandbox to be ready...")
		for i := 0; i < 60; i++ {
			infoCmd := exec.Command("daytona", "info", workspaceName)
			output, err := infoCmd.Output()
			if err == nil && strings.Contains(string(output), "State           STARTED") {
				fmt.Println("Sandbox is ready!")
				break
			}
			if i == 59 {
				return fmt.Errorf("timeout waiting for sandbox to start")
			}
			time.Sleep(1 * time.Second)
		}

		// Note: Daytona provides public URLs automatically via preview-url command
		// Port forwarding is handled differently than Docker
		if len(spec.Ports) > 0 {
			fmt.Println("Note: Daytona provides automatic public URLs for exposed ports")
			fmt.Println("Use 'daytona preview-url' command to access your ports")
		}
	} else {
		fmt.Printf("Using existing Daytona sandbox: %s\n", workspaceName)
	}

	// For interactive sessions, use SSH with API token for proper TTY support
	if spec.Interactive {
		fmt.Println("Starting interactive SSH session...")
		return p.runInteractiveSSH(workspaceName, spec.Args)
	}

	// For non-interactive sessions (piped input, scripts), use exec
	execArgs := []string{"exec", workspaceName, "--"}
	execArgs = append(execArgs, spec.Args...)

	cmd := exec.Command("daytona", execArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Shell opens a shell in a workspace
func (p *DaytonaProvider) Shell(spec *provider.RunSpec) error {
	workspaceName := spec.Name

	// Check if workspace exists
	if !p.Exists(workspaceName) {
		// Create sandbox (Daytona v0.138+ terminology)
		fmt.Printf("Creating Daytona sandbox: %s\n", workspaceName)
		createArgs := []string{"create", "--name", workspaceName}

		// Use embedded Dockerfile and entrypoint to build custom snapshot with Claude Code
		if len(p.embeddedDockerfile) > 0 && len(p.embeddedEntrypoint) > 0 {
			fmt.Println("Building custom Daytona sandbox with Claude Code installed...")
			fmt.Println("This will take a few minutes on first build...")

			// Create temp directory for build context
			tmpDir, err := os.MkdirTemp("", "daytona-build-*")
			if err != nil {
				return fmt.Errorf("failed to create temp directory: %w", err)
			}
			defer os.RemoveAll(tmpDir)

			// Write Dockerfile to temp directory
			dockerfilePath := filepath.Join(tmpDir, "Dockerfile")
			if err := os.WriteFile(dockerfilePath, p.embeddedDockerfile, 0644); err != nil {
				return fmt.Errorf("failed to write Dockerfile: %w", err)
			}

			// Write entrypoint script to temp directory
			entrypointPath := filepath.Join(tmpDir, "daytona-entrypoint.sh")
			if err := os.WriteFile(entrypointPath, p.embeddedEntrypoint, 0755); err != nil {
				return fmt.Errorf("failed to write entrypoint: %w", err)
			}

			// Add Dockerfile and context flags
			createArgs = append(createArgs, "--dockerfile", dockerfilePath)
			createArgs = append(createArgs, "--context", tmpDir)
		} else if spec.ImageName != "" {
			// Fall back to using a snapshot if specified
			createArgs = append(createArgs, "--snapshot", spec.ImageName)
		} else {
			// Use default Daytona snapshot
			fmt.Println("Note: Using default snapshot. Claude Code not pre-installed.")
		}

		// Note: Daytona cloud sandboxes don't support mounting local filesystem paths
		// The --volume flag is for Daytona-managed volumes only
		// We skip volume mounting for cloud sandboxes
		// TODO: Consider uploading files via other means or using Daytona volumes

		// Add environment variables from env file if specified
		if envFile := spec.Env["ADDT_ENV_FILE"]; envFile != "" {
			createArgs = append(createArgs, p.loadEnvFile(envFile)...)
		}

		// Add environment variables
		for k, v := range spec.Env {
			if k != "ADDT_ENV_FILE" { // Skip the env file path itself
				createArgs = append(createArgs, "--env", fmt.Sprintf("%s=%s", k, v))
			}
		}

		// Add GPG_TTY if GPG forwarding is enabled
		if spec.GPGForward != "" && spec.GPGForward != "off" && spec.GPGForward != "false" {
			createArgs = append(createArgs, "--env", "GPG_TTY=/dev/console")
		}

		cmd := exec.Command("daytona", createArgs...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to create Daytona workspace: %w", err)
		}

		// Wait for sandbox to be fully started (max 60 seconds)
		fmt.Println("Waiting for sandbox to be ready...")
		for i := 0; i < 60; i++ {
			infoCmd := exec.Command("daytona", "info", workspaceName)
			output, err := infoCmd.Output()
			if err == nil && strings.Contains(string(output), "State           STARTED") {
				fmt.Println("Sandbox is ready!")
				break
			}
			if i == 59 {
				return fmt.Errorf("timeout waiting for sandbox to start")
			}
			time.Sleep(1 * time.Second)
		}

		// Note: Daytona provides public URLs automatically via preview-url command
		// Port forwarding is handled differently than Docker
		if len(spec.Ports) > 0 {
			fmt.Println("Note: Daytona provides automatic public URLs for exposed ports")
			fmt.Println("Use 'daytona preview-url' command to access your ports")
		}
	} else {
		fmt.Printf("Using existing Daytona sandbox: %s\n", workspaceName)
	}

	// Connect to sandbox with SSH for interactive shell
	fmt.Println("Opening SSH session to Daytona sandbox...")
	// Pass empty args to run the entrypoint which will start claude by default
	return p.runInteractiveSSH(workspaceName, []string{})
}

// Cleanup cleans up resources
func (p *DaytonaProvider) Cleanup() error {
	// Nothing to clean up for Daytona
	return nil
}

// GetStatus returns a status string for display
func (p *DaytonaProvider) GetStatus(cfg *provider.Config, envName string) string {
	status := fmt.Sprintf("Provider:%s Mode:sandbox", p.GetName())

	// Sandbox name
	if cfg.Persistent {
		status += fmt.Sprintf(" | Sandbox:%s", envName)
	}

	// GitHub token status
	if os.Getenv("GH_TOKEN") != "" {
		status += " | GH:✓"
	} else {
		status += " | GH:-"
	}

	// SSH forwarding status
	switch cfg.SSHForward {
	case "agent":
		status += " | SSH:builtin"
	case "keys":
		status += " | SSH:keys"
	default:
		status += " | SSH:builtin"
	}

	// GPG forwarding status
	if cfg.GPGForward != "" && cfg.GPGForward != "off" && cfg.GPGForward != "false" {
		status += fmt.Sprintf(" | GPG:%s", cfg.GPGForward)
	} else {
		status += " | GPG:-"
	}

	// Docker support - note that Daytona may not support this
	if cfg.DockerDindMode != "" {
		status += " | Docker:limited"
	}

	// Environment variables
	if len(cfg.EnvVars) > 0 || cfg.EnvFile != "" {
		status += " | Env:✓"
	}

	// Port mappings
	if len(cfg.Ports) > 0 {
		status += fmt.Sprintf(" | Ports:%d", len(cfg.Ports))
	}

	return status
}

// GeneratePersistentName generates a sandbox name for persistent mode
func (p *DaytonaProvider) GeneratePersistentName() string {
	// Use configured workdir or fall back to current directory
	workdir := p.config.Workdir
	if workdir == "" {
		var err error
		workdir, err = os.Getwd()
		if err != nil {
			return "addt-sandbox"
		}
	}

	// Get directory name
	dirname := filepath.Base(workdir)

	// Sanitize directory name (lowercase, remove special chars, max 20 chars)
	re := regexp.MustCompile(`[^a-z0-9-]+`)
	dirname = strings.ToLower(dirname)
	dirname = re.ReplaceAllString(dirname, "-")
	dirname = strings.Trim(dirname, "-")
	if len(dirname) > 20 {
		dirname = dirname[:20]
	}

	// Create hash of full path for uniqueness
	hash := md5.Sum([]byte(workdir))
	hashStr := fmt.Sprintf("%x", hash)[:8]

	return fmt.Sprintf("addt-sandbox-%s-%s", dirname, hashStr)
}

// GenerateEphemeralName generates a unique sandbox name for ephemeral mode
func (p *DaytonaProvider) GenerateEphemeralName() string {
	return fmt.Sprintf("addt-%s-%d", time.Now().Format("20060102-150405"), os.Getpid())
}

// BuildIfNeeded is a no-op for Daytona (no image building needed)
func (p *DaytonaProvider) BuildIfNeeded(rebuild bool, rebuildBase bool) error {
	return nil
}

// DetermineImageName returns empty string for Daytona (no image concept)
func (p *DaytonaProvider) DetermineImageName() string {
	return ""
}

// GetExtensionEnvVars returns extension-required environment variables
// For Daytona, we don't have local image metadata, so return nil
// TODO: Read extension config from sandbox or config file
func (p *DaytonaProvider) GetExtensionEnvVars(imageName string) []string {
	return nil
}

// loadEnvFile reads an env file and converts it to --env flags
// Daytona doesn't support --env-file, so we parse it manually
func (p *DaytonaProvider) loadEnvFile(envFilePath string) []string {
	var args []string

	if envFilePath == "" {
		return args
	}

	file, err := os.Open(envFilePath)
	if err != nil {
		fmt.Printf("Warning: Failed to open env file %s: %v\n", envFilePath, err)
		return args
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse KEY=VALUE
		if strings.Contains(line, "=") {
			args = append(args, "--env", line)
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Printf("Warning: Error reading env file: %v\n", err)
	}

	return args
}

// runInteractiveSSH uses the Daytona API to get an SSH token and connects via native ssh
func (p *DaytonaProvider) runInteractiveSSH(sandboxName string, args []string) error {
	ctx := context.Background()

	fmt.Println("Creating Daytona API client...")
	// Create API client configuration with proper settings
	cfg := apiclient.NewConfiguration()

	// Configure API URL (default to app.daytona.io/api)
	apiURL := os.Getenv("DAYTONA_API_URL")
	if apiURL == "" {
		apiURL = "https://app.daytona.io/api"
	}
	cfg.Servers[0].URL = apiURL
	fmt.Printf("Using API URL: %s\n", apiURL)

	// Add API key authentication
	apiKey := os.Getenv("DAYTONA_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("DAYTONA_API_KEY environment variable not set")
	}
	cfg.AddDefaultHeader("Authorization", "Bearer "+apiKey)

	// Add organization ID if set
	orgID := os.Getenv("DAYTONA_ORGANIZATION_ID")
	if orgID != "" {
		cfg.AddDefaultHeader("X-Daytona-Organization-ID", orgID)
		fmt.Printf("Using Organization ID: %s\n", orgID)
	}

	apiClient := apiclient.NewAPIClient(cfg)

	fmt.Printf("Requesting SSH token for sandbox: %s\n", sandboxName)
	// Generate SSH token (30 minutes expiry)
	expiresIn := float32(30)
	req := apiClient.SandboxAPI.CreateSshAccess(ctx, sandboxName).ExpiresInMinutes(expiresIn)
	sshAccess, resp, err := req.Execute()
	if err != nil {
		fmt.Printf("API Error: %v\n", err)
		if resp != nil {
			fmt.Printf("Response Status: %s\n", resp.Status)
		}
		return fmt.Errorf("failed to create SSH access: %w", err)
	}

	if sshAccess.Token == "" {
		return fmt.Errorf("SSH token is empty")
	}

	fmt.Printf("Got SSH token: %s...\n", sshAccess.Token[:10])

	// Build command to run - use entrypoint script which handles port mapping and setup
	entrypointCmd := "/usr/local/bin/daytona-entrypoint.sh"
	var cmdToRun string
	if len(args) > 0 {
		cmdToRun = entrypointCmd + " " + strings.Join(args, " ")
	} else {
		cmdToRun = entrypointCmd
	}

	// Use native SSH with -t flag for TTY allocation
	fmt.Println("Connecting via SSH with proper TTY...")
	sshCmd := exec.Command("ssh",
		"-t",
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "LogLevel=ERROR",
		fmt.Sprintf("%s@ssh.app.daytona.io", sshAccess.Token),
		cmdToRun)
	sshCmd.Stdin = os.Stdin
	sshCmd.Stdout = os.Stdout
	sshCmd.Stderr = os.Stderr

	return sshCmd.Run()
}
