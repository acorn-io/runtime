package info

import (
	"context"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/info"
	"github.com/acorn-io/acorn/pkg/tables"
	"k8s.io/apimachinery/pkg/apis/meta/internalversion"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/registry/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewStorage(c client.WithWatch) *Storage {
	return &Storage{
		TableConvertor: tables.InfoConverter,
		client:         c,
	}
}

type Storage struct {
	rest.TableConvertor

	client client.WithWatch
}

func (s *Storage) NewList() runtime.Object {
	return &apiv1.InfoList{}
}

func (s *Storage) NamespaceScoped() bool {
	return true
}

func (s *Storage) New() runtime.Object {
	return &apiv1.Info{}
}

func (s *Storage) List(ctx context.Context, options *internalversion.ListOptions) (runtime.Object, error) {
	i, err := info.Get(ctx, s.client)
	if err != nil {
		return nil, err
	}

	return &apiv1.InfoList{
		Items: []apiv1.Info{
			*i,
		},
	}, nil
}
