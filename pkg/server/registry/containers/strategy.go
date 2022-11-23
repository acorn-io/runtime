package containers

import (
	"github.com/acorn-io/acorn/pkg/tables"
	"github.com/acorn-io/mink/pkg/strategy"
	"github.com/acorn-io/mink/pkg/strategy/remote"
	"github.com/acorn-io/mink/pkg/strategy/translation"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apiserver/pkg/registry/rest"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func NewStrategy(client kclient.WithWatch) (*Strategy, error) {
	storageStrategy, err := newStorageStrategy(client)
	if err != nil {
		return nil, err
	}

	return &Strategy{
		CompleteStrategy: storageStrategy,
		TableConvertor:   tables.ContainerConverter,
	}, nil
}

type Strategy struct {
	strategy.CompleteStrategy
	rest.TableConvertor
}

func newStorageStrategy(client kclient.WithWatch) (strategy.CompleteStrategy, error) {
	backend := remote.NewRemote(&corev1.Pod{}, &corev1.PodList{}, client)
	return translation.NewTranslationStrategy(&Translator{
		client: client,
	}, backend), nil
}
