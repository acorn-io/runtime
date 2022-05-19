package main

import (
	"os"

	acorn "github.com/acorn-io/acorn/pkg/cli"
	"github.com/acorn-io/acorn/pkg/version"
	"github.com/rancher/wrangler/pkg/signals"
)

var (
	Version = "v0.0.0-dev"
)

func main() {
	cmd := acorn.New()
	cmd.Version = version.Version(Version)
	version.Tag = Version
	cmd.InitDefaultVersionFlag()

	ctx := signals.SetupSignalContext()
	if err := cmd.ExecuteContext(ctx); err != nil {
		os.Exit(1)
	}
}
