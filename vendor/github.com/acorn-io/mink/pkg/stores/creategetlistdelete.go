package stores

import (
	"github.com/acorn-io/mink/pkg/strategy"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/registry/rest"
)

var (
	_ rest.Getter             = (*CreateGetListDeleteStore)(nil)
	_ rest.Lister             = (*CreateGetListDeleteStore)(nil)
	_ rest.RESTDeleteStrategy = (*CreateGetListDeleteStore)(nil)
	_ strategy.Base           = (*CreateGetListDeleteStore)(nil)
)

type CreateGetListDeleteStore struct {
	*strategy.GetAdapter
	*strategy.CreateAdapter
	*strategy.ListAdapter
	*strategy.DeleteAdapter
	*strategy.DestroyAdapter
	*strategy.TableAdapter
}

func (c *CreateGetListDeleteStore) New() runtime.Object {
	return c.CreateAdapter.New()
}

func (r *CreateGetListDeleteStore) NamespaceScoped() bool {
	return r.ListAdapter.NamespaceScoped()
}
