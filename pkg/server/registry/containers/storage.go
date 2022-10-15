package containers

import (
	"github.com/acorn-io/mink/pkg/db"
	"github.com/acorn-io/mink/pkg/stores"
	"k8s.io/apiserver/pkg/registry/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewStorage(c client.WithWatch, db *db.Factory) (rest.Storage, rest.Storage, error) {
	strategy, err := NewStrategy(c, db)
	if err != nil {
		return nil, nil, err
	}

	if db == nil {
		return stores.NewReadDelete(c.Scheme(), strategy), nil, nil
	}
	store, status := stores.NewWithStatus(c.Scheme(), strategy)
	return store, status, nil
}
