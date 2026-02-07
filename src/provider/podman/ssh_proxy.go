package podman

import (
	"fmt"
	"os"
	"runtime"

	"github.com/jedi4ever/addt/config/security"
)

// handleSSHProxyForwarding creates a filtered SSH agent proxy
func (p *PodmanProvider) handleSSHProxyForwarding(homeDir, username string, allowedKeys []string) []string {
	var args []string

	sshAuthSock := os.Getenv("SSH_AUTH_SOCK")
	if sshAuthSock == "" {
		fmt.Println("Warning: SSH_AUTH_SOCK not set, cannot create SSH proxy")
		return args
	}

	// On macOS, podman runs in a VM and can't mount Unix sockets via virtiofs.
	// Use TCP mode: proxy listens on TCP, container connects via socat.
	if runtime.GOOS == "darwin" {
		return p.handleSSHProxyForwardingTCP(sshAuthSock, homeDir, username, allowedKeys)
	}

	// Linux: use Unix socket (can be mounted directly)
	proxy, err := security.NewSSHProxyAgent(sshAuthSock, allowedKeys)
	if err != nil {
		fmt.Printf("Warning: failed to create SSH proxy: %v\n", err)
		return args
	}

	if err := proxy.Start(); err != nil {
		fmt.Printf("Warning: failed to start SSH proxy: %v\n", err)
		return args
	}

	p.sshProxy = proxy

	proxySocket := proxy.SocketPath()
	args = append(args, "-v", fmt.Sprintf("%s:/ssh-agent", proxySocket))
	args = append(args, "-e", "SSH_AUTH_SOCK=/ssh-agent")
	args = append(args, p.mountSafeSSHFiles(homeDir, username)...)

	if len(allowedKeys) > 0 {
		fmt.Printf("SSH proxy active: only keys matching %v are accessible\n", allowedKeys)
	} else {
		fmt.Println("SSH proxy active: all keys accessible")
	}

	return args
}

// handleSSHProxyForwardingTCP creates a TCP-based SSH agent proxy for macOS.
// The proxy listens on a TCP port on the host; the container connects via socat.
func (p *PodmanProvider) handleSSHProxyForwardingTCP(sshAuthSock, homeDir, username string, allowedKeys []string) []string {
	var args []string

	proxy, err := security.NewSSHProxyAgentTCP(sshAuthSock, allowedKeys)
	if err != nil {
		fmt.Printf("Warning: failed to create SSH TCP proxy: %v\n", err)
		return args
	}

	if err := proxy.Start(); err != nil {
		fmt.Printf("Warning: failed to start SSH TCP proxy: %v\n", err)
		return args
	}

	p.sshProxy = proxy

	// Get host IP reachable from the container
	hostIP, err := getHostGatewayIP()
	if err != nil {
		fmt.Printf("Warning: could not detect host IP for SSH proxy: %v\n", err)
		proxy.Stop()
		return args
	}

	// Pass connection info as env vars — entrypoint uses socat to bridge TCP→Unix socket
	args = append(args, "-e", fmt.Sprintf("ADDT_SSH_PROXY_HOST=%s", hostIP))
	args = append(args, "-e", fmt.Sprintf("ADDT_SSH_PROXY_PORT=%d", proxy.TCPPort()))

	// Mount safe SSH files only (config, known_hosts, public keys)
	args = append(args, p.mountSafeSSHFiles(homeDir, username)...)

	if len(allowedKeys) > 0 {
		fmt.Printf("SSH proxy active (TCP): only keys matching %v are accessible\n", allowedKeys)
	} else {
		fmt.Println("SSH proxy active (TCP): all keys accessible")
	}

	return args
}
