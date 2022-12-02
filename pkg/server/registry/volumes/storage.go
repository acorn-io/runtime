package volumes

import (
	"github.com/acorn-io/acorn/pkg/scheme"
	"github.com/acorn-io/mink/pkg/stores"
	"k8s.io/apiserver/pkg/registry/rest"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func NewStorage(c kclient.WithWatch) (rest.Storage, error) {
	strategy, err := NewStrategy(c)
	if err != nil {
		return nil, err
	}
	return stores.NewReadDelete(scheme.Scheme, strategy), nil
}
