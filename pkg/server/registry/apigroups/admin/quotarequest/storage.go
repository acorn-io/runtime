package quotarequest

import (
	"github.com/acorn-io/mink/pkg/stores"
	"github.com/acorn-io/mink/pkg/strategy/remote"
	"github.com/acorn-io/mink/pkg/strategy/translation"
	adminv1 "github.com/acorn-io/runtime/pkg/apis/admin.acorn.io/v1"
	internaladminv1 "github.com/acorn-io/runtime/pkg/apis/internal.admin.acorn.io/v1"
	"k8s.io/apiserver/pkg/registry/rest"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func NewStorage(c kclient.WithWatch) rest.Storage {
	remoteResource := translation.NewSimpleTranslationStrategy(&Translator{},
		remote.NewRemote(&internaladminv1.QuotaRequestInstance{}, c))

	return stores.NewBuilder(c.Scheme(), &adminv1.QuotaRequest{}).
		WithGet(remoteResource).
		WithList(remoteResource).
		WithWatch(remoteResource).
		Build()
}
