package stores

import (
	"github.com/acorn-io/mink/pkg/strategy"
	"k8s.io/apiserver/pkg/registry/rest"
)

var (
	_ rest.Getter   = (*GetOnlyStore)(nil)
	_ strategy.Base = (*GetOnlyStore)(nil)
)

type GetOnlyStore struct {
	*strategy.GetAdapter
	*strategy.NewAdapter
	*strategy.DestroyAdapter
	*strategy.ScoperAdapter
	*strategy.TableAdapter
}
