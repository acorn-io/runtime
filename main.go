package main

import (
	"os"

	acorn "github.com/acorn-io/acorn/pkg/cli"
	"github.com/acorn-io/acorn/pkg/version"
	"github.com/rancher/wrangler/pkg/signals"
)

func main() {
	cmd := acorn.New()
	cmd.Version = version.Get().String()
	cmd.InitDefaultVersionFlag()

	ctx := signals.SetupSignalContext()
	if err := cmd.ExecuteContext(ctx); err != nil {
		os.Exit(1)
	}
}
