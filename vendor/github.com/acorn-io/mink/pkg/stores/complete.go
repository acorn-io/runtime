package stores

import (
	"github.com/acorn-io/mink/pkg/strategy"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/registry/rest"
)

var (
	_ strategy.Base = (*Complete)(nil)
)

func NewComplete(scheme *runtime.Scheme, s strategy.CompleteStrategy) rest.Storage {
	store, _ := newComplete(scheme, s)
	return store
}

type Complete struct {
	*strategy.CreateAdapter
	*strategy.UpdateAdapter
	*strategy.GetAdapter
	*strategy.ListAdapter
	*strategy.DeleteAdapter
	*strategy.WatchAdapter

	strategy strategy.CompleteStrategy
}

func (c *Complete) NamespaceScoped() bool {
	return c.CreateAdapter.NamespaceScoped()
}

func (c *Complete) Destroy() {
	c.strategy.Destroy()
}

func newComplete(scheme *runtime.Scheme, s strategy.CompleteStrategy) (*Complete, *strategy.Status) {
	return &Complete{
		CreateAdapter: strategy.NewCreate(scheme, s),
		UpdateAdapter: strategy.NewUpdate(scheme, s),
		GetAdapter:    strategy.NewGet(s),
		ListAdapter:   strategy.NewList(s),
		DeleteAdapter: strategy.NewDelete(scheme, s),
		WatchAdapter:  strategy.NewWatch(s),
		strategy:      s,
	}, strategy.NewStatus(scheme, s)
}
