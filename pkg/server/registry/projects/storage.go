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

func NewStorage(c kclient.WithWatch) rest.Storage {
	remoteResource := remote.NewWithTranslation(&Translator{},
		&corev1.Namespace{}, c)
	strategy := &Strategy{
		c:       c,
		lister:  remoteResource,
		creater: remoteResource,
		deleter: remoteResource,
	}
	return stores.NewBuilder(c.Scheme(), &apiv1.Project{}).
		WithCreate(strategy).
		WithDelete(strategy).
		WithGet(strategy).
		WithList(strategy).
		WithTableConverter(tables.ProjectConverter).
		Build()
}
