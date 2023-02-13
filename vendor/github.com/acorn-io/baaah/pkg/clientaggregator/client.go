package clientaggregator

import (
	"context"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

type Client struct {
	defaultClient kclient.WithWatch
	perGroup      map[string]kclient.WithWatch
	perGroupKind  map[schema.GroupKind]kclient.WithWatch
}

func New(c kclient.WithWatch) *Client {
	return &Client{
		defaultClient: c,
		perGroup:      map[string]kclient.WithWatch{},
		perGroupKind:  map[schema.GroupKind]kclient.WithWatch{},
	}
}

func (c *Client) AddGroup(group string, client kclient.WithWatch) {
	c.perGroup[group] = client
}

func (c *Client) AddGroupKind(groupKind schema.GroupKind, client kclient.WithWatch) {
	c.perGroupKind[groupKind] = client
}

func (c *Client) getClient(obj runtime.Object) kclient.WithWatch {
	gvk, err := apiutil.GVKForObject(obj, c.defaultClient.Scheme())
	if c, ok := c.perGroup[gvk.Group]; err == nil && ok {
		return c
	}
	if c, ok := c.perGroupKind[gvk.GroupKind()]; err == nil && ok {
		return c
	}
	return c.defaultClient
}

func (c *Client) Get(ctx context.Context, key kclient.ObjectKey, obj kclient.Object) error {
	return c.getClient(obj).Get(ctx, key, obj)
}

func (c *Client) List(ctx context.Context, list kclient.ObjectList, opts ...kclient.ListOption) error {
	return c.getClient(list).List(ctx, list, opts...)
}

func (c *Client) Create(ctx context.Context, obj kclient.Object, opts ...kclient.CreateOption) error {
	return c.getClient(obj).Create(ctx, obj, opts...)
}

func (c *Client) Delete(ctx context.Context, obj kclient.Object, opts ...kclient.DeleteOption) error {
	return c.getClient(obj).Delete(ctx, obj, opts...)
}

func (c *Client) Update(ctx context.Context, obj kclient.Object, opts ...kclient.UpdateOption) error {
	return c.getClient(obj).Update(ctx, obj, opts...)
}

func (c *Client) Patch(ctx context.Context, obj kclient.Object, patch kclient.Patch, opts ...kclient.PatchOption) error {
	return c.getClient(obj).Patch(ctx, obj, patch, opts...)
}

func (c *Client) DeleteAllOf(ctx context.Context, obj kclient.Object, opts ...kclient.DeleteAllOfOption) error {
	return c.getClient(obj).DeleteAllOf(ctx, obj, opts...)
}

func (c *Client) Status() kclient.StatusWriter {
	return &StatusWriter{c}
}

func (c *Client) Scheme() *runtime.Scheme {
	return c.defaultClient.Scheme()
}

func (c *Client) RESTMapper() meta.RESTMapper {
	panic("not implemented")
}

func (c *Client) Watch(ctx context.Context, obj kclient.ObjectList, opts ...kclient.ListOption) (watch.Interface, error) {
	return c.getClient(obj).Watch(ctx, obj, opts...)
}

type StatusWriter struct {
	c *Client
}

func (s *StatusWriter) Update(ctx context.Context, obj kclient.Object, opts ...kclient.UpdateOption) error {
	return s.c.getClient(obj).Status().Update(ctx, obj, opts...)
}

func (s *StatusWriter) Patch(ctx context.Context, obj kclient.Object, patch kclient.Patch, opts ...kclient.PatchOption) error {
	return s.c.getClient(obj).Status().Patch(ctx, obj, patch, opts...)
}
