package podman

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// HandleFirewallConfig configures firewall for Podman containers using pasta networking.
// Pasta (passt) provides userspace networking with better performance and security than slirp4netns.
// When firewall is enabled, we use pasta's network isolation capabilities.
func (p *PodmanProvider) HandleFirewallConfig(homeDir, username string) []string {
	var args []string

	if !p.config.FirewallEnabled {
		return args
	}

	// Use pasta networking for firewall mode
	// pasta provides userspace networking with configurable isolation
	pastaOpts := buildPastaOptions(p.config.FirewallMode, homeDir)
	if pastaOpts != "" {
		args = append(args, "--network", pastaOpts)
	}

	// Mount firewall config directory for the init script
	firewallConfigDir := filepath.Join(homeDir, ".addt", "firewall")
	if err := ensureFirewallConfigDir(firewallConfigDir); err == nil {
		args = append(args, "-v", fmt.Sprintf("%s:/home/%s/.addt/firewall", firewallConfigDir, username))
	}

	// Still need NET_ADMIN for iptables rules inside container (for domain-based filtering)
	// pasta handles the network namespace, iptables handles domain filtering
	args = append(args, "--cap-add", "NET_ADMIN")

	return args
}

// buildPastaOptions constructs pasta network options based on firewall mode
func buildPastaOptions(firewallMode string, homeDir string) string {
	switch firewallMode {
	case "off", "disabled":
		// No pasta, use default networking
		return ""
	case "permissive":
		// Use pasta but allow all traffic (logging only in container)
		return "pasta"
	case "strict", "":
		// Use pasta with restricted configuration
		return buildStrictPastaOptions(homeDir)
	default:
		return "pasta"
	}
}

// buildStrictPastaOptions creates pasta options for strict firewall mode
func buildStrictPastaOptions(homeDir string) string {
	// Start with base pasta network
	opts := []string{"pasta"}

	// Configure DNS - allow DNS queries (needed for domain resolution)
	// pasta automatically forwards DNS by default

	// Allow specific TCP ports for common development needs
	// These are outbound ports that the container can connect to
	allowedPorts := getAllowedPorts(homeDir)
	if len(allowedPorts) > 0 {
		// pasta -T option: TCP port forwarding for outbound
		for _, port := range allowedPorts {
			opts = append(opts, fmt.Sprintf("-T %s", port))
		}
	}

	// Join options with commas for pasta network specification
	if len(opts) == 1 {
		return "pasta"
	}

	// pasta options are passed after colon
	// Format: pasta:option1,option2,...
	return strings.Join(opts, ":")
}

// getAllowedPorts reads allowed ports from config or returns defaults
func getAllowedPorts(homeDir string) []string {
	// Default ports for common services
	defaults := []string{
		"443",  // HTTPS (APIs, git, etc.)
		"80",   // HTTP
		"22",   // SSH (git over SSH)
		"53",   // DNS
		"5432", // PostgreSQL (common for dev)
		"3306", // MySQL (common for dev)
		"6379", // Redis (common for dev)
	}

	// Try to read custom ports from config
	portsFile := filepath.Join(homeDir, ".addt", "firewall", "allowed-ports.txt")
	if ports, err := readPortsFile(portsFile); err == nil && len(ports) > 0 {
		return ports
	}

	return defaults
}

// readPortsFile reads allowed ports from a file (one per line)
func readPortsFile(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var ports []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "#") {
			ports = append(ports, line)
		}
	}

	return ports, scanner.Err()
}

// ensureFirewallConfigDir creates the firewall config directory if it doesn't exist
func ensureFirewallConfigDir(dir string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return os.MkdirAll(dir, 0755)
	}
	return nil
}

// GetAllowedDomainsFile returns the path to the allowed domains file
func GetAllowedDomainsFile(homeDir string) string {
	return filepath.Join(homeDir, ".addt", "firewall", "allowed-domains.txt")
}

// WriteAllowedDomains writes the combined allowed domains to a file for the container
func WriteAllowedDomains(homeDir string, domains []string) error {
	firewallDir := filepath.Join(homeDir, ".addt", "firewall")
	if err := os.MkdirAll(firewallDir, 0755); err != nil {
		return err
	}

	filename := GetAllowedDomainsFile(homeDir)
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	for _, domain := range domains {
		fmt.Fprintln(file, domain)
	}

	return nil
}
