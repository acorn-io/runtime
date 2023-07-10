package volumes

import (
	"github.com/acorn-io/mink/pkg/stores"
	"github.com/acorn-io/mink/pkg/strategy/remote"
	"github.com/acorn-io/mink/pkg/strategy/translation"
	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/publicname"
	"github.com/acorn-io/runtime/pkg/tables"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apiserver/pkg/registry/rest"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func NewStorage(c kclient.WithWatch) rest.Storage {
	translated := translation.NewTranslationStrategy(&Translator{
		c: c,
	}, remote.NewRemote(&corev1.PersistentVolume{}, c))
	remoteResource := publicname.NewStrategy(translated)

	return stores.NewBuilder(c.Scheme(), &apiv1.Volume{}).
		WithGet(remoteResource).
		WithList(remoteResource).
		WithDelete(remoteResource).
		WithWatch(remoteResource).
		WithTableConverter(tables.VolumeConverter).
		Build()
}
