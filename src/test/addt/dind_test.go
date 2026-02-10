//go:build addt

package addt

import (
	"os"
	"strings"
	"testing"

	configcmd "github.com/jedi4ever/addt/cmd/config"
)

// --- Config tests (in-process, no container needed) ---

func TestDind_Addt_ConfigLoaded(t *testing.T) {
	// Scenario: User sets docker.dind.mode to "isolated" in the project config,
	// then checks the config list to verify it appears with source=project.
	_, cleanup := setupAddtDir(t, "", `
docker:
  dind:
    mode: "isolated"
`)
	defer cleanup()

	output := captureOutput(t, func() {
		configcmd.HandleCommand([]string{"list"})
	})

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "docker.dind.mode") {
			if !strings.Contains(line, "isolated") {
				t.Errorf("Expected docker.dind.mode=isolated, got line: %s", line)
			}
			if !strings.Contains(line, "project") {
				t.Errorf("Expected docker.dind.mode source=project, got line: %s", line)
			}
			return
		}
	}
	t.Errorf("Expected docker.dind.mode in config list, got:\n%s", output)
}

func TestDind_Addt_DefaultValues(t *testing.T) {
	// Scenario: User starts with no DinD config and checks the defaults.
	// docker.dind.enable should default to false, docker.dind.mode to "isolated".
	_, cleanup := setupAddtDir(t, "", ``)
	defer cleanup()

	output := captureOutput(t, func() {
		configcmd.HandleCommand([]string{"list"})
	})

	lines := strings.Split(output, "\n")

	// Check docker.dind.enable defaults to false
	foundEnable := false
	for _, line := range lines {
		if strings.Contains(line, "docker.dind.enable") {
			foundEnable = true
			if !strings.Contains(line, "false") {
				t.Errorf("Expected docker.dind.enable default=false, got line: %s", line)
			}
			if !strings.Contains(line, "default") {
				t.Errorf("Expected docker.dind.enable source=default, got line: %s", line)
			}
		}
	}
	if !foundEnable {
		t.Errorf("Expected docker.dind.enable in config list, got:\n%s", output)
	}

	// Check docker.dind.mode defaults to "isolated"
	foundMode := false
	for _, line := range lines {
		if strings.Contains(line, "docker.dind.mode") {
			foundMode = true
			if !strings.Contains(line, "isolated") {
				t.Errorf("Expected docker.dind.mode default=isolated, got line: %s", line)
			}
			if !strings.Contains(line, "default") {
				t.Errorf("Expected docker.dind.mode source=default, got line: %s", line)
			}
		}
	}
	if !foundMode {
		t.Errorf("Expected docker.dind.mode in config list, got:\n%s", output)
	}
}

func TestDind_Addt_ConfigSetGet(t *testing.T) {
	// Scenario: User sets docker.dind.mode to "host" via the config set command,
	// then reads it back with config list to verify the change.
	_, cleanup := setupAddtDir(t, "", ``)
	defer cleanup()

	captureOutput(t, func() {
		configcmd.HandleCommand([]string{"set", "docker.dind.mode", "host"})
	})

	output := captureOutput(t, func() {
		configcmd.HandleCommand([]string{"list"})
	})

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "docker.dind.mode") {
			if !strings.Contains(line, "host") {
				t.Errorf("Expected docker.dind.mode=host after config set, got line: %s", line)
			}
			if !strings.Contains(line, "project") {
				t.Errorf("Expected docker.dind.mode source=project after config set, got line: %s", line)
			}
			return
		}
	}
	t.Errorf("Expected docker.dind.mode in config list, got:\n%s", output)
}

// --- Container tests (subprocess, require providers) ---

func TestDind_Addt_IsolatedDockerAvailable(t *testing.T) {
	// Scenario: User enables DinD in isolated mode, then runs the provider's
	// info command inside the container to verify nested containers work.
	// Docker: starts dockerd, validates with "docker info"
	// Podman: validates with "podman info" (daemonless, no startup needed)
	providers := requireProviders(t)

	for _, prov := range providers {
		t.Run(prov, func(t *testing.T) {
			// Podman-in-Podman requires nested user namespaces which aren't
			// supported in the Podman VM on macOS (newuidmap fails).
			if prov == "podman" {
				t.Skip("Podman-in-Podman not supported on macOS (nested user namespaces unavailable in Podman VM)")
			}

			dir, cleanup := setupAddtDirWithExtensions(t, prov, `
docker:
  dind:
    mode: "isolated"
`)
			defer cleanup()
			ensureAddtImage(t, dir, "debug")

			// Set ADDT_DOCKER_DIND_MODE env var for subprocess
			origMode := os.Getenv("ADDT_DOCKER_DIND_MODE")
			os.Setenv("ADDT_DOCKER_DIND_MODE", "isolated")
			defer func() {
				if origMode != "" {
					os.Setenv("ADDT_DOCKER_DIND_MODE", origMode)
				} else {
					os.Unsetenv("ADDT_DOCKER_DIND_MODE")
				}
			}()

			// Pick the right info command based on provider
			var infoCmd string
			switch prov {
			case "docker", "rancher", "orbstack":
				infoCmd = "docker info --format '{{.ServerVersion}}' && echo DIND_TEST:ok"
			case "podman":
				infoCmd = "podman info --format '{{.Version.Version}}' && echo DIND_TEST:ok"
			}

			output, err := runRunSubcommand(t, dir, "debug", "-c", infoCmd)
			t.Logf("DinD isolated output (%s):\n%s", prov, output)
			if err != nil {
				t.Fatalf("DinD isolated test failed (%s): %v\nOutput:\n%s", prov, err, output)
			}

			result := extractMarker(output, "DIND_TEST:")
			if result != "ok" {
				t.Errorf("Expected DIND_TEST:ok (%s info succeeded), got %q\nFull output:\n%s", prov, result, output)
			}
		})
	}
}

func TestDind_Addt_DisabledByDefault(t *testing.T) {
	// Scenario: User runs a container without any DinD config. Nested containers
	// should NOT be available (no daemon for Docker, no privileged mode for Podman).
	providers := requireProviders(t)

	for _, prov := range providers {
		t.Run(prov, func(t *testing.T) {
			dir, cleanup := setupAddtDirWithExtensions(t, prov, ``)
			defer cleanup()
			ensureAddtImage(t, dir, "debug")

			// Pick the right check command based on provider
			var checkCmd string
			switch prov {
			case "docker", "rancher", "orbstack":
				checkCmd = "if docker info >/dev/null 2>&1; then echo DIND_DEFAULT:available; else echo DIND_DEFAULT:unavailable; fi"
			case "podman":
				// Without --privileged, podman run should fail (no user namespaces)
				checkCmd = "if podman run --rm alpine echo ok >/dev/null 2>&1; then echo DIND_DEFAULT:available; else echo DIND_DEFAULT:unavailable; fi"
			}

			output, err := runRunSubcommand(t, dir, "debug", "-c", checkCmd)
			t.Logf("DinD disabled output (%s):\n%s", prov, output)
			if err != nil {
				t.Fatalf("DinD disabled test failed (%s): %v\nOutput:\n%s", prov, err, output)
			}

			result := extractMarker(output, "DIND_DEFAULT:")
			if result != "unavailable" {
				t.Errorf("Expected DIND_DEFAULT:unavailable (nested containers not available by default), got %q\nFull output:\n%s", result, output)
			}
		})
	}
}
