package util

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

// Spinner displays an animated spinner for indeterminate operations
type Spinner struct {
	message  string
	frames   []string
	interval time.Duration
	stop     chan struct{}
	done     chan struct{}
	mu       sync.Mutex
	writer   io.Writer
	active   bool
}

// NewSpinner creates a new spinner with the given message
func NewSpinner(message string) *Spinner {
	return &Spinner{
		message:  message,
		frames:   []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
		interval: 80 * time.Millisecond,
		stop:     make(chan struct{}),
		done:     make(chan struct{}),
		writer:   os.Stderr,
	}
}

// Start begins the spinner animation
func (s *Spinner) Start() {
	s.mu.Lock()
	if s.active {
		s.mu.Unlock()
		return
	}
	s.active = true
	s.mu.Unlock()

	go func() {
		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()
		defer close(s.done)

		i := 0
		for {
			select {
			case <-s.stop:
				// Clear the spinner line
				fmt.Fprintf(s.writer, "\r%s\r", strings.Repeat(" ", len(s.message)+5))
				return
			case <-ticker.C:
				s.mu.Lock()
				msg := s.message
				s.mu.Unlock()
				fmt.Fprintf(s.writer, "\r%s %s ", s.frames[i%len(s.frames)], msg)
				i++
			}
		}
	}()
}

// UpdateMessage changes the spinner message while running
func (s *Spinner) UpdateMessage(message string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.message = message
}

// Stop stops the spinner and optionally shows a final message
func (s *Spinner) Stop() {
	s.mu.Lock()
	if !s.active {
		s.mu.Unlock()
		return
	}
	s.active = false
	s.mu.Unlock()

	close(s.stop)
	<-s.done
}

// StopWithSuccess stops the spinner and shows a success message
func (s *Spinner) StopWithSuccess(message string) {
	s.Stop()
	fmt.Fprintf(s.writer, "\r%s %s\n", SuccessIcon, message)
}

// StopWithError stops the spinner and shows an error message
func (s *Spinner) StopWithError(message string) {
	s.Stop()
	fmt.Fprintf(s.writer, "\r%s %s\n", ErrorIcon, message)
}

// StopWithWarning stops the spinner and shows a warning message
func (s *Spinner) StopWithWarning(message string) {
	s.Stop()
	fmt.Fprintf(s.writer, "\r%s %s\n", WarningIcon, message)
}

// Progress icons
const (
	SuccessIcon = "✓"
	ErrorIcon   = "✗"
	WarningIcon = "!"
	InfoIcon    = "→"
	BuildIcon   = "🔨"
	PackageIcon = "📦"
	RocketIcon  = "🚀"
)

// ProgressBar displays a progress bar for determinate operations
type ProgressBar struct {
	total    int
	current  int
	width    int
	message  string
	mu       sync.Mutex
	writer   io.Writer
	complete bool
}

// NewProgressBar creates a new progress bar
func NewProgressBar(total int, message string) *ProgressBar {
	return &ProgressBar{
		total:   total,
		width:   40,
		message: message,
		writer:  os.Stderr,
	}
}

// Update updates the progress bar
func (p *ProgressBar) Update(current int, message string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.current = current
	if message != "" {
		p.message = message
	}
	p.render()
}

// Increment increments the progress by one
func (p *ProgressBar) Increment(message string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.current++
	if message != "" {
		p.message = message
	}
	p.render()
}

func (p *ProgressBar) render() {
	if p.total <= 0 {
		return
	}

	percent := float64(p.current) / float64(p.total)
	filled := int(percent * float64(p.width))
	if filled > p.width {
		filled = p.width
	}

	bar := strings.Repeat("█", filled) + strings.Repeat("░", p.width-filled)
	fmt.Fprintf(p.writer, "\r[%s] %3.0f%% %s", bar, percent*100, p.message)
}

// Complete marks the progress bar as complete
func (p *ProgressBar) Complete(message string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.current = p.total
	p.complete = true
	p.render()
	fmt.Fprintf(p.writer, "\n%s %s\n", SuccessIcon, message)
}

// Fail marks the progress bar as failed
func (p *ProgressBar) Fail(message string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.complete = true
	fmt.Fprintf(p.writer, "\n%s %s\n", ErrorIcon, message)
}

// StepProgress tracks progress through discrete steps
type StepProgress struct {
	steps   []string
	current int
	mu      sync.Mutex
	writer  io.Writer
}

// NewStepProgress creates a new step progress tracker
func NewStepProgress(steps []string) *StepProgress {
	return &StepProgress{
		steps:  steps,
		writer: os.Stderr,
	}
}

// Start begins tracking progress
func (sp *StepProgress) Start() {
	fmt.Fprintf(sp.writer, "\n")
	sp.printStep(0, "pending")
}

// NextStep moves to the next step
func (sp *StepProgress) NextStep() {
	sp.mu.Lock()
	defer sp.mu.Unlock()

	if sp.current < len(sp.steps) {
		sp.printStepComplete(sp.current)
		sp.current++
		if sp.current < len(sp.steps) {
			sp.printStep(sp.current, "active")
		}
	}
}

// CompleteStep marks current step as complete and shows message
func (sp *StepProgress) CompleteStep(message string) {
	sp.mu.Lock()
	defer sp.mu.Unlock()

	if sp.current < len(sp.steps) {
		sp.printStepWithMessage(sp.current, message)
		sp.current++
		if sp.current < len(sp.steps) {
			sp.printStep(sp.current, "active")
		}
	}
}

// FailStep marks current step as failed
func (sp *StepProgress) FailStep(message string) {
	sp.mu.Lock()
	defer sp.mu.Unlock()

	fmt.Fprintf(sp.writer, "\r%s Step %d/%d: %s - %s\n",
		ErrorIcon, sp.current+1, len(sp.steps), sp.steps[sp.current], message)
}

// Complete marks all steps as complete
func (sp *StepProgress) Complete() {
	sp.mu.Lock()
	defer sp.mu.Unlock()

	fmt.Fprintf(sp.writer, "\n%s All steps completed!\n", SuccessIcon)
}

func (sp *StepProgress) printStep(idx int, status string) {
	icon := "○"
	if status == "active" {
		icon = "●"
	}
	fmt.Fprintf(sp.writer, "%s Step %d/%d: %s\n", icon, idx+1, len(sp.steps), sp.steps[idx])
}

func (sp *StepProgress) printStepComplete(idx int) {
	// Move cursor up and overwrite
	fmt.Fprintf(sp.writer, "\033[1A\r%s Step %d/%d: %s\n",
		SuccessIcon, idx+1, len(sp.steps), sp.steps[idx])
}

func (sp *StepProgress) printStepWithMessage(idx int, message string) {
	fmt.Fprintf(sp.writer, "\033[1A\r%s Step %d/%d: %s - %s\n",
		SuccessIcon, idx+1, len(sp.steps), sp.steps[idx], message)
}

// PrintStatus prints a status message with an icon
func PrintStatus(icon, message string) {
	fmt.Fprintf(os.Stderr, "%s %s\n", icon, message)
}

// PrintSuccess prints a success message
func PrintSuccess(message string) {
	PrintStatus(SuccessIcon, message)
}

// PrintError prints an error message
func PrintError(message string) {
	PrintStatus(ErrorIcon, message)
}

// PrintWarning prints a warning message
func PrintWarning(message string) {
	PrintStatus(WarningIcon, message)
}

// PrintInfo prints an info message
func PrintInfo(message string) {
	PrintStatus(InfoIcon, message)
}

// BuildProgress provides progress tracking specifically for container builds
type BuildProgress struct {
	spinner      *Spinner
	layerCount   int
	currentLayer int
	startTime    time.Time
}

// NewBuildProgress creates a new build progress tracker
func NewBuildProgress() *BuildProgress {
	return &BuildProgress{
		spinner:   NewSpinner("Building container image..."),
		startTime: time.Now(),
	}
}

// Start begins tracking the build
func (bp *BuildProgress) Start() {
	PrintInfo("Starting container image build")
	bp.spinner.Start()
}

// UpdateLayer updates the current build layer
func (bp *BuildProgress) UpdateLayer(layer, total int, message string) {
	bp.layerCount = total
	bp.currentLayer = layer
	if message != "" {
		bp.spinner.UpdateMessage(fmt.Sprintf("Building [%d/%d] %s", layer, total, message))
	} else {
		bp.spinner.UpdateMessage(fmt.Sprintf("Building layer %d/%d", layer, total))
	}
}

// UpdateStep updates with a build step message
func (bp *BuildProgress) UpdateStep(message string) {
	bp.spinner.UpdateMessage(message)
}

// Complete marks the build as complete
func (bp *BuildProgress) Complete() {
	elapsed := time.Since(bp.startTime).Round(time.Second)
	bp.spinner.StopWithSuccess(fmt.Sprintf("Build completed in %s", elapsed))
}

// Fail marks the build as failed
func (bp *BuildProgress) Fail(err string) {
	bp.spinner.StopWithError(fmt.Sprintf("Build failed: %s", err))
}
