package bwrap

import "strconv"

// addSecurityArgs translates security configuration to bwrap arguments.
//
// Bwrap natively supports:
//   - Network isolation (--unshare-net)
//   - UTS namespace (--unshare-uts)
//
// Network isolation is enabled by:
//   - NetworkMode=="none" → full isolation (loopback only, no proxy)
//   - FirewallEnabled → --unshare-net + HTTP proxy for per-domain filtering
//     (proxy setup is handled separately in setupNetworkProxy)
//
// NOT translatable to bwrap:
//   - Process limits (pids_limit) — no cgroup access
//   - Ulimits — must be set on the host
//   - Capability dropping — bwrap runs unprivileged by design
//   - Seccomp profiles — bwrap uses raw BPF, not Docker JSON format
//   - Memory swap limits — requires cgroup access
func (b *BwrapProvider) addSecurityArgs(args []string) []string {
	sec := b.config.Security

	// Network isolation:
	//  - NetworkMode=="none" → full isolation (loopback only, no proxy bypass)
	//  - FirewallEnabled → --unshare-net + proxy for selective domain access
	//  - Default → share host network (ports directly accessible)
	if sec.NetworkMode == "none" || b.config.FirewallEnabled {
		args = append(args, "--unshare-net")
	}

	// UTS namespace — isolate hostname
	args = append(args, "--unshare-uts")
	args = append(args, "--hostname", "addt")

	// Time limit — pass as env var for the command to enforce
	if sec.TimeLimit > 0 {
		args = append(args, "--setenv", "ADDT_TIME_LIMIT_SECONDS",
			strconv.Itoa(sec.TimeLimit*60))
	}

	return args
}
