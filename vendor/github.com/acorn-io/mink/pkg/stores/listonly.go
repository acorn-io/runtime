package stores

import (
	"github.com/acorn-io/mink/pkg/strategy"
	"k8s.io/apiserver/pkg/registry/rest"
)

var (
	_ rest.Lister   = (*ListOnlyStore)(nil)
	_ strategy.Base = (*ListOnlyStore)(nil)
)

type ListOnlyStore struct {
	*strategy.ListAdapter
	*strategy.DestroyAdapter
	*strategy.NewAdapter
}

func (r *ListOnlyStore) NamespaceScoped() bool {
	return r.ListAdapter.NamespaceScoped()
}
