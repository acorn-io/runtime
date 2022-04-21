package build

import (
	"context"

	"github.com/acorn-io/acorn/pkg/k8sclient"
	"github.com/acorn-io/acorn/pkg/remoteopts"
	"github.com/google/go-containerregistry/pkg/v1/remote"
)

func GetRemoteOptions(ctx context.Context) ([]remote.Option, error) {
	c, err := k8sclient.Default()
	if err != nil {
		return nil, err
	}
	return remoteopts.GetRemoteOptions(ctx, c)
}
