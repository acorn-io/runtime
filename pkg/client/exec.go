package client

import (
	"context"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/client/term"
	"github.com/acorn-io/acorn/pkg/scheme"
)

func (c *client) execContainer(ctx context.Context, container *apiv1.ContainerReplica, args []string, tty bool, opts *ContainerReplicaExecOptions) (*term.ExecIO, error) {
	req := c.RESTClient.Get().
		Namespace(container.Namespace).
		Resource("containerreplicas").
		Name(container.Name).
		SubResource("exec").
		VersionedParams(&apiv1.ContainerReplicaExecOptions{
			TTY:        tty,
			Command:    args,
			DebugImage: opts.DebugImage,
		}, scheme.ParameterCodec)

	conn, err := c.Dialer.DialContext(ctx, req.URL().String(), nil)
	if err != nil {
		return nil, err
	}

	return conn.ToExecIO(tty), nil
}

func (c *client) ContainerReplicaExec(ctx context.Context, containerName string, args []string, tty bool, opts *ContainerReplicaExecOptions) (*term.ExecIO, error) {
	con, err := c.ContainerReplicaGet(ctx, containerName)
	if err != nil {
		return nil, err
	}

	if opts == nil {
		opts = &ContainerReplicaExecOptions{}
	}

	return c.execContainer(ctx, con, args, tty, opts)
}
