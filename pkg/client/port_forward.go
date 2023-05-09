package client

import (
	"context"
	"net"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/scheme"
	"github.com/sirupsen/logrus"
)

func (c *DefaultClient) execPortForward(container *apiv1.ContainerReplica, port int) (PortForwardDialer, error) {
	req := c.RESTClient.Get().
		Namespace(container.Namespace).
		Resource("containerreplicas").
		Name(container.Name).
		SubResource("portforward").
		VersionedParams(&apiv1.ContainerReplicaPortForwardOptions{
			Port: port,
		}, scheme.ParameterCodec)

	url := req.URL().String()
	logrus.Debugf("Exec URL: %s", url)
	return func(ctx context.Context) (net.Conn, error) {
		conn, err := c.Dialer.DialMultiplexed(ctx, url, nil)
		if err != nil {
			return nil, err
		}
		return conn.ForStream(0), nil
	}, nil
}

func (c *DefaultClient) ContainerReplicaPortForward(ctx context.Context, containerName string, port int) (PortForwardDialer, error) {
	con, err := c.ContainerReplicaGet(ctx, containerName)
	if err != nil {
		return nil, err
	}

	return c.execPortForward(con, port)
}
