package remoteopts

import (
	"context"
	"net"
	"net/http"
	"strings"

	"github.com/acorn-io/acorn/pkg/build/buildkit"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetRemoteWriteOptions(ctx context.Context, c client.WithWatch) ([]remote.Option, error) {
	return []remote.Option{
		remote.WithContext(ctx),
		remote.WithAuthFromKeychain(authn.DefaultKeychain),
	}, nil
}

func GetRemoteOptions(ctx context.Context, c client.WithWatch) ([]remote.Option, error) {
	opts, err := GetRemoteWriteOptions(ctx, c)
	if err != nil {
		return nil, err
	}

	dialer, err := buildkit.GetRegistryDialer(ctx, c)
	if err != nil {
		return nil, err
	}

	return append(opts, remote.WithTransport(&http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			if strings.HasPrefix(addr, "127.0.0.1") {
				return dialer(ctx, "")
			}
			n := net.Dialer{}
			return n.DialContext(ctx, network, addr)
		},
	})), nil
}
