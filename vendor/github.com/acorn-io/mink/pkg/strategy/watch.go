package strategy

import (
	"context"

	"github.com/acorn-io/mink/pkg/types"
	metainternalversion "k8s.io/apimachinery/pkg/apis/meta/internalversion"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/rest"
	"k8s.io/apiserver/pkg/storage"
)

type watchResult struct {
	ctx    context.Context
	cancel func()
	c      <-chan watch.Event
}

func (w *watchResult) Stop() {
	w.cancel()
}

func (w *watchResult) ResultChan() <-chan watch.Event {
	return w.c
}

var _ rest.Watcher = (*WatchAdapter)(nil)

type Watcher interface {
	Watch(ctx context.Context, namespace string, opts storage.ListOptions) (<-chan watch.Event, error)
	New() types.Object
}

type WatchAdapter struct {
	strategy Watcher
}

func NewWatch(strategy Watcher) *WatchAdapter {
	return &WatchAdapter{
		strategy: strategy,
	}
}

func (w *WatchAdapter) Watch(ctx context.Context, options *metainternalversion.ListOptions) (watch.Interface, error) {
	label := labels.Everything()
	if options != nil && options.LabelSelector != nil {
		label = options.LabelSelector
	}
	field := fields.Everything()
	if options != nil && options.FieldSelector != nil {
		field = options.FieldSelector
	}
	predicate := w.predicate(label, field)

	resourceVersion := ""
	if options != nil {
		resourceVersion = options.ResourceVersion
		predicate.AllowWatchBookmarks = options.AllowWatchBookmarks
	}
	return w.WatchPredicate(ctx, predicate, resourceVersion)
}

func (w *WatchAdapter) WatchPredicate(ctx context.Context, p storage.SelectionPredicate, resourceVersion string) (watch.Interface, error) {
	storageOpts := storage.ListOptions{ResourceVersion: resourceVersion, Predicate: p, Recursive: true}

	ns, _ := request.NamespaceFrom(ctx)
	ctx, cancel := context.WithCancel(ctx)
	c, err := w.strategy.Watch(ctx, ns, storageOpts)
	if err != nil {
		cancel()
		return nil, err
	}

	return &watchResult{
		ctx:    ctx,
		cancel: cancel,
		c:      c,
	}, nil
}

func (w *WatchAdapter) NamespaceScoped() bool {
	if o, ok := w.strategy.(NamespaceScoper); ok {
		return o.NamespaceScoped()
	}
	if o, ok := w.strategy.New().(NamespaceScoper); ok {
		return o.NamespaceScoped()
	}
	return true
}

func (w *WatchAdapter) predicate(label labels.Selector, field fields.Selector) storage.SelectionPredicate {
	result := storage.SelectionPredicate{
		Label: label,
		Field: field,
	}
	if w.NamespaceScoped() {
		result.GetAttrs = storage.DefaultNamespaceScopedAttr
	} else {
		result.GetAttrs = storage.DefaultClusterScopedAttr
	}
	return result
}
