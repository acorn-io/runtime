package computeclass

import (
	adminv1 "github.com/acorn-io/acorn/pkg/apis/admin.acorn.io/v1"
	v1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	internaladminv1 "github.com/acorn-io/acorn/pkg/apis/internal.admin.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/tables"
	"github.com/acorn-io/mink/pkg/stores"
	"github.com/acorn-io/mink/pkg/strategy/remote"
	"github.com/acorn-io/mink/pkg/strategy/translation"
	"k8s.io/apiserver/pkg/registry/rest"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func NewClusterStorage(c kclient.WithWatch) rest.Storage {
	remoteResource := translation.NewSimpleTranslationStrategy(&ClusterTranslator{},
		remote.NewRemote(&internaladminv1.ClusterComputeClassInstance{}, c))
	validator := NewClusterValidator(c)

	return stores.NewBuilder(c.Scheme(), &adminv1.ClusterComputeClass{}).
		WithCompleteCRUD(remoteResource).
		WithValidateCreate(validator).
		WithValidateUpdate(validator).
		WithTableConverter(tables.ComputeClassConverter).
		Build()
}

func NewProjectStorage(c kclient.WithWatch) rest.Storage {
	remoteResource := translation.NewSimpleTranslationStrategy(&ProjectTranslator{},
		remote.NewRemote(&internaladminv1.ProjectComputeClassInstance{}, c))
	validator := NewProjectValidator(c)

	return stores.NewBuilder(c.Scheme(), &adminv1.ProjectComputeClass{}).
		WithCompleteCRUD(remoteResource).
		WithValidateCreate(validator).
		WithValidateUpdate(validator).
		WithTableConverter(tables.ComputeClassConverter).
		Build()
}

func NewAggregateStorage(c kclient.WithWatch) rest.Storage {
	return stores.NewBuilder(c.Scheme(), &v1.ComputeClass{}).
		WithGet(NewStrategy(c)).
		WithList(NewStrategy(c)).
		WithTableConverter(tables.ComputeClassConverter).
		Build()
}
