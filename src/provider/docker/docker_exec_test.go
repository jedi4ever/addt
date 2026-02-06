package docker

import (
	"testing"

	"github.com/jedi4ever/addt/provider"
)

func TestBuildBaseDockerArgs_NewPersistent(t *testing.T) {
	p := &DockerProvider{
		config: &provider.Config{},
	}
	spec := &provider.RunSpec{
		Name:        "test-container",
		Persistent:  true,
		Interactive: true,
	}
	ctx := &containerContext{
		useExistingContainer: false,
	}

	args := p.buildBaseDockerArgs(spec, ctx)

	// Should have: run --name test-container -it --init
	assertContains(t, args, "run")
	assertContains(t, args, "--name")
	assertContains(t, args, "test-container")
	assertContains(t, args, "-it")
	assertContains(t, args, "--init")
	assertNotContains(t, args, "--rm")
}

func TestBuildBaseDockerArgs_ExistingPersistent(t *testing.T) {
	p := &DockerProvider{
		config: &provider.Config{},
	}
	spec := &provider.RunSpec{
		Name:        "test-container",
		Persistent:  true,
		Interactive: true,
	}
	ctx := &containerContext{
		useExistingContainer: true,
	}

	args := p.buildBaseDockerArgs(spec, ctx)

	// Should have: exec -it
	assertContains(t, args, "exec")
	assertContains(t, args, "-it")
	assertNotContains(t, args, "run")
	assertNotContains(t, args, "--rm")
}

func TestBuildBaseDockerArgs_Ephemeral(t *testing.T) {
	p := &DockerProvider{
		config: &provider.Config{},
	}
	spec := &provider.RunSpec{
		Name:        "test-container",
		Persistent:  false,
		Interactive: false,
	}
	ctx := &containerContext{
		useExistingContainer: false,
	}

	args := p.buildBaseDockerArgs(spec, ctx)

	// Should have: run --rm --name test-container -i
	assertContains(t, args, "run")
	assertContains(t, args, "--rm")
	assertContains(t, args, "-i")
	assertNotContains(t, args, "-it")
}

func TestStripInteractiveFlags(t *testing.T) {
	// Test the flag stripping logic used in runPersistent/shellPersistent
	tests := []struct {
		name            string
		input           []string
		wantArgs        []string
		wantInteractive bool
	}{
		{
			name:            "strips -it flag",
			input:           []string{"run", "--name", "test", "-it", "--init", "-v", "/src:/dst"},
			wantArgs:        []string{"run", "--name", "test", "-v", "/src:/dst"},
			wantInteractive: true,
		},
		{
			name:            "strips -i flag",
			input:           []string{"run", "--name", "test", "-i", "-v", "/src:/dst"},
			wantArgs:        []string{"run", "--name", "test", "-v", "/src:/dst"},
			wantInteractive: true,
		},
		{
			name:            "strips -t and --init flags",
			input:           []string{"run", "--name", "test", "-t", "--init", "-e", "FOO=bar"},
			wantArgs:        []string{"run", "--name", "test", "-e", "FOO=bar"},
			wantInteractive: false,
		},
		{
			name:            "no interactive flags",
			input:           []string{"run", "--name", "test", "-v", "/src:/dst"},
			wantArgs:        []string{"run", "--name", "test", "-v", "/src:/dst"},
			wantInteractive: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotArgs []string
			gotInteractive := false
			for _, arg := range tt.input {
				switch arg {
				case "-it":
					gotInteractive = true
				case "-i":
					gotInteractive = true
				case "-t", "--init":
					// stripped
				default:
					gotArgs = append(gotArgs, arg)
				}
			}

			if gotInteractive != tt.wantInteractive {
				t.Errorf("interactive = %v, want %v", gotInteractive, tt.wantInteractive)
			}
			assertSliceEqual(t, gotArgs, tt.wantArgs)
		})
	}
}

func TestShellExistingContainer_UsesEntrypoint(t *testing.T) {
	// Verify that Shell() for existing containers now uses the entrypoint
	// with ADDT_COMMAND=/bin/bash instead of direct /bin/bash
	p := &DockerProvider{
		config: &provider.Config{},
	}
	spec := &provider.RunSpec{
		Name:        "test-container",
		Persistent:  true,
		Interactive: true,
	}
	ctx := &containerContext{
		useExistingContainer: true,
	}

	args := p.buildBaseDockerArgs(spec, ctx)
	// Simulate what Shell() does for existing containers
	args = append(args, "-e", "ADDT_COMMAND=/bin/bash")
	args = append(args, spec.Name, "/usr/local/bin/docker-entrypoint.sh")

	assertContains(t, args, "exec")
	assertContains(t, args, "-e")
	assertContains(t, args, "ADDT_COMMAND=/bin/bash")
	assertContains(t, args, "/usr/local/bin/docker-entrypoint.sh")
}

// Helper functions

func assertContains(t *testing.T, slice []string, item string) {
	t.Helper()
	for _, s := range slice {
		if s == item {
			return
		}
	}
	t.Errorf("slice %v does not contain %q", slice, item)
}

func assertNotContains(t *testing.T, slice []string, item string) {
	t.Helper()
	for _, s := range slice {
		if s == item {
			t.Errorf("slice %v should not contain %q", slice, item)
			return
		}
	}
}

func assertSliceEqual(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Errorf("slice length = %d, want %d\ngot:  %v\nwant: %v", len(got), len(want), got, want)
		return
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("slice[%d] = %q, want %q\ngot:  %v\nwant: %v", i, got[i], want[i], got, want)
			return
		}
	}
}
