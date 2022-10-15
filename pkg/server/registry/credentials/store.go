package credentials

import (
	"github.com/acorn-io/acorn/pkg/scheme"
	"github.com/acorn-io/mink/pkg/stores"
	"k8s.io/apiserver/pkg/registry/rest"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func NewStore(c kclient.WithWatch) rest.Storage {
	return stores.NewComplete(scheme.Scheme, NewStrategy(c, false))
}
