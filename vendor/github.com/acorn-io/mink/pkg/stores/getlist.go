package stores

import (
	"github.com/acorn-io/mink/pkg/strategy"
	"k8s.io/apiserver/pkg/registry/rest"
)

var (
	_ rest.Getter   = (*GetListStore)(nil)
	_ rest.Lister   = (*GetListStore)(nil)
	_ strategy.Base = (*GetListStore)(nil)
)

type GetListStore struct {
	*strategy.GetAdapter
	*strategy.ListAdapter
	*strategy.DestroyAdapter
	*strategy.NewAdapter
	*strategy.TableAdapter
}

func (r *GetListStore) NamespaceScoped() bool {
	return r.ListAdapter.NamespaceScoped()
}
