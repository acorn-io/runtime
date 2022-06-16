package builders

import (
	"context"

	api "github.com/acorn-io/acorn/pkg/apis/api.acorn.io"
	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/build/buildkit"
	"github.com/acorn-io/acorn/pkg/tables"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewStorage(c client.WithWatch) *Storage {
	return &Storage{
		TableConvertor: tables.ContainerConverter,
		client:         c,
	}
}

type Storage struct {
	rest.TableConvertor

	client client.WithWatch
}

func (s *Storage) NewList() runtime.Object {
	return &apiv1.BuilderList{}
}

func (s *Storage) NamespaceScoped() bool {
	return true
}

func (s *Storage) New() runtime.Object {
	return &apiv1.Builder{}
}

func (s *Storage) Create(ctx context.Context, obj runtime.Object, createValidation rest.ValidateObjectFunc, options *metav1.CreateOptions) (runtime.Object, error) {
	if createValidation != nil {
		if err := createValidation(ctx, obj); err != nil {
			return nil, err
		}
	}

	_, _, err := buildkit.GetBuildkitPod(ctx, s.client)
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, "", nil)
}

func (s *Storage) Delete(ctx context.Context, name string, deleteValidation rest.ValidateObjectFunc, options *metav1.DeleteOptions) (runtime.Object, bool, error) {
	obj, err := s.Get(ctx, name, nil)
	if err != nil {
		return nil, false, err
	}
	builder := obj.(*apiv1.Builder)
	if deleteValidation != nil {
		if err := deleteValidation(ctx, builder); err != nil {
			return nil, false, err
		}
	}
	return obj, true, buildkit.Delete(ctx)
}

func (s *Storage) Get(ctx context.Context, name string, options *metav1.GetOptions) (runtime.Object, error) {
	nsName, _ := request.NamespaceFrom(ctx)
	if ok, err := buildkit.Exists(ctx, s.client); err != nil {
		return nil, err
	} else if !ok {
		return nil, apierrors.NewNotFound(schema.GroupResource{
			Group:    api.Group,
			Resource: "builders",
		}, name)
	}

	_, pod, err := buildkit.GetBuildkitPod(ctx, s.client)
	if err != nil {
		return nil, err
	}

	builder := &apiv1.Builder{
		ObjectMeta: pod.ObjectMeta,
		Status: apiv1.BuilderStatus{
			Ready: true,
		},
	}
	builder.Namespace = nsName
	return builder, nil
}

func (s *Storage) List(ctx context.Context, options *internalversion.ListOptions) (runtime.Object, error) {
	obj, err := s.Get(ctx, "", nil)
	if apierrors.IsNotFound(err) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	return &apiv1.BuilderList{
		Items: []apiv1.Builder{*obj.(*apiv1.Builder)},
	}, nil
}
