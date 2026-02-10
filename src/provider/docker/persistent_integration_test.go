//go:build integration

package docker

import (
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/jedi4ever/addt/provider"
)

// checkDockerForPersistent verifies Docker is available
func checkDockerForPersistent(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping container test in short mode")
	}
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("Docker not found in PATH, skipping integration test")
	}
	cmd := provider.DockerCmd("default", "info")
	if err := cmd.Run(); err != nil {
		t.Skip("Docker daemon not running, skipping integration test")
	}
}

// createPersistentTestProvider creates a DockerProvider for persistent tests
func createPersistentTestProvider(workdir, extensions string) *DockerProvider {
	return &DockerProvider{
		config: &provider.Config{
			Workdir:    workdir,
			Extensions: extensions,
		},
		tempDirs: []string{},
	}
}

// cleanupContainer removes a container if it exists
func cleanupContainer(name string) {
	provider.DockerCmd("default", "rm", "-f", name).Run()
}

func TestPersistent_Integration_GenerateContainerName(t *testing.T) {
	checkDockerForPersistent(t)

	testCases := []struct {
		name       string
		workdir    string
		extensions string
		wantPrefix string
	}{
		{
			name:       "simple workdir",
			workdir:    "/home/user/myproject",
			extensions: "claude",
			wantPrefix: "addt-persistent-myproject-",
		},
		{
			name:       "workdir with special chars",
			workdir:    "/home/user/My Project!",
			extensions: "claude",
			wantPrefix: "addt-persistent-my-project-",
		},
		{
			name:       "long workdir name",
			workdir:    "/home/user/this-is-a-very-long-directory-name-that-should-be-truncated",
			extensions: "claude",
			wantPrefix: "addt-persistent-this-is-a-very-long-", // Truncated to 20 chars
		},
		{
			name:       "multiple extensions",
			workdir:    "/home/user/project",
			extensions: "claude,codex",
			wantPrefix: "addt-persistent-project-",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			prov := createPersistentTestProvider(tc.workdir, tc.extensions)
			name := prov.GenerateContainerName()

			if !strings.HasPrefix(name, tc.wantPrefix) {
				t.Errorf("GenerateContainerName() = %q, want prefix %q", name, tc.wantPrefix)
			}

			// Name should be consistent
			name2 := prov.GenerateContainerName()
			if name != name2 {
				t.Errorf("GenerateContainerName() not consistent: %q != %q", name, name2)
			}
		})
	}
}

func TestPersistent_Integration_DifferentExtensionsDifferentNames(t *testing.T) {
	checkDockerForPersistent(t)

	workdir := "/home/user/testproject"

	prov1 := createPersistentTestProvider(workdir, "claude")
	prov2 := createPersistentTestProvider(workdir, "codex")
	prov3 := createPersistentTestProvider(workdir, "claude,codex")

	name1 := prov1.GenerateContainerName()
	name2 := prov2.GenerateContainerName()
	name3 := prov3.GenerateContainerName()

	// All names should be different due to different extensions
	if name1 == name2 {
		t.Errorf("Same name for different extensions: %q", name1)
	}
	if name1 == name3 {
		t.Errorf("Same name for different extensions: %q", name1)
	}
	if name2 == name3 {
		t.Errorf("Same name for different extensions: %q", name2)
	}
}

func TestPersistent_Integration_ExtensionOrderDoesNotMatter(t *testing.T) {
	checkDockerForPersistent(t)

	workdir := "/home/user/testproject"

	prov1 := createPersistentTestProvider(workdir, "claude,codex")
	prov2 := createPersistentTestProvider(workdir, "codex,claude")

	name1 := prov1.GenerateContainerName()
	name2 := prov2.GenerateContainerName()

	// Names should be the same regardless of extension order
	if name1 != name2 {
		t.Errorf("Extension order matters: %q != %q", name1, name2)
	}
}

func TestPersistent_Integration_GenerateEphemeralName(t *testing.T) {
	checkDockerForPersistent(t)

	prov := createPersistentTestProvider("/tmp", "claude")

	name1 := prov.GenerateEphemeralName()
	time.Sleep(time.Second) // Ensure different timestamp
	name2 := prov.GenerateEphemeralName()

	if !strings.HasPrefix(name1, "addt-") {
		t.Errorf("Ephemeral name should start with 'addt-': %q", name1)
	}

	if strings.HasPrefix(name1, "addt-persistent-") {
		t.Errorf("Ephemeral name should not be persistent: %q", name1)
	}

	// Names should be different due to different timestamps
	if name1 == name2 {
		t.Errorf("Ephemeral names should be unique: %q == %q", name1, name2)
	}
}

func TestPersistent_Integration_ContainerLifecycle(t *testing.T) {
	checkDockerForPersistent(t)

	containerName := "addt-persistent-test-lifecycle"
	cleanupContainer(containerName)
	defer cleanupContainer(containerName)

	prov := &DockerProvider{}

	// Initially should not exist
	if prov.Exists(containerName) {
		t.Fatal("Container should not exist initially")
	}

	if prov.IsRunning(containerName) {
		t.Fatal("Container should not be running initially")
	}

	// Create a container
	cmd := provider.DockerCmd("default", "run", "-d", "--name", containerName, "alpine:latest", "sleep", "60")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to create container: %v", err)
	}

	// Should exist and be running
	if !prov.Exists(containerName) {
		t.Error("Container should exist after creation")
	}

	if !prov.IsRunning(containerName) {
		t.Error("Container should be running after creation")
	}

	// Stop the container
	if err := prov.Stop(containerName); err != nil {
		t.Fatalf("Failed to stop container: %v", err)
	}

	// Should exist but not be running
	if !prov.Exists(containerName) {
		t.Error("Container should still exist after stop")
	}

	if prov.IsRunning(containerName) {
		t.Error("Container should not be running after stop")
	}

	// Start the container
	if err := prov.Start(containerName); err != nil {
		t.Fatalf("Failed to start container: %v", err)
	}

	// Should be running again
	if !prov.IsRunning(containerName) {
		t.Error("Container should be running after start")
	}

	// Remove the container
	if err := prov.Remove(containerName); err != nil {
		t.Fatalf("Failed to remove container: %v", err)
	}

	// Should not exist
	if prov.Exists(containerName) {
		t.Error("Container should not exist after removal")
	}
}

func TestPersistent_Integration_ListContainers(t *testing.T) {
	checkDockerForPersistent(t)

	// Create a few test persistent containers
	testContainers := []string{
		"addt-persistent-test-list-1",
		"addt-persistent-test-list-2",
	}

	for _, name := range testContainers {
		cleanupContainer(name)
	}
	defer func() {
		for _, name := range testContainers {
			cleanupContainer(name)
		}
	}()

	// Create containers
	for _, name := range testContainers {
		cmd := provider.DockerCmd("default", "run", "-d", "--name", name, "alpine:latest", "sleep", "60")
		if err := cmd.Run(); err != nil {
			t.Fatalf("Failed to create container %s: %v", name, err)
		}
	}

	prov := &DockerProvider{}
	envs, err := prov.List()
	if err != nil {
		t.Fatalf("List() failed: %v", err)
	}

	// Check our test containers are in the list
	foundContainers := make(map[string]bool)
	for _, env := range envs {
		foundContainers[env.Name] = true
	}

	for _, name := range testContainers {
		if !foundContainers[name] {
			t.Errorf("Container %s not found in list", name)
		}
	}
}

func TestPersistent_Integration_IsPersistentContainer(t *testing.T) {
	testCases := []struct {
		name string
		want bool
	}{
		{"addt-persistent-myproject-abc123", true},
		{"addt-persistent-test-xyz", true},
		{"addt-20240101-123456-1234", false},
		{"some-other-container", false},
		{"addt-", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := IsPersistentContainer(tc.name)
			if got != tc.want {
				t.Errorf("IsPersistentContainer(%q) = %v, want %v", tc.name, got, tc.want)
			}
		})
	}
}

func TestPersistent_Integration_IsEphemeralContainer(t *testing.T) {
	testCases := []struct {
		name string
		want bool
	}{
		{"addt-20240101-123456-1234", true},
		{"addt-test-123", true},
		{"addt-persistent-myproject-abc123", false},
		{"some-other-container", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := IsEphemeralContainer(tc.name)
			if got != tc.want {
				t.Errorf("IsEphemeralContainer(%q) = %v, want %v", tc.name, got, tc.want)
			}
		})
	}
}

func TestPersistent_Integration_GetContainerWorkdir(t *testing.T) {
	testCases := []struct {
		name string
		want string
	}{
		{"addt-persistent-myproject-abc123", "myproject"},
		{"addt-persistent-test-xyz", "test"},
		{"addt-persistent-my-long-name-abc", "my-long-name"},
		{"addt-20240101-123456-1234", ""},
		{"some-other-container", ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := GetContainerWorkdir(tc.name)
			if got != tc.want {
				t.Errorf("GetContainerWorkdir(%q) = %q, want %q", tc.name, got, tc.want)
			}
		})
	}
}

func TestPersistent_Integration_ReuseExistingContainer(t *testing.T) {
	checkDockerForPersistent(t)

	containerName := "addt-persistent-test-reuse"
	cleanupContainer(containerName)
	defer cleanupContainer(containerName)

	// Create a container with a marker file
	cmd := provider.DockerCmd("default", "run", "-d", "--name", containerName,
		"alpine:latest", "sh", "-c", "echo 'marker' > /tmp/marker && sleep 60")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to create container: %v", err)
	}

	// Verify marker file exists
	checkCmd := provider.DockerCmd("default", "exec", containerName, "cat", "/tmp/marker")
	output, err := checkCmd.Output()
	if err != nil {
		t.Fatalf("Failed to check marker file: %v", err)
	}

	if !strings.Contains(string(output), "marker") {
		t.Error("Marker file not found in container")
	}

	// Stop the container
	prov := &DockerProvider{}
	if err := prov.Stop(containerName); err != nil {
		t.Fatalf("Failed to stop container: %v", err)
	}

	// Start it again
	if err := prov.Start(containerName); err != nil {
		t.Fatalf("Failed to start container: %v", err)
	}

	// Marker file should still exist (persistent state)
	checkCmd = provider.DockerCmd("default", "exec", containerName, "cat", "/tmp/marker")
	output, err = checkCmd.Output()
	if err != nil {
		t.Fatalf("Failed to check marker file after restart: %v", err)
	}

	if !strings.Contains(string(output), "marker") {
		t.Error("Marker file lost after container restart - state not persisted")
	}
}

func TestPersistent_Integration_WorkdirFromEnv(t *testing.T) {
	checkDockerForPersistent(t)

	// Test with empty workdir (should use current directory)
	prov := createPersistentTestProvider("", "claude")

	name := prov.GenerateContainerName()

	// Should generate a name based on current directory
	if !strings.HasPrefix(name, "addt-persistent-") {
		t.Errorf("Name should start with 'addt-persistent-': %q", name)
	}

	// Get current directory name for comparison
	cwd, _ := os.Getwd()
	t.Logf("Generated name: %s (from cwd: %s)", name, cwd)
}

func TestPersistent_Integration_GeneratePersistentNameAlias(t *testing.T) {
	checkDockerForPersistent(t)

	prov := createPersistentTestProvider("/tmp/test", "claude")

	// GeneratePersistentName should be an alias for GenerateContainerName
	name1 := prov.GenerateContainerName()
	name2 := prov.GeneratePersistentName()

	if name1 != name2 {
		t.Errorf("GeneratePersistentName() should equal GenerateContainerName(): %q != %q", name1, name2)
	}
}
