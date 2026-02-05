package main

import (
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

//go:embed static/*
var staticFiles embed.FS

// Session represents an addt session
type Session struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Status    string    `json:"status"` // running, stopped
	StartedAt time.Time `json:"started_at,omitempty"`
	WorkDir   string    `json:"work_dir"`
	Image     string    `json:"image,omitempty"`
}

// SessionManager manages addt sessions
type SessionManager struct {
	mu       sync.RWMutex
	sessions map[string]*Session
}

// NewSessionManager creates a new session manager
func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions: make(map[string]*Session),
	}
}

// List returns all sessions
func (sm *SessionManager) List() []*Session {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	// Get sessions from docker/podman
	sessions := sm.getContainerSessions()
	return sessions
}

// getContainerSessions queries docker/podman for addt containers
func (sm *SessionManager) getContainerSessions() []*Session {
	var sessions []*Session

	// Try docker first, then podman
	cmd := exec.Command("docker", "ps", "-a", "--filter", "name=addt-", "--format", "{{.Names}}\t{{.Status}}\t{{.Labels}}")
	output, err := cmd.Output()
	if err != nil {
		// Try podman
		cmd = exec.Command("podman", "ps", "-a", "--filter", "name=addt-", "--format", "{{.Names}}\t{{.Status}}\t{{.Labels}}")
		output, err = cmd.Output()
		if err != nil {
			return sessions
		}
	}

	// Parse output
	lines := splitLines(string(output))
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := splitTabs(line)
		if len(parts) < 2 {
			continue
		}

		name := parts[0]
		statusStr := parts[1]

		status := "stopped"
		if containsString(statusStr, "Up") {
			status = "running"
		}

		sessions = append(sessions, &Session{
			ID:     name,
			Name:   name,
			Status: status,
		})
	}

	return sessions
}

// Start starts a new or existing session
func (sm *SessionManager) Start(name, workDir string) (*Session, error) {
	// Check if session exists and is stopped
	cmd := exec.Command("docker", "start", name)
	if err := cmd.Run(); err != nil {
		// Try podman
		cmd = exec.Command("podman", "start", name)
		if err := cmd.Run(); err != nil {
			return nil, fmt.Errorf("failed to start session: %w", err)
		}
	}

	return &Session{
		ID:        name,
		Name:      name,
		Status:    "running",
		StartedAt: time.Now(),
		WorkDir:   workDir,
	}, nil
}

// Stop stops a session
func (sm *SessionManager) Stop(name string) error {
	cmd := exec.Command("docker", "stop", name)
	if err := cmd.Run(); err != nil {
		// Try podman
		cmd = exec.Command("podman", "stop", name)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to stop session: %w", err)
		}
	}
	return nil
}

// Remove removes a session
func (sm *SessionManager) Remove(name string) error {
	cmd := exec.Command("docker", "rm", "-f", name)
	if err := cmd.Run(); err != nil {
		// Try podman
		cmd = exec.Command("podman", "rm", "-f", name)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to remove session: %w", err)
		}
	}
	return nil
}

// Server is the orchestrator HTTP server
type Server struct {
	sm       *SessionManager
	upgrader websocket.Upgrader
	port     int
}

// NewServer creates a new server
func NewServer(port int) *Server {
	return &Server{
		sm:   NewSessionManager(),
		port: port,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins for local use
			},
		},
	}
}

// handleSessions returns list of sessions
func (s *Server) handleSessions(w http.ResponseWriter, r *http.Request) {
	sessions := s.sm.List()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sessions)
}

// handleSessionStart starts a session
func (s *Server) handleSessionStart(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if name == "" {
		http.Error(w, "name required", http.StatusBadRequest)
		return
	}

	session, err := s.sm.Start(name, "")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(session)
}

// handleSessionStop stops a session
func (s *Server) handleSessionStop(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if name == "" {
		http.Error(w, "name required", http.StatusBadRequest)
		return
	}

	if err := s.sm.Stop(name); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// handleSessionRemove removes a session
func (s *Server) handleSessionRemove(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if name == "" {
		http.Error(w, "name required", http.StatusBadRequest)
		return
	}

	if err := s.sm.Remove(name); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// handleTerminal handles WebSocket connections for terminal access
func (s *Server) handleTerminal(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if name == "" {
		http.Error(w, "name required", http.StatusBadRequest)
		return
	}

	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}
	defer conn.Close()

	// Start docker/podman exec with PTY
	cmd := exec.Command("docker", "exec", "-it", name, "/bin/bash")
	ptmx, err := startPty(cmd)
	if err != nil {
		// Try podman
		cmd = exec.Command("podman", "exec", "-it", name, "/bin/bash")
		ptmx, err = startPty(cmd)
		if err != nil {
			log.Printf("Failed to start PTY: %v", err)
			conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("Error: %v", err)))
			return
		}
	}
	defer ptmx.Close()

	// Read from PTY, write to WebSocket
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := ptmx.Read(buf)
			if err != nil {
				return
			}
			if err := conn.WriteMessage(websocket.BinaryMessage, buf[:n]); err != nil {
				return
			}
		}
	}()

	// Read from WebSocket, write to PTY
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			break
		}
		if _, err := ptmx.Write(message); err != nil {
			break
		}
	}

	cmd.Process.Kill()
}

// Run starts the server
func (s *Server) Run() error {
	mux := http.NewServeMux()

	// API routes
	mux.HandleFunc("/api/sessions", s.handleSessions)
	mux.HandleFunc("/api/sessions/start", s.handleSessionStart)
	mux.HandleFunc("/api/sessions/stop", s.handleSessionStop)
	mux.HandleFunc("/api/sessions/remove", s.handleSessionRemove)
	mux.HandleFunc("/api/terminal", s.handleTerminal)

	// Static files
	staticFS, err := fs.Sub(staticFiles, "static")
	if err != nil {
		return fmt.Errorf("failed to get static files: %w", err)
	}
	mux.Handle("/", http.FileServer(http.FS(staticFS)))

	addr := fmt.Sprintf(":%d", s.port)
	fmt.Printf("🚀 addt-orchestrator running at http://localhost%s\n", addr)
	return http.ListenAndServe(addr, mux)
}

// Helper functions
func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func splitTabs(s string) []string {
	var parts []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\t' {
			parts = append(parts, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		parts = append(parts, s[start:])
	}
	return parts
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr) >= 0
}

func findSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func main() {
	port := flag.Int("port", 8080, "Port to listen on")
	flag.Parse()

	server := NewServer(*port)
	if err := server.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
