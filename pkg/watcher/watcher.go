package watcher

import (
	"context"
	"fmt"
	"time"

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

func doWatch[T client.Object](ctx context.Context, watchFunc watchFunc, cb func(obj T) (bool, error)) (cont bool, nonTerminal error, terminal error) {
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
			o := typed.New[T]()
			return false, nil, fmt.Errorf("terminating watch on type %T: %w", o, ctx.Err())
		case event, open := <-result.ResultChan():

			if !open {
				return false, nil, nil
			}
			switch event.Type {
			case watch.Deleted:
				o := event.Object.DeepCopyObject()
				mo := o.(client.Object)
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
			case watch.Error:
				return false, nil, apierrors.FromObject(event.Object)
			}
		}
	}
}

func retryWatch[T client.Object](ctx context.Context, watchFunc watchFunc, cb func(obj T) (bool, error)) (T, error) {
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
		}
		select {
		case <-ctx.Done():
			return last, ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
}

type Watcher[T client.Object] struct {
	client client.WithWatch
	scheme *runtime.Scheme
}

func New[T client.Object](client client.WithWatch) *Watcher[T] {
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
	return w.bySelector(ctx, namespace, nil, fields.SelectorFromSet(map[string]string{
		"metadata.name": name,
	}), cb)
}

func (w *Watcher[T]) BySelector(ctx context.Context, namespace string, selector labels.Selector, cb func(obj T) (bool, error)) (def T, _ error) {
	return w.bySelector(ctx, namespace, selector, nil, cb)
}

func (w *Watcher[T]) bySelector(ctx context.Context, namespace string, selector labels.Selector, fieldSelector fields.Selector, cb func(obj T) (bool, error)) (def T, _ error) {
	listObj, err := w.newListObj()
	if err != nil {
		return def, err
	}

	logrus.Debugf("&&&&&&&&&!!!!!!!!!!!!!!!!!!!!!!!!!!! Selecting %v %v %v", namespace, selector, fieldSelector)
	err = w.client.List(ctx, listObj, &client.ListOptions{
		Namespace:     namespace,
		LabelSelector: selector,
		FieldSelector: fieldSelector,
	})
	if err != nil {
		return def, err
	}
	rev := listObj.GetResourceVersion()

	logrus.Debugf("&!&!&!&&!&!&!&!&&!&!&!&&!&!&!&!&&!&!& count %v", listObj.GetRemainingItemCount())

	var (
		doneObj T
		doneSet bool
	)
	err = meta2.EachListItem(listObj, func(object runtime.Object) error {
		logrus.Debugf("CHECKING DONE ..................... %#v", object)
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
			FieldSelector: fieldSelector,
			Namespace:     namespace,
			Raw: &metav1.ListOptions{
				ResourceVersion: rev,
			},
		})
	}, cb)
}
