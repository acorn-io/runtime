package backend

import (
	"context"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/cache"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type Callback func(gvk schema.GroupVersionKind, key string, obj runtime.Object) (runtime.Object, error)

type Trigger interface {
	Trigger(gvk schema.GroupVersionKind, key string, delay time.Duration) error
}

type Watcher interface {
	Watch(ctx context.Context, gvk schema.GroupVersionKind, name string, cb Callback) error
}

type Backend interface {
	Trigger
	CacheFactory
	Watcher
	kclient.Client

	Start(ctx context.Context) error
	GVKForObject(obj runtime.Object, scheme *runtime.Scheme) (schema.GroupVersionKind, error)
}

type CacheFactory interface {
	GetInformerForKind(ctx context.Context, gvk schema.GroupVersionKind) (cache.SharedIndexInformer, error)
}
