package main

import (
	acorn "github.com/acorn-io/runtime/pkg/cli"
	"github.com/rancher/wrangler/pkg/signals"

	// Include cloud auth clients
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

func main() {
	cmd := acorn.New()

	ctx := signals.SetupSignalContext()
	acorn.RunAndHandleError(ctx, cmd)
}
