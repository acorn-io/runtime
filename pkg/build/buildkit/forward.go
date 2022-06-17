package buildkit

import (
	"context"
	"net"

	"github.com/acorn-io/acorn/pkg/portforwarder"
	"github.com/acorn-io/acorn/pkg/system"
	"github.com/acorn-io/baaah/pkg/restconfig"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type DialerContext func(ctx context.Context, address string) (net.Conn, error)

func GetRegistryDialer(ctx context.Context, client kclient.WithWatch) (DialerContext, error) {
	_, pod, err := GetBuildkitPod(ctx, client)
	if err != nil {
		return nil, err
	}

	cfg, err := restconfig.New(client.Scheme())
	if err != nil {
		return nil, err
	}

	dialer, err := portforwarder.NewWebSocketDialer(cfg, pod, uint32(system.RegistryPort))
	if err != nil {
		return nil, err
	}
	return dialer.DialContext, nil
}
