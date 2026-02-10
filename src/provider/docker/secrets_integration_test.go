//go:build integration

package docker

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/jedi4ever/addt/assets"
	"github.com/jedi4ever/addt/config/security"
	"github.com/jedi4ever/addt/extensions"
	"github.com/jedi4ever/addt/provider"
)

// checkDockerForSecrets verifies Docker is available
func checkDockerForSecrets(t *testing.T) {
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

func TestIsolateSecrets_Integration_EnvVarsNotPassed(t *testing.T) {
	checkDockerForSecrets(t)

	// Create a provider with isolate_secrets enabled
	secCfg := security.DefaultConfig()
	secCfg.IsolateSecrets = true

	cfg := &provider.Config{
		Security: secCfg,
	}

	prov := &DockerProvider{
		config:   cfg,
		tempDirs: []string{},
	}

	// Simulate env with secrets
	env := map[string]string{
		"ANTHROPIC_API_KEY": "sk-ant-test-key-12345",
		"TERM":              "xterm-256color",
		"HOME":              "/home/addt",
	}

	// Get extension env vars (simulate what would come from extension config)
	secretVarNames := []string{"ANTHROPIC_API_KEY"}

	// Filter the secret env vars from the env map
	prov.filterSecretEnvVars(env, secretVarNames)

	// Verify ANTHROPIC_API_KEY was removed from env
	if _, exists := env["ANTHROPIC_API_KEY"]; exists {
		t.Error("ANTHROPIC_API_KEY should be filtered out when isolate_secrets is enabled")
	}

	// Verify non-secret env vars remain
	if env["TERM"] != "xterm-256color" {
		t.Errorf("TERM should remain, got %q", env["TERM"])
	}
}

func TestIsolateSecrets_Integration_TmpfsSecretsReadable(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping container test in short mode")
	}
	checkDockerForSecrets(t)

	// Create secrets JSON (as host would prepare)
	secrets := map[string]string{
		"ANTHROPIC_API_KEY": "sk-ant-test-key-integration-12345",
	}
	jsonBytes, _ := json.Marshal(secrets)

	containerName := fmt.Sprintf("addt-secrets-tmpfs-test-%d", os.Getpid())
	defer provider.DockerCmd("default", "rm", "-f", containerName).Run()

	// Start container detached with wait loop
	startCmd := provider.DockerCmd("default", "run", "-d",
		"--name", containerName,
		"--tmpfs", "/run/secrets:size=1m,mode=0777",
		"node:22-slim",
		"sh", "-c", "while [ ! -f /run/secrets/.secrets ]; do sleep 0.1; done; cat /run/secrets/ANTHROPIC_API_KEY")

	if output, err := startCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to start container: %v\nOutput: %s", err, string(output))
	}

	// Write secrets to container via docker exec (not docker cp, which can't write to tmpfs)
	writeCmd := provider.DockerCmd("default", "exec", "-i", containerName,
		"sh", "-c", "cat > /run/secrets/.secrets && chmod 644 /run/secrets/.secrets")
	writeCmd.Stdin = strings.NewReader(string(jsonBytes))
	if output, err := writeCmd.CombinedOutput(); err != nil {
		t.Fatalf("docker exec write secrets failed: %v\nOutput: %s", err, string(output))
	}

	// Also write the individual secret file (simulating entrypoint behavior)
	writeSecretCmd := provider.DockerCmd("default", "exec", "-i", containerName,
		"sh", "-c", "cat > /run/secrets/ANTHROPIC_API_KEY && chmod 600 /run/secrets/ANTHROPIC_API_KEY")
	writeSecretCmd.Stdin = strings.NewReader(secrets["ANTHROPIC_API_KEY"])
	if output, err := writeSecretCmd.CombinedOutput(); err != nil {
		t.Fatalf("docker exec write secret failed: %v\nOutput: %s", err, string(output))
	}

	// Wait for container to finish
	waitCmd := provider.DockerCmd("default", "wait", containerName)
	waitCmd.Run()

	// Get logs
	logsCmd := provider.DockerCmd("default", "logs", containerName)
	output, _ := logsCmd.CombinedOutput()

	expected := secrets["ANTHROPIC_API_KEY"]
	if strings.TrimSpace(string(output)) != expected {
		t.Errorf("Expected secret value %q, got %q", expected, string(output))
	}
}

func TestIsolateSecrets_Integration_SecretsNotInEnvWhenDisabled(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping container test in short mode")
	}
	checkDockerForSecrets(t)

	// When isolate_secrets is disabled, secrets should be passed as env vars
	// This test verifies the default behavior
	secretValue := "sk-ant-test-direct-env-12345"

	cmd := provider.DockerCmd("default", "run", "--rm",
		"-e", "ANTHROPIC_API_KEY="+secretValue,
		"alpine:latest",
		"printenv", "ANTHROPIC_API_KEY")

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to run container: %v\nOutput: %s", err, string(output))
	}

	if strings.TrimSpace(string(output)) != secretValue {
		t.Errorf("Expected env var value %q, got %q", secretValue, string(output))
	}
}

func TestIsolateSecrets_Integration_TmpfsPermissions(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping container test in short mode")
	}
	checkDockerForSecrets(t)

	containerName := fmt.Sprintf("addt-tmpfs-perms-test-%d", os.Getpid())
	defer provider.DockerCmd("default", "rm", "-f", containerName).Run()

	// Start container that writes a secret file and checks permissions
	cmd := provider.DockerCmd("default", "run", "--rm",
		"--name", containerName,
		"--tmpfs", "/run/secrets:size=1m,mode=0777",
		"alpine:latest",
		"sh", "-c", "echo 'secret' > /run/secrets/TEST_SECRET && chmod 600 /run/secrets/TEST_SECRET && stat -c '%a' /run/secrets/TEST_SECRET")

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to run container: %v\nOutput: %s", err, string(output))
	}

	// File should have 0600 permissions
	if strings.TrimSpace(string(output)) != "600" {
		t.Errorf("Secret file should have 600 permissions, got %s", string(output))
	}
}

func TestIsolateSecrets_Integration_ProviderBuildsCorrectArgs(t *testing.T) {
	checkDockerForSecrets(t)

	// Create a provider with isolate_secrets enabled
	secCfg := security.DefaultConfig()
	secCfg.IsolateSecrets = true

	cfg := &provider.Config{
		Security: secCfg,
	}

	prov := &DockerProvider{
		config:   cfg,
		tempDirs: []string{},
	}

	// Create a RunSpec with env vars
	spec := &provider.RunSpec{
		Name:      "test-secrets-args",
		ImageName: "alpine:latest",
		Env: map[string]string{
			"TERM": "xterm",
			"HOME": "/home/test",
		},
	}

	// Create container context
	ctx := &containerContext{
		homeDir:              "/tmp",
		username:             "addt",
		useExistingContainer: false,
	}

	// Build docker args
	dockerArgs := prov.buildBaseDockerArgs(spec, ctx)
	dockerArgs, cleanup := prov.addContainerVolumesAndEnv(dockerArgs, spec, ctx)
	defer cleanup()

	// Check for --tmpfs mount (should be present when isolate_secrets is enabled)
	foundTmpfsMount := false
	foundSecretsEnvVar := false

	for i, arg := range dockerArgs {
		if arg == "--tmpfs" && i+1 < len(dockerArgs) {
			if strings.HasPrefix(dockerArgs[i+1], "/run/secrets:") {
				foundTmpfsMount = true
			}
		}
		// Should NOT have any ADDT_SECRETS_B64 env var with new approach
		if arg == "-e" && i+1 < len(dockerArgs) {
			if strings.HasPrefix(dockerArgs[i+1], "ADDT_SECRETS_B64=") {
				foundSecretsEnvVar = true
			}
		}
	}

	if !foundTmpfsMount {
		t.Error("tmpfs mount for /run/secrets should be present")
	}

	if foundSecretsEnvVar {
		t.Error("ADDT_SECRETS_B64 env var should NOT be present with docker cp approach")
	}

	t.Logf("Docker args: %v", dockerArgs)
}

const testSecretsImageName = "addt-test-secrets"

// createSecretsTestProvider creates a provider for secrets tests
func createSecretsTestProvider(t *testing.T, cfg *provider.Config) *DockerProvider {
	t.Helper()
	prov, err := NewDockerProvider(
		cfg,
		assets.DockerDockerfile,
		assets.DockerDockerfileBase,
		assets.DockerEntrypoint,
		assets.DockerInitFirewall,
		assets.DockerInstallSh,
		extensions.FS,
	)
	if err != nil {
		t.Fatalf("Failed to create Docker provider: %v", err)
	}
	// Type assert to *DockerProvider
	dockerProv, ok := prov.(*DockerProvider)
	if !ok {
		t.Fatal("Provider is not a DockerProvider")
	}
	return dockerProv
}

// ensureSecretsTestImage builds the test image if needed
func ensureSecretsTestImage(t *testing.T) {
	t.Helper()

	if testing.Short() {
		t.Skip("skipping image build test in short mode")
	}

	cmd := provider.DockerCmd("default", "image", "inspect", testSecretsImageName)
	if cmd.Run() == nil {
		return // Image exists
	}

	t.Log("Building test image for secrets integration test...")

	cfg := &provider.Config{
		AddtVersion: "0.0.0-test",
		Extensions:  "claude",
		NodeVersion: "22",
		GoVersion:   "latest",
		UvVersion:   "latest",
		ImageName:   testSecretsImageName,
	}

	prov := createSecretsTestProvider(t, cfg)
	if err := prov.Initialize(cfg); err != nil {
		t.Fatalf("Failed to initialize provider: %v", err)
	}

	if err := prov.BuildIfNeeded(false, false); err != nil {
		t.Fatalf("Failed to build test image: %v", err)
	}
}

// TestIsolateSecrets_Integration_DockerExecApproach tests the docker exec approach
// where secrets are piped via stdin into a running container's tmpfs
func TestIsolateSecrets_Integration_DockerExecApproach(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping container test in short mode")
	}
	checkDockerForSecrets(t)

	secretValue := "sk-ant-docker-exec-test-12345"
	secrets := map[string]string{
		"ANTHROPIC_API_KEY": secretValue,
	}
	secretsJSON, _ := json.Marshal(secrets)

	containerName := fmt.Sprintf("addt-docker-exec-test-%d", os.Getpid())
	defer provider.DockerCmd("default", "rm", "-f", containerName).Run()

	// 1. Start container detached with wait loop (simulating runWithSecrets)
	waitScript := `while [ ! -f /run/secrets/.secrets ]; do sleep 0.05; done
node -e '
const fs = require("fs");
const data = fs.readFileSync("/run/secrets/.secrets", "utf8");
const secrets = JSON.parse(data);
for (const [key, value] of Object.entries(secrets)) {
    fs.writeFileSync("/run/secrets/" + key, value, { mode: 0o600 });
}
'
rm -f /run/secrets/.secrets
export ANTHROPIC_API_KEY="$(cat /run/secrets/ANTHROPIC_API_KEY)"
echo "LOADED_SECRET=$ANTHROPIC_API_KEY"
`

	startCmd := provider.DockerCmd("default", "run", "-d",
		"--name", containerName,
		"--tmpfs", "/run/secrets:size=1m,mode=0777",
		"node:22-slim",
		"bash", "-c", waitScript)

	if output, err := startCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to start container: %v\nOutput: %s", err, string(output))
	}

	// 2. Write secrets to container via docker exec (not docker cp, which can't write to tmpfs)
	writeCmd := provider.DockerCmd("default", "exec", "-i", containerName,
		"sh", "-c", "cat > /run/secrets/.secrets && chmod 644 /run/secrets/.secrets")
	writeCmd.Stdin = strings.NewReader(string(secretsJSON))
	if output, err := writeCmd.CombinedOutput(); err != nil {
		t.Fatalf("docker exec write secrets failed: %v\nOutput: %s", err, string(output))
	}

	// 3. Wait for container to finish
	waitCmd := provider.DockerCmd("default", "wait", containerName)
	waitCmd.Run()

	// 4. Check logs
	logsCmd := provider.DockerCmd("default", "logs", containerName)
	output, _ := logsCmd.CombinedOutput()

	outputStr := string(output)
	t.Logf("Container output:\n%s", outputStr)

	expected := "LOADED_SECRET=" + secretValue
	if !strings.Contains(outputStr, expected) {
		t.Errorf("Expected %q in output, got: %s", expected, outputStr)
	}
}

// TestIsolateSecrets_Integration_SecretsNotInProcEnviron verifies that
// secrets are NOT visible in /proc/1/environ when using docker cp approach
func TestIsolateSecrets_Integration_SecretsNotInProcEnviron(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping container test in short mode")
	}
	checkDockerForSecrets(t)

	secretValue := "sk-ant-proc-environ-test"
	secrets := map[string]string{
		"MY_SECRET": secretValue,
	}
	secretsJSON, _ := json.Marshal(secrets)

	containerName := fmt.Sprintf("addt-proc-environ-test-%d", os.Getpid())
	defer provider.DockerCmd("default", "rm", "-f", containerName).Run()

	// Start container that checks /proc/1/environ
	checkScript := `
while [ ! -f /run/secrets/.secrets ]; do sleep 0.05; done

# Check if secret is in initial /proc/1/environ (should NOT be)
if grep -q "MY_SECRET" /proc/1/environ 2>/dev/null; then
    echo "FAIL: MY_SECRET found in /proc/1/environ"
else
    echo "PASS: MY_SECRET NOT in /proc/1/environ"
fi

# Load secret from file
node -e '
const fs = require("fs");
const data = fs.readFileSync("/run/secrets/.secrets", "utf8");
const secrets = JSON.parse(data);
for (const [key, value] of Object.entries(secrets)) {
    fs.writeFileSync("/run/secrets/" + key, value, { mode: 0o600 });
}
'
rm -f /run/secrets/.secrets
export MY_SECRET="$(cat /run/secrets/MY_SECRET)"

# Secret is now in current shell's env, but not in /proc/1/environ
echo "MY_SECRET value: $MY_SECRET"
`

	startCmd := provider.DockerCmd("default", "run", "-d",
		"--name", containerName,
		"--tmpfs", "/run/secrets:size=1m,mode=0777",
		"node:22-slim",
		"bash", "-c", checkScript)

	if output, err := startCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to start container: %v\nOutput: %s", err, string(output))
	}

	// Write secrets to container via docker exec (not docker cp, which can't write to tmpfs)
	writeCmd := provider.DockerCmd("default", "exec", "-i", containerName,
		"sh", "-c", "cat > /run/secrets/.secrets && chmod 644 /run/secrets/.secrets")
	writeCmd.Stdin = strings.NewReader(string(secretsJSON))
	if output, err := writeCmd.CombinedOutput(); err != nil {
		t.Fatalf("docker exec write secrets failed: %v\nOutput: %s", err, string(output))
	}

	// Wait and check logs
	provider.DockerCmd("default", "wait", containerName).Run()

	logsCmd := provider.DockerCmd("default", "logs", containerName)
	output, _ := logsCmd.CombinedOutput()
	outputStr := string(output)
	t.Logf("Container output:\n%s", outputStr)

	if !strings.Contains(outputStr, "PASS: MY_SECRET NOT in /proc/1/environ") {
		t.Error("Secret should NOT be in /proc/1/environ")
	}
	if !strings.Contains(outputStr, "MY_SECRET value: "+secretValue) {
		t.Error("Secret should be loadable from tmpfs")
	}
}

// TestIsolateSecrets_Integration_RealEntrypoint tests the actual docker-entrypoint.sh
// with secrets copied via docker cp
func TestIsolateSecrets_Integration_RealEntrypoint(t *testing.T) {
	checkDockerForSecrets(t)
	ensureSecretsTestImage(t)

	secretValue := "sk-ant-real-entrypoint-test-12345"
	secrets := map[string]string{
		"ANTHROPIC_API_KEY": secretValue,
	}
	secretsJSON, _ := json.Marshal(secrets)

	containerName := fmt.Sprintf("addt-secrets-real-entrypoint-%d", os.Getpid())
	defer provider.DockerCmd("default", "rm", "-f", containerName).Run()

	// Start container detached with wait loop that calls entrypoint
	waitScript := `while [ ! -f /run/secrets/.secrets ] && [ ! -f /run/secrets/.ready ]; do sleep 0.05; done; exec /usr/local/bin/docker-entrypoint.sh "$@"`

	startCmd := provider.DockerCmd("default", "run", "-d",
		"--name", containerName,
		"--tmpfs", "/run/secrets:size=1m,mode=0777",
		"-e", "ADDT_COMMAND=sh",
		"--entrypoint", "/bin/sh",
		testSecretsImageName,
		"-c", waitScript, "--",
		"-c", `echo "ANTHROPIC_API_KEY=${ANTHROPIC_API_KEY:-<not set>}"`)

	if output, err := startCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to start container: %v\nOutput: %s", err, string(output))
	}

	// Write secrets to container via docker exec (not docker cp, which can't write to tmpfs)
	writeCmd := provider.DockerCmd("default", "exec", "-i", containerName,
		"sh", "-c", "cat > /run/secrets/.secrets && chmod 644 /run/secrets/.secrets")
	writeCmd.Stdin = strings.NewReader(string(secretsJSON))
	if output, err := writeCmd.CombinedOutput(); err != nil {
		t.Fatalf("docker exec write secrets failed: %v\nOutput: %s", err, string(output))
	}

	// Wait and check
	provider.DockerCmd("default", "wait", containerName).Run()

	logsCmd := provider.DockerCmd("default", "logs", containerName)
	output, _ := logsCmd.CombinedOutput()
	outputStr := string(output)
	t.Logf("Container output:\n%s", outputStr)

	expected := "ANTHROPIC_API_KEY=" + secretValue
	if !strings.Contains(outputStr, expected) {
		t.Errorf("Entrypoint should have loaded ANTHROPIC_API_KEY from tmpfs, expected %q", expected)
	}
}

// TestPrepareSecretsJSON tests the prepareSecretsJSON function
func TestPrepareSecretsJSON(t *testing.T) {
	secCfg := security.DefaultConfig()
	secCfg.IsolateSecrets = true

	cfg := &provider.Config{
		Security:   secCfg,
		Extensions: "claude",
	}

	prov := &DockerProvider{
		config:   cfg,
		tempDirs: []string{},
	}

	env := map[string]string{
		"ANTHROPIC_API_KEY":    "sk-test-key",
		"GH_TOKEN":             "ghp_test",
		"TERM":                 "xterm",
		"ADDT_CREDENTIAL_VARS": "ANTHROPIC_API_KEY,GH_TOKEN",
	}

	jsonStr, secretVarNames, err := prov.prepareSecretsJSON("addt-test", env)
	if err != nil {
		t.Fatalf("prepareSecretsJSON failed: %v", err)
	}

	// Verify JSON contains secrets
	var parsed map[string]string
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		t.Fatalf("Invalid JSON: %v", err)
	}

	if parsed["ANTHROPIC_API_KEY"] != "sk-test-key" {
		t.Errorf("ANTHROPIC_API_KEY not in JSON")
	}
	if parsed["GH_TOKEN"] != "ghp_test" {
		t.Errorf("GH_TOKEN not in JSON")
	}

	// Verify secret var names returned
	if len(secretVarNames) != 2 {
		t.Errorf("Expected 2 secret var names, got %d", len(secretVarNames))
	}
}
