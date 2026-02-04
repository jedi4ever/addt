package main

import (
	"github.com/jedi4ever/addt/cmd"
	"github.com/jedi4ever/addt/util"
)

// Version can be overridden at build time with -ldflags "-X main.Version=x.y.z"
var Version = "0.0.7"

const (
	DefaultNodeVersion    = "22"
	DefaultGoVersion      = "latest"
	DefaultUvVersion      = "latest"
	DefaultPortRangeStart = 30000
)

func main() {
	// Setup cleanup on exit
	util.SetupCleanup()

	// Execute CLI
	cmd.Execute(Version, DefaultNodeVersion, DefaultGoVersion, DefaultUvVersion, DefaultPortRangeStart)
}
