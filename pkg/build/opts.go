package build

import (
	"context"
	"fmt"
	"net"
	"net/http"

	"github.com/google/go-containerregistry/pkg/authn"
	ggcrv1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/ibuildthecloud/herd/pkg/build/buildkit"
	"github.com/ibuildthecloud/herd/pkg/k8sclient"
)

func GetRemoteOptions(ctx context.Context) ([]remote.Option, error) {
	c, err := k8sclient.Default()
	if err != nil {
		return nil, err
	}

	dialer, err := buildkit.GetRegistryDialer(ctx, c)
	if err != nil {
		return nil, err
	}

	progress := make(chan ggcrv1.Update)
	go func() {
		for p := range progress {
			if p.Error == nil {
				fmt.Println(p.Complete, "/", p.Total)
			} else {
				fmt.Println(p.Complete, "/", p.Total, p.Error)
			}
		}
	}()
	return []remote.Option{
		remote.WithProgress(progress),
		remote.WithContext(ctx),
		remote.WithAuthFromKeychain(authn.DefaultKeychain),
		remote.WithTransport(&http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return dialer(ctx, "")
			},
		}),
	}, nil
}
