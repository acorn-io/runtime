package builders

import (
	"context"

	api "github.com/acorn-io/acorn/pkg/apis/api.acorn.io"
	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/build/buildkit"
	"github.com/acorn-io/acorn/pkg/system"
	"github.com/acorn-io/acorn/pkg/tables"
	"github.com/acorn-io/mink/pkg/types"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/apiserver/pkg/registry/rest"
	"k8s.io/apiserver/pkg/storage"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func NewStrategy(c kclient.WithWatch) *Strategy {
	return &Strategy{
		TableConvertor: tables.BuilderConverter,
		client:         c,
	}
}

type Strategy struct {
	rest.TableConvertor
	client kclient.WithWatch
}

func (s *Strategy) Get(ctx context.Context, namespace, name string) (types.Object, error) {
	return s.get(ctx, false, namespace, name)
}

func (s *Strategy) get(ctx context.Context, create bool, namespace, name string) (types.Object, error) {
	if namespace != system.Namespace || name != system.BuildKitName {
		return nil, apierrors.NewNotFound(schema.GroupResource{
			Group:    api.Group,
			Resource: "builders",
		}, name)
	}

	if ok, err := buildkit.Exists(ctx, s.client); err != nil {
		return nil, err
	} else if !ok && !create {
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
	builder.Name = system.BuildKitName
	builder.Namespace = system.Namespace
	builder.UID = builder.UID + "-build"
	return builder, nil
}

func (s *Strategy) Validate(ctx context.Context, object runtime.Object) (errors field.ErrorList) {
	obj := object.(kclient.Object)
	if obj.GetName() != system.BuildKitName {
		errors = append(errors, field.Invalid(field.NewPath("metadata", "name"),
			obj.GetName(), "name be must equal to "+system.BuildKitName))
	}
	if obj.GetNamespace() != system.Namespace {
		errors = append(errors, field.Invalid(field.NewPath("metadata", "namespace"),
			obj.GetNamespace(), "namespace be must equal to "+system.Namespace))
	}
	return
}

func (s *Strategy) Create(ctx context.Context, obj types.Object) (types.Object, error) {
	return s.get(ctx, true, obj.GetNamespace(), obj.GetName())
}

func (s *Strategy) List(ctx context.Context, namespace string, opts storage.ListOptions) (types.ObjectList, error) {
	if namespace == "" || namespace == system.Namespace {
		obj, err := s.Get(ctx, system.Namespace, system.BuildKitName)
		if apierrors.IsNotFound(err) {
			return &apiv1.BuilderList{}, nil
		} else if err != nil {
			return nil, err
		}
		return &apiv1.BuilderList{
			Items: []apiv1.Builder{*obj.(*apiv1.Builder)},
		}, nil
	}
	return &apiv1.BuilderList{}, nil
}

func (s *Strategy) New() types.Object {
	return &apiv1.Builder{}
}

func (s *Strategy) NewList() types.ObjectList {
	return &apiv1.BuilderList{}
}

func (s *Strategy) Delete(ctx context.Context, obj types.Object) (types.Object, error) {
	return obj, buildkit.Delete(ctx)
}
