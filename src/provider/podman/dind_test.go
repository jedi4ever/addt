package podman

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestHandlePodmanForwarding_Disabled(t *testing.T) {
	p := &PodmanProvider{}

	testCases := []string{"", "off", "false", "none"}

	for _, mode := range testCases {
		t.Run(mode, func(t *testing.T) {
			args := p.HandlePodmanForwarding(mode, "test-container")
			if len(args) != 0 {
				t.Errorf("HandlePodmanForwarding(%q) returned %v, want empty", mode, args)
			}
		})
	}
}

func TestHandlePodmanForwarding_Isolated(t *testing.T) {
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
			args := p.HandlePodmanForwarding(tc.mode, tc.containerName)

			// Should include --privileged
			if !containsArg(args, "--privileged") {
				t.Errorf("HandlePodmanForwarding(%q) missing --privileged flag", tc.mode)
			}

			// Should include volume for Podman container storage
			expectedVolume := "addt-podman-" + tc.containerName + ":/home/addt/.local/share/containers"
			if !containsVolume(args, expectedVolume) {
				t.Errorf("HandlePodmanForwarding(%q) missing volume %q, got %v", tc.mode, expectedVolume, args)
			}

			// Should set ADDT_DOCKER_DIND_ENABLE=true env var
			if !containsEnv(args, "ADDT_DOCKER_DIND_ENABLE=true") {
				t.Errorf("HandlePodmanForwarding(%q) missing ADDT_DOCKER_DIND_ENABLE=true env var", tc.mode)
			}

			// Should NOT have --device /dev/fuse (--privileged subsumes it)
			if containsArg(args, "--device") {
				t.Errorf("HandlePodmanForwarding(%q) should not have --device (--privileged subsumes it)", tc.mode)
			}

			// Should NOT have --security-opt label=disable (--privileged subsumes it)
			if containsArg(args, "--security-opt") {
				t.Errorf("HandlePodmanForwarding(%q) should not have --security-opt (--privileged subsumes it)", tc.mode)
			}
		})
	}
}

func TestHandlePodmanForwarding_Host(t *testing.T) {
	p := &PodmanProvider{}

	args := p.HandlePodmanForwarding("host", "test-container")

	// Host mode depends on whether the Podman socket exists
	podmanSocket := os.Getenv("XDG_RUNTIME_DIR")
	if podmanSocket == "" {
		podmanSocket = fmt.Sprintf("/run/user/%d", os.Getuid())
	}
	socketPath := filepath.Join(podmanSocket, "podman", "podman.sock")

	if _, err := os.Stat(socketPath); err == nil {
		// Socket exists: should mount it
		expectedMount := socketPath + ":/run/podman/podman.sock"
		if !containsVolume(args, expectedMount) {
			t.Errorf("HandlePodmanForwarding(\"host\") missing socket mount %q, got %v", expectedMount, args)
		}

		// Should set DOCKER_HOST env
		if !containsEnvPrefix(args, "DOCKER_HOST=") {
			t.Errorf("HandlePodmanForwarding(\"host\") missing DOCKER_HOST env var")
		}

		// Should NOT have --privileged (only isolated mode has that)
		if containsArg(args, "--privileged") {
			t.Errorf("HandlePodmanForwarding(\"host\") should not have --privileged")
		}
	} else {
		// Without Podman socket, host mode should return empty
		if len(args) != 0 {
			t.Errorf("HandlePodmanForwarding(\"host\") without Podman socket returned %v, want empty", args)
		}
	}
}

func TestHandlePodmanForwarding_IsolatedVolumeNaming(t *testing.T) {
	p := &PodmanProvider{}

	// Test that different container names get different volumes
	containers := []string{"app-1", "app-2", "my-special-container"}

	for _, name := range containers {
		args := p.HandlePodmanForwarding("isolated", name)

		expectedVolume := "addt-podman-" + name + ":/home/addt/.local/share/containers"
		if !containsVolume(args, expectedVolume) {
			t.Errorf("Container %q should have volume %q, got %v", name, expectedVolume, args)
		}
	}
}

// containsArg checks if a specific argument appears in the args slice
func containsArg(args []string, target string) bool {
	for _, arg := range args {
		if arg == target {
			return true
		}
	}
	return false
}

// containsVolume, containsEnv, containsEnvPrefix are defined in ssh_test.go
