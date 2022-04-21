package watcher

import (
	"context"
	"fmt"

	"github.com/acorn-io/baaah/pkg/meta"
	"github.com/acorn-io/baaah/pkg/typed"
	"github.com/sirupsen/logrus"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	meta2 "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

func doWatch[T meta.Object](ctx context.Context, watchFunc watchFunc, cb func(obj T) (bool, error)) (cont bool, nonTerminal error, terminal error) {
	result, err := watchFunc()
	if err != nil {
		return false, err, nil
	}
	defer func() {
		result.Stop()
		for range result.ResultChan() {
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return false, nil, fmt.Errorf("terminating watch: %w", ctx.Err())
		case event, open := <-result.ResultChan():
			if !open {
				return false, nil, nil
			}
			switch event.Type {
			case watch.Deleted:
				o := event.Object.DeepCopyObject()
				mo := o.(meta.Object)
				if mo.GetDeletionTimestamp().IsZero() {
					now := metav1.Now()
					mo.SetDeletionTimestamp(&now)
					event.Object = mo
				}
				fallthrough
			case watch.Added, watch.Modified:
				done, err := cb(event.Object.(T))
				if apierrors.IsConflict(err) {
					// if we got a conflict, return a false (not done) and nil for error
					return false, err, nil
				}
				if err != nil {
					return false, nil, err
				}
				if done {
					return true, nil, nil
				}
			}
		}
	}
}

func retryWatch[T meta.Object](ctx context.Context, watchFunc watchFunc, cb func(obj T) (bool, error)) (T, error) {
	var last T
	newCB := func(obj T) (bool, error) {
		last = obj
		return cb(obj)
	}
	for {
		done, err, terminalErr := doWatch(ctx, watchFunc, newCB)
		if err != nil {
			o := typed.New[T]()
			logrus.Debugf("error while watching type %T: %v", o, err)
		}
		if terminalErr != nil {
			return last, terminalErr
		} else if done {
			return last, nil
		} else {
			select {
			case <-ctx.Done():
				return last, ctx.Err()
			default:
			}
		}
	}
}

type Watcher[T meta.Object] struct {
	client client.WithWatch
	scheme *runtime.Scheme
}

func New[T meta.Object](client client.WithWatch) *Watcher[T] {
	return &Watcher[T]{
		client: client,
		scheme: client.Scheme(),
	}
}

func (w *Watcher[T]) newListObj() (client.ObjectList, error) {
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

func (w *Watcher[T]) ByObject(ctx context.Context, obj T, cb func(obj T) (bool, error)) (def T, _ error) {
	return w.ByName(ctx, obj.GetNamespace(), obj.GetName(), cb)
}

func (w *Watcher[T]) ByName(ctx context.Context, namespace, name string, cb func(obj T) (bool, error)) (def T, _ error) {
	listObj, err := w.newListObj()
	if err != nil {
		return def, err
	}

	obj := typed.New[T]()
	if err := w.client.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, obj); apierrors.IsNotFound(err) {
	} else if err != nil {
		return def, err
	} else {
		if done, err := cb(obj); done || err != nil {
			return obj, err
		}
	}

	rev := obj.GetResourceVersion()
	return retryWatch(ctx, func() (watch.Interface, error) {
		return w.client.Watch(ctx, listObj, &client.ListOptions{
			Raw: &metav1.ListOptions{
				ResourceVersion: rev,
			},
			FieldSelector: fields.OneTermEqualSelector("metadata.name", name),
			Namespace:     namespace,
		})
	}, cb)
}

func (w *Watcher[T]) BySelector(ctx context.Context, namespace string, selector labels.Selector, cb func(obj T) (bool, error)) (def T, _ error) {
	listObj, err := w.newListObj()
	if err != nil {
		return def, err
	}

	err = w.client.List(ctx, listObj, &client.ListOptions{
		Namespace:     namespace,
		LabelSelector: selector,
	})
	if err != nil {
		return def, err
	}
	rev := listObj.GetResourceVersion()

	var (
		doneObj T
		doneSet bool
	)
	err = meta2.EachListItem(listObj, func(object runtime.Object) error {
		done, err := cb(object.(T))
		if done {
			doneObj = object.(T)
			doneSet = true
		}
		return err
	})
	if doneSet || err != nil {
		return doneObj, err
	}

	return retryWatch(ctx, func() (watch.Interface, error) {
		return w.client.Watch(ctx, listObj, &client.ListOptions{
			LabelSelector: selector,
			Namespace:     namespace,
			Raw: &metav1.ListOptions{
				ResourceVersion: rev,
			},
		})
	}, cb)
}
