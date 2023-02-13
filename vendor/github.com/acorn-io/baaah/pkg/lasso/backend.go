package lasso

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/acorn-io/baaah/pkg/backend"
	"github.com/acorn-io/baaah/pkg/fields"
	"github.com/acorn-io/baaah/pkg/router"
	"github.com/acorn-io/baaah/pkg/uncached"
	controllerruntime "github.com/rancher/lasso/controller-runtime"
	"github.com/rancher/lasso/pkg/controller"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	kcache "k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

type Backend struct {
	*cacheClient

	cacheFactory controller.SharedControllerFactory
	cache        cache.Cache
	started      atomic.Bool
}

func newBackend(cacheFactory controller.SharedControllerFactory, client *cacheClient, cache cache.Cache) *Backend {
	return &Backend{
		cacheClient:  client,
		cacheFactory: cacheFactory,
		cache:        cache,
	}
}

func (b *Backend) Start(ctx context.Context) (err error) {
	defer func() {
		if err == nil {
			b.started.Store(true)
		}
	}()
	if err := b.cacheFactory.Start(ctx, 5); err != nil {
		return err
	}
	go func() {
		if err := b.cache.Start(ctx); err != nil {
			panic(err)
		}
	}()
	if !b.cache.WaitForCacheSync(ctx) {
		return fmt.Errorf("failed to wait for caches to sync")
	}
	if !b.started.Load() {
		b.cacheClient.startPurge(ctx)
	}
	return nil
}

func (b *Backend) Trigger(gvk schema.GroupVersionKind, key string, delay time.Duration) error {
	controller, err := b.cacheFactory.ForKind(gvk)
	if err != nil {
		return err
	}
	if delay > 0 {
		ns, name, ok := strings.Cut(key, "/")
		if ok {
			controller.EnqueueAfter(ns, name, delay)
		} else {
			controller.EnqueueAfter("", key, delay)
		}
	} else {
		controller.EnqueueKey(router.TriggerPrefix + key)
	}
	return nil
}

func (b *Backend) addIndexer(ctx context.Context, gvk schema.GroupVersionKind) error {
	obj, err := b.Scheme().New(gvk)
	if err != nil {
		return err
	}
	f, ok := obj.(fields.Fields)
	if !ok {
		return nil
	}
	caches := b.cacheFactory.SharedCacheFactory().WaitForCacheSync(ctx)
	if caches[gvk] {
		return nil
	}

	cache, err := b.cacheFactory.SharedCacheFactory().ForKind(gvk)
	if err != nil {
		return err
	}

	indexers := map[string]kcache.IndexFunc{}
	for _, field := range f.FieldNames() {
		field := field
		indexers["field:"+field] = kcache.IndexFunc(func(obj interface{}) ([]string, error) {
			f, ok := obj.(fields.Fields)
			if !ok {
				return nil, nil
			}
			v := f.Get(field)
			if v == "" {
				return nil, nil
			}
			vals := []string{controllerruntime.KeyToNamespacedKey("", v)}
			if ko, ok := obj.(kclient.Object); ok && ko.GetNamespace() != "" {
				vals = append(vals, controllerruntime.KeyToNamespacedKey(ko.GetNamespace(), v))
			}
			return vals, nil
		})
	}
	return cache.AddIndexers(indexers)
}

func (b *Backend) Watch(ctx context.Context, gvk schema.GroupVersionKind, name string, cb backend.Callback) error {
	c, err := b.cacheFactory.ForKind(gvk)
	if err != nil {
		return err
	}
	if err := b.addIndexer(ctx, gvk); err != nil {
		return err
	}
	handler := controller.SharedControllerHandlerFunc(func(key string, obj runtime.Object) (runtime.Object, error) {
		return cb(gvk, key, obj)
	})
	c.RegisterHandler(ctx, fmt.Sprintf("%s %v", name, gvk), handler)

	if b.started.Load() {
		return c.Start(ctx, 5)
	}
	return nil
}

func (b *Backend) GVKForObject(obj runtime.Object, scheme *runtime.Scheme) (schema.GroupVersionKind, error) {
	return apiutil.GVKForObject(uncached.Unwrap(obj), scheme)
}

func (b *Backend) GetInformerForKind(ctx context.Context, gvk schema.GroupVersionKind) (kcache.SharedIndexInformer, error) {
	return b.cacheFactory.SharedCacheFactory().ForKind(gvk)
}
