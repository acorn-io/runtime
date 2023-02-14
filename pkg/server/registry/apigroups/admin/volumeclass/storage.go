package volumeclass

import (
	adminv1 "github.com/acorn-io/acorn/pkg/apis/admin.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/tables"
	"github.com/acorn-io/mink/pkg/stores"
	"github.com/acorn-io/mink/pkg/strategy/remote"
	"k8s.io/apiserver/pkg/registry/rest"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func NewClusterStorage(c kclient.WithWatch) rest.Storage {
	remoteResource := remote.NewWithSimpleTranslation(&ClusterTranslator{}, &adminv1.ClusterVolumeClass{}, c)
	validator := NewClusterValidator(c)

	return stores.NewBuilder(c.Scheme(), &adminv1.ClusterVolumeClass{}).
		WithCompleteCRUD(remoteResource).
		WithValidateUpdate(validator).
		WithValidateCreate(validator).
		WithTableConverter(tables.VolumeClassConverter).
		Build()
}

func NewProjectStorage(c kclient.WithWatch) rest.Storage {
	remoteResource := remote.NewWithSimpleTranslation(&ProjectTranslator{}, &adminv1.ProjectVolumeClass{}, c)
	validator := NewProjectValidator(c)

	return stores.NewBuilder(c.Scheme(), &adminv1.ProjectVolumeClass{}).
		WithCompleteCRUD(remoteResource).
		WithValidateUpdate(validator).
		WithValidateCreate(validator).
		WithTableConverter(tables.VolumeClassConverter).
		Build()
}
