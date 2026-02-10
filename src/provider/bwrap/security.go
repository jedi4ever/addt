package bwrap

import "strconv"

// addSecurityArgs translates security configuration to bwrap arguments.
//
// Bwrap natively supports:
//   - Network isolation (--unshare-net): enabled by NetworkMode=="none" or FirewallEnabled
//   - User namespace isolation (--unshare-user)
//   - UTS namespace (--unshare-uts)
//
// NOT translatable to bwrap:
//   - Per-domain firewall rules — bwrap only supports full on/off via --unshare-net
//   - Process limits (pids_limit) — bwrap doesn't manage cgroups
//   - Ulimits — must be set on the host or via a wrapper
//   - Capability dropping — bwrap runs unprivileged by design
//   - Seccomp profiles — bwrap uses raw BPF, not Docker JSON format
//   - Read-only rootfs — bwrap uses --ro-bind per mount (handled in buildBwrapArgs)
//   - Memory swap limits — requires cgroup access
func (b *BwrapProvider) addSecurityArgs(args []string) []string {
	sec := b.config.Security

	// Network isolation:
	//  - NetworkMode=="none" → full isolation (loopback only)
	//  - FirewallEnabled → maps to --unshare-net since bwrap has no per-domain filtering
	//  - Other modes (bridge, host, "") → share host network (ports directly accessible)
	if sec.NetworkMode == "none" || b.config.FirewallEnabled {
		args = append(args, "--unshare-net")
	}

	// UTS namespace — isolate hostname
	args = append(args, "--unshare-uts")
	args = append(args, "--hostname", "addt")

	// Time limit — pass as env var for the command to enforce
	// (No built-in bwrap timeout; the caller can use the timeout command)
	if sec.TimeLimit > 0 {
		args = append(args, "--setenv", "ADDT_TIME_LIMIT_SECONDS",
			strconv.Itoa(sec.TimeLimit*60))
	}

	return args
}
