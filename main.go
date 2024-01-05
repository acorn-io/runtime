package main

import (
	acorn "github.com/acorn-io/runtime/pkg/cli"
	controllerruntime "sigs.k8s.io/controller-runtime"

	// Include cloud auth clients
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

func main() {
	cmd := acorn.New()

	ctx := controllerruntime.SetupSignalHandler()
	acorn.RunAndHandleError(ctx, cmd)
}
