package stores

import (
	"github.com/acorn-io/mink/pkg/strategy"
	"k8s.io/apiserver/pkg/registry/rest"
)

var (
	_ rest.Getter             = (*ReadDeleteStore)(nil)
	_ rest.Lister             = (*ReadDeleteStore)(nil)
	_ rest.Watcher            = (*ReadDeleteStore)(nil)
	_ strategy.Base           = (*ReadDeleteStore)(nil)
	_ rest.RESTDeleteStrategy = (*ReadDeleteStore)(nil)
)

type ReadDeleteStore struct {
	*strategy.GetAdapter
	*strategy.ListAdapter
	*strategy.WatchAdapter
	*strategy.DeleteAdapter
	*strategy.DestroyAdapter
	*strategy.NewAdapter
}

func (r *ReadDeleteStore) NamespaceScoped() bool {
	return r.ListAdapter.NamespaceScoped()
}
