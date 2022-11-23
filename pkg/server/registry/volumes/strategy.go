package volumes

import (
	"github.com/acorn-io/acorn/pkg/tables"
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

func NewStrategy(c kclient.WithWatch) (strategy.CompleteStrategy, error) {
	storageStrategy, err := newStorageStrategy(c)
	if err != nil {
		return nil, err
	}
	return NewStrategyWithStorage(c, storageStrategy)
}

func NewStrategyWithStorage(c kclient.WithWatch, storage strategy.CompleteStrategy) (strategy.CompleteStrategy, error) {
	return &Strategy{
		CompleteStrategy: storage,
		TableConvertor:   tables.VolumeConverter,
	}, nil
}

func newStorageStrategy(c kclient.WithWatch) (strategy.CompleteStrategy, error) {
	return translation.NewTranslationStrategy(&Translator{
		c: c,
	}, remote.NewRemote(&corev1.PersistentVolume{}, &corev1.PersistentVolumeList{}, c)), nil
}
