package bwrap

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/jedi4ever/addt/config/security"
)

// setupNetworkProxy creates a domain-filtering HTTP proxy for bwrap.
// Uses --unshare-net for total isolation, then bridges allowed traffic
// through a Unix socket + socat, similar to Claude Code's sandbox-runtime.
//
// Returns the proxy (caller must defer Stop) and wrapper directory path.
func (b *BwrapProvider) setupNetworkProxy() (*NetworkProxy, string, error) {
	if !b.config.FirewallEnabled {
		return nil, "", nil
	}

	// socat is required to bridge TCP↔Unix socket across the network namespace
	if _, err := exec.LookPath("socat"); err != nil {
		return nil, "", fmt.Errorf("socat is required for bwrap firewall support.\n" +
			"Install it with:\n" +
			"  Ubuntu/Debian: sudo apt install socat\n" +
			"  Fedora:        sudo dnf install socat\n" +
			"  Arch:          sudo pacman -S socat")
	}

	allowed := mergeUniqueDomains(
		b.config.GlobalFirewallAllowed,
		b.config.ProjectFirewallAllowed,
		b.config.ExtensionFirewallAllowed,
	)
	denied := mergeUniqueDomains(
		b.config.GlobalFirewallDenied,
		b.config.ProjectFirewallDenied,
		b.config.ExtensionFirewallDenied,
	)
	mode := b.config.FirewallMode
	if mode == "" {
		mode = "strict"
	}

	proxy, err := NewNetworkProxy(allowed, denied, mode)
	if err != nil {
		return nil, "", err
	}
	if err := security.WritePIDFile(proxy.SocketDir()); err != nil {
		proxy.Stop()
		return nil, "", err
	}
	b.tempDirs = append(b.tempDirs, proxy.SocketDir())

	if err := proxy.Start(); err != nil {
		proxy.Stop()
		return nil, "", fmt.Errorf("failed to start network proxy: %w", err)
	}

	// Create wrapper script that starts socat inside the sandbox
	wrapperDir, err := os.MkdirTemp("", "bwrap-net-*")
	if err != nil {
		proxy.Stop()
		return nil, "", err
	}
	if err := os.Chmod(wrapperDir, 0755); err != nil {
		os.RemoveAll(wrapperDir)
		proxy.Stop()
		return nil, "", err
	}
	b.tempDirs = append(b.tempDirs, wrapperDir)

	script := `#!/bin/bash
# Bridge TCP:3128 inside sandbox to Unix socket on host via socat
socat TCP-LISTEN:3128,fork,reuseaddr UNIX-CONNECT:/run/addt-proxy/proxy.sock &
SOCAT_PID=$!
sleep 0.1

# Route all HTTP(S) traffic through the filtering proxy
export HTTP_PROXY="http://127.0.0.1:3128"
export HTTPS_PROXY="http://127.0.0.1:3128"
export http_proxy="http://127.0.0.1:3128"
export https_proxy="http://127.0.0.1:3128"

cleanup() { kill $SOCAT_PID 2>/dev/null; }
trap cleanup EXIT
exec "$@"
`
	wrapperPath := filepath.Join(wrapperDir, "net-proxy.sh")
	if err := os.WriteFile(wrapperPath, []byte(script), 0755); err != nil {
		proxy.Stop()
		return nil, "", fmt.Errorf("failed to write proxy wrapper: %w", err)
	}

	fmt.Printf("Firewall: proxy active (mode=%s, allowed=%d domains, denied=%d domains)\n",
		mode, len(allowed), len(denied))
	return proxy, wrapperDir, nil
}

// addNetworkProxyArgs adds bwrap bind-mounts for the proxy socket and wrapper.
func (b *BwrapProvider) addNetworkProxyArgs(args []string, proxy *NetworkProxy, wrapperDir string) []string {
	if proxy == nil {
		return args
	}
	args = append(args, "--bind", proxy.SocketDir(), "/run/addt-proxy")
	args = append(args, "--ro-bind", wrapperDir, "/run/addt-net")
	return args
}

// mergeUniqueDomains deduplicates and combines multiple domain lists.
func mergeUniqueDomains(lists ...[]string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, list := range lists {
		for _, d := range list {
			if !seen[d] {
				seen[d] = true
				result = append(result, d)
			}
		}
	}
	return result
}
