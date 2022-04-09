package main

import (
	"os"

	herd "github.com/ibuildthecloud/herd/pkg/cli"
	"github.com/rancher/wrangler/pkg/signals"
)

func main() {
	cmd := herd.New()
	ctx := signals.SetupSignalContext()
	if err := cmd.ExecuteContext(ctx); err != nil {
		os.Exit(1)
	}
}
