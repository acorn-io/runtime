package build

import (
	"context"

	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/ibuildthecloud/herd/pkg/k8sclient"
	"github.com/ibuildthecloud/herd/pkg/remoteopts"
)

func GetRemoteOptions(ctx context.Context) ([]remote.Option, error) {
	c, err := k8sclient.Default()
	if err != nil {
		return nil, err
	}
	return remoteopts.GetRemoteOptions(ctx, c)
}
