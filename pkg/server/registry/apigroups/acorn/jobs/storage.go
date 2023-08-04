package jobs

import (
	"github.com/acorn-io/mink/pkg/stores"
	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/tables"
	"k8s.io/apiserver/pkg/registry/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewStorage(c client.WithWatch) rest.Storage {
	strategy := NewStrategy(c)

	return stores.NewBuilder(c.Scheme(), &apiv1.Job{}).
		WithGet(strategy).
		WithList(strategy).
		WithTableConverter(tables.JobConverter).
		Build()
}
