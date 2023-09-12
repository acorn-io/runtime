package imageroleauthorizations

import (
	"github.com/acorn-io/mink/pkg/stores"
	"github.com/acorn-io/mink/pkg/strategy/remote"
	"github.com/acorn-io/mink/pkg/strategy/translation"
	adminv1 "github.com/acorn-io/runtime/pkg/apis/admin.acorn.io/v1"
	internaladminv1 "github.com/acorn-io/runtime/pkg/apis/internal.admin.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/tables"
	"k8s.io/apiserver/pkg/registry/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewStorage(c client.WithWatch) rest.Storage {
	remoteResource := translation.NewSimpleTranslationStrategy(&Translator{},
		remote.NewRemote(&internaladminv1.ImageRoleAuthorizationInstance{}, c))

	validator := &Validator{}

	return stores.NewBuilder(c.Scheme(), &adminv1.ImageRoleAuthorization{}).
		WithValidateCreate(validator).
		WithValidateUpdate(validator).
		WithCompleteCRUD(remoteResource).
		WithTableConverter(tables.ImageRoleAuthorizationConverter).
		Build()
}

func NewClusterStorage(c client.WithWatch) rest.Storage {
	remoteResource := translation.NewSimpleTranslationStrategy(&ClusterTranslator{},
		remote.NewRemote(&internaladminv1.ClusterImageRoleAuthorizationInstance{}, c))

	validator := &ClusterValidator{}

	return stores.NewBuilder(c.Scheme(), &adminv1.ClusterImageRoleAuthorization{}).
		WithValidateCreate(validator).
		WithValidateUpdate(validator).
		WithCompleteCRUD(remoteResource).
		WithTableConverter(tables.ImageRoleAuthorizationConverter).
		Build()
}
