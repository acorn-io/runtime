package builders

import (
	"github.com/acorn-io/mink/pkg/db"
	"github.com/acorn-io/mink/pkg/stores"
	"k8s.io/apiserver/pkg/registry/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewStorage(c client.WithWatch, db *db.Factory) (rest.Storage, rest.Storage, error) {
	if db == nil {
		return stores.NewCreateGetListDelete(c.Scheme(), NewDynamicStrategy(c)), nil, nil
	}
	dbStrategy, err := NewDBStrategy(db)
	if err != nil {
		return nil, nil, err
	}
	store, status := stores.NewWithStatus(c.Scheme(), dbStrategy)
	return store, status, nil
}
