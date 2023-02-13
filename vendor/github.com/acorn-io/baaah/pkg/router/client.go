package router

import (
	"context"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type TriggerRegistry interface {
	Watch(obj runtime.Object, namespace, name string, selector labels.Selector, fields fields.Selector) error
	WatchingGVKs() []schema.GroupVersionKind
}

type client struct {
	reader
	writer
	status
}

func (c *client) Scheme() *runtime.Scheme {
	return c.reader.client.Scheme()
}

func (c *client) RESTMapper() meta.RESTMapper {
	return c.reader.client.RESTMapper()
}

type writer struct {
	client   kclient.Client
	registry TriggerRegistry
}

func (w *writer) DeleteAllOf(ctx context.Context, obj kclient.Object, opts ...kclient.DeleteAllOfOption) error {
	delOpts := &kclient.DeleteAllOfOptions{}
	for _, opt := range opts {
		opt.ApplyToDeleteAllOf(delOpts)
	}
	if err := w.registry.Watch(obj, delOpts.Namespace, "", delOpts.LabelSelector, delOpts.FieldSelector); err != nil {
		return err
	}
	return w.client.DeleteAllOf(ctx, obj, opts...)
}

func (w *writer) Delete(ctx context.Context, obj kclient.Object, opts ...kclient.DeleteOption) error {
	if err := w.registry.Watch(obj, obj.GetNamespace(), obj.GetName(), nil, nil); err != nil {
		return err
	}
	return w.client.Delete(ctx, obj, opts...)
}

func (w *writer) Patch(ctx context.Context, obj kclient.Object, patch kclient.Patch, opts ...kclient.PatchOption) error {
	if err := w.registry.Watch(obj, obj.GetNamespace(), obj.GetName(), nil, nil); err != nil {
		return err
	}
	return w.client.Patch(ctx, obj, patch, opts...)
}

func (w *writer) Update(ctx context.Context, obj kclient.Object, opts ...kclient.UpdateOption) error {
	if err := w.registry.Watch(obj, obj.GetNamespace(), obj.GetName(), nil, nil); err != nil {
		return err
	}
	return w.client.Update(ctx, obj, opts...)
}

func (w *writer) Create(ctx context.Context, obj kclient.Object, opts ...kclient.CreateOption) error {
	if err := w.registry.Watch(obj, obj.GetNamespace(), obj.GetName(), nil, nil); err != nil {
		return err
	}
	return w.client.Create(ctx, obj, opts...)
}

type statusClient struct {
	client   kclient.Client
	registry TriggerRegistry
}

type status struct {
	client   kclient.Client
	registry TriggerRegistry
}

func (s *status) Status() kclient.StatusWriter {
	return &statusClient{
		client:   s.client,
		registry: s.registry,
	}
}

func (s *statusClient) Update(ctx context.Context, obj kclient.Object, opts ...kclient.UpdateOption) error {
	if err := s.registry.Watch(obj, obj.GetNamespace(), obj.GetName(), nil, nil); err != nil {
		return err
	}
	return s.client.Status().Update(ctx, obj, opts...)
}

func (s *statusClient) Patch(ctx context.Context, obj kclient.Object, patch kclient.Patch, opts ...kclient.PatchOption) error {
	if err := s.registry.Watch(obj, obj.GetNamespace(), obj.GetName(), nil, nil); err != nil {
		return err
	}
	return s.client.Status().Patch(ctx, obj, patch, opts...)
}

type reader struct {
	scheme   *runtime.Scheme
	client   kclient.Client
	registry TriggerRegistry
}

func (a *reader) Get(ctx context.Context, key kclient.ObjectKey, obj kclient.Object) error {
	if err := a.registry.Watch(obj, key.Namespace, key.Name, nil, nil); err != nil {
		return err
	}

	return a.client.Get(ctx, key, obj)
}

func (a *reader) List(ctx context.Context, list kclient.ObjectList, opts ...kclient.ListOption) error {
	listOpt := &kclient.ListOptions{}
	for _, opt := range opts {
		opt.ApplyToList(listOpt)
	}

	if err := a.registry.Watch(list, listOpt.Namespace, "", listOpt.LabelSelector, listOpt.FieldSelector); err != nil {
		return err
	}

	return a.client.List(ctx, list, listOpt)
}
