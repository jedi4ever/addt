package security

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// GPGProxyAgent creates a filtering proxy for gpg-agent
// It intercepts the Assuan protocol and can filter signing operations by key ID
type GPGProxyAgent struct {
	upstreamSocket string
	proxySocket    string
	allowedKeyIDs  []string // Key IDs (fingerprints) that are allowed
	listener       net.Listener
	mu             sync.Mutex
	running        bool
	wg             sync.WaitGroup
	useTCP         bool // listen on TCP instead of Unix socket (macOS + podman)
	tcpPort        int  // TCP port when useTCP is true
}

// NewGPGProxyAgent creates a new GPG proxy agent
// allowedKeyIDs can be full fingerprints or last 8/16 chars (short/long key ID)
func NewGPGProxyAgent(upstreamSocket string, allowedKeyIDs []string) (*GPGProxyAgent, error) {
	// Create socket directory in ~/.addt/sockets/ so Podman machine can access it
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home dir: %w", err)
	}

	socketsDir := filepath.Join(homeDir, ".addt", "sockets")
	if err := os.MkdirAll(socketsDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create sockets dir: %w", err)
	}

	// Create unique subdirectory for this proxy instance
	tmpDir, err := os.MkdirTemp(socketsDir, "gpg-proxy-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}

	// Set restrictive permissions on directory (owner only)
	if err := os.Chmod(tmpDir, 0700); err != nil {
		os.RemoveAll(tmpDir)
		return nil, fmt.Errorf("failed to set temp dir permissions: %w", err)
	}

	// Write PID file for cleanup to identify orphaned directories
	if err := WritePIDFile(tmpDir); err != nil {
		os.RemoveAll(tmpDir)
		return nil, fmt.Errorf("failed to write PID file: %w", err)
	}

	proxySocket := filepath.Join(tmpDir, "S.gpg-agent")

	return &GPGProxyAgent{
		upstreamSocket: upstreamSocket,
		proxySocket:    proxySocket,
		allowedKeyIDs:  normalizeKeyIDs(allowedKeyIDs),
	}, nil
}

// normalizeKeyIDs converts key IDs to uppercase for comparison
func normalizeKeyIDs(keyIDs []string) []string {
	normalized := make([]string, len(keyIDs))
	for i, id := range keyIDs {
		normalized[i] = strings.ToUpper(strings.TrimSpace(id))
	}
	return normalized
}

// NewGPGProxyAgentTCP creates a GPG proxy agent that listens on TCP.
// Used on macOS where podman can't mount Unix sockets from the host.
func NewGPGProxyAgentTCP(upstreamSocket string, allowedKeyIDs []string) (*GPGProxyAgent, error) {
	return &GPGProxyAgent{
		upstreamSocket: upstreamSocket,
		allowedKeyIDs:  normalizeKeyIDs(allowedKeyIDs),
		useTCP:         true,
	}, nil
}

// TCPPort returns the TCP port the proxy is listening on (only valid after Start with useTCP)
func (p *GPGProxyAgent) TCPPort() int {
	return p.tcpPort
}

// Start begins listening on the proxy socket
func (p *GPGProxyAgent) Start() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.running {
		return nil
	}

	var listener net.Listener
	if p.useTCP {
		// TCP mode: listen on all interfaces so podman VM can reach us
		l, err := net.Listen("tcp", "0.0.0.0:0")
		if err != nil {
			return fmt.Errorf("failed to listen on TCP: %w", err)
		}
		p.tcpPort = l.Addr().(*net.TCPAddr).Port
		listener = l
	} else {
		// Remove existing socket if present
		os.Remove(p.proxySocket)

		l, err := net.Listen("unix", p.proxySocket)
		if err != nil {
			return fmt.Errorf("failed to listen on proxy socket: %w", err)
		}

		// Set socket permissions
		if err := os.Chmod(p.proxySocket, 0600); err != nil {
			l.Close()
			return fmt.Errorf("failed to set socket permissions: %w", err)
		}
		listener = l
	}

	p.listener = listener
	p.running = true

	p.wg.Add(1)
	go p.acceptLoop()

	return nil
}

// Stop stops the proxy agent
func (p *GPGProxyAgent) Stop() error {
	p.mu.Lock()
	if !p.running {
		p.mu.Unlock()
		return nil
	}
	p.running = false
	p.mu.Unlock()

	if p.listener != nil {
		p.listener.Close()
	}

	p.wg.Wait()

	// Clean up socket directory (Unix socket mode only)
	if !p.useTCP && p.proxySocket != "" {
		socketDir := filepath.Dir(p.proxySocket)
		os.RemoveAll(socketDir)
	}

	return nil
}

// SocketPath returns the path to the proxy socket
func (p *GPGProxyAgent) SocketPath() string {
	return p.proxySocket
}

// SocketDir returns the directory containing the proxy socket
func (p *GPGProxyAgent) SocketDir() string {
	return filepath.Dir(p.proxySocket)
}

func (p *GPGProxyAgent) acceptLoop() {
	defer p.wg.Done()

	for {
		conn, err := p.listener.Accept()
		if err != nil {
			p.mu.Lock()
			running := p.running
			p.mu.Unlock()
			if !running {
				return
			}
			continue
		}

		p.wg.Add(1)
		go p.handleConnection(conn)
	}
}

func (p *GPGProxyAgent) handleConnection(clientConn net.Conn) {
	defer p.wg.Done()
	defer clientConn.Close()

	// Connect to upstream gpg-agent
	upstreamConn, err := net.Dial("unix", p.upstreamSocket)
	if err != nil {
		return
	}
	defer upstreamConn.Close()

	// If no filtering, just proxy everything
	if len(p.allowedKeyIDs) == 0 {
		go io.Copy(upstreamConn, clientConn)
		io.Copy(clientConn, upstreamConn)
		return
	}

	// With filtering, we need to intercept and check commands
	p.proxyWithFiltering(clientConn, upstreamConn)
}

// proxyWithFiltering intercepts Assuan protocol commands
func (p *GPGProxyAgent) proxyWithFiltering(client, upstream net.Conn) {
	// Read the initial OK from gpg-agent
	upstreamReader := bufio.NewReader(upstream)
	clientWriter := bufio.NewWriter(client)

	// Forward initial greeting
	greeting, err := upstreamReader.ReadString('\n')
	if err != nil {
		return
	}
	clientWriter.WriteString(greeting)
	clientWriter.Flush()

	clientReader := bufio.NewReader(client)
	upstreamWriter := bufio.NewWriter(upstream)

	var currentKeyID string

	for {
		// Read command from client
		line, err := clientReader.ReadString('\n')
		if err != nil {
			return
		}

		cmd := strings.TrimSpace(line)
		upperCmd := strings.ToUpper(cmd)

		// Track SIGKEY/SETKEY commands to know which key is being used
		if strings.HasPrefix(upperCmd, "SIGKEY ") || strings.HasPrefix(upperCmd, "SETKEY ") {
			parts := strings.Fields(cmd)
			if len(parts) >= 2 {
				currentKeyID = strings.ToUpper(parts[1])
			}
		}

		// Check PKSIGN operation
		if strings.HasPrefix(upperCmd, "PKSIGN") {
			if !p.isKeyAllowed(currentKeyID) {
				LogGPGSign(currentKeyID, false, "key not in allowed list")
				clientWriter.WriteString("ERR 67108903 Key not allowed by proxy\n")
				clientWriter.Flush()
				continue
			}
			LogGPGSign(currentKeyID, true, "")
		}

		// Check PKDECRYPT operation
		if strings.HasPrefix(upperCmd, "PKDECRYPT") {
			if !p.isKeyAllowed(currentKeyID) {
				LogGPGDecrypt(currentKeyID, false, "key not in allowed list")
				clientWriter.WriteString("ERR 67108903 Key not allowed by proxy\n")
				clientWriter.Flush()
				continue
			}
			LogGPGDecrypt(currentKeyID, true, "")
		}

		// Forward command to upstream
		upstreamWriter.WriteString(line)
		upstreamWriter.Flush()

		// Read and forward response(s)
		for {
			response, err := upstreamReader.ReadString('\n')
			if err != nil {
				return
			}

			clientWriter.WriteString(response)
			clientWriter.Flush()

			trimmed := strings.TrimSpace(response)
			// Assuan responses end with OK, ERR, or END
			if strings.HasPrefix(trimmed, "OK") ||
				strings.HasPrefix(trimmed, "ERR") ||
				trimmed == "END" {
				break
			}
		}
	}
}

// isKeyAllowed checks if a key ID is in the allowed list
func (p *GPGProxyAgent) isKeyAllowed(keyID string) bool {
	if len(p.allowedKeyIDs) == 0 {
		return true
	}

	keyID = strings.ToUpper(keyID)

	for _, allowed := range p.allowedKeyIDs {
		// Match full fingerprint or suffix (short/long key ID)
		if keyID == allowed ||
			strings.HasSuffix(keyID, allowed) ||
			strings.HasSuffix(allowed, keyID) {
			return true
		}
	}

	return false
}
