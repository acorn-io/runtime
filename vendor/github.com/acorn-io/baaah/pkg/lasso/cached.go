package lasso

import (
	"context"
	"strconv"
	"sync"
	"time"

	"github.com/acorn-io/baaah/pkg/uncached"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

const (
	cacheDuration = 10 * time.Second
)

type objectKey struct {
	gvk             schema.GroupVersionKind
	namespace, name string
}

type objectValue struct {
	Object   kclient.Object
	Inserted time.Time
}

type cacheClient struct {
	uncached, cached kclient.Client

	recent     map[objectKey]objectValue
	recentLock sync.Mutex
}

func newer(oldRV, newRV string) bool {
	if len(oldRV) == len(newRV) {
		return oldRV < newRV
	}
	oldI, err := strconv.Atoi(oldRV)
	if err != nil {
		return true
	}
	newI, err := strconv.Atoi(newRV)
	if err != nil {
		return false
	}
	return oldI < newI
}

func newCacheClient(uncached, cached kclient.Client) *cacheClient {
	return &cacheClient{
		uncached: uncached,
		cached:   cached,
		recent:   map[objectKey]objectValue{},
	}
}

func (c *cacheClient) startPurge(ctx context.Context) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(cacheDuration):
			}

			now := time.Now()
			c.recentLock.Lock()
			for k, v := range c.recent {
				if v.Inserted.Add(cacheDuration).Before(now) {
					delete(c.recent, k)
				}
			}
			c.recentLock.Unlock()
		}
	}()
}

func (c *cacheClient) deleteStore(obj kclient.Object) {
	gvk, err := apiutil.GVKForObject(obj, c.Scheme())
	if err != nil {
		return
	}
	c.recentLock.Lock()
	delete(c.recent, objectKey{
		gvk:       gvk,
		namespace: obj.GetNamespace(),
		name:      obj.GetName(),
	})
	c.recentLock.Unlock()
}

func (c *cacheClient) store(obj kclient.Object) {
	gvk, err := apiutil.GVKForObject(obj, c.Scheme())
	if err != nil {
		return
	}
	c.recentLock.Lock()
	c.recent[objectKey{
		gvk:       gvk,
		namespace: obj.GetNamespace(),
		name:      obj.GetName(),
	}] = objectValue{
		Object:   obj,
		Inserted: time.Now(),
	}
	c.recentLock.Unlock()
}

func (c *cacheClient) Get(ctx context.Context, key kclient.ObjectKey, obj kclient.Object) error {
	if u, ok := obj.(*uncached.Holder); ok {
		return c.uncached.Get(ctx, key, u.Object)
	}

	err := c.cached.Get(ctx, key, obj)
	if err != nil {
		return err
	}

	gvk, err := apiutil.GVKForObject(obj, c.Scheme())
	if err != nil {
		return err
	}

	cacheKey := objectKey{
		gvk:       gvk,
		namespace: obj.GetNamespace(),
		name:      obj.GetName(),
	}

	c.recentLock.Lock()
	cachedObj, ok := c.recent[cacheKey]
	c.recentLock.Unlock()
	if ok && newer(obj.GetResourceVersion(), cachedObj.Object.GetResourceVersion()) {
		return CopyInto(obj, cachedObj.Object)
	}

	return nil
}

func (c *cacheClient) List(ctx context.Context, list kclient.ObjectList, opts ...kclient.ListOption) error {
	if u, ok := list.(*uncached.HolderList); ok {
		return c.uncached.List(ctx, u.ObjectList, opts...)
	}
	return c.cached.List(ctx, list, opts...)
}

func (c *cacheClient) Create(ctx context.Context, obj kclient.Object, opts ...kclient.CreateOption) error {
	err := c.cached.Create(ctx, obj, opts...)
	if err != nil {
		return err
	}
	c.store(obj)
	return nil
}

func (c *cacheClient) Delete(ctx context.Context, obj kclient.Object, opts ...kclient.DeleteOption) error {
	err := c.cached.Delete(ctx, obj, opts...)
	if err != nil {
		return err
	}
	c.deleteStore(obj)
	return nil
}

func (c *cacheClient) Update(ctx context.Context, obj kclient.Object, opts ...kclient.UpdateOption) error {
	err := c.cached.Update(ctx, obj, opts...)
	if err != nil {
		return err
	}
	c.store(obj)
	return nil
}

func (c *cacheClient) Patch(ctx context.Context, obj kclient.Object, patch kclient.Patch, opts ...kclient.PatchOption) error {
	err := c.cached.Patch(ctx, obj, patch, opts...)
	if err != nil {
		return err
	}
	c.store(obj)
	return nil
}

func (c *cacheClient) DeleteAllOf(ctx context.Context, obj kclient.Object, opts ...kclient.DeleteAllOfOption) error {
	return c.cached.DeleteAllOf(ctx, obj, opts...)
}

func (c *cacheClient) Status() kclient.StatusWriter {
	return &statusWriter{
		c:      c,
		status: c.cached.Status(),
	}
}

func (c *cacheClient) Scheme() *runtime.Scheme {
	return c.cached.Scheme()
}

func (c *cacheClient) RESTMapper() meta.RESTMapper {
	return c.cached.RESTMapper()
}

type statusWriter struct {
	c      *cacheClient
	status kclient.StatusWriter
}

func (s *statusWriter) Update(ctx context.Context, obj kclient.Object, opts ...kclient.UpdateOption) error {
	err := s.status.Update(ctx, obj, opts...)
	if err != nil {
		return err
	}
	s.c.store(obj)
	return nil
}

func (s *statusWriter) Patch(ctx context.Context, obj kclient.Object, patch kclient.Patch, opts ...kclient.PatchOption) error {
	err := s.status.Patch(ctx, obj, patch, opts...)
	if err != nil {
		return err
	}
	s.c.store(obj)
	return nil
}
