package images

import (
	"github.com/acorn-io/acorn/pkg/client"
	"github.com/acorn-io/mink/pkg/stores"
	"k8s.io/apiserver/pkg/registry/rest"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func NewStorage(c kclient.WithWatch, clientFactory *client.Factory) (rest.Storage, error) {
	strategy, err := NewStrategy(c, clientFactory)
	if err != nil {
		return nil, err
	}
	return stores.NewComplete(c.Scheme(), strategy), nil
}
