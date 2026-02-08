# Test Plan

## Extensions
- [ ] CLAUDE NATIVE
- [ ] CLAUDE ENV / API
- [ ] CLAUDE AUTOMOUNT
- [ ] RESUME SESSION CLAUDE
- [ ] versions latest, stable
- [ ] auth versions


- [ ] OPENAI env (codex)

## Forwarding / Mounts
- [ ] SECRETS / FILE
- [ ] HISTORY
- [x] SSH (ssh_test.go: 7 tests - config, proxy/keys/agent modes, disabled, custom dir, github connect)
- [x] GPG (gpg_test.go: 6 tests - config, custom dir, proxy/keys/agent modes, disabled)
- [x] PORT FORWARD (ports_test.go: 7 tests - config, defaults, port map, system prompt, service accessible, no ports)
- [x] GITHUB auth (github_test.go: 5 tests - defaults, config via set, config loaded, token forwarded, token disabled)
- [x] GITHUB config settings (github_test.go: defaults and config via set tests)
- [x] GIT config (git_config_test.go: 4 tests - default enabled, config via set, config path, completion)
- [x] TMUX (tmux_test.go: 7 tests - default value, config loaded, config via set, env var forwarded, socket accessible, disabled no forwarding, no session no error)
- [ ] OTEL
- [x] ENV file (envfile_test.go: 10 tests - default values, config loaded, config via set, disabled via config, vars loaded, multiple vars, custom file path, disabled no vars, comments and empty lines, missing file no error)
- [x] NPM install with readonly root (npm_readonly_test.go: 7 tests - default value, config loaded, config via set, npm install with readonly, npm install without readonly, readonly root write fails, tmp writable)

## Container Lifecycle
- [x] SHELL (shell_test.go: 5 tests - basic execution, workdir mounted, bash is default, env vars forwarded, user is addt)
- [x] RUN (run_test.go: 5 tests - basic execution, workdir mounted, entrypoint is extension, env vars forwarded, user is addt)
- [x] PERSISTENT (persistent_test.go: 7 tests - default value, config loaded, config via set, state preserved, ephemeral no state, container listed, container cleaned)
- [ ] REBUILD logic
- [ ] PRUNE
- [X] DIND


## Arguments / Flags
- [ ] PASS ARG
- [x] YOLO (claude_yolo_test.go: 4 tests - config sets env, not set by default, args transformation, args via env)
- [ ] OVERRIDES ORDER

## Security
- [ ] SECRETS FILES
- [x] FIREWALL (firewall_test.go: 13 tests - config, defaults, set, allow/deny, remove, reset, strict blocks, allowed reachable, disabled allows all)
- [ ] SECURITY SETTINGS

## Config System
- [ ] GLOBAL, PROJECT, EXTENSION precedence
- [ ] LOGGER

## Provider Support
- [X] DOCKER / PODMAN (parallel test execution)

## Extensions System
- [ ] LOCAL EXTENSIONS / NEW
- [ ] MULTI EXTENSION / DEPENDENCY
- [x] COMPLETIONS (completion_test.go: 6 tests - bash/zsh/fish output, help, extensions included, config keys included)
- [ ] ALIASES

## Other
- [ ] UPDATING CLI / extension
- [ ] DOCTOR
- [X] `<extension> addt` command
- [x] CODEX API KEY (codex_test.go: 3 tests - key forwarded, not leaked to env, absent when unset)
- [ ] Automount
- [ ] Seccomp
