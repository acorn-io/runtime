package stores

import (
	"github.com/acorn-io/mink/pkg/strategy"
	"k8s.io/apiserver/pkg/registry/rest"
)

var (
	_ rest.Getter             = (*ReadWriteWatchStore)(nil)
	_ rest.Lister             = (*ReadWriteWatchStore)(nil)
	_ rest.Updater            = (*ReadWriteWatchStore)(nil)
	_ rest.Watcher            = (*ReadWriteWatchStore)(nil)
	_ rest.Creater            = (*ReadWriteWatchStore)(nil)
	_ rest.RESTDeleteStrategy = (*ReadWriteWatchStore)(nil)
	_ strategy.Base           = (*ReadWriteWatchStore)(nil)
)

type ReadWriteWatchStore struct {
	*strategy.CreateAdapter
	*strategy.GetAdapter
	*strategy.ListAdapter
	*strategy.UpdateAdapter
	*strategy.DeleteAdapter
	*strategy.WatchAdapter
	*strategy.DestroyAdapter
	*strategy.TableAdapter
}

func (r *ReadWriteWatchStore) NamespaceScoped() bool {
	return r.ListAdapter.NamespaceScoped()
}
