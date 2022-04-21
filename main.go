package main

import (
	"os"

	acorn "github.com/acorn-io/acorn/pkg/cli"
	"github.com/rancher/wrangler/pkg/signals"
)

func main() {
	cmd := acorn.New()
	ctx := signals.SetupSignalContext()
	if err := cmd.ExecuteContext(ctx); err != nil {
		os.Exit(1)
	}
}
