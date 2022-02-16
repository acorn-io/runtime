package waiter

import (
	"context"
	"fmt"

	"github.com/ibuildthecloud/baaah/pkg/meta"
	"github.com/ibuildthecloud/baaah/pkg/typed"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

var (
	WatchTimeoutSeconds int64 = 120
)

type watchFunc func() (watch.Interface, error)

func doWatch[T meta.Object](ctx context.Context, watchFunc watchFunc, cb func(obj T) bool) (bool, error) {
	result, err := watchFunc()
	if err != nil {
		return false, err
	}
	defer func() {
		result.Stop()
		for range result.ResultChan() {
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return false, fmt.Errorf("timeout waiting condition: %w", ctx.Err())
		case event, open := <-result.ResultChan():
			if !open {
				return false, nil
			}
			switch event.Type {
			case watch.Added, watch.Modified, watch.Deleted:
				done := cb(event.Object.(T))
				if done {
					if apierrors.IsConflict(err) {
						// if we got a conflict, return a false (not done) and nil for error
						return false, nil
					}
					if err != nil {
						return false, err
					}
					return true, nil
				}
			}
		}
	}
}

func retryWatch[T meta.Object](ctx context.Context, watchFunc watchFunc, cb func(obj T) bool) (T, error) {
	var last T
	newCB := func(obj T) bool {
		last = obj
		return cb(obj)
	}
	for {
		if done, err := doWatch(ctx, watchFunc, newCB); err != nil {
			return last, err
		} else if done {
			return last, nil
		}
	}
}

type Waiter[T meta.Object] struct {
	client client.WithWatch
	scheme *runtime.Scheme
}

func New[T meta.Object](client client.WithWatch) *Waiter[T] {
	return &Waiter[T]{
		client: client,
		scheme: client.Scheme(),
	}
}

func (w *Waiter[T]) newListObj() (client.ObjectList, error) {
	obj := typed.New[T]()
	gvk, err := apiutil.GVKForObject(obj, w.scheme)
	if err != nil {
		return nil, err
	}
	gvk.Kind += "List"
	listObj, err := w.scheme.New(gvk)
	if err != nil {
		return nil, err
	}
	clientListObj, ok := listObj.(client.ObjectList)
	if !ok {
		return nil, fmt.Errorf("%T is not a client.ObjectList", listObj)
	}
	return clientListObj, nil
}

func (w *Waiter[T]) ByName(ctx context.Context, namespace, name string, cb func(obj T) bool) (def T, _ error) {
	listObj, err := w.newListObj()
	if err != nil {
		return def, err
	}

	return retryWatch(ctx, func() (watch.Interface, error) {
		return w.client.Watch(ctx, listObj, &client.ListOptions{
			FieldSelector: fields.OneTermEqualSelector("metadata.name", name),
			Namespace:     namespace,
		})
	}, cb)
}

func (w *Waiter[T]) BySelector(ctx context.Context, namespace string, selector labels.Selector, cb func(obj T) bool) (def T, _ error) {
	listObj, err := w.newListObj()
	if err != nil {
		return def, err
	}

	return retryWatch(ctx, func() (watch.Interface, error) {
		return w.client.Watch(ctx, listObj, &client.ListOptions{
			LabelSelector: selector,
			Namespace:     namespace,
		})
	}, cb)
}
