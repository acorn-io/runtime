package client

import (
	"context"
	"fmt"
	"net"
	"time"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/portforwarder"
	"github.com/acorn-io/acorn/pkg/scheme"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	client2 "sigs.k8s.io/controller-runtime/pkg/client"
)

func (c *client) BuilderCreate(ctx context.Context) (*apiv1.Builder, error) {
	builder := &apiv1.Builder{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "builder",
			Namespace: c.Namespace,
		},
	}
	return builder, c.Client.Create(ctx, builder)
}

func (c *client) BuilderDelete(ctx context.Context) (*apiv1.Builder, error) {
	builder, err := c.BuilderGet(ctx)
	if apierrors.IsNotFound(err) {
		return nil, nil
	}

	return builder, c.Client.Delete(ctx, &apiv1.App{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "builder",
			Namespace: c.Namespace,
		},
	})
}

func (c *client) BuilderGet(ctx context.Context) (*apiv1.Builder, error) {
	builder := &apiv1.Builder{}
	err := c.Client.Get(ctx, client2.ObjectKey{
		Name:      "builder",
		Namespace: c.Namespace,
	}, builder)
	if err != nil {
		return nil, err
	}

	return builder, nil
}

func (c *client) builderCreatePrint(ctx context.Context) (*apiv1.Builder, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	go func() {
		select {
		case <-ctx.Done():
		case <-time.After(3 * time.Second):
			fmt.Print("Waiting for builder to start... ")
			<-ctx.Done()
			fmt.Println("Ready")
		}
	}()
	return c.BuilderCreate(ctx)
}

func (c *client) BuilderDialer(ctx context.Context) (func(ctx context.Context) (net.Conn, error), error) {
	builder, err := c.BuilderGet(ctx)
	if apierrors.IsNotFound(err) {
		builder, err = c.builderCreatePrint(ctx)
		if err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}

	req := c.RESTClient.Get().
		Namespace(builder.Namespace).
		Resource("builders").
		Name(builder.Name).
		SubResource("port").
		VersionedParams(&apiv1.BuilderPortOptions{}, scheme.ParameterCodec)

	dialer, err := portforwarder.NewWebSocketDialerForURL(c.RESTConfig, req.URL())
	if err != nil {
		return nil, err
	}
	return func(ctx context.Context) (net.Conn, error) {
		return dialer.DialContext(ctx, "")
	}, nil
}

func (c *client) BuilderRegistryDialer(ctx context.Context) (func(ctx context.Context) (net.Conn, error), error) {
	builder, err := c.BuilderGet(ctx)
	if apierrors.IsNotFound(err) {
		builder, err = c.builderCreatePrint(ctx)
		if err != nil {
			return nil, err
		}
	}

	req := c.RESTClient.Get().
		Namespace(builder.Namespace).
		Resource("builders").
		Name(builder.Name).
		SubResource("registryport").
		VersionedParams(&apiv1.BuilderPortOptions{}, scheme.ParameterCodec)

	dialer, err := portforwarder.NewWebSocketDialerForURL(c.RESTConfig, req.URL())
	if err != nil {
		return nil, err
	}
	return func(ctx context.Context) (net.Conn, error) {
		return dialer.DialContext(ctx, "")
	}, nil
}
