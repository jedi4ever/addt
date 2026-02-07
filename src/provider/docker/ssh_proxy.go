package docker

import (
	"fmt"
	"os"

	"github.com/jedi4ever/addt/config/security"
)

// handleSSHProxyForwarding creates a filtered SSH agent proxy
func (p *DockerProvider) handleSSHProxyForwarding(homeDir, username string, allowedKeys []string) []string {
	var args []string

	sshAuthSock := os.Getenv("SSH_AUTH_SOCK")
	if sshAuthSock == "" {
		fmt.Println("Warning: SSH_AUTH_SOCK not set, cannot create SSH proxy")
		return args
	}

	// Create the proxy agent
	proxy, err := security.NewSSHProxyAgent(sshAuthSock, allowedKeys)
	if err != nil {
		fmt.Printf("Warning: failed to create SSH proxy: %v\n", err)
		return args
	}

	// Start the proxy
	if err := proxy.Start(); err != nil {
		fmt.Printf("Warning: failed to start SSH proxy: %v\n", err)
		return args
	}

	// Store proxy for cleanup
	p.sshProxy = proxy

	// Mount the proxy socket
	proxySocket := proxy.SocketPath()
	args = append(args, "-v", fmt.Sprintf("%s:/ssh-agent", proxySocket))
	args = append(args, "-e", "SSH_AUTH_SOCK=/ssh-agent")

	// Mount safe SSH files only (config, known_hosts, public keys)
	args = append(args, p.mountSafeSSHFiles(homeDir, username)...)

	if len(allowedKeys) > 0 {
		fmt.Printf("SSH proxy active: only keys matching %v are accessible\n", allowedKeys)
	} else {
		fmt.Println("SSH proxy active: all keys accessible")
	}

	return args
}
