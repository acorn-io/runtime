package buildkit

import (
	"context"
	"net"

	"github.com/ibuildthecloud/baaah/pkg/restconfig"
	"github.com/ibuildthecloud/herd/pkg/portforwarder"
	"github.com/ibuildthecloud/herd/pkg/system"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type DialerContext func(ctx context.Context, address string) (net.Conn, error)

func GetBuildkitDialer(ctx context.Context, client client.WithWatch) (int, DialerContext, error) {
	port, pod, err := GetBuildkitPod(ctx, client)
	if err != nil {
		return 0, nil, err
	}

	cfg, err := restconfig.New(client.Scheme())
	if err != nil {
		return 0, nil, err
	}

	dialer, err := portforwarder.NewWebSocketDialer(cfg, pod, uint32(system.BuildkitPort))
	if err != nil {
		return 0, nil, err
	}
	return port, dialer.DialContext, nil
}
