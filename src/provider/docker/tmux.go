package docker

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// HandleTmuxForwarding handles tmux socket forwarding to the container.
// When enabled, it detects the active tmux socket and mounts it into the container,
// allowing the container to interact with the host's tmux session.
func (p *DockerProvider) HandleTmuxForwarding(enabled bool) []string {
	if !enabled {
		return nil
	}

	// Check if we're inside a tmux session
	tmuxEnv := os.Getenv("TMUX")
	if tmuxEnv == "" {
		// Not in a tmux session, nothing to forward
		return nil
	}

	// TMUX env format: /tmp/tmux-1000/default,12345,0
	// First part is the socket path
	parts := strings.Split(tmuxEnv, ",")
	if len(parts) < 1 {
		return nil
	}

	socketPath := parts[0]
	if socketPath == "" {
		return nil
	}

	// Verify socket exists
	if _, err := os.Stat(socketPath); os.IsNotExist(err) {
		return nil
	}

	// Get the socket directory (e.g., /tmp/tmux-1000)
	socketDir := filepath.Dir(socketPath)

	var args []string

	// Mount the tmux socket directory
	args = append(args, "-v", fmt.Sprintf("%s:%s", socketDir, socketDir))

	// Pass the TMUX environment variable so tmux commands work inside container
	args = append(args, "-e", fmt.Sprintf("TMUX=%s", tmuxEnv))

	// Also pass TMUX_PANE if set
	if tmuxPane := os.Getenv("TMUX_PANE"); tmuxPane != "" {
		args = append(args, "-e", fmt.Sprintf("TMUX_PANE=%s", tmuxPane))
	}

	return args
}
