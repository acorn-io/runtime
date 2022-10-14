package images

import (
	"github.com/acorn-io/mink/pkg/db"
	"github.com/acorn-io/mink/pkg/stores"
	"k8s.io/apiserver/pkg/registry/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewStorage(c client.WithWatch, db *db.Factory) (rest.Storage, error) {
	if db == nil {
		return stores.NewGetListDelete(c.Scheme(), NewDynamicStrategy(c)), nil
	}

	dbStrategy, err := NewDBStrategy(db)
	if err != nil {
		return nil, err
	}
	return stores.NewComplete(c.Scheme(), dbStrategy), nil
}
