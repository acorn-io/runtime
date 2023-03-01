package helper

import (
	"context"
	"fmt"
	"testing"
	"time"

	hclient "github.com/acorn-io/acorn/pkg/client"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/watch"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	WatchTimeoutSeconds int64 = 240
)

type WatchFunc func(ctx context.Context, obj client.ObjectList, opts ...client.ListOption) (watch.Interface, error)
type watchFunc func() (watch.Interface, error)

// doWatch watches for changes in a Kubernetes API resource and checks if the object satisfies a condition specified
// in the cb function over either context's deadline, or 1 minute if not set.
func doWatch[T client.Object](t *testing.T, watchFunc watchFunc, cb func(obj T) bool) bool {
	t.Helper()

	ctx := GetCTX(t)
	var cancel context.CancelFunc

	if deadline, ok := t.Deadline(); ok {
		ctx, cancel = context.WithDeadline(ctx, deadline)
	} else {
		ctx, cancel = context.WithTimeout(ctx, 1*time.Minute)
	}

	defer cancel()

	result, err := watchFunc()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		result.Stop()
		for range result.ResultChan() {
		}
	}()

	for {
		select {
		case <-ctx.Done():
			t.Fatal(fmt.Errorf("timeout waiting condition: %w", ctx.Err()))
		case event, open := <-result.ResultChan():
			if !open {
				return false
			}
			switch event.Type {
			case watch.Added, watch.Modified, watch.Deleted:
				done := cb(event.Object.(T))
				if done {
					if apierrors.IsConflict(err) {
						// if we got a conflict, return a false (not done) and nil for error
						return false
					}
					if err != nil {
						t.Fatal(err)
					}
					return true
				}
			}
		}
	}
}

// retryWatch repeatedly uses watchFunc handler to call cb until the function
// returns true. or the function errored other than a conflict, it results in a
// fatal cb is the callback to confirm the object exists.
func retryWatch[T client.Object](t *testing.T, watchFunc watchFunc, cb func(obj T) bool) {
	t.Helper()

	for {
		if done := doWatch(t, watchFunc, cb); done {
			return
		}
	}
}

// Wait waits for a specific object to appear in a Kubernetes API server and returns the last object of type T
// and populates client.ObjectList pointer
func Wait[T client.Object](t *testing.T, watchFunc WatchFunc, list client.ObjectList, cb func(obj T) bool) T {
	t.Helper()

	var last T
	retryWatch(t, func() (watch.Interface, error) {
		ctx := GetCTX(t)
		return watchFunc(ctx, list)
	}, func(obj T) bool {
		last = obj
		return cb(obj)
	})
	return last
}

func Watcher(t *testing.T, c hclient.Client) WatchFunc {
	t.Helper()
	wc, err := c.GetClient()
	if err != nil {
		t.Fatal(err)
	}
	return wc.Watch
}

func WaitForObject[T client.Object](t *testing.T, watchFunc WatchFunc, list client.ObjectList, obj T, cb func(obj T) bool) T {
	t.Helper()

	if done := cb(obj); done {
		return obj
	}

	var last T
	// keep calling watch function
	retryWatch(t, func() (watch.Interface, error) {
		ctx := GetCTX(t)
		return watchFunc(ctx, list, &client.ListOptions{
			FieldSelector: fields.OneTermEqualSelector("metadata.name", obj.GetName()),
			Namespace:     obj.GetNamespace(),
		})
	}, func(obj T) bool {
		last = obj
		return cb(obj)
	})
	return last
}

func EnsureDoesNotExist(ctx context.Context, getter func() (client.Object, error)) error {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	ctx, cancel := context.WithTimeout(ctx, time.Duration(WatchTimeoutSeconds)*time.Second)
	defer cancel()

	_, err := getter()
	if apierrors.IsNotFound(err) {
		return nil
	}

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for deletion: %w", ctx.Err())
		case <-ticker.C:
			_, err := getter()
			if apierrors.IsNotFound(err) {
				return nil
			} else if err != nil {
				return err
			}
		}
	}
}
