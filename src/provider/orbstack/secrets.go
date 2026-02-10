package orbstack

import (
	"encoding/json"
	"fmt"
	"strings"
)

// prepareSecretsJSON collects secret environment variables and returns them as JSON
// Returns the JSON string and the list of secret variable names
func (p *OrbStackProvider) prepareSecretsJSON(imageName string, env map[string]string) (string, []string, error) {
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

// copySecretsToContainer writes secrets JSON directly into the container's tmpfs.
// Uses docker exec instead of docker cp because docker cp writes to the overlay
// layer beneath tmpfs mounts, making the file invisible inside the container.
func (p *OrbStackProvider) copySecretsToContainer(containerName, secretsJSON string) error {
	cmd := p.dockerCmd("exec", "-i", containerName,
		"sh", "-c", "cat > /run/secrets/.secrets && chmod 644 /run/secrets/.secrets")
	cmd.Stdin = strings.NewReader(secretsJSON)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("docker exec write secrets failed: %w\n%s", err, string(output))
	}
	return nil
}

// addTmpfsSecretsMount adds a tmpfs mount for secrets at /run/secrets
// World-writable so the entrypoint (running as addt) can read and delete
// the secrets file. The tmpfs is ephemeral and secrets are deleted immediately
// after parsing, so the broad permissions are acceptable.
func (p *OrbStackProvider) addTmpfsSecretsMount(dockerArgs []string) []string {
	return append(dockerArgs, "--tmpfs", "/run/secrets:size=1m,mode=0777")
}

// filterSecretEnvVars removes secret env vars from the env map
// This prevents secrets from being passed as -e flags
func (p *OrbStackProvider) filterSecretEnvVars(env map[string]string, secretVarNames []string) {
	for _, varName := range secretVarNames {
		delete(env, varName)
	}
}
