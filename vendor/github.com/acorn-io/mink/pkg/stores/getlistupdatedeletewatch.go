package stores

import (
	"github.com/acorn-io/mink/pkg/strategy"
	"k8s.io/apiserver/pkg/registry/rest"
)

var (
	_ rest.Getter             = (*GetListUpdateDeleteWatchStore)(nil)
	_ rest.Lister             = (*GetListUpdateDeleteWatchStore)(nil)
	_ rest.Updater            = (*GetListUpdateDeleteWatchStore)(nil)
	_ rest.Watcher            = (*GetListUpdateDeleteWatchStore)(nil)
	_ rest.RESTDeleteStrategy = (*GetListUpdateDeleteWatchStore)(nil)
	_ strategy.Base           = (*GetListUpdateDeleteWatchStore)(nil)
)

type GetListUpdateDeleteWatchStore struct {
	*strategy.GetAdapter
	*strategy.UpdateAdapter
	*strategy.ListAdapter
	*strategy.DeleteAdapter
	*strategy.WatchAdapter
	*strategy.DestroyAdapter
	*strategy.TableAdapter
}

func (g *GetListUpdateDeleteWatchStore) NamespaceScoped() bool {
	return g.WatchAdapter.NamespaceScoped()
}
