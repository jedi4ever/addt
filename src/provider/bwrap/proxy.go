package bwrap

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const proxyDialTimeout = 10 * time.Second

// NetworkProxy is an HTTP/CONNECT proxy that filters requests by domain.
// It listens on a Unix socket so it can bridge into a network-isolated
// bwrap sandbox via bind-mount + socat.
//
// Architecture:
//
//	Host: Go proxy listens on Unix socket
//	bwrap: socket dir bind-mounted to /run/addt-proxy/
//	Sandbox: socat TCP-LISTEN:3128 → UNIX-CONNECT:/run/addt-proxy/proxy.sock
//	Sandbox: HTTP_PROXY=http://127.0.0.1:3128
type NetworkProxy struct {
	listener       net.Listener
	socketDir      string
	socketPath     string
	allowedDomains []string
	deniedDomains  []string
	mode           string // "strict" or "permissive"
	server         *http.Server
	mu             sync.Mutex
	running        bool
}

// NewNetworkProxy creates a new domain-filtering proxy.
func NewNetworkProxy(allowedDomains, deniedDomains []string, mode string) (*NetworkProxy, error) {
	socketDir, err := os.MkdirTemp("", "bwrap-proxy-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create proxy socket dir: %w", err)
	}
	if err := os.Chmod(socketDir, 0700); err != nil {
		os.RemoveAll(socketDir)
		return nil, err
	}

	return &NetworkProxy{
		socketDir:      socketDir,
		socketPath:     filepath.Join(socketDir, "proxy.sock"),
		allowedDomains: allowedDomains,
		deniedDomains:  deniedDomains,
		mode:           mode,
	}, nil
}

// Start starts the proxy server on the Unix socket.
func (p *NetworkProxy) Start() error {
	listener, err := net.Listen("unix", p.socketPath)
	if err != nil {
		return fmt.Errorf("failed to listen on proxy socket: %w", err)
	}
	p.listener = listener
	p.server = &http.Server{Handler: p}

	p.mu.Lock()
	p.running = true
	p.mu.Unlock()

	go p.server.Serve(listener)
	bwrapLogger.Debugf("Network proxy started on %s (mode=%s, allowed=%d, denied=%d)",
		p.socketPath, p.mode, len(p.allowedDomains), len(p.deniedDomains))
	return nil
}

// Stop stops the proxy and removes the socket directory.
func (p *NetworkProxy) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.running {
		return
	}
	p.running = false
	if p.server != nil {
		p.server.Close()
	}
	if p.listener != nil {
		p.listener.Close()
	}
	os.RemoveAll(p.socketDir)
}

// SocketDir returns the directory containing the proxy socket.
func (p *NetworkProxy) SocketDir() string { return p.socketDir }

// ServeHTTP handles both CONNECT (HTTPS) and regular HTTP requests.
func (p *NetworkProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodConnect {
		p.handleConnect(w, r)
	} else {
		p.handleHTTP(w, r)
	}
}

func (p *NetworkProxy) handleConnect(w http.ResponseWriter, r *http.Request) {
	host, _, err := net.SplitHostPort(r.Host)
	if err != nil {
		host = r.Host
	}

	if !p.isDomainAllowed(host) {
		bwrapLogger.Debugf("Proxy: blocked CONNECT to %s", host)
		http.Error(w, fmt.Sprintf("Connection blocked by network allowlist: %s", host), http.StatusForbidden)
		return
	}

	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "Hijacking not supported", http.StatusInternalServerError)
		return
	}

	targetConn, err := net.DialTimeout("tcp", r.Host, proxyDialTimeout)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to connect: %v", err), http.StatusBadGateway)
		return
	}

	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		targetConn.Close()
		return
	}

	clientConn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		io.Copy(targetConn, clientConn)
	}()
	go func() {
		defer wg.Done()
		io.Copy(clientConn, targetConn)
	}()
	wg.Wait()
	clientConn.Close()
	targetConn.Close()
}

func (p *NetworkProxy) handleHTTP(w http.ResponseWriter, r *http.Request) {
	host := r.URL.Hostname()
	if host == "" {
		host = r.Host
		if h, _, err := net.SplitHostPort(host); err == nil {
			host = h
		}
	}

	if !p.isDomainAllowed(host) {
		bwrapLogger.Debugf("Proxy: blocked HTTP to %s", host)
		http.Error(w, fmt.Sprintf("Connection blocked by network allowlist: %s", host), http.StatusForbidden)
		return
	}

	resp, err := http.DefaultTransport.RoundTrip(r)
	if err != nil {
		http.Error(w, fmt.Sprintf("Proxy error: %v", err), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	for k, vv := range resp.Header {
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

// isDomainAllowed checks domain against deny/allow lists.
func (p *NetworkProxy) isDomainAllowed(domain string) bool {
	domain = strings.ToLower(strings.TrimSpace(domain))

	// Deny list always wins
	for _, d := range p.deniedDomains {
		if matchDomain(domain, d) {
			return false
		}
	}

	// Permissive mode: allow everything not denied
	if p.mode == "permissive" {
		return true
	}

	// Strict mode: must be in allow list
	for _, a := range p.allowedDomains {
		if matchDomain(domain, a) {
			return true
		}
	}

	// Always allow localhost
	if domain == "localhost" || domain == "127.0.0.1" || domain == "::1" {
		return true
	}

	return false
}

// matchDomain checks if domain matches a pattern (supports *.example.com wildcards).
func matchDomain(domain, pattern string) bool {
	domain = strings.ToLower(domain)
	pattern = strings.ToLower(pattern)

	if domain == pattern {
		return true
	}

	// *.example.com matches foo.example.com and bar.example.com
	if strings.HasPrefix(pattern, "*.") {
		suffix := pattern[1:] // ".example.com"
		if strings.HasSuffix(domain, suffix) {
			return true
		}
	}

	return false
}
