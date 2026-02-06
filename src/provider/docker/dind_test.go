package docker

import (
	"os"
	"strings"
	"testing"
)

func TestHandleDockerForwarding_Disabled(t *testing.T) {
	p := &DockerProvider{}

	testCases := []string{"", "off", "false", "none"}

	for _, mode := range testCases {
		t.Run(mode, func(t *testing.T) {
			args := p.HandleDockerForwarding(mode, "test-container")
			if len(args) != 0 {
				t.Errorf("HandleDockerForwarding(%q) returned %v, want empty", mode, args)
			}
		})
	}
}

func TestHandleDockerForwarding_Isolated(t *testing.T) {
	p := &DockerProvider{}

	testCases := []struct {
		mode          string
		containerName string
	}{
		{"isolated", "my-container"},
		{"true", "another-container"},
	}

	for _, tc := range testCases {
		t.Run(tc.mode, func(t *testing.T) {
			args := p.HandleDockerForwarding(tc.mode, tc.containerName)

			// Should include --privileged
			if !containsArg(args, "--privileged") {
				t.Errorf("HandleDockerForwarding(%q) missing --privileged flag", tc.mode)
			}

			// Should include volume for Docker data
			expectedVolume := "addt-docker-" + tc.containerName + ":/var/lib/docker"
			if !containsVolume(args, expectedVolume) {
				t.Errorf("HandleDockerForwarding(%q) missing volume %q, got %v", tc.mode, expectedVolume, args)
			}

			// Should set ADDT_DOCKER_DIND_ENABLE=true env var
			if !containsEnv(args, "ADDT_DOCKER_DIND_ENABLE=true") {
				t.Errorf("HandleDockerForwarding(%q) missing ADDT_DOCKER_DIND_ENABLE=true env var", tc.mode)
			}
		})
	}
}

func TestHandleDockerForwarding_Host(t *testing.T) {
	p := &DockerProvider{}

	// Check if Docker socket exists
	socketPath := "/var/run/docker.sock"
	socketExists := false
	if _, err := os.Stat(socketPath); err == nil {
		socketExists = true
	}

	args := p.HandleDockerForwarding("host", "test-container")

	if socketExists {
		// Should mount the Docker socket
		expectedMount := socketPath + ":" + socketPath
		if !containsVolume(args, expectedMount) {
			t.Errorf("HandleDockerForwarding(\"host\") missing socket mount %q, got %v", expectedMount, args)
		}

		// Should add group memberships
		if !containsArg(args, "--group-add") {
			t.Errorf("HandleDockerForwarding(\"host\") missing --group-add flags")
		}

		// Should NOT have --privileged (only isolated mode has that)
		if containsArg(args, "--privileged") {
			t.Errorf("HandleDockerForwarding(\"host\") should not have --privileged")
		}

		// Should NOT set ADDT_DOCKER_DIND_ENABLE env var (only isolated mode)
		if containsEnv(args, "ADDT_DOCKER_DIND_ENABLE=true") {
			t.Errorf("HandleDockerForwarding(\"host\") should not set ADDT_DOCKER_DIND_ENABLE=true")
		}
	} else {
		// Without Docker socket, host mode should return empty
		if len(args) != 0 {
			t.Errorf("HandleDockerForwarding(\"host\") without Docker socket returned %v, want empty", args)
		}
	}
}

func TestHandleDockerForwarding_IsolatedVolumeNaming(t *testing.T) {
	p := &DockerProvider{}

	// Test that different container names get different volumes
	containers := []string{"app-1", "app-2", "my-special-container"}

	for _, name := range containers {
		args := p.HandleDockerForwarding("isolated", name)

		expectedVolume := "addt-docker-" + name + ":/var/lib/docker"
		if !containsVolume(args, expectedVolume) {
			t.Errorf("Container %q should have volume %q, got %v", name, expectedVolume, args)
		}
	}
}

func TestGetDockerSocketGID(t *testing.T) {
	socketPath := "/var/run/docker.sock"

	// Only test if socket exists
	if _, err := os.Stat(socketPath); err != nil {
		t.Skip("Docker socket not available, skipping GID test")
	}

	gid := getDockerSocketGID(socketPath)

	// GID should be positive (0 indicates failure)
	if gid <= 0 {
		t.Errorf("getDockerSocketGID() = %d, want positive GID", gid)
	}
}

func TestGetDockerGroupArgs(t *testing.T) {
	socketPath := "/var/run/docker.sock"

	// Only test if socket exists
	if _, err := os.Stat(socketPath); err != nil {
		t.Skip("Docker socket not available, skipping group args test")
	}

	args := getDockerGroupArgs(socketPath)

	// Should have at least one --group-add
	if !containsArg(args, "--group-add") {
		t.Errorf("getDockerGroupArgs() missing --group-add, got %v", args)
	}

	// Count --group-add occurrences (should be at least 1)
	count := 0
	for _, arg := range args {
		if arg == "--group-add" {
			count++
		}
	}
	if count < 1 {
		t.Errorf("getDockerGroupArgs() has %d --group-add flags, want at least 1", count)
	}
}

// Helper functions

func containsArg(args []string, target string) bool {
	for _, arg := range args {
		if arg == target {
			return true
		}
	}
	return false
}

func containsVolume(args []string, volume string) bool {
	for i, arg := range args {
		if arg == "-v" && i+1 < len(args) && args[i+1] == volume {
			return true
		}
	}
	return false
}

func containsEnv(args []string, env string) bool {
	for i, arg := range args {
		if arg == "-e" && i+1 < len(args) && args[i+1] == env {
			return true
		}
	}
	return false
}

func containsEnvPrefix(args []string, prefix string) bool {
	for i, arg := range args {
		if arg == "-e" && i+1 < len(args) && strings.HasPrefix(args[i+1], prefix) {
			return true
		}
	}
	return false
}
