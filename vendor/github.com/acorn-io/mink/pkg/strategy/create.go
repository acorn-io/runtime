package strategy

import (
	"context"

	"github.com/acorn-io/mink/pkg/types"
	"github.com/acorn-io/mink/pkg/validator"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/rest"
	"k8s.io/apiserver/pkg/storage/names"
)

type CompleteCRUD interface {
	Lister
	Watcher
	Creater
	Updater
	Deleter
}

type Creater interface {
	Create(ctx context.Context, object types.Object) (types.Object, error)
	New() types.Object
}

type WarningsOnCreator interface {
	WarningsOnCreate(ctx context.Context, obj runtime.Object) []string
}

type NameValidator interface {
	ValidateName(ctx context.Context, obj runtime.Object) field.ErrorList
}

type Validator interface {
	Validate(ctx context.Context, obj runtime.Object) field.ErrorList
}

type PrepareForCreator interface {
	PrepareForCreate(ctx context.Context, obj runtime.Object)
}

type NamespaceScoper interface {
	NamespaceScoped() bool
}

var _ rest.Creater = (*CreateAdapter)(nil)

func NewCreate(schema *runtime.Scheme, strategy Creater) *CreateAdapter {
	return &CreateAdapter{
		NameGenerator: names.SimpleNameGenerator,
		Scheme:        schema,
		strategy:      strategy,
	}
}

type CreateAdapter struct {
	names.NameGenerator
	*runtime.Scheme
	strategy          Creater
	Warner            WarningsOnCreator
	Validator         Validator
	NameValidator     NameValidator
	PrepareForCreater PrepareForCreator
}

func (a *CreateAdapter) New() runtime.Object {
	return a.strategy.New()
}

func (a *CreateAdapter) Create(ctx context.Context, obj runtime.Object, createValidation rest.ValidateObjectFunc, options *metav1.CreateOptions) (runtime.Object, error) {
	if objectMeta, err := meta.Accessor(obj); err == nil {
		rest.FillObjectMetaSystemFields(objectMeta)
		if objectMeta.GetName() == "" {
			requestInfo, ok := request.RequestInfoFrom(ctx)
			if ok && requestInfo.Name != "" {
				objectMeta.SetName(requestInfo.Name)
			}
		}
	} else {
		return nil, err
	}

	if err := rest.BeforeCreate(a, ctx, obj); err != nil {
		return nil, err
	}

	// at this point we have a fully formed object.  It is time to call the validators that the apiserver
	// handling chain wants to enforce.
	if createValidation != nil {
		if err := createValidation(ctx, obj.DeepCopyObject()); err != nil {
			return nil, err
		}
	}

	return a.strategy.Create(ctx, obj.(types.Object))
}

func (a *CreateAdapter) PrepareForCreate(ctx context.Context, obj runtime.Object) {
	if a.PrepareForCreater != nil {
		a.PrepareForCreater.PrepareForCreate(ctx, obj)
	} else if o, ok := a.strategy.(PrepareForCreator); ok {
		o.PrepareForCreate(ctx, obj)
	}
}

func checkNamespace(nsed bool, obj runtime.Object) *field.Error {
	o := obj.(types.Object)
	if nsed && o.GetNamespace() == "" {
		return field.Forbidden(field.NewPath("metadata", "namespace"), "namespace must be set for namespaced scoped resource")
	} else if !nsed && o.GetNamespace() != "" {
		return field.Forbidden(field.NewPath("metadata", "namespace"), "namespace must not be set for cluster scoped resource")
	}
	return nil
}

func (a *CreateAdapter) Validate(ctx context.Context, obj runtime.Object) (result field.ErrorList) {
	if a.NameValidator != nil {
		result = append(result, a.NameValidator.ValidateName(ctx, obj)...)
	} else if o, ok := a.strategy.(NameValidator); ok {
		result = append(result, o.ValidateName(ctx, obj)...)
	} else {
		result = append(result, validator.ValidDNSLabel.ValidateName(ctx, obj)...)
	}
	if err := checkNamespace(a.NamespaceScoped(), obj); err != nil {
		result = append(result, err)
	}
	if a.Validator != nil {
		result = append(result, a.Validator.Validate(ctx, obj)...)
	} else if o, ok := a.strategy.(Validator); ok {
		result = append(result, o.Validate(ctx, obj)...)
	}
	return
}

func (a *CreateAdapter) WarningsOnCreate(ctx context.Context, obj runtime.Object) []string {
	if a.Warner != nil {
		return a.Warner.WarningsOnCreate(ctx, obj)
	}
	if o, ok := a.strategy.(WarningsOnCreator); ok {
		return o.WarningsOnCreate(ctx, obj)
	}
	return nil
}

func (a *CreateAdapter) Canonicalize(obj runtime.Object) {
}

func (a *CreateAdapter) NamespaceScoped() bool {
	if o, ok := a.strategy.(NamespaceScoper); ok {
		return o.NamespaceScoped()
	}
	if o, ok := a.strategy.New().(NamespaceScoper); ok {
		return o.NamespaceScoped()
	}
	return true
}
