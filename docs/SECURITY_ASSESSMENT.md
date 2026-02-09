# Security Assessment: addt (AI Don't Do That)

**Date:** 2026-02-09
**Scope:** Full codebase security review of addt v0.0.10
**Methodology:** Static analysis of source code, configuration, and architecture

---

## Executive Summary

addt is a containerized execution environment for AI coding agents (Claude Code, Codex, Gemini, etc.) that provides isolation between untrusted AI agents and the host system. The tool demonstrates **strong security architecture** with defense-in-depth practices including secret isolation via tmpfs, SSH/GPG agent proxy filtering, capability dropping, and audit logging.

This assessment identified **2 high-severity**, **4 medium-severity**, and **5 low-severity** findings. The critical findings relate to TCP proxy binding scope and entrypoint script injection risks. No hardcoded secrets or critical vulnerabilities allowing remote code execution were found.

### Risk Rating: **MODERATE**

The tool is well-suited for its intended use case (single-developer workstations). Additional hardening is recommended for shared or multi-user environments.

---

## Table of Contents

1. [Architecture Security Overview](#1-architecture-security-overview)
2. [Findings Summary](#2-findings-summary)
3. [Detailed Findings](#3-detailed-findings)
4. [Strengths](#4-strengths)
5. [Recommendations](#5-recommendations)

---

## 1. Architecture Security Overview

### Trust Boundaries

```
 Host System (TRUSTED)
 +------------------------------------------+
 |  addt CLI                                |
 |  SSH/GPG Proxy (host-side)               |
 |  Credentials (keychain, env vars)        |
 +------------------+-----------------------+
                    |
          Container Boundary (ISOLATION)
                    |
 +------------------v-----------------------+
 |  Container (SEMI-TRUSTED)                |
 |  AI Agent (Claude, Codex, etc.)          |
 |  Workspace (mounted from host)           |
 |  Filtered SSH/GPG access                 |
 +------------------------------------------+
```

### Security Layers

| Layer | Mechanism | Default |
|-------|-----------|---------|
| Process Isolation | Container (Docker/Podman) | Enabled |
| Capability Restriction | `--cap-drop ALL` + minimal adds | Enabled |
| Privilege Escalation | `--security-opt no-new-privileges` | Enabled |
| Secret Isolation | tmpfs at `/run/secrets` | Enabled (`isolate_secrets: true`) |
| Credential Cleanup | Overwrite + unset after use | Enabled |
| SSH Key Filtering | Protocol-level proxy | Available |
| GPG Key Filtering | Assuan protocol proxy | Available |
| Network Isolation | Optional firewall rules | Available |
| Filesystem | Read-only rootfs option | Available |
| Syscall Filtering | Seccomp profiles | Available |
| Resource Limits | PIDs, memory, file descriptors | Configured |
| Audit Logging | JSON event log | Available |

---

## 2. Findings Summary

| ID | Severity | Category | Title |
|----|----------|----------|-------|
| SEC-01 | **HIGH** | Network | TCP proxies bind to 0.0.0.0 without authentication |
| SEC-02 | **HIGH** | Injection | Entrypoint secret JSON keys not validated before shell export |
| SEC-03 | **MEDIUM** | Concurrency | Race condition in SSH proxy filterIdentities |
| SEC-04 | **MEDIUM** | Container | Docker-in-Docker socket world-accessible (0666) |
| SEC-05 | **MEDIUM** | Supply Chain | No cryptographic verification of Podman binary downloads |
| SEC-06 | **MEDIUM** | Configuration | Empty allowlist defaults to all keys permitted |
| SEC-07 | **LOW** | Container | `NOPASSWD:ALL` sudo in container |
| SEC-08 | **LOW** | Configuration | `.env` file parsing accepts arbitrary variable names |
| SEC-09 | **LOW** | Configuration | `yolo` mode bypasses security checks |
| SEC-10 | **LOW** | Container | Secrets tmpfs uses mode 0777 |
| SEC-11 | **LOW** | Entrypoint | Fragile JSON parsing via grep/sed in entrypoint |

---

## 3. Detailed Findings

### SEC-01: TCP Proxies Bind to 0.0.0.0 Without Authentication [HIGH]

**Files:**
- `src/config/security/ssh_proxy.go:119`
- `src/config/security/gpg_proxy.go:107`

**Description:**
When running on macOS (where Unix sockets cannot be forwarded into the container VM), both SSH and GPG agent proxies fall back to TCP mode. The listeners bind to `0.0.0.0:0` (all interfaces, random port) with **no authentication mechanism**.

```go
// ssh_proxy.go:119
l, err := net.Listen("tcp", "0.0.0.0:0")
```

**Impact:**
- Any process on the host (or reachable network) can connect to the proxy
- On multi-user systems, other users can discover the port via `netstat`/`lsof` and use the SSH/GPG keys
- SSH signing and GPG signing/decryption operations can be performed without authorization

**Recommendation:**
- Bind to `127.0.0.1` instead of `0.0.0.0` (containers can still reach host via gateway)
- Consider adding a shared-secret token exchanged via environment variable
- Document the single-user assumption prominently

---

### SEC-02: Entrypoint Secret JSON Keys Not Validated [HIGH]

**Files:**
- `src/assets/docker/docker-entrypoint.sh:143-152`
- `src/assets/podman/podman-entrypoint.sh:143-152`

**Description:**
The entrypoint scripts load secrets from a JSON file and export them as environment variables using `eval`. While values are properly escaped (single-quote escaping), the JSON **keys** (used as variable names) are not validated against a safe pattern.

```bash
eval "$(node -e '
    const secrets = JSON.parse(data);
    for (const [key, value] of Object.entries(secrets)) {
        const escaped = value.replace(/...single quote escape.../);
        console.log(`export ${key}=...`);  # key is NOT validated
    }
')"
```

**Impact:**
If an attacker can influence the secrets JSON (e.g., through a malicious credential script or compromised extension), they could inject shell commands via crafted key names like `$(malicious_command)` or `foo;evil_command`.

**Mitigating factors:**
- The secrets JSON is constructed server-side by addt from trusted credential sources
- Credential scripts have a 5-second timeout and validated output format
- The `env.go` credential pipeline validates env var names (`isValidEnvVarName`)

**Recommendation:**
- Add key validation in the Node.js parser: `if (!/^[A-Za-z_][A-Za-z0-9_]*$/.test(key)) continue;`
- This creates defense-in-depth even though the upstream pipeline validates

---

### SEC-03: Race Condition in SSH Proxy filterIdentities [MEDIUM]

**File:** `src/config/security/ssh_proxy.go:318-331`

**Description:**
The `filterIdentities()` method writes to shared maps (`blobComments`, `allowedBlobs`) without acquiring the mutex, while `checkSignRequest()` reads the same maps with the mutex held. In Go, concurrent map reads/writes cause a runtime panic.

```go
func (p *SSHProxyAgent) filterIdentities(msg []byte) []byte {
    // ...
    p.blobComments[blobStr] = key.comment    // WRITE without lock
    p.allowedBlobs[blobStr] = true           // WRITE without lock
}

func (p *SSHProxyAgent) checkSignRequest(msg []byte) (bool, string) {
    p.mu.Lock()
    comment := p.blobComments[blobStr]       // READ with lock
    p.mu.Unlock()
}
```

**Impact:**
- Runtime panic if two connections trigger simultaneous identity listing and signing
- Denial of service to the SSH proxy
- Low probability in typical single-user usage, higher in automated/scripted scenarios

**Recommendation:**
- Acquire `p.mu.Lock()` in `filterIdentities()` before map writes

---

### SEC-04: Docker-in-Docker Socket World-Accessible [MEDIUM]

**File:** `src/assets/docker/docker-entrypoint.sh` (DinD initialization)

**Description:**
When Docker-in-Docker (DinD) isolated mode is enabled, the nested Docker daemon socket is created with permissions `0666` (world-readable/writable). While this is inside the container, it allows any process within the container to access the nested Docker daemon.

**Impact:**
- An AI agent running inside the container could start additional containers
- Container escape via the nested Docker daemon is theoretically possible
- The AI agent could use DinD to run privileged operations

**Mitigating factors:**
- DinD is an opt-in feature, not enabled by default
- The outer container is already capability-restricted
- The purpose of DinD is explicitly to give the agent Docker access

**Recommendation:**
- Restrict socket to the `addt` user/group (`0660` with proper group)
- Document that DinD mode expands the container's privilege boundary

---

### SEC-05: No Cryptographic Verification of Podman Downloads [MEDIUM]

**File:** `src/config/podman_install.go:75-157`

**Description:**
When downloading the Podman binary from GitHub releases, the tool:
1. Downloads via HTTPS (transport encryption only)
2. Extracts the tar.gz
3. Verifies the binary runs (`podman --version`)

There is **no cryptographic signature verification** (GPG signature, SHA256 checksum) of the downloaded archive.

**Impact:**
- A compromised GitHub CDN or MITM (even unlikely with HTTPS) could serve a malicious binary
- The `podman --version` check only verifies the binary executes, not its integrity

**Recommendation:**
- Verify SHA256 checksum of the downloaded archive against a pinned value
- Consider verifying GPG signatures from the Podman release signing key

---

### SEC-06: Empty Allowlist Defaults to All Keys Permitted [MEDIUM]

**Files:**
- `src/config/security/ssh_proxy.go` (isKeyAllowed)
- `src/config/security/gpg_proxy.go` (isKeyAllowed)

**Description:**
When no keys are specified in the SSH or GPG allowlists, the proxies default to permitting **all** keys:

```go
func (p *SSHProxyAgent) isKeyAllowed(comment string, blob []byte) bool {
    if len(p.allowedKeys) == 0 {
        return true  // ALL KEYS ALLOWED
    }
}
```

Several code paths pass `nil` as the allowlist:
```go
// podman/gpg.go:41
return p.handleGPGProxyForwarding(gpgDir, username, nil)
```

**Impact:**
- Users who enable the proxy but forget to configure key filtering get no benefit from the proxy's filtering capability
- The proxy provides a false sense of security when misconfigured

**Recommendation:**
- Log a warning when proxy mode is active with an empty allowlist
- Consider requiring explicit `allow_all: true` rather than defaulting open

---

### SEC-07: NOPASSWD Sudo in Container [LOW]

**File:** `src/assets/docker/Dockerfile.base:60`

**Description:**
The container user `addt` is granted passwordless sudo access to all commands.

**Impact:**
- An AI agent could escalate to root within the container
- Combined with certain capabilities, this could facilitate container escape
- Acceptable trade-off for developer usability in a container context

**Mitigating factors:**
- `--security-opt no-new-privileges` is enabled by default
- Capabilities are dropped (only CHOWN, SETUID, SETGID retained)
- The container is ephemeral

---

### SEC-08: .env File Parsing Accepts Arbitrary Variable Names [LOW]

**File:** `src/config/env.go:11-45`

**Description:**
The `.env` file parser accepts any `KEY=value` pair and sets it via `os.Setenv()` without validating the key name. A malicious `.env` file in a cloned repository could override critical environment variables (e.g., `PATH`, `HOME`, `LD_PRELOAD`).

**Mitigating factors:**
- `.env` files are typically user-created, not from untrusted sources
- The `.addt.yaml` project config is similarly trusted

**Recommendation:**
- Validate variable names against a safe pattern
- Consider prefixing or namespacing allowed variables

---

### SEC-09: Yolo Mode Bypasses Security Checks [LOW]

**Files:**
- `src/config/security/types.go`
- `src/core/env.go:55-58`

**Description:**
The `security.yolo: true` configuration and `ADDT_SECURITY_YOLO=true` environment variable bypass permission checks in AI agents (e.g., Claude's `--dangerously-skip-permissions`). This is passed into the container environment.

**Impact:**
- AI agents run without safety guardrails
- Explicitly opt-in and documented as dangerous

**Recommendation:**
- Log a visible warning when yolo mode is active
- Consider requiring confirmation for first-time use

---

### SEC-10: Secrets tmpfs Uses Mode 0777 [LOW]

**File:** `src/provider/docker/secrets.go`

**Description:**
The tmpfs mount for secrets uses `mode=0777`:
```
--tmpfs /run/secrets:size=1m,mode=0777
```

**Impact:**
- Any process in the container can read/write to `/run/secrets`
- Secrets are briefly readable by any container process

**Mitigating factors:**
- Secrets are deleted within milliseconds by the entrypoint
- The container runs as a single user in practice
- The file is overwritten with random data before deletion

**Recommendation:**
- Use `mode=0700` or `mode=0755` with proper ownership

---

### SEC-11: Fragile JSON Parsing in Entrypoint [LOW]

**Files:**
- `src/assets/docker/docker-entrypoint.sh:383-389`
- `src/assets/podman/podman-entrypoint.sh:394-400`

**Description:**
Extension configuration JSON is parsed using `grep`, `sed`, and `tr` rather than a proper JSON parser. This is fragile and could produce unexpected results with malformed input.

**Impact:**
- Incorrect parsing could lead to wrong command execution
- Not exploitable for injection (values are used as array elements, not shell-evaluated)

**Recommendation:**
- Use `node -e` or `jq` for JSON parsing consistency

---

## 4. Strengths

The codebase demonstrates several security best practices that deserve recognition:

### 4.1 Defense-in-Depth Secret Handling
The three-layer approach to secrets (tmpfs delivery -> env var export -> cryptographic scrub + unset) is notably thorough. The use of `crypto/rand` for file overwriting and explicit `/proc/1/environ` protection shows awareness of common container secret leakage vectors.

### 4.2 Protocol-Level Key Filtering
The SSH and GPG proxies operate at the protocol level rather than simply forwarding sockets. They parse and filter individual messages (SSH_AGENTC_SIGN_REQUEST, PKSIGN, PKDECRYPT), providing granular access control that goes beyond typical implementations.

### 4.3 Safe Command Execution
All `exec.Command` invocations use array-based argument passing rather than shell string concatenation. No instances of `sh -c` with user-controlled input were found.

### 4.4 Capability Minimization
Default configuration drops ALL capabilities and re-adds only CHOWN, SETUID, and SETGID. The `no-new-privileges` security option is enabled by default.

### 4.5 Credential Variable Lifecycle
Credential variables are tracked via `ADDT_CREDENTIAL_VARS`, overwritten with random data of the same length, then unset. This prevents leakage via `/proc/<pid>/environ` and shell history.

### 4.6 Audit Logging
The audit system provides structured JSON logging of all SSH key access, SSH signing, GPG signing, and GPG decryption operations with key identification and allow/deny decisions.

### 4.7 Restrictive File Permissions
Security-sensitive files consistently use 0600 (sockets, audit logs, PID files) and 0700 (proxy directories). No world-writable files outside the intentional tmpfs.

### 4.8 Seccomp Profile
The restrictive seccomp profile uses a default-deny approach (`SCMP_ACT_ERRNO`) with an explicit allowlist of ~250 syscalls. Dangerous syscalls like `ptrace`, `mount`, `reboot`, and `keyctl` are not in the allowlist.

### 4.9 Stale Resource Cleanup
PID-based orphan detection for proxy sockets and temp directories prevents resource leaks and potential conflicts.

---

## 5. Recommendations

### Priority 1 (Address Soon)

1. **Bind TCP proxies to 127.0.0.1** instead of 0.0.0.0 (SEC-01)
2. **Add JSON key validation** in entrypoint secret parsing (SEC-02)
3. **Fix race condition** in SSH proxy's `filterIdentities` (SEC-03)

### Priority 2 (Planned Improvement)

4. **Add checksum verification** for Podman binary downloads (SEC-05)
5. **Log warnings** for empty proxy allowlists (SEC-06)
6. **Restrict DinD socket permissions** to 0660 (SEC-04)
7. **Tighten secrets tmpfs** permissions to 0700 (SEC-10)

### Priority 3 (Hardening)

8. **Validate .env variable names** against a safe pattern (SEC-08)
9. **Add visible warning** for yolo mode activation (SEC-09)
10. **Use proper JSON parser** in entrypoint for extension config (SEC-11)
11. **Document security model assumptions** (single-user, trusted config files)

---

## Appendix A: Files Reviewed

| Category | Key Files |
|----------|-----------|
| CLI & Config | `src/cmd/root.go`, `src/config/loader.go`, `src/config/file.go`, `src/config/env.go` |
| Security | `src/config/security/ssh_proxy.go`, `src/config/security/gpg_proxy.go`, `src/config/security/types.go`, `src/config/security/audit.go`, `src/config/security/cleanup.go` |
| Container Exec | `src/provider/docker/docker_exec.go`, `src/provider/podman/podman_exec.go` |
| Secrets | `src/provider/docker/secrets.go`, `src/util/scrub.go` |
| Entrypoints | `src/assets/docker/docker-entrypoint.sh`, `src/assets/podman/podman-entrypoint.sh` |
| Dockerfiles | `src/assets/docker/Dockerfile.base`, `src/assets/podman/Dockerfile.base` |
| Extensions | `src/extensions/credentials.go`, `src/extensions/loader.go` |
| Core | `src/core/env.go`, `src/core/options.go`, `src/core/volumes.go` |
| Supply Chain | `src/go.mod`, `src/config/podman_install.go` |
| Seccomp | `src/assets/seccomp/restrictive.json` |

## Appendix B: Dependency Review

| Dependency | Version | Risk | Notes |
|------------|---------|------|-------|
| `golang.org/x/sys` | v0.30.0 | Low | Official Go extended library |
| `gopkg.in/yaml.v3` | v3.0.1 | Low | Widely used YAML parser |
| `github.com/daytonaio/daytona` | v0.138.0 | Low | API client for Daytona provider |
| `github.com/creack/pty` | v1.1.24 | Low | PTY handling |
| `github.com/gorilla/websocket` | v1.5.3 | Low | WebSocket (indirect, via Daytona) |
| `github.com/muesli/termenv` | v0.16.0 | Low | Terminal styling |

The dependency footprint is minimal (6 direct + 6 indirect), reducing supply chain attack surface. All are well-maintained, widely-used Go libraries.
