package bwrap

import "testing"

func TestMatchDomain_Exact(t *testing.T) {
	tests := []struct {
		domain  string
		pattern string
		want    bool
	}{
		{"github.com", "github.com", true},
		{"GITHUB.COM", "github.com", true},
		{"github.com", "GITHUB.COM", true},
		{"github.com", "gitlab.com", false},
		{"api.github.com", "github.com", false},
	}

	for _, tc := range tests {
		got := matchDomain(tc.domain, tc.pattern)
		if got != tc.want {
			t.Errorf("matchDomain(%q, %q) = %v, want %v", tc.domain, tc.pattern, got, tc.want)
		}
	}
}

func TestMatchDomain_Wildcard(t *testing.T) {
	tests := []struct {
		domain  string
		pattern string
		want    bool
	}{
		{"api.example.com", "*.example.com", true},
		{"sub.api.example.com", "*.example.com", true},
		{"example.com", "*.example.com", false},
		{"foo.bar.com", "*.example.com", false},
	}

	for _, tc := range tests {
		got := matchDomain(tc.domain, tc.pattern)
		if got != tc.want {
			t.Errorf("matchDomain(%q, %q) = %v, want %v", tc.domain, tc.pattern, got, tc.want)
		}
	}
}

func TestIsDomainAllowed_StrictMode(t *testing.T) {
	// Strict mode: only explicitly allowed domains pass
	p := &NetworkProxy{
		allowedDomains: []string{"github.com", "*.npmjs.org"},
		deniedDomains:  []string{"evil.com"},
		mode:           "strict",
	}

	tests := []struct {
		domain string
		want   bool
	}{
		{"github.com", true},
		{"registry.npmjs.org", true},
		{"example.com", false},
		{"evil.com", false},
		{"localhost", true},
		{"127.0.0.1", true},
	}

	for _, tc := range tests {
		got := p.isDomainAllowed(tc.domain)
		if got != tc.want {
			t.Errorf("strict: isDomainAllowed(%q) = %v, want %v", tc.domain, got, tc.want)
		}
	}
}

func TestIsDomainAllowed_PermissiveMode(t *testing.T) {
	// Permissive mode: everything not denied is allowed
	p := &NetworkProxy{
		allowedDomains: []string{"github.com"},
		deniedDomains:  []string{"evil.com", "*.malware.net"},
		mode:           "permissive",
	}

	tests := []struct {
		domain string
		want   bool
	}{
		{"github.com", true},
		{"example.com", true},
		{"random-site.org", true},
		{"evil.com", false},
		{"sub.malware.net", false},
	}

	for _, tc := range tests {
		got := p.isDomainAllowed(tc.domain)
		if got != tc.want {
			t.Errorf("permissive: isDomainAllowed(%q) = %v, want %v", tc.domain, got, tc.want)
		}
	}
}

func TestIsDomainAllowed_DenyOverridesAllow(t *testing.T) {
	// Deny list should take priority over allow list
	p := &NetworkProxy{
		allowedDomains: []string{"evil.com"},
		deniedDomains:  []string{"evil.com"},
		mode:           "strict",
	}

	if p.isDomainAllowed("evil.com") {
		t.Error("expected deny to override allow for evil.com")
	}
}

func TestMergeUniqueDomains(t *testing.T) {
	result := mergeUniqueDomains(
		[]string{"a.com", "b.com"},
		[]string{"b.com", "c.com"},
		[]string{"c.com", "d.com"},
	)

	if len(result) != 4 {
		t.Errorf("expected 4 unique domains, got %d: %v", len(result), result)
	}

	expected := map[string]bool{"a.com": true, "b.com": true, "c.com": true, "d.com": true}
	for _, d := range result {
		if !expected[d] {
			t.Errorf("unexpected domain in result: %s", d)
		}
	}
}

func TestMergeUniqueDomains_Empty(t *testing.T) {
	result := mergeUniqueDomains(nil, []string{}, nil)
	if len(result) != 0 {
		t.Errorf("expected 0 domains from empty lists, got %d", len(result))
	}
}
