package info

import (
	"github.com/acorn-io/mink/pkg/stores"
	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/tables"
	"k8s.io/apiserver/pkg/registry/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewStorage(c client.WithWatch) rest.Storage {
	return stores.NewBuilder(c.Scheme(), &apiv1.Info{}).
		WithList(NewStrategy(c)).
		WithTableConverter(tables.InfoConverter).
		Build()
}
