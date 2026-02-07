package podman

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// prepareSecretsJSON collects secret environment variables and returns them as JSON
// Returns the JSON string and the list of secret variable names
func (p *PodmanProvider) prepareSecretsJSON(imageName string, env map[string]string) (string, []string, error) {
	// Get extension env vars (these are the "secrets")
	secretVarNames := p.GetExtensionEnvVars(imageName)

	// Also include credential script vars (e.g. CLAUDE_OAUTH_CREDENTIALS)
	if credVars, ok := env["ADDT_CREDENTIAL_VARS"]; ok && credVars != "" {
		for _, v := range strings.Split(credVars, ",") {
			secretVarNames = append(secretVarNames, strings.TrimSpace(v))
		}
	}

	if len(secretVarNames) == 0 {
		return "", nil, nil
	}

	// Collect secrets that have values
	secrets := make(map[string]string)
	writtenSecrets := []string{}
	for _, varName := range secretVarNames {
		value, exists := env[varName]
		if !exists || value == "" {
			continue
		}
		secrets[varName] = value
		writtenSecrets = append(writtenSecrets, varName)
	}

	if len(writtenSecrets) == 0 {
		return "", nil, nil
	}

	// Encode as JSON
	jsonBytes, err := json.Marshal(secrets)
	if err != nil {
		return "", nil, fmt.Errorf("failed to marshal secrets: %w", err)
	}

	return string(jsonBytes), writtenSecrets, nil
}

// copySecretsToContainer copies secrets JSON to the container's tmpfs via podman cp
func (p *PodmanProvider) copySecretsToContainer(containerName, secretsJSON string) error {
	// Write secrets to a temp file
	tmpFile, err := os.CreateTemp("", "addt-secrets-*.json")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if _, err := tmpFile.WriteString(secretsJSON); err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to write secrets: %w", err)
	}
	tmpFile.Close()

	// Set readable permissions — podman cp preserves these inside the container,
	// and the entrypoint runs as addt (not root) so it needs to read the file.
	// The file lives in a tmpfs and is deleted immediately after parsing.
	if err := os.Chmod(tmpPath, 0644); err != nil {
		return fmt.Errorf("failed to set permissions: %w", err)
	}

	// Copy to container's /run/secrets/.secrets
	cmd := exec.Command("podman", "cp", tmpPath, containerName+":/run/secrets/.secrets")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("podman cp failed: %w\n%s", err, string(output))
	}

	return nil
}

// addTmpfsSecretsMount adds a tmpfs mount for secrets at /run/secrets
// World-writable so the entrypoint (running as addt) can read and delete
// the secrets file. The tmpfs is ephemeral and secrets are deleted immediately
// after parsing, so the broad permissions are acceptable.
func (p *PodmanProvider) addTmpfsSecretsMount(podmanArgs []string) []string {
	return append(podmanArgs, "--tmpfs", "/run/secrets:size=1m,mode=0777")
}

// filterSecretEnvVars removes secret env vars from the env map
// This prevents secrets from being passed as -e flags
func (p *PodmanProvider) filterSecretEnvVars(env map[string]string, secretVarNames []string) {
	for _, varName := range secretVarNames {
		delete(env, varName)
	}
}
