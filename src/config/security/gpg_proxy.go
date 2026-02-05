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
}

// NewGPGProxyAgent creates a new GPG proxy agent
// allowedKeyIDs can be full fingerprints or last 8/16 chars (short/long key ID)
func NewGPGProxyAgent(upstreamSocket string, allowedKeyIDs []string) (*GPGProxyAgent, error) {
	// Create temp directory for proxy socket
	tmpDir, err := os.MkdirTemp("", "gpg-proxy-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
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

// Start begins listening on the proxy socket
func (p *GPGProxyAgent) Start() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.running {
		return nil
	}

	// Remove existing socket if present
	os.Remove(p.proxySocket)

	listener, err := net.Listen("unix", p.proxySocket)
	if err != nil {
		return fmt.Errorf("failed to listen on proxy socket: %w", err)
	}

	// Set socket permissions
	if err := os.Chmod(p.proxySocket, 0600); err != nil {
		listener.Close()
		return fmt.Errorf("failed to set socket permissions: %w", err)
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

	// Clean up socket directory
	socketDir := filepath.Dir(p.proxySocket)
	os.RemoveAll(socketDir)

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

		// Check PKSIGN and PKDECRYPT operations
		if strings.HasPrefix(upperCmd, "PKSIGN") || strings.HasPrefix(upperCmd, "PKDECRYPT") {
			if !p.isKeyAllowed(currentKeyID) {
				// Deny the operation
				clientWriter.WriteString("ERR 67108903 Key not allowed by proxy\n")
				clientWriter.Flush()
				continue
			}
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
