package portforward

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/acorn-io/acorn/pkg/client"
	"inet.af/tcpproxy"
)

func PortForward(ctx context.Context, c client.Client, containerName string, address string, portDef string) error {
	src, dest, ok := strings.Cut(portDef, ":")
	if !ok {
		dest = src
	}

	port, err := strconv.Atoi(dest)
	if err != nil {
		return err
	}

	dialer, err := c.ContainerReplicaPortForward(ctx, containerName, port)
	if err != nil {
		return err
	}

	p := tcpproxy.Proxy{}
	p.AddRoute(address+":"+src, &tcpproxy.DialProxy{
		DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
			return dialer(ctx)
		},
	})
	p.ListenFunc = func(_, laddr string) (net.Listener, error) {
		l, err := net.Listen("tcp", laddr)
		if err != nil {
			return nil, err
		}
		fmt.Printf("Forwarding %s => %d for container [%s]\n", l.Addr().String(), port, containerName)
		return l, err
	}
	go func() {
		<-ctx.Done()
		_ = p.Close()
	}()
	if err := p.Start(); err != nil {
		return err
	}
	return p.Wait()
}
