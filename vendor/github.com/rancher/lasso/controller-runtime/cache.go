package controllerruntime

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	lcache "github.com/rancher/lasso/pkg/cache"
	"github.com/rancher/lasso/pkg/dynamic"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	kcache "k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

type Cache struct {
	startLock    sync.RWMutex
	started      bool
	schema       *runtime.Scheme
	cacheFactory lcache.SharedCacheFactory
	dynamic      *dynamic.Controller
}

func NewNewCacheFunc(cacheFactory lcache.SharedCacheFactory, dynamic *dynamic.Controller) cache.NewCacheFunc {
	return func(config *rest.Config, opts cache.Options) (cache.Cache, error) {
		s := opts.Scheme
		if s == nil {
			s = scheme.Scheme
		}
		return &Cache{
			schema:       s,
			cacheFactory: cacheFactory,
			dynamic:      dynamic,
		}, nil
	}
}

func (c *Cache) Get(ctx context.Context, key client.ObjectKey, out client.Object) error {
	gvk, err := apiutil.GVKForObject(out, c.schema)
	if err != nil {
		return err
	}
	reader, err := c.getReader(ctx, gvk)
	if err != nil {
		return err
	}
	return reader.Get(ctx, key, out)
}

func (c *Cache) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	gvk, err := apiutil.GVKForObject(list, c.schema)
	if err != nil {
		return err
	}
	if strings.HasSuffix(gvk.Kind, "List") {
		gvk.Kind = gvk.Kind[:len(gvk.Kind)-4]
	}

	reader, err := c.getReader(ctx, gvk)
	if err != nil {
		return err
	}

	return reader.List(ctx, list, opts...)
}

func (c *Cache) GetInformer(ctx context.Context, obj client.Object) (cache.Informer, error) {
	gvk, err := apiutil.GVKForObject(obj, c.schema)
	if err != nil {
		return nil, err
	}
	return c.GetInformerForKind(ctx, gvk)
}

func (c *Cache) GetInformerForKind(ctx context.Context, gvk schema.GroupVersionKind) (cache.Informer, error) {
	var (
		errNotStarted *cache.ErrCacheNotStarted
	)

	informer, err := c.getInformer(ctx, gvk)
	if errors.As(err, &errNotStarted) {
		return informer, nil
	} else if err != nil {
		return nil, err
	}
	return informer, err
}

func (c *Cache) Start(ctx context.Context) error {
	c.startLock.Lock()
	defer c.startLock.Unlock()
	if err := c.cacheFactory.Start(ctx); err != nil {
		return err
	}
	c.started = true
	return nil
}

func (c *Cache) WaitForCacheSync(ctx context.Context) bool {
	all := c.cacheFactory.WaitForCacheSync(ctx)
	for _, v := range all {
		if !v {
			return false
		}
	}
	return true
}

func (c *Cache) IndexField(ctx context.Context, obj client.Object, field string, extractValue client.IndexerFunc) error {
	informer, err := c.GetInformer(ctx, obj)
	if err != nil {
		return err
	}
	return indexByField(informer, field, extractValue)
}

func (c *Cache) getReader(ctx context.Context, gvk schema.GroupVersionKind) (*CacheReader, error) {
	informer, err := c.getInformer(ctx, gvk)
	if err != nil {
		return nil, err
	}

	nsed, err := c.cacheFactory.SharedClientFactory().IsNamespaced(gvk)
	if err != nil {
		return nil, err
	}

	reader := &CacheReader{
		indexer:          informer.GetIndexer(),
		groupVersionKind: gvk,
		scopeName:        meta.RESTScopeNameRoot,
	}
	if nsed {
		reader.scopeName = meta.RESTScopeNameNamespace
	}
	return reader, nil
}

func (c *Cache) getGVK(out interface{}) (schema.GroupVersionKind, error) {
	if out, ok := out.(client.ObjectList); ok {
		gvk, err := apiutil.GVKForObject(out, c.schema)
		if err != nil {
			return schema.GroupVersionKind{}, err
		}
		gvk.Kind = gvk.Kind[:len(gvk.Kind)-4]
		return gvk, nil
	}

	if out, ok := out.(client.Object); ok {
		return apiutil.GVKForObject(out, c.schema)
	}

	return schema.GroupVersionKind{}, fmt.Errorf("unknown kind for %T", out)
}

func (c *Cache) getInformer(ctx context.Context, gvk schema.GroupVersionKind) (kcache.SharedIndexInformer, error) {
	var (
		informer kcache.SharedIndexInformer
		err      error
		started  bool
	)

	if c.schema.Recognizes(gvk) {
		informer, err = c.cacheFactory.ForKind(gvk)
		if informer != nil {
			started = informer.HasSynced()
		}
	} else {
		informer, started, err = c.dynamic.GetCache(ctx, gvk)
	}
	if err != nil {
		return nil, err
	} else if !started {
		c.startLock.RLock()
		started := c.started
		c.startLock.RUnlock()
		if started {
			if err := c.cacheFactory.StartGVK(ctx, gvk); err != nil {
				return nil, err
			}
			c.cacheFactory.WaitForCacheSync(ctx)
			return informer, nil
		} else {
			return informer, &cache.ErrCacheNotStarted{}
		}
	}

	return informer, nil
}
