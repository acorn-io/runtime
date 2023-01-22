package client

import (
	"context"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/client/term"
	"github.com/acorn-io/acorn/pkg/scheme"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (c *DefaultClient) execContainer(ctx context.Context, container *apiv1.ContainerReplica, args []string, tty bool, opts *ContainerReplicaExecOptions) (*term.ExecIO, error) {
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

	logrus.Debugf("Exec URL: %s", req.URL().String())
	conn, err := c.Dialer.DialContext(ctx, req.URL().String(), nil)
	if err != nil {
		return nil, err
	}

	return conn.ToExecIO(tty), nil
}

func (c *DefaultClient) ContainerReplicaExec(ctx context.Context, containerName string, args []string, tty bool, opts *ContainerReplicaExecOptions) (*term.ExecIO, error) {
	if containerName == "_" && opts != nil && opts.DebugImage != "" {
		return c.execContainer(ctx, &apiv1.ContainerReplica{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "_",
				Namespace: c.Namespace,
			},
		}, args, tty, opts)
	}
	con, err := c.ContainerReplicaGet(ctx, containerName)
	if err != nil {
		return nil, err
	}

	if opts == nil {
		opts = &ContainerReplicaExecOptions{}
	}

	return c.execContainer(ctx, con, args, tty, opts)
}
