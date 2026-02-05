package podman

import (
	"testing"
)

func TestFilterSecretEnvVars(t *testing.T) {
	p := &PodmanProvider{}

	env := map[string]string{
		"ANTHROPIC_API_KEY":   "secret123",
		"GH_TOKEN":            "ghp_xxx",
		"PATH":                "/usr/bin",
		"OPENAI_API_KEY":      "sk-xxx",
		"NON_SECRET_VAR":      "value",
	}

	secretVars := []string{"ANTHROPIC_API_KEY", "GH_TOKEN", "OPENAI_API_KEY"}

	p.filterSecretEnvVars(env, secretVars)

	// Secret vars should be removed
	for _, secretVar := range secretVars {
		if _, exists := env[secretVar]; exists {
			t.Errorf("filterSecretEnvVars should have removed %s", secretVar)
		}
	}

	// Non-secret vars should remain
	if _, exists := env["PATH"]; !exists {
		t.Error("filterSecretEnvVars removed PATH which should remain")
	}
	if _, exists := env["NON_SECRET_VAR"]; !exists {
		t.Error("filterSecretEnvVars removed NON_SECRET_VAR which should remain")
	}
}

func TestAddSecretsMount_Empty(t *testing.T) {
	p := &PodmanProvider{}

	args := []string{"-it", "--rm"}
	result := p.addSecretsMount(args, "")

	// Should return original args unchanged
	if len(result) != len(args) {
		t.Errorf("addSecretsMount with empty secretsDir changed args: got %v, want %v", result, args)
	}
}

func TestAddSecretsMount_Valid(t *testing.T) {
	p := &PodmanProvider{}

	args := []string{"-it", "--rm"}
	secretsDir := "/tmp/addt-secrets-123"
	result := p.addSecretsMount(args, secretsDir)

	// Should add volume mount
	expectedMount := secretsDir + ":/run/secrets:ro"
	found := false
	for i := 0; i < len(result)-1; i++ {
		if result[i] == "-v" && result[i+1] == expectedMount {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("addSecretsMount did not add expected mount %q, got %v", expectedMount, result)
	}
}
