//go:build integration

package core

import (
	"fmt"
	"net"
	"testing"
	"time"
)

func TestIsPortAvailable_Integration_UnusedPort(t *testing.T) {
	// Find a port that's likely to be unused (high port number)
	port := 59123

	// Make sure it's not in use
	listener, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", port))
	if err != nil {
		t.Skipf("Port %d already in use, skipping test", port)
	}
	listener.Close()

	// Give OS time to release the port
	time.Sleep(100 * time.Millisecond)

	if !IsPortAvailable(port) {
		t.Errorf("Expected port %d to be available", port)
	}
}

func TestIsPortAvailable_Integration_UsedPort(t *testing.T) {
	// Start a listener on a random port
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}
	defer listener.Close()

	// Get the assigned port
	port := listener.Addr().(*net.TCPAddr).Port

	// Port should be in use
	if IsPortAvailable(port) {
		t.Errorf("Expected port %d to be unavailable (in use)", port)
	}
}

func TestFindAvailablePort_Integration_Basic(t *testing.T) {
	startPort := 59200

	port := FindAvailablePort(startPort)

	if port < startPort {
		t.Errorf("Expected port >= %d, got %d", startPort, port)
	}

	// Verify it's actually available
	if !IsPortAvailable(port) {
		t.Errorf("FindAvailablePort returned %d but it's not available", port)
	}
}

func TestFindAvailablePort_Integration_SkipsUsedPorts(t *testing.T) {
	startPort := 59300

	// Occupy the first few ports
	var listeners []net.Listener
	for i := 0; i < 3; i++ {
		listener, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", startPort+i))
		if err != nil {
			t.Skipf("Could not occupy port %d: %v", startPort+i, err)
		}
		listeners = append(listeners, listener)
	}
	defer func() {
		for _, l := range listeners {
			l.Close()
		}
	}()

	// FindAvailablePort should skip the occupied ports
	port := FindAvailablePort(startPort)

	if port < startPort+3 {
		t.Errorf("Expected port >= %d (first 3 occupied), got %d", startPort+3, port)
	}

	// Verify it's actually available
	if !IsPortAvailable(port) {
		t.Errorf("FindAvailablePort returned %d but it's not available", port)
	}
}

func TestFindAvailablePort_Integration_ConsecutiveCalls(t *testing.T) {
	startPort := 59400

	// Find first available port
	port1 := FindAvailablePort(startPort)

	// Occupy it
	listener, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", port1))
	if err != nil {
		t.Fatalf("Failed to occupy port %d: %v", port1, err)
	}
	defer listener.Close()

	// Find next available port
	port2 := FindAvailablePort(startPort)

	if port2 == port1 {
		t.Errorf("Second call returned same port %d which is now occupied", port1)
	}

	if port2 < port1 {
		t.Errorf("Expected port2 >= port1, got port1=%d, port2=%d", port1, port2)
	}
}

func TestFindAvailablePort_Integration_MultiplePortsInUse(t *testing.T) {
	startPort := 59500

	// Occupy ports in a scattered pattern: 0, 2, 4
	var listeners []net.Listener
	for _, offset := range []int{0, 2, 4} {
		listener, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", startPort+offset))
		if err != nil {
			continue // Port might already be in use
		}
		listeners = append(listeners, listener)
	}
	defer func() {
		for _, l := range listeners {
			l.Close()
		}
	}()

	// Find available port starting from startPort
	port := FindAvailablePort(startPort)

	// Should find one of the gaps (1, 3) or after 4
	if !IsPortAvailable(port) {
		t.Errorf("FindAvailablePort returned %d but it's not available", port)
	}

	t.Logf("Found available port: %d (start was %d)", port, startPort)
}

func TestIsPortAvailable_Integration_MultiplePorts(t *testing.T) {
	// Test a range of ports
	testPorts := []int{59600, 59601, 59602, 59603, 59604}

	for _, port := range testPorts {
		available := IsPortAvailable(port)
		t.Logf("Port %d available: %v", port, available)
	}
}

func TestIsPortAvailable_Integration_Timeout(t *testing.T) {
	// Test that checking an unavailable port doesn't hang
	// Use a port that's occupied

	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}
	defer listener.Close()

	port := listener.Addr().(*net.TCPAddr).Port

	start := time.Now()
	IsPortAvailable(port)
	elapsed := time.Since(start)

	// Should complete within reasonable time (the function has 1s timeout)
	if elapsed > 2*time.Second {
		t.Errorf("IsPortAvailable took too long: %v", elapsed)
	}
}

func TestFindAvailablePort_Integration_LargeRange(t *testing.T) {
	// Test finding a port in a larger range
	startPort := 60000

	port := FindAvailablePort(startPort)

	if port < startPort {
		t.Errorf("Expected port >= %d, got %d", startPort, port)
	}

	// Verify we can actually bind to it
	listener, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", port))
	if err != nil {
		t.Errorf("Could not bind to returned port %d: %v", port, err)
	} else {
		listener.Close()
	}
}

func TestIsPortAvailable_Integration_PrivilegedPorts(t *testing.T) {
	// Low ports (< 1024) require root, but we can still check availability
	// These should typically be in use or unavailable

	port := 80 // HTTP

	// Just test that the function doesn't panic/hang
	available := IsPortAvailable(port)
	t.Logf("Port %d (HTTP) available: %v", port, available)
}

func TestFindAvailablePort_Integration_Concurrency(t *testing.T) {
	// Test that concurrent calls work correctly
	startPort := 60100
	results := make(chan int, 10)

	for i := 0; i < 10; i++ {
		go func(offset int) {
			port := FindAvailablePort(startPort + offset*10)
			results <- port
		}(i)
	}

	ports := make(map[int]bool)
	for i := 0; i < 10; i++ {
		port := <-results
		if port == 0 {
			t.Error("FindAvailablePort returned 0")
			continue
		}
		if ports[port] {
			// Same port could be returned if they run at different times
			// This is expected behavior
			t.Logf("Port %d returned multiple times (expected if timing overlaps)", port)
		}
		ports[port] = true
	}

	t.Logf("Found %d unique ports", len(ports))
}
