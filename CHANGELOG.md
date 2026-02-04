# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Firewall command with global/project/extension scopes
- Layered firewall rule evaluation (defaults → extension → global → project)
- Project-level rules can override global rules
- Per-extension firewall configuration
- `addt firewall global|project|extension` subcommands
- `CheckDomain()` function for programmatic firewall checks

### Changed
- Firewall rules now stored in config files (`~/.addt/config.yaml`, `.addt.yaml`)
- Moved firewall code to dedicated `cmd/firewall/` package

## [0.0.5] - 2025-02-03

### Added
- Support for `addt-<extension>` symlink naming (e.g., `addt-claude`, `addt-codex`)

## [0.0.4] - 2025-02-03

### Added
- Two-stage Docker build for faster extension builds
- Base image (`addt-base:nodeXX-uidXXX`) caches Node, Go, UV, and system packages
- Extension images build FROM base, taking only ~10-30 seconds
- `--addt-rebuild-base` flag to force base image rebuild

### Changed
- `--addt-rebuild` now only rebuilds extension layer (uses cached base)
- Dockerfile split into `Dockerfile.base` and `Dockerfile`

## [0.0.3] - 2025-02-03

### Fixed
- Look up actual entrypoint from extension config instead of using extension name
- Fixes extensions where name differs from entrypoint: kiro→kiro-cli, beads→bd, gastown→gt, etc.

## [0.0.2] - 2025-02-03

### Fixed
- Auto-detect command from `ADDT_EXTENSIONS` when `ADDT_COMMAND` is not set
- Running `ADDT_EXTENSIONS=codex addt` now correctly runs codex instead of defaulting to claude

## [0.0.1] - 2025-02-02

### Added
- Initial alpha release of addt (AI Don't Do That)
- Complete rewrite in Go with provider-based architecture
- Multi-agent support via extension system
- 14 extensions available:
  - **AI Agents**: claude, codex, gemini, copilot, amp, cursor, kiro
  - **Claude Ecosystem**: claude-flow, claude-sneakpeek, openclaw, tessl, gastown
  - **Utilities**: beads, backlog-md
- Docker provider with image building and container management
- Daytona provider (experimental) for cloud-based workspaces
- Symlink-based agent selection (binary name determines which agent runs)
- Automatic environment variable forwarding based on extension config
- Extension dependency resolution
- Network firewall with whitelist-based domain filtering
- SSH agent forwarding and GPG key forwarding
- Docker-in-Docker support (isolated and host modes)
- Automatic port mapping with host port detection
- Persistent container mode for faster startup
- Command logging support

### Technical
- Go 1.21+ with embedded assets (Dockerfile, extensions)
- Cross-platform builds: darwin/amd64, darwin/arm64, linux/amd64, linux/arm64
- Provider interface for pluggable container runtimes
- Extension metadata in config.yaml with install.sh, setup.sh, args.sh scripts

### Notes
- This is an alpha release - expect breaking changes
- Previously known as "dclaude" and "nddt" (Nope, Don't Do That)
- Renamed to "addt" (AI Don't Do That) to reflect multi-agent support
