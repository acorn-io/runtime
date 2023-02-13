package stores

import (
	"github.com/acorn-io/mink/pkg/strategy"
	"k8s.io/apiserver/pkg/registry/rest"
)

var (
	_ rest.Getter             = (*GetListDeleteStore)(nil)
	_ rest.Lister             = (*GetListDeleteStore)(nil)
	_ rest.RESTDeleteStrategy = (*GetListDeleteStore)(nil)
	_ strategy.Base           = (*GetListDeleteStore)(nil)
)

type GetListDeleteStore struct {
	*strategy.GetAdapter
	*strategy.ListAdapter
	*strategy.DeleteAdapter
	*strategy.DestroyAdapter
	*strategy.NewAdapter
	*strategy.TableAdapter
}

func (r *GetListDeleteStore) NamespaceScoped() bool {
	return r.ListAdapter.NamespaceScoped()
}
