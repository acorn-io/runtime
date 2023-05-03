package cli

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"

	cli "github.com/acorn-io/acorn/pkg/cli/builder"
	"github.com/acorn-io/acorn/pkg/client"
	"github.com/spf13/cobra"
	"inet.af/tcpproxy"
)

func NewPortForward(c CommandContext) *cobra.Command {
	exec := &PortForward{client: c.ClientFactory}
	cmd := cli.Command(exec, cobra.Command{
		Use:               "port-forward [flags] APP_NAME|CONTAINER_NAME PORT",
		SilenceUsage:      true,
		Short:             "Forward a container port locally",
		Long:              "Forward a container port locally",
		ValidArgsFunction: newCompletion(c.ClientFactory, onlyAppsWithAcornContainer(exec.Container)).complete,
		Args:              cobra.ExactArgs(2),
	})

	// This will produce an error if the container flag doesn't exist or a completion function has already
	// been registered for this flag. Not returning the error since neither of these is likely occur.
	if err := cmd.RegisterFlagCompletionFunc("container", newCompletion(c.ClientFactory, acornContainerCompletion).complete); err != nil {
		cmd.Printf("Error registering completion function for -c flag: %v\n", err)
	}

	return cmd
}

type PortForward struct {
	Container string `usage:"Name of container to port forward into" short:"c"`
	Address   string `usage:"The IP address to listen on" default:"127.0.0.1"`
	client    ClientFactory
}

func (s *PortForward) forwardPort(ctx context.Context, c client.Client, containerName string, portDef string) error {
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
	p.AddRoute(s.Address+":"+src, &tcpproxy.DialProxy{
		DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
			return dialer(ctx)
		},
	})
	p.ListenFunc = func(_, laddr string) (net.Listener, error) {
		l, err := net.Listen("tcp", laddr)
		if err != nil {
			return nil, err
		}
		fmt.Printf("Forwarding %s => %d\n", l.Addr().String(), port)
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

func (s *PortForward) Run(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	c, err := s.client.CreateDefault()
	if err != nil {
		return err
	}

	name, portDef := args[0], args[1]
	if err != nil {
		return err
	}

	app, appErr := c.AppGet(ctx, name)
	if appErr == nil {
		name, err = getContainerForApp(ctx, c, app, s.Container, true)
		if err != nil {
			return err
		}
	}
	return s.forwardPort(ctx, c, name, portDef)
}
