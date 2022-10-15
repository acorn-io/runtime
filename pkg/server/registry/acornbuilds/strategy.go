package acornbuilds

import (
	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/tables"
	"github.com/acorn-io/mink/pkg/db"
	"github.com/acorn-io/mink/pkg/strategy"
	"github.com/acorn-io/mink/pkg/strategy/remote"
	"github.com/acorn-io/mink/pkg/strategy/translation"
	"k8s.io/apiserver/pkg/registry/rest"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type Strategy struct {
	strategy.CompleteStrategy
	rest.TableConvertor
}

func NewStrategy(c kclient.WithWatch, db *db.Factory) (*Strategy, error) {
	storageStrategy, err := newStorageStrategy(c, db)
	if err != nil {
		return nil, err
	}
	return &Strategy{
		CompleteStrategy: storageStrategy,
		TableConvertor:   tables.AcornBuildConverter,
	}, nil
}

func newStorageStrategy(c kclient.WithWatch, db *db.Factory) (strategy.CompleteStrategy, error) {
	if db != nil {
		return db.NewDBStrategy(&apiv1.AcornBuild{})
	}
	return translation.NewTranslationStrategy(&Translator{}, remote.NewRemote(&v1.AcornBuild{}, &v1.AcornBuildList{}, c)), nil
}
