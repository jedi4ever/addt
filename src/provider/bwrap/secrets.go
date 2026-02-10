package bwrap

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jedi4ever/addt/config/security"
	"github.com/jedi4ever/addt/provider"
)

// prepareSecrets collects secret environment variables from the env map.
// Returns a map of secret name→value and the list of variable names.
func (b *BwrapProvider) prepareSecrets(env map[string]string) (map[string]string, []string) {
	var secretVarNames []string

	// Extension env vars (read from embedded configs)
	for _, spec := range b.GetExtensionEnvVars("") {
		// Handle "VAR=default" format — extract just the var name
		varName := spec
		if idx := strings.Index(spec, "="); idx != -1 {
			varName = spec[:idx]
		}
		secretVarNames = append(secretVarNames, varName)
	}

	// Credential vars from credential scripts
	if credVars, ok := env["ADDT_CREDENTIAL_VARS"]; ok && credVars != "" {
		for _, v := range strings.Split(credVars, ",") {
			v = strings.TrimSpace(v)
			if v != "" {
				secretVarNames = append(secretVarNames, v)
			}
		}
	}

	if len(secretVarNames) == 0 {
		return nil, nil
	}

	// Collect secrets that have values
	secrets := make(map[string]string)
	var written []string
	for _, varName := range secretVarNames {
		val, ok := env[varName]
		if !ok || val == "" {
			continue
		}
		secrets[varName] = val
		written = append(written, varName)
	}

	if len(written) == 0 {
		return nil, nil
	}
	return secrets, written
}

// writeSecretsDir creates a temp directory with a shell-sourceable secrets file
// and a wrapper script that loads secrets, scrubs the file, and execs the command.
// Returns the temp directory path, or "" if no secrets to write.
func (b *BwrapProvider) writeSecretsDir(secrets map[string]string) (string, error) {
	if len(secrets) == 0 {
		return "", nil
	}

	tmpDir, err := os.MkdirTemp("", "bwrap-secrets-*")
	if err != nil {
		return "", fmt.Errorf("failed to create secrets temp dir: %w", err)
	}

	if err := os.Chmod(tmpDir, 0700); err != nil {
		os.RemoveAll(tmpDir)
		return "", err
	}
	if err := security.WritePIDFile(tmpDir); err != nil {
		os.RemoveAll(tmpDir)
		return "", err
	}
	b.tempDirs = append(b.tempDirs, tmpDir)

	// Write secrets as shell-sourceable export lines
	envFile := filepath.Join(tmpDir, ".secrets")
	var lines []string
	for k, v := range secrets {
		// Escape single quotes in value
		escaped := strings.ReplaceAll(v, "'", "'\\''")
		lines = append(lines, fmt.Sprintf("export %s='%s'", k, escaped))
	}
	if err := os.WriteFile(envFile, []byte(strings.Join(lines, "\n")+"\n"), 0600); err != nil {
		return "", fmt.Errorf("failed to write secrets: %w", err)
	}

	// Write wrapper script that loads and scrubs secrets
	wrapper := filepath.Join(tmpDir, ".wrapper.sh")
	script := "#!/bin/bash\n" +
		"# Load secrets into environment\n" +
		". /run/secrets/.secrets\n" +
		"# Scrub secrets file with random data before deleting\n" +
		"filesize=$(stat -c %s /run/secrets/.secrets 2>/dev/null || echo 256)\n" +
		"dd if=/dev/urandom of=/run/secrets/.secrets bs=\"$filesize\" count=1 conv=notrunc 2>/dev/null\n" +
		"rm -f /run/secrets/.secrets\n" +
		"# Execute the original command\n" +
		"exec \"$@\"\n"
	if err := os.WriteFile(wrapper, []byte(script), 0755); err != nil {
		return "", fmt.Errorf("failed to write wrapper script: %w", err)
	}

	return tmpDir, nil
}

// prepareAndFilterSecrets handles secret isolation setup for a run.
// Collects secrets, writes them to a temp directory, and removes them from
// the env map so they won't appear as --setenv flags.
// Returns the temp directory path containing secrets, or "" if not applicable.
func (b *BwrapProvider) prepareAndFilterSecrets(spec *provider.RunSpec) string {
	if !b.config.Security.IsolateSecrets {
		return ""
	}

	secrets, secretVarNames := b.prepareSecrets(spec.Env)
	if len(secrets) == 0 {
		return ""
	}

	dir, err := b.writeSecretsDir(secrets)
	if err != nil {
		bwrapLogger.Debugf("Warning: failed to prepare secrets: %v", err)
		return ""
	}

	b.filterSecretEnvVars(spec.Env, secretVarNames)
	return dir
}

// filterSecretEnvVars removes secret env vars from the env map
// to prevent them from being passed as --setenv flags
func (b *BwrapProvider) filterSecretEnvVars(env map[string]string, secretVarNames []string) {
	for _, name := range secretVarNames {
		delete(env, name)
	}
	// ADDT_CREDENTIAL_VARS is no longer needed — secrets are in the file
	delete(env, "ADDT_CREDENTIAL_VARS")
}
