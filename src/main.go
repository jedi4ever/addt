package main

import (
	"github.com/jedi4ever/dclaude/cmd"
	"github.com/jedi4ever/dclaude/internal/util"
)

// Version can be overridden at build time with -ldflags "-X main.Version=x.y.z"
var Version = "1.1.0"

const (
	DefaultNodeVersion    = "20"
	DefaultGoVersion      = "1.23"
	DefaultPortRangeStart = 30000
)

func main() {
	// Setup cleanup on exit
	util.SetupCleanup()

	// Execute CLI
	cmd.Execute(Version, DefaultNodeVersion, DefaultGoVersion, DefaultPortRangeStart)
}
