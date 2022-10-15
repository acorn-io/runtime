package volumes

import (
	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/tables"
	"github.com/acorn-io/mink/pkg/db"
	"github.com/acorn-io/mink/pkg/strategy"
	"github.com/acorn-io/mink/pkg/strategy/remote"
	"github.com/acorn-io/mink/pkg/strategy/translation"
	corev1 "k8s.io/api/core/v1"
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
		TableConvertor:   tables.VolumeConverter,
	}, nil
}

func newStorageStrategy(c kclient.WithWatch, db *db.Factory) (strategy.CompleteStrategy, error) {
	if db != nil {
		return db.NewDBStrategy(&apiv1.Volume{})
	}
	return translation.NewTranslationStrategy(&Translator{
		c: c,
	}, remote.NewRemote(&corev1.PersistentVolume{}, &corev1.PersistentVolumeList{}, c)), nil
}
