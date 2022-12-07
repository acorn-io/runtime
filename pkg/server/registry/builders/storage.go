package builders

import (
	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/mink/pkg/stores"
	"k8s.io/apiserver/pkg/registry/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewStorage(c client.WithWatch) rest.Storage {
	strategy := NewStrategy(c)
	return stores.NewBuilder(c.Scheme(), &apiv1.Builder{}).
		WithCreate(strategy).
		WithGet(strategy).
		WithList(strategy).
		WithDelete(strategy).
		WithValidateCreate(strategy).
		Build()
}
