package core

import (
	"net"
	"strings"
	"testing"

	"github.com/jedi4ever/addt/provider"
)

func TestIsPortAvailable_Available(t *testing.T) {
	// Use a high port that's unlikely to be in use
	port := 59123

	if !IsPortAvailable(port) {
		t.Skip("Port 59123 is in use, skipping test")
	}

	if !IsPortAvailable(port) {
		t.Errorf("IsPortAvailable(%d) = false, want true", port)
	}
}

func TestIsPortAvailable_InUse(t *testing.T) {
	// Start a listener on a port
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}
	defer listener.Close()

	// Get the port that was assigned
	port := listener.Addr().(*net.TCPAddr).Port

	if IsPortAvailable(port) {
		t.Errorf("IsPortAvailable(%d) = true, want false (port is in use)", port)
	}
}

func TestFindAvailablePort_FromStart(t *testing.T) {
	startPort := 59200

	port := FindAvailablePort(startPort)

	if port < startPort {
		t.Errorf("FindAvailablePort(%d) = %d, want >= %d", startPort, port, startPort)
	}

	// The returned port should be available
	if !IsPortAvailable(port) {
		t.Errorf("FindAvailablePort returned %d but it's not available", port)
	}
}

func TestFindAvailablePort_SkipsInUse(t *testing.T) {
	// Start a listener on a specific port
	listener, err := net.Listen("tcp", "localhost:59300")
	if err != nil {
		t.Skip("Could not bind to port 59300, skipping test")
	}
	defer listener.Close()

	// FindAvailablePort should skip the in-use port
	port := FindAvailablePort(59300)

	if port == 59300 {
		t.Errorf("FindAvailablePort(59300) = 59300, but that port is in use")
	}

	if port < 59300 {
		t.Errorf("FindAvailablePort(59300) = %d, want >= 59300", port)
	}
}

func TestBuildPorts_Empty(t *testing.T) {
	cfg := &provider.Config{
		Ports:          []string{},
		PortRangeStart: 30000,
	}

	ports := BuildPorts(cfg)

	if len(ports) != 0 {
		t.Errorf("Expected 0 ports, got %d", len(ports))
	}
}

func TestBuildPorts_Single(t *testing.T) {
	cfg := &provider.Config{
		Ports:          []string{"3000"},
		PortRangeStart: 30000,
	}

	ports := BuildPorts(cfg)

	if len(ports) != 1 {
		t.Fatalf("Expected 1 port, got %d", len(ports))
	}

	if ports[0].Container != 3000 {
		t.Errorf("Container port = %d, want 3000", ports[0].Container)
	}

	if ports[0].Host < 30000 {
		t.Errorf("Host port = %d, want >= 30000", ports[0].Host)
	}
}

func TestBuildPorts_Multiple(t *testing.T) {
	cfg := &provider.Config{
		Ports:          []string{"3000", "8080", "5432"},
		PortRangeStart: 30000,
	}

	ports := BuildPorts(cfg)

	if len(ports) != 3 {
		t.Fatalf("Expected 3 ports, got %d", len(ports))
	}

	expectedContainerPorts := []int{3000, 8080, 5432}
	for i, expectedPort := range expectedContainerPorts {
		if ports[i].Container != expectedPort {
			t.Errorf("Port %d: Container = %d, want %d", i, ports[i].Container, expectedPort)
		}
	}

	// Host ports should be unique and >= 30000
	usedPorts := make(map[int]bool)
	for i, port := range ports {
		if port.Host < 30000 {
			t.Errorf("Port %d: Host = %d, want >= 30000", i, port.Host)
		}
		if usedPorts[port.Host] {
			t.Errorf("Port %d: Host port %d is duplicated", i, port.Host)
		}
		usedPorts[port.Host] = true
	}
}

func TestBuildPorts_WhitespaceHandling(t *testing.T) {
	cfg := &provider.Config{
		Ports:          []string{" 3000 ", "8080", " 5432"},
		PortRangeStart: 30000,
	}

	ports := BuildPorts(cfg)

	if len(ports) != 3 {
		t.Fatalf("Expected 3 ports, got %d", len(ports))
	}

	if ports[0].Container != 3000 {
		t.Errorf("Port 0: Container = %d, want 3000 (whitespace not trimmed)", ports[0].Container)
	}
}

func TestBuildPortMapString_Empty(t *testing.T) {
	cfg := &provider.Config{
		Ports:          []string{},
		PortRangeStart: 30000,
	}

	portMap := BuildPortMapString(cfg)

	if portMap != "" {
		t.Errorf("Expected empty string, got %q", portMap)
	}
}

func TestBuildPortMapString_Single(t *testing.T) {
	cfg := &provider.Config{
		Ports:          []string{"3000"},
		PortRangeStart: 30000,
	}

	portMap := BuildPortMapString(cfg)

	if !strings.HasPrefix(portMap, "3000:") {
		t.Errorf("Port map = %q, want prefix '3000:'", portMap)
	}
}

func TestBuildPortMapString_Multiple(t *testing.T) {
	cfg := &provider.Config{
		Ports:          []string{"3000", "8080"},
		PortRangeStart: 30000,
	}

	portMap := BuildPortMapString(cfg)

	parts := strings.Split(portMap, ",")
	if len(parts) != 2 {
		t.Errorf("Expected 2 port mappings, got %d: %q", len(parts), portMap)
	}

	if !strings.HasPrefix(parts[0], "3000:") {
		t.Errorf("First mapping = %q, want prefix '3000:'", parts[0])
	}

	if !strings.HasPrefix(parts[1], "8080:") {
		t.Errorf("Second mapping = %q, want prefix '8080:'", parts[1])
	}
}

func TestBuildPortMapString_Format(t *testing.T) {
	cfg := &provider.Config{
		Ports:          []string{"3000", "8080", "5432"},
		PortRangeStart: 30000,
	}

	portMap := BuildPortMapString(cfg)

	// Verify format: "containerPort:hostPort,containerPort:hostPort,..."
	parts := strings.Split(portMap, ",")
	for i, part := range parts {
		mapping := strings.Split(part, ":")
		if len(mapping) != 2 {
			t.Errorf("Mapping %d = %q, expected format 'container:host'", i, part)
		}
	}

	t.Logf("ADDT_PORT_MAP = %q", portMap)
}

func TestBuildPortDisplayString_Empty(t *testing.T) {
	cfg := &provider.Config{
		Ports:          []string{},
		PortRangeStart: 30000,
	}

	display := BuildPortDisplayString(cfg)

	if display != "" {
		t.Errorf("Expected empty string, got %q", display)
	}
}

func TestBuildPortDisplayString_Format(t *testing.T) {
	cfg := &provider.Config{
		Ports:          []string{"3000", "8080"},
		PortRangeStart: 30000,
	}

	display := BuildPortDisplayString(cfg)

	// Should use → instead of :
	if !strings.Contains(display, "→") {
		t.Errorf("Display string should contain '→', got %q", display)
	}

	if strings.Contains(display, "3000:") {
		t.Errorf("Display string should use '→' not ':', got %q", display)
	}
}
