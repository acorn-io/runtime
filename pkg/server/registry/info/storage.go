package info

import (
	"github.com/acorn-io/mink/pkg/stores"
	"k8s.io/apiserver/pkg/registry/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewStorage(c client.WithWatch) rest.Storage {
	return stores.NewListOnly(NewStrategy(c))
}
