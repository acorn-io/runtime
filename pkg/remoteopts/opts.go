package remoteopts

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/acorn-io/acorn/pkg/build/buildkit"
	"github.com/acorn-io/acorn/pkg/client"
	"github.com/acorn-io/acorn/pkg/portforwarder"
	"github.com/acorn-io/acorn/pkg/system"
	"github.com/acorn-io/baaah/pkg/restconfig"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func Common(ctx context.Context) ([]remote.Option, error) {
	return []remote.Option{
		remote.WithContext(ctx),
	}, nil
}

func WithClientDialer(ctx context.Context, c client.Client) ([]remote.Option, error) {
	opts, err := Common(ctx)
	if err != nil {
		return nil, err
	}

	dialer, err := c.BuilderRegistryDialer(ctx)
	if err != nil {
		return nil, err
	}

	return append(opts, remote.WithTransport(&http.Transport{
		MaxIdleConns: -1,
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			n := net.Dialer{}
			if strings.HasPrefix(addr, "127.0.0.") {
				return dialer(ctx)
			}
			return n.DialContext(ctx, network, addr)
		},
	})), nil
}

func WithServerDialer(ctx context.Context, c kclient.WithWatch) ([]remote.Option, error) {
	opts, err := Common(ctx)
	if err != nil {
		return nil, err
	}

	// This is for dev/unit tests where k8s dns is not available
	if os.Getenv("KUBERNETES_SERVICE_HOST") == "" {
		cfg, err := restconfig.Default()
		if err != nil {
			return nil, err
		}

		_, pod, err := buildkit.GetBuildkitPod(ctx, c)
		if err != nil {
			return nil, err
		}

		dialer, err := portforwarder.NewWebSocketDialer(cfg, pod, uint32(system.RegistryPort))
		if err != nil {
			return nil, err
		}

		return append(opts, remote.WithTransport(&http.Transport{
			MaxIdleConns: -1,
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				n := net.Dialer{}
				if strings.HasPrefix(addr, "127.0.0.") {
					return dialer.DialContext(ctx, addr)
				}
				return n.DialContext(ctx, network, addr)
			},
		})), nil
	}

	return append(opts, remote.WithTransport(&http.Transport{
		MaxIdleConns: -1,
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			n := net.Dialer{}
			if strings.HasPrefix(addr, "127.0.0.") {
				return n.DialContext(ctx, network, fmt.Sprintf("%s.%s:%d", system.RegistryName, system.Namespace, system.RegistryPort))
			}
			return n.DialContext(ctx, network, addr)
		},
	})), nil
}
