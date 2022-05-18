package client

import (
	"context"
	"encoding/json"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/client/term"
	"github.com/acorn-io/acorn/pkg/scheme"
	"github.com/sirupsen/logrus"
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

	exit := make(chan term.ExitCode, 1)
	go func() {
		exit <- term.ToExitCode(conn.ForStream(3))
	}()

	resize := make(chan term.TermSize, 1)
	go func() {
		for size := range resize {
			data, err := json.Marshal(size)
			if err != nil {
				logrus.Errorf("failed to marshall term size %v: %v", size, err)
				continue
			}
			_, err = conn.Write(4, data)
			if err != nil {
				break
			}
		}
	}()

	return &term.ExecIO{
		Stdin:    conn.ForStream(0),
		Stdout:   conn.ForStream(1),
		Stderr:   conn.ForStream(2),
		ExitCode: exit,
		Resize:   resize,
	}, nil
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
