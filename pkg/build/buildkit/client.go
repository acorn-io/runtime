package buildkit

import (
	"context"
	"os"

	"github.com/acorn-io/runtime/pkg/build/depot"
	"github.com/moby/buildkit/client"
)

var (
	depotToken   = os.Getenv("DEPOT_TOKEN")
	depotProject = os.Getenv("DEPOT_PROJECT_ID")
)

func newClient(ctx context.Context, image, platform string) (*client.Client, func(error), error) {
	if depotToken != "" && depotProject != "" {
		return depot.Client(ctx, depotProject, depotToken, image, platform)
	}
	bkc, err := client.New(ctx, "")
	if err != nil {
		return nil, nil, err
	}
	return bkc, func(_ error) {
		_ = bkc.Close()
	}, nil
}
