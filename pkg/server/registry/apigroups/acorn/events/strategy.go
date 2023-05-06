package events

import (
	"context"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/mink/pkg/strategy"
	"github.com/acorn-io/mink/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/apiserver/pkg/storage"
)

type creatorListerWatcher interface {
	strategy.Creater
	strategy.Lister
	strategy.Watcher
}

type Strategy struct {
	delegate creatorListerWatcher
}

func (s *Strategy) Create(ctx context.Context, object types.Object) (types.Object, error) {
	// TODO(njhale): Implement me!
	return s.delegate.Create(ctx, object)
}

func (s *Strategy) List(ctx context.Context, namespace string, opts storage.ListOptions) (types.ObjectList, error) {
	// TODO(njhale): Implement me!
	return s.delegate.List(ctx, namespace, opts)
}

func (s *Strategy) Watch(ctx context.Context, namespace string, opts storage.ListOptions) (<-chan watch.Event, error) {
	// TODO(njhale): Implement me!
	return s.delegate.Watch(ctx, namespace, opts)
}

func (s *Strategy) New() types.Object {
	return &apiv1.Event{}
}

func (s *Strategy) NewList() types.ObjectList {
	return &apiv1.EventList{}
}
