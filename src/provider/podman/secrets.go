package podman

import (
	"fmt"
	"os"
	"path/filepath"
)

// writeSecretsToFiles writes secret environment variables to files and returns the secrets directory
// The directory should be mounted as /run/secrets:ro in the container
func (p *PodmanProvider) writeSecretsToFiles(imageName string, env map[string]string) (string, []string, error) {
	// Get extension env vars (these are the "secrets")
	secretVarNames := p.GetExtensionEnvVars(imageName)
	if len(secretVarNames) == 0 {
		return "", nil, nil
	}

	// Create temp directory for secrets
	secretsDir, err := os.MkdirTemp("", "addt-secrets-")
	if err != nil {
		return "", nil, fmt.Errorf("failed to create secrets directory: %w", err)
	}

	// Set restrictive permissions on secrets directory
	if err := os.Chmod(secretsDir, 0700); err != nil {
		os.RemoveAll(secretsDir)
		return "", nil, fmt.Errorf("failed to set secrets directory permissions: %w", err)
	}

	// Write each secret to a file
	writtenSecrets := []string{}
	for _, varName := range secretVarNames {
		value, exists := env[varName]
		if !exists || value == "" {
			continue
		}

		secretPath := filepath.Join(secretsDir, varName)
		if err := os.WriteFile(secretPath, []byte(value), 0600); err != nil {
			os.RemoveAll(secretsDir)
			return "", nil, fmt.Errorf("failed to write secret %s: %w", varName, err)
		}
		writtenSecrets = append(writtenSecrets, varName)
	}

	if len(writtenSecrets) == 0 {
		// No secrets to write, clean up
		os.RemoveAll(secretsDir)
		return "", nil, nil
	}

	return secretsDir, writtenSecrets, nil
}

// addSecretsMount adds the secrets directory mount to podman args
func (p *PodmanProvider) addSecretsMount(podmanArgs []string, secretsDir string) []string {
	if secretsDir == "" {
		return podmanArgs
	}
	// Mount as read-only
	return append(podmanArgs, "-v", secretsDir+":/run/secrets:ro")
}

// filterSecretEnvVars removes secret env vars from the env map and returns new args
// This prevents secrets from being passed as -e flags when secrets_to_files is enabled
func (p *PodmanProvider) filterSecretEnvVars(env map[string]string, secretVarNames []string) {
	for _, varName := range secretVarNames {
		delete(env, varName)
	}
}
