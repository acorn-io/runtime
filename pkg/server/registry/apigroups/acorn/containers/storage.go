package containers

import (
	"github.com/acorn-io/mink/pkg/stores"
	"github.com/acorn-io/mink/pkg/strategy/remote"
	"github.com/acorn-io/mink/pkg/strategy/translation"
	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/tables"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apiserver/pkg/registry/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewStorage(c client.WithWatch) rest.Storage {
	strategy := translation.NewTranslationStrategy(&Translator{
		client: c,
	}, remote.NewRemote(&corev1.Pod{}, c))

	return stores.NewBuilder(c.Scheme(), &apiv1.ContainerReplica{}).
		WithGet(strategy).
		WithList(strategy).
		WithDelete(strategy).
		WithWatch(strategy).
		WithTableConverter(tables.ContainerConverter).
		Build()
}
