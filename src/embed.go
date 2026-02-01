package main

import (
	_ "embed"
)

// Embedded assets - these files are copied from parent directory at build time
// See Makefile target 'assets'

//go:embed assets/Dockerfile
var EmbeddedDockerfile []byte

//go:embed assets/docker-entrypoint.sh
var EmbeddedEntrypoint []byte
