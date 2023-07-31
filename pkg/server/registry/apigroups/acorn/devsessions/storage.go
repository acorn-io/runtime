package devsessions

import (
	"github.com/acorn-io/mink/pkg/stores"
	"github.com/acorn-io/mink/pkg/strategy/remote"
	"github.com/acorn-io/mink/pkg/strategy/translation"
	"github.com/acorn-io/mink/pkg/validator"
	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/client"
	"github.com/acorn-io/runtime/pkg/server/registry/apigroups/acorn/apps"
	"k8s.io/apiserver/pkg/registry/rest"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func NewStorage(c kclient.WithWatch, cf *client.Factory) rest.Storage {
	remoteResource := translation.NewSimpleTranslationStrategy(&Translator{},
		remote.NewRemote(&v1.DevSessionInstance{}, c))

	appValidator := apps.NewValidator(c, cf, nil, nil)
	devSessionValidator := NewValidator(c, appValidator)

	return stores.NewBuilder(c.Scheme(), &apiv1.DevSession{}).
		WithValidateCreate(devSessionValidator).
		WithValidateUpdate(devSessionValidator).
		WithCompleteCRUD(remoteResource).
		WithValidateName(validator.ValidDNSSubdomain).
		Build()
}
