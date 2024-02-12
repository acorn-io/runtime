package portforward

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/acorn-io/runtime/pkg/client"
	"inet.af/tcpproxy"
)

func PortForward(ctx context.Context, c client.Client, containerName string, address string, portDef string) error {
	var anyPort bool
	src, dest, ok := strings.Cut(portDef, ":")
	if !ok {
		dest = src
		anyPort = true
	} else if src == dest {
		anyPort = true
	}

	port, err := strconv.Atoi(dest)
	if err != nil {
		return err
	}

	var (
		listener      net.Listener
		listenAddress = address + ":" + src
		// this is only used when anyPort is true which assumes dest == src
		currentSrcPort = port
	)

	for {
		l, err := net.Listen("tcp", listenAddress)
		if err != nil && anyPort && strings.Contains(err.Error(), "address already in use") {
			currentSrcPort++
			listenAddress = fmt.Sprintf("%s:%d", address, currentSrcPort)
			continue
		} else if err != nil {
			return err
		}
		listener = l
		defer listener.Close()
		break
	}

	dialer, err := c.ContainerReplicaPortForward(ctx, containerName, port)
	if err != nil {
		return err
	}

	p := tcpproxy.Proxy{}
	p.AddRoute(listenAddress, &tcpproxy.DialProxy{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return dialer(ctx)
		},
	})
	p.ListenFunc = func(_, _ string) (net.Listener, error) {
		fmt.Printf("Forwarding %s => %d for container [%s]\n", listener.Addr().String(), port, containerName)
		return listener, err
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
