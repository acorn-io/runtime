package main

import (
	acorn "github.com/acorn-io/acorn/pkg/cli"
	"github.com/acorn-io/acorn/pkg/version"
	"github.com/rancher/wrangler/pkg/signals"

	// Include cloud auth clients
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

func main() {
	cmd := acorn.New()
	cmd.Version = version.Get().String()
	cmd.InitDefaultVersionFlag()

	ctx := signals.SetupSignalContext()
	acorn.RunAndHandleError(ctx, cmd)
}
