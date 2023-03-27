package projects

import (
	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/tables"
	"github.com/acorn-io/mink/pkg/stores"
	"github.com/acorn-io/mink/pkg/strategy/remote"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apiserver/pkg/registry/rest"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const localRegion = "local"

func NewStorage(c kclient.WithWatch) rest.Storage {
	remoteResource := remote.NewWithTranslation(&Translator{localRegion}, &corev1.Namespace{}, c)
	strategy := &Strategy{
		c:       c,
		lister:  remoteResource,
		creater: remoteResource,
		updater: remoteResource,
		deleter: remoteResource,
	}
	validator := &Validator{localRegion}
	return stores.NewBuilder(c.Scheme(), &apiv1.Project{}).
		WithCreate(strategy).
		WithUpdate(strategy).
		WithDelete(strategy).
		WithGet(strategy).
		// Watch is not enabled because the dynamic privileges can do weird things...
		WithList(strategy).
		WithValidateCreate(validator).
		WithValidateUpdate(validator).
		WithTableConverter(tables.ProjectConverter).
		Build()
}
