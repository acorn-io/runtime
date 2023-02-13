package stores

import (
	"github.com/acorn-io/mink/pkg/strategy"
	"k8s.io/apiserver/pkg/registry/rest"
)

var (
	_ rest.Getter   = (*CreateGetStore)(nil)
	_ rest.Creater  = (*CreateGetStore)(nil)
	_ strategy.Base = (*CreateGetStore)(nil)
)

type CreateGetStore struct {
	*strategy.CreateAdapter
	*strategy.GetAdapter
	*strategy.DestroyAdapter
	*strategy.TableAdapter
}
