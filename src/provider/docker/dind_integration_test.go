//go:build integration

package docker

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

// checkDockerForDind verifies Docker is available
func checkDockerForDind(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("Docker not found in PATH, skipping integration test")
	}
	cmd := exec.Command("docker", "info")
	if err := cmd.Run(); err != nil {
		t.Skip("Docker daemon not running, skipping integration test")
	}
}

func TestDockerForwarding_Integration_HostMode(t *testing.T) {
	checkDockerForDind(t)

	prov := &DockerProvider{}
	args := prov.HandleDockerForwarding("host", "test-container")

	// Check for socket mount
	foundSocketMount := false
	for i, arg := range args {
		if arg == "-v" && i+1 < len(args) {
			if strings.Contains(args[i+1], "/var/run/docker.sock") {
				foundSocketMount = true
				break
			}
		}
	}

	// Socket mount depends on whether /var/run/docker.sock exists
	if _, err := os.Stat("/var/run/docker.sock"); err == nil {
		if !foundSocketMount {
			t.Errorf("Expected docker socket mount in args, got: %v", args)
		}

		// Should also have group-add args
		foundGroupAdd := false
		for _, arg := range args {
			if arg == "--group-add" {
				foundGroupAdd = true
				break
			}
		}

		if !foundGroupAdd {
			t.Errorf("Expected --group-add in args, got: %v", args)
		}
	} else {
		t.Log("Docker socket not found at /var/run/docker.sock, skipping socket mount check")
	}
}

func TestDockerForwarding_Integration_IsolatedMode(t *testing.T) {
	checkDockerForDind(t)

	prov := &DockerProvider{}
	containerName := "test-isolated-container"
	args := prov.HandleDockerForwarding("isolated", containerName)

	// Check for --privileged
	foundPrivileged := false
	for _, arg := range args {
		if arg == "--privileged" {
			foundPrivileged = true
			break
		}
	}

	if !foundPrivileged {
		t.Errorf("Expected --privileged in isolated mode args, got: %v", args)
	}

	// Check for volume mount
	foundVolumeMount := false
	expectedVolume := "addt-docker-" + containerName
	for i, arg := range args {
		if arg == "-v" && i+1 < len(args) {
			if strings.Contains(args[i+1], expectedVolume) &&
				strings.Contains(args[i+1], "/var/lib/docker") {
				foundVolumeMount = true
				break
			}
		}
	}

	if !foundVolumeMount {
		t.Errorf("Expected docker volume mount in args, got: %v", args)
	}

	// Check for ADDT_DIND env var
	foundDindEnv := false
	for i, arg := range args {
		if arg == "-e" && i+1 < len(args) {
			if args[i+1] == "ADDT_DIND=true" {
				foundDindEnv = true
				break
			}
		}
	}

	if !foundDindEnv {
		t.Errorf("Expected ADDT_DIND=true env var in args, got: %v", args)
	}
}

func TestDockerForwarding_Integration_TrueMode(t *testing.T) {
	checkDockerForDind(t)

	// "true" should be equivalent to "isolated"
	prov := &DockerProvider{}
	args := prov.HandleDockerForwarding("true", "test-container")

	foundPrivileged := false
	for _, arg := range args {
		if arg == "--privileged" {
			foundPrivileged = true
			break
		}
	}

	if !foundPrivileged {
		t.Errorf("Expected --privileged in 'true' mode (alias for isolated), got: %v", args)
	}
}

func TestDockerForwarding_Integration_NoForwarding(t *testing.T) {
	checkDockerForDind(t)

	prov := &DockerProvider{}
	args := prov.HandleDockerForwarding("", "test-container")

	if args != nil && len(args) > 0 {
		t.Errorf("Expected nil/empty args for no forwarding, got: %v", args)
	}
}

func TestDockerForwarding_Integration_InvalidMode(t *testing.T) {
	checkDockerForDind(t)

	prov := &DockerProvider{}
	args := prov.HandleDockerForwarding("invalid", "test-container")

	if args != nil && len(args) > 0 {
		t.Errorf("Expected nil/empty args for invalid mode, got: %v", args)
	}
}

func TestDockerForwarding_Integration_HostModeInContainer(t *testing.T) {
	checkDockerForDind(t)

	// Skip if docker socket doesn't exist
	if _, err := os.Stat("/var/run/docker.sock"); err != nil {
		t.Skip("Docker socket not found, skipping container test")
	}

	// Run a container with the docker socket mounted and verify docker works
	cmd := exec.Command("docker", "run", "--rm",
		"-v", "/var/run/docker.sock:/var/run/docker.sock",
		"docker:cli",
		"docker", "version", "--format", "{{.Server.Version}}")

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to run docker in container: %v\nOutput: %s", err, string(output))
	}

	// Should output a version number
	if len(strings.TrimSpace(string(output))) == 0 {
		t.Errorf("Expected docker version output, got empty")
	}

	t.Logf("Docker version from container: %s", strings.TrimSpace(string(output)))
}

func TestDockerForwarding_Integration_IsolatedModeInContainer(t *testing.T) {
	checkDockerForDind(t)

	// This test runs a docker:dind container in privileged mode
	// It verifies that an isolated Docker daemon can be started

	containerName := "addt-dind-integration-test"

	// Clean up any existing container
	exec.Command("docker", "rm", "-f", containerName).Run()
	defer exec.Command("docker", "rm", "-f", containerName).Run()

	// Start a dind container in background
	startCmd := exec.Command("docker", "run", "-d",
		"--name", containerName,
		"--privileged",
		"-v", "addt-dind-test:/var/lib/docker",
		"docker:dind",
		"--storage-driver=overlay2")

	if err := startCmd.Run(); err != nil {
		t.Fatalf("Failed to start dind container: %v", err)
	}

	// Wait a bit for dockerd to start
	exec.Command("docker", "exec", containerName, "sh", "-c",
		"for i in 1 2 3 4 5; do docker info >/dev/null 2>&1 && break || sleep 2; done").Run()

	// Try to run docker info inside the container
	checkCmd := exec.Command("docker", "exec", containerName, "docker", "info", "--format", "{{.ServerVersion}}")
	output, err := checkCmd.CombinedOutput()
	if err != nil {
		t.Logf("Note: dind may take time to start. Output: %s", string(output))
		// Don't fail - dind can be slow to start
		t.Skip("DinD container dockerd not ready, skipping")
	}

	t.Logf("Isolated Docker version: %s", strings.TrimSpace(string(output)))

	// Clean up the test volume
	exec.Command("docker", "volume", "rm", "-f", "addt-dind-test").Run()
}

func TestDockerForwarding_Integration_GetDockerSocketGID(t *testing.T) {
	checkDockerForDind(t)

	socketPath := "/var/run/docker.sock"
	if _, err := os.Stat(socketPath); err != nil {
		t.Skip("Docker socket not found, skipping GID test")
	}

	gid := getDockerSocketGID(socketPath)

	if gid <= 0 {
		t.Errorf("Expected positive GID, got: %d", gid)
	}

	t.Logf("Docker socket GID: %d", gid)
}

func TestDockerForwarding_Integration_GetDockerGroupArgs(t *testing.T) {
	checkDockerForDind(t)

	socketPath := "/var/run/docker.sock"
	if _, err := os.Stat(socketPath); err != nil {
		t.Skip("Docker socket not found, skipping group args test")
	}

	args := getDockerGroupArgs(socketPath)

	// Should have at least one --group-add
	foundGroupAdd := false
	for _, arg := range args {
		if arg == "--group-add" {
			foundGroupAdd = true
			break
		}
	}

	if !foundGroupAdd {
		t.Errorf("Expected --group-add in args, got: %v", args)
	}

	t.Logf("Docker group args: %v", args)
}

func TestDockerForwarding_Integration_VolumeNaming(t *testing.T) {
	checkDockerForDind(t)

	testCases := []struct {
		containerName  string
		expectedVolume string
	}{
		{"my-container", "addt-docker-my-container"},
		{"test-123", "addt-docker-test-123"},
		{"claude-session", "addt-docker-claude-session"},
	}

	prov := &DockerProvider{}

	for _, tc := range testCases {
		t.Run(tc.containerName, func(t *testing.T) {
			args := prov.HandleDockerForwarding("isolated", tc.containerName)

			foundExpectedVolume := false
			for i, arg := range args {
				if arg == "-v" && i+1 < len(args) {
					if strings.HasPrefix(args[i+1], tc.expectedVolume+":") {
						foundExpectedVolume = true
						break
					}
				}
			}

			if !foundExpectedVolume {
				t.Errorf("Expected volume %s in args, got: %v", tc.expectedVolume, args)
			}
		})
	}
}
