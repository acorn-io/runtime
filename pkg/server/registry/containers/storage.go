package containers

import (
	"github.com/acorn-io/mink/pkg/stores"
	"k8s.io/apiserver/pkg/registry/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewStorage(c client.WithWatch) (rest.Storage, error) {
	strategy, err := NewStrategy(c)
	if err != nil {
		return nil, err
	}

	return stores.NewReadDelete(c.Scheme(), strategy), nil
}
