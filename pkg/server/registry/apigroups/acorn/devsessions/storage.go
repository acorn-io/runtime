package devsessions

import (
	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/client"
	"github.com/acorn-io/acorn/pkg/server/registry/apigroups/acorn/apps"
	"github.com/acorn-io/mink/pkg/stores"
	"github.com/acorn-io/mink/pkg/strategy/remote"
	"github.com/acorn-io/mink/pkg/strategy/translation"
	"github.com/acorn-io/mink/pkg/validator"
	"k8s.io/apiserver/pkg/registry/rest"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func NewStorage(c kclient.WithWatch, cf *client.Factory) rest.Storage {
	remoteResource := translation.NewSimpleTranslationStrategy(&Translator{},
		remote.NewRemote(&v1.DevSessionInstance{}, c))

	appValidator := apps.NewValidator(c, cf, nil)
	strategy := NewValidator(c, appValidator)

	return stores.NewBuilder(c.Scheme(), &apiv1.DevSession{}).
		WithValidateCreate(strategy).
		WithValidateUpdate(strategy).
		WithCompleteCRUD(remoteResource).
		WithValidateName(validator.ValidDNSSubdomain).
		Build()
}
