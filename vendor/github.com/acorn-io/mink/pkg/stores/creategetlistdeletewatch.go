package stores

import (
	"github.com/acorn-io/mink/pkg/strategy"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/registry/rest"
)

var (
	_ rest.Getter             = (*CreateGetListDeleteWatchStore)(nil)
	_ rest.Lister             = (*CreateGetListDeleteWatchStore)(nil)
	_ rest.Watcher            = (*CreateGetListDeleteWatchStore)(nil)
	_ rest.RESTDeleteStrategy = (*CreateGetListDeleteWatchStore)(nil)
	_ strategy.Base           = (*CreateGetListDeleteWatchStore)(nil)
)

type CreateGetListDeleteWatchStore struct {
	*strategy.GetAdapter
	*strategy.CreateAdapter
	*strategy.ListAdapter
	*strategy.DeleteAdapter
	*strategy.DestroyAdapter
	*strategy.TableAdapter
	*strategy.WatchAdapter
}

func (c *CreateGetListDeleteWatchStore) New() runtime.Object {
	return c.CreateAdapter.New()
}

func (r *CreateGetListDeleteWatchStore) NamespaceScoped() bool {
	return r.ListAdapter.NamespaceScoped()
}
