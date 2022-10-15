package apps

import (
	"github.com/acorn-io/acorn/pkg/client"
	"github.com/acorn-io/mink/pkg/db"
	"github.com/acorn-io/mink/pkg/stores"
	"k8s.io/apiserver/pkg/registry/rest"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func NewStorage(c kclient.WithWatch, clientFactory *client.Factory, db *db.Factory) (rest.Storage, rest.Storage, error) {
	strategy, err := NewStrategy(c, clientFactory, db)
	if err != nil {
		return nil, nil, err
	}
	store, statusStore := stores.NewWithStatus(c.Scheme(), strategy)
	return store, statusStore, nil
}
