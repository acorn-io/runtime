package class

import (
	"github.com/acorn-io/mink/pkg/stores"
	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/tables"
	"k8s.io/apiserver/pkg/registry/rest"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func NewClassStorage(c kclient.WithWatch) rest.Storage {
	strategy := &Strategy{c}

	return stores.NewBuilder(c.Scheme(), &apiv1.VolumeClass{}).
		WithGet(strategy).
		WithList(strategy).
		WithTableConverter(tables.VolumeClassConverter).
		Build()
}
