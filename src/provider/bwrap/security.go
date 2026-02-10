package bwrap

// addSecurityArgs translates security configuration to bwrap arguments.
//
// Bwrap natively supports:
//   - Network isolation (--unshare-net)
//   - User namespace isolation (--unshare-user)
//   - UTS namespace (--unshare-uts)
//
// NOT translatable to bwrap:
//   - Process limits (pids_limit) — bwrap doesn't manage cgroups
//   - Ulimits — must be set on the host or via a wrapper
//   - Capability dropping — bwrap runs unprivileged by design
//   - Seccomp profiles — bwrap uses raw BPF, not Docker JSON format
//   - Read-only rootfs — bwrap uses --ro-bind per mount (handled in buildBwrapArgs)
//   - Memory swap limits — requires cgroup access
func (b *BwrapProvider) addSecurityArgs(args []string) []string {
	sec := b.config.Security

	// Network isolation: "none" maps to --unshare-net (fully isolated)
	// Other modes (bridge, host, "") share the host network
	if sec.NetworkMode == "none" {
		args = append(args, "--unshare-net")
	}

	// IPC isolation (already set via --unshare-ipc in buildBwrapArgs,
	// but honor explicit disable_ipc=false to skip it)
	// The default is to isolate IPC; this is handled in buildBwrapArgs.

	// UTS namespace — isolate hostname
	args = append(args, "--unshare-uts")
	args = append(args, "--hostname", "addt")

	// Time limit — pass as env var for the command to enforce
	// (No built-in bwrap timeout; the caller can use the timeout command)
	if sec.TimeLimit > 0 {
		args = append(args, "--setenv", "ADDT_TIME_LIMIT_SECONDS",
			formatInt(sec.TimeLimit*60))
	}

	return args
}

// formatInt converts an int to string without importing strconv
func formatInt(n int) string {
	if n == 0 {
		return "0"
	}
	result := ""
	negative := false
	if n < 0 {
		negative = true
		n = -n
	}
	for n > 0 {
		result = string(rune('0'+n%10)) + result
		n /= 10
	}
	if negative {
		result = "-" + result
	}
	return result
}
