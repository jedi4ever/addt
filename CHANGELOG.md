# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.5.0] - 2025-02-01

### Added
- Go language support with configurable version (DCLAUDE_GO_VERSION, default: 1.23.5)
- UV Python package manager with configurable version (DCLAUDE_UV_VERSION, default: 0.5.11)
- DCLAUDE_MOUNT_WORKDIR flag to control mounting working directory (default: true)
- DCLAUDE_MOUNT_CLAUDE_CONFIG flag to control mounting ~/.claude config (default: true)
- Go binary installed at /usr/local/go/bin/go with PATH configured
- UV and uvx binaries installed at /usr/local/bin with full functionality
- .bashrc configuration for Go PATH in interactive shells

### Changed
- Go and UV are now available in both container processes and interactive bash shells
- Improved documentation for mount configuration options
- Pre-installed tools list updated to include Go and UV

### Tested
- Go version 1.23.5 working in interactive shells
- UV 0.5.11 with full project workflow (init, add, run)
- UVX tool runner for on-demand Python tools
- Mount configuration flags working correctly

## [1.4.4] - 2025-02-01

### Fixed
- macOS binary "Killed: 9" error by adding codesign step to installation instructions
- Added `codesign --sign - --force` to all macOS installation commands
- Added prominent troubleshooting section for macOS code signing issues

### Changed
- Updated installation instructions to use `xattr -c` and `codesign` on macOS
- Clarified that codesign is necessary for proper execution on macOS

## [1.4.3] - 2025-02-01

### Added
- Automatic "latest" tag support for GitHub releases
- Users can now install without specifying version numbers
- Installation URLs now use `/releases/latest/download/` for convenience

### Changed
- Release workflow automatically updates "latest" git tag on each release
- Updated installation instructions to use latest tag by default
- Specific version installation still available for reproducibility

## [1.4.2] - 2025-02-01

### Fixed
- Fixed "Killed: 9" error on macOS when using binaries from GitHub releases
- Added `CGO_ENABLED=0` to Makefile dist target for clean cross-compilation
- Ensures static binaries without C dependencies when cross-compiling from Linux

### Changed
- All release binaries are now built with CGO disabled for better portability

## [1.4.1] - 2025-02-01

### Changed
- Rebuild release to fix binary issues

## [1.4.0] - 2025-02-01

### Fixed
- Container username is now always "claude" instead of using host username
- Uses host UID/GID for proper file permissions while maintaining consistent username
- Fixed DinD shell mode to properly open bash instead of Claude
- Fixed cross-compilation for all platforms (darwin/linux, amd64/arm64)
- Version checking now uses prefix matching (20 matches 20.x.x)
- Fixed entrypoint argument passing

### Changed
- Renamed DCLAUDE_DOCKER_FORWARD to DCLAUDE_DIND_MODE for clarity
- All mount paths now use /home/claude/ instead of /home/{username}/
- Automatic code formatting added to build process

### Added
- VERSION file dependency in Makefile for proper rebuild triggers

### Tested
- Port forwarding (container→host)
- Docker-in-Docker (isolated and host modes)
- SSH key forwarding
- GPG forwarding
- Logging
- Persistent containers
- Version detection and auto-rebuild

---

## Release Links

- [v1.5.0](https://github.com/jedi4ever/dclaude/releases/tag/v1.5.0) - Latest
- [v1.4.4](https://github.com/jedi4ever/dclaude/releases/tag/v1.4.4)
- [v1.4.3](https://github.com/jedi4ever/dclaude/releases/tag/v1.4.3)
- [v1.4.2](https://github.com/jedi4ever/dclaude/releases/tag/v1.4.2)
- [v1.4.1](https://github.com/jedi4ever/dclaude/releases/tag/v1.4.1)
- [v1.4.0](https://github.com/jedi4ever/dclaude/releases/tag/v1.4.0)

## Installation

Download the latest version:

```bash
# macOS Apple Silicon (M1/M2/M3)
curl -fsSL https://github.com/jedi4ever/dclaude/releases/latest/download/dclaude-darwin-arm64 -o dclaude
chmod +x dclaude
xattr -c dclaude && codesign --sign - --force dclaude
sudo mv dclaude /usr/local/bin/

# macOS Intel
curl -fsSL https://github.com/jedi4ever/dclaude/releases/latest/download/dclaude-darwin-amd64 -o dclaude
chmod +x dclaude
xattr -c dclaude && codesign --sign - --force dclaude
sudo mv dclaude /usr/local/bin/

# Linux x86_64
curl -fsSL https://github.com/jedi4ever/dclaude/releases/latest/download/dclaude-linux-amd64 -o dclaude
chmod +x dclaude
sudo mv dclaude /usr/local/bin/

# Linux ARM64
curl -fsSL https://github.com/jedi4ever/dclaude/releases/latest/download/dclaude-linux-arm64 -o dclaude
chmod +x dclaude
sudo mv dclaude /usr/local/bin/
```
