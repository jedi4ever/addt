copy and paste blocks in claude in container
- bracketed paste works in container bash shell but NOT in Claude Code (Ink/Node.js)
  Ink needs to enable bracketed paste mode (\e[?2004h) itself — not an addt issue
- add config setting to enable/disable OSC support (terminal.osc or similar)
====
- resume sessions - mount or .addt ?
    or in ~/.addt outside of dir
    also for project settings
- config_dirs in .addt ?
========
- shell history / it is per workdir - even it not reused 
===========
- seccomp - TODO 
- profiles - to review



=======
check config for github nog , etc...
=======
add config setting explainer - impact of setting on security / config audit, severity / security group
====
workdir - automount
====================
so I'm confused some vars in install.sh say CLAUDE_VERSION and in setup they seem to be prefixed       
  with ADDT_EXT                                                                                         
                                                                                                         
⏺ Good observation. They're two different contexts:                                                      
                                                                                                         
  install.sh — runs at build time (Docker image build)                                                   
  - Uses tool-native vars: CLAUDE_VERSION, CODEX_VERSION
  - Set by the parent assets/*/install.sh which parses EXTENSION_VERSIONS build arg                      
                                                                                                   
  setup.sh — runs at runtime (container start, via entrypoint)
  - Uses ADDT_EXT_* vars: ADDT_EXT_AUTH_AUTOLOGIN, ADDT_EXT_WORKDIR_AUTOTRUST
  - Set by the entrypoint script which reads from extensions.json + per-extension env var overrides

  Let me trace the flow to confirm.

⏺ Searched for 2 patterns, read 1 file (ctrl+o to expand)     

⏺ So here's the summary:                                                                                 
  ┌────────────────────┬─────────────────┬──────────────────┬──────────────────────────────┐
  │      Context       │      When       │ Variable pattern │           Example            │             
  ├────────────────────┼─────────────────┼──────────────────┼──────────────────────────────┤       
  │ install.sh (build) │ docker build    │ <TOOL>_VERSION   │ CLAUDE_VERSION=stable        │             
  ├────────────────────┼─────────────────┼──────────────────┼──────────────────────────────┤             
  │ setup.sh (runtime) │ container start │ ADDT_EXT_*       │ ADDT_EXT_AUTH_AUTOLOGIN=true │
  └────────────────────┴─────────────────┴──────────────────┴──────────────────────────────┘
  The <TOOL>_VERSION vars (line 413: export "$version_env_var=$ext_version") are set by the parent
  installer and consumed by each extension's install.sh. The ADDT_EXT_* vars (lines 224-226) are set by
  the entrypoint and consumed by each extension's setup.sh.

  They're intentionally different — install.sh uses the tool's own naming convention (so the tool's
  installer can pick it up natively), while setup.sh uses the addt-namespaced convention for config that
  addt manages.

─────────────────────────────────────────────────────────────────────────────────────────────────────────
❯  
─────────────────────────────────────────────────────────────────────────────────────────────────────────
  ⏵⏵ accept edits on (shift+tab to cycle) · 2 files +2 -0 · ctrl+t to hide tasks
====================
auth
    method: auto, env, native
    autologin : false/ true
==========
- config mount
===========
credentials_Script under auth ?
====
- resume sessions - mount or .addt ?
    or in ~/.addt outside of dir
    also for project settings
- config_dirs in .addt ?
========
- shell history
=============================
- add-dir , extra mount
===============================
- mount vs copy workspace
    filter .env, secrets
    ro ?
=============================
- proxy tmiux, gpg, ssh -> auto

- autodect claude native

- mcp ?
- ENV wilcards in extensions ?
- refresh token Claude local

===========================
- .gitconfig ro ? 

- cursor, gemini - saving it to disk !
- write the DOCS! from a dev
- rust
- env vars that get passed
- unset envok

- tmux spawn with new shell
- no banner etc.. when used in non-intersctive mode ? json outout
- mock clsude keychain for now in test
- init command (before dowmload)

claude yolo test
update if homebrew keep that
podman DINDok 
dind isolate
no network
docker/podman install based on provider

- select default shell config
- profiles
- vcr record 
- website

- do we still need the npm version of claude then ?
- integrity check claude
- port 0.09.0.0 listen
- fmt -> logging / denub
- system prune
- secrets alwats trye , why not remove
- secrets into vault
/usr/local/bin/podman-entrypoint.sh: line 271: /home/addt/.local/bin/claude: Permission denied
- cursor IDE integration
- google verify claude

- addt params via args
- logging output build to debug
- -v of config list needs better formatting
- Opencode ?
- unset all env vars
- check Ingmar security measures
- firewall allow in extensions

- git.config_path has no default
- mark extensions - release, beta, experimental
- DISABLE output in headless ?
- multi account claude

golang enable
uv enable

- gpg socket socat issue


https://code.claude.com/docs/en/setup#auto-updates
=============
config global , project , extension

something dev , ops and sec want

===
sockets don't work in podman

===
readonly rootfs tmpfs: podman uses mode=1777 for /home/addt (podman doesn't support uid/gid tmpfs params).
Consider starting as root + chown in entrypoint for better security (same pattern as firewall/DinD).
Docker/OrbStack use uid/gid properly.

Podman-in-Podman (DinD isolated): podman binary now installed in container image,
but nested user namespaces fail on macOS (newuidmap: write to uid_map failed: Operation not permitted).
This is a Podman VM kernel limitation — PinP only works on native Linux hosts.

 "installMethod": "native",
  "autoUpdates": false,
===
https://code.claude.com/docs/en/settings#environment-variables