package podman

import (
	"os"
	"strings"
	"testing"
)

func TestHandleDockerForwarding_Disabled(t *testing.T) {
	p := &PodmanProvider{}

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
	p := &PodmanProvider{}

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

			// Should set ADDT_DIND=true env var
			if !containsEnv(args, "ADDT_DIND=true") {
				t.Errorf("HandleDockerForwarding(%q) missing ADDT_DIND=true env var", tc.mode)
			}
		})
	}
}

func TestHandleDockerForwarding_Host(t *testing.T) {
	p := &PodmanProvider{}

	// Check if Docker or Podman socket exists
	socketPaths := []string{
		"/var/run/docker.sock",
		"/var/run/podman/podman.sock",
	}
	socketExists := false
	for _, sp := range socketPaths {
		if _, err := os.Stat(sp); err == nil {
			socketExists = true
			break
		}
	}

	args := p.HandleDockerForwarding("host", "test-container")

	if socketExists {
		// Should mount a socket
		hasSocketMount := false
		for i, arg := range args {
			if arg == "-v" && i+1 < len(args) && strings.Contains(args[i+1], "/var/run/docker.sock") {
				hasSocketMount = true
				break
			}
		}
		if !hasSocketMount {
			t.Errorf("HandleDockerForwarding(\"host\") missing socket mount, got %v", args)
		}

		// Should add group memberships
		if !containsArg(args, "--group-add") {
			t.Errorf("HandleDockerForwarding(\"host\") missing --group-add flags")
		}

		// Should NOT have --privileged (only isolated mode has that)
		if containsArg(args, "--privileged") {
			t.Errorf("HandleDockerForwarding(\"host\") should not have --privileged")
		}

		// Should NOT set ADDT_DIND env var (only isolated mode)
		if containsEnv(args, "ADDT_DIND=true") {
			t.Errorf("HandleDockerForwarding(\"host\") should not set ADDT_DIND=true")
		}
	} else {
		// Without Docker/Podman socket, host mode should return empty
		if len(args) != 0 {
			t.Errorf("HandleDockerForwarding(\"host\") without socket returned %v, want empty", args)
		}
	}
}

func TestHandleDockerForwarding_IsolatedVolumeNaming(t *testing.T) {
	p := &PodmanProvider{}

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
