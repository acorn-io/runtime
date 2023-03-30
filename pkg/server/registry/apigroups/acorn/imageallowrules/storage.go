package imageallowrules

import (
	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/tables"
	"github.com/acorn-io/mink/pkg/stores"
	"github.com/acorn-io/mink/pkg/strategy/remote"
	"k8s.io/apiserver/pkg/registry/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewStorage(c client.WithWatch) rest.Storage {
	remoteResource := remote.NewWithSimpleTranslation(&Translator{}, &apiv1.ImageAllowRule{}, c)

	return stores.NewBuilder(c.Scheme(), &apiv1.ImageAllowRule{}).
		WithValidateCreate(&Validator{}).
		WithValidateUpdate(&Validator{}).
		WithCompleteCRUD(remoteResource).
		WithTableConverter(tables.ImageAllowRuleConverter).
		Build()
}
