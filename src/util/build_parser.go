package util

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

// BuildOutput represents parsed build output
type BuildOutput struct {
	Step     int
	Total    int
	Message  string
	IsStep   bool
	IsCached bool
	IsError  bool
	Raw      string
}

// BuildRunner executes a container build command with progress output
type BuildRunner struct {
	Command     string // "docker" or "podman"
	Args        []string
	Verbose     bool
	startTime   time.Time
	currentStep int
	totalSteps  int
	spinner     *Spinner
}

// NewBuildRunner creates a new build runner
func NewBuildRunner(command string, args []string) *BuildRunner {
	return &BuildRunner{
		Command: command,
		Args:    args,
		Verbose: os.Getenv("ADDT_VERBOSE") == "true",
	}
}

// Run executes the build with progress indication
func (br *BuildRunner) Run() error {
	br.startTime = time.Now()

	// If verbose mode, just run normally
	if br.Verbose {
		cmd := exec.Command(br.Command, br.Args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}

	// Create command with piped output
	cmd := exec.Command(br.Command, br.Args...)

	// Combine stdout and stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start spinner
	br.spinner = NewSpinner("Preparing build...")
	br.spinner.Start()

	// Start the command
	if err := cmd.Start(); err != nil {
		br.spinner.StopWithError("Failed to start build")
		return fmt.Errorf("failed to start build: %w", err)
	}

	// Process output in goroutines
	done := make(chan struct{})
	var buildErr error

	go func() {
		br.processOutput(io.MultiReader(stdout, stderr))
		close(done)
	}()

	// Wait for output processing to complete
	<-done

	// Wait for command to finish
	if err := cmd.Wait(); err != nil {
		br.spinner.StopWithError(fmt.Sprintf("Build failed: %v", err))
		return err
	}

	// Show completion
	elapsed := time.Since(br.startTime).Round(time.Second)
	br.spinner.StopWithSuccess(fmt.Sprintf("Build completed in %s", elapsed))

	return buildErr
}

func (br *BuildRunner) processOutput(reader io.Reader) {
	scanner := bufio.NewScanner(reader)

	// Regex patterns for parsing build output
	// Docker BuildKit format: #5 [2/4] RUN npm install
	buildkitStepRegex := regexp.MustCompile(`#\d+\s+\[(\d+)/(\d+)\]\s+(.+)`)
	// Legacy format: Step 2/4 : RUN npm install
	legacyStepRegex := regexp.MustCompile(`Step\s+(\d+)/(\d+)\s*:\s*(.+)`)
	// Cached step: CACHED
	cachedRegex := regexp.MustCompile(`(?i)CACHED`)
	// Error patterns
	errorRegex := regexp.MustCompile(`(?i)(error|failed|cannot|unable to)`)
	// Podman format similar to Docker
	podmanStepRegex := regexp.MustCompile(`STEP\s+(\d+)/(\d+):\s*(.+)`)

	for scanner.Scan() {
		line := scanner.Text()
		output := br.parseLine(line, buildkitStepRegex, legacyStepRegex, podmanStepRegex, cachedRegex, errorRegex)

		if output.IsStep {
			br.currentStep = output.Step
			br.totalSteps = output.Total

			msg := output.Message
			if len(msg) > 50 {
				msg = msg[:47] + "..."
			}

			status := fmt.Sprintf("[%d/%d] %s", output.Step, output.Total, msg)
			if output.IsCached {
				status += " (cached)"
			}
			br.spinner.UpdateMessage(status)
		} else if output.IsError {
			// Store error for later display
			fmt.Fprintf(os.Stderr, "\n%s\n", output.Raw)
		}
	}
}

func (br *BuildRunner) parseLine(line string, buildkitRegex, legacyRegex, podmanRegex, cachedRegex, errorRegex *regexp.Regexp) BuildOutput {
	output := BuildOutput{Raw: line}

	// Try BuildKit format
	if matches := buildkitRegex.FindStringSubmatch(line); matches != nil {
		output.IsStep = true
		fmt.Sscanf(matches[1], "%d", &output.Step)
		fmt.Sscanf(matches[2], "%d", &output.Total)
		output.Message = strings.TrimSpace(matches[3])
		output.IsCached = cachedRegex.MatchString(line)
		return output
	}

	// Try legacy Docker format
	if matches := legacyRegex.FindStringSubmatch(line); matches != nil {
		output.IsStep = true
		fmt.Sscanf(matches[1], "%d", &output.Step)
		fmt.Sscanf(matches[2], "%d", &output.Total)
		output.Message = strings.TrimSpace(matches[3])
		output.IsCached = cachedRegex.MatchString(line)
		return output
	}

	// Try Podman format
	if matches := podmanRegex.FindStringSubmatch(line); matches != nil {
		output.IsStep = true
		fmt.Sscanf(matches[1], "%d", &output.Step)
		fmt.Sscanf(matches[2], "%d", &output.Total)
		output.Message = strings.TrimSpace(matches[3])
		output.IsCached = cachedRegex.MatchString(line)
		return output
	}

	// Check for errors
	if errorRegex.MatchString(line) {
		output.IsError = true
	}

	return output
}

// RunBuildCommand runs a docker/podman build with progress indication
func RunBuildCommand(command string, args []string) error {
	runner := NewBuildRunner(command, args)
	return runner.Run()
}

// SimpleSpinnerRun runs a command with a simple spinner
func SimpleSpinnerRun(message string, cmd *exec.Cmd) error {
	spinner := NewSpinner(message)
	spinner.Start()

	// Suppress output unless verbose
	if os.Getenv("ADDT_VERBOSE") != "true" {
		cmd.Stdout = nil
		cmd.Stderr = nil
	} else {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	err := cmd.Run()

	if err != nil {
		spinner.StopWithError(fmt.Sprintf("%s - failed", message))
		return err
	}

	spinner.StopWithSuccess(message)
	return nil
}

// WithSpinner wraps a function with spinner display
func WithSpinner(message string, fn func() error) error {
	spinner := NewSpinner(message)
	spinner.Start()

	err := fn()

	if err != nil {
		spinner.StopWithError(fmt.Sprintf("%s - failed: %v", message, err))
		return err
	}

	spinner.StopWithSuccess(message)
	return nil
}

// PrintBuildStart prints build start message
func PrintBuildStart(imageName string) {
	fmt.Println()
	PrintInfo(fmt.Sprintf("Building image: %s", imageName))
}

// PrintBuildComplete prints build completion message with timing
func PrintBuildComplete(imageName string, elapsed time.Duration) {
	PrintSuccess(fmt.Sprintf("Build completed in %s", elapsed.Round(time.Second)))
	fmt.Printf("   Image: %s\n", imageName)
}

// PrintCacheHit prints a cache hit message
func PrintCacheHit(imageName string) {
	PrintSuccess(fmt.Sprintf("Using cached image: %s", imageName))
}
