package firewall

import (
	"github.com/jedi4ever/addt/config"
)

// CheckResult represents the result of a firewall check
type CheckResult int

const (
	// CheckNoMatch means no rule matched at this layer
	CheckNoMatch CheckResult = iota
	// CheckAllowed means domain is explicitly allowed
	CheckAllowed
	// CheckDenied means domain is explicitly denied
	CheckDenied
)

// CheckDomain checks if a domain is allowed based on layered rules.
// Order: Defaults → Extension → Global → Project (project wins)
// Returns: allowed (bool), matched layer (string)
func CheckDomain(domain string, cfg *config.Config, extensionName string) (bool, string) {
	// Layer 4: Project (most specific, checked first)
	if result := checkLayer(domain, cfg.ProjectFirewallDenied, cfg.ProjectFirewallAllowed); result != CheckNoMatch {
		return result == CheckAllowed, "project"
	}

	// Layer 3: Global
	if result := checkLayer(domain, cfg.GlobalFirewallDenied, cfg.GlobalFirewallAllowed); result != CheckNoMatch {
		return result == CheckAllowed, "global"
	}

	// Layer 2: Extension
	if result := checkLayer(domain, cfg.ExtensionFirewallDenied, cfg.ExtensionFirewallAllowed); result != CheckNoMatch {
		return result == CheckAllowed, "extension"
	}

	// Layer 1: Defaults (allow only)
	if containsString(DefaultAllowedDomains(), domain) {
		return true, "defaults"
	}

	// No match - depends on firewall mode
	return false, "none"
}

// checkLayer checks a single layer's deny and allow lists
func checkLayer(domain string, denied, allowed []string) CheckResult {
	// Check deny first
	if containsString(denied, domain) {
		return CheckDenied
	}
	// Then check allow
	if containsString(allowed, domain) {
		return CheckAllowed
	}
	return CheckNoMatch
}
