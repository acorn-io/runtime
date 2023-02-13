package strategy

import (
	"context"

	"github.com/acorn-io/mink/pkg/types"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/apiserver/pkg/endpoints/request"
	genericapirequest "k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/rest"
)

type PrepareForUpdater interface {
	PrepareForUpdate(ctx context.Context, obj, old runtime.Object)
}

type ValidateUpdater interface {
	ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList
}

type WarningsOnUpdater interface {
	WarningsOnUpdate(ctx context.Context, obj, old runtime.Object) []string
}

type Updater interface {
	Getter
	Creater

	Update(ctx context.Context, obj types.Object) (types.Object, error)
}

type updaterCommon interface {
	Getter
	Creater
}

var _ rest.Updater = (*UpdateAdapter)(nil)

type UpdateAdapter struct {
	*CreateAdapter
	status            bool
	strategy          updaterCommon
	PrepareForUpdater PrepareForUpdater
	WarningsOnUpdater WarningsOnUpdater
	ValidateUpdater   ValidateUpdater
}

func NewUpdate(schema *runtime.Scheme, strategy Updater) *UpdateAdapter {
	return &UpdateAdapter{
		CreateAdapter: NewCreate(schema, strategy),
		strategy:      strategy,
	}
}

func (a *UpdateAdapter) AllowUnconditionalUpdate() bool {
	return false
}

func (a *UpdateAdapter) AllowCreateOnUpdate() bool {
	return false
}

func (a *UpdateAdapter) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {
	if a.PrepareForUpdater != nil {
		a.PrepareForUpdater.PrepareForUpdate(ctx, obj, old)
	} else if o, ok := a.strategy.(PrepareForUpdater); ok {
		o.PrepareForUpdate(ctx, obj, old)
	}
}

func (a *UpdateAdapter) ValidateUpdate(ctx context.Context, obj, old runtime.Object) (result field.ErrorList) {
	if err := checkNamespace(a.NamespaceScoped(), obj); err != nil {
		result = append(result, err)
	}
	if a.ValidateUpdater != nil {
		result = append(result, a.ValidateUpdater.ValidateUpdate(ctx, obj, old)...)
	} else if o, ok := a.strategy.(ValidateUpdater); ok {
		result = append(result, o.ValidateUpdate(ctx, obj, old)...)
	}
	return
}

func (a *UpdateAdapter) WarningsOnUpdate(ctx context.Context, obj, old runtime.Object) []string {
	if a.WarningsOnUpdater != nil {
		return a.WarningsOnUpdater.WarningsOnUpdate(ctx, obj, old)
	}
	if o, ok := a.strategy.(WarningsOnUpdater); ok {
		return o.WarningsOnUpdate(ctx, obj, old)
	}
	return nil
}

func (a *UpdateAdapter) Update(ctx context.Context, name string, objInfo rest.UpdatedObjectInfo, createValidation rest.ValidateObjectFunc, updateValidation rest.ValidateObjectUpdateFunc, forceAllowCreate bool, options *metav1.UpdateOptions) (runtime.Object, bool, error) {
	return a.update(ctx, a.status, name, objInfo, createValidation, updateValidation, forceAllowCreate, options)
}

func (a *UpdateAdapter) update(ctx context.Context, status bool, name string, objInfo rest.UpdatedObjectInfo, createValidation rest.ValidateObjectFunc, updateValidation rest.ValidateObjectUpdateFunc, forceAllowCreate bool, options *metav1.UpdateOptions) (runtime.Object, bool, error) {
	doCreate := false

	ns, _ := genericapirequest.NamespaceFrom(ctx)
	var existing runtime.Object
	existing, err := a.strategy.Get(ctx, ns, name)
	if apierrors.IsNotFound(err) && forceAllowCreate {
		existing = a.strategy.New()
		doCreate = true
	} else if err != nil {
		return nil, false, err
	}

	// Given the existing object, get the new object
	obj, err := objInfo.UpdatedObject(ctx, existing)
	if err != nil {
		return nil, false, err
	}

	if obj.(types.Object).GetResourceVersion() == "" && existing.(types.Object).GetResourceVersion() != "" {
		obj.(types.Object).SetResourceVersion(existing.(types.Object).GetResourceVersion())
	}

	if doCreate {
		if objectMeta, err := meta.Accessor(obj); err == nil {
			rest.FillObjectMetaSystemFields(objectMeta)
			if objectMeta.GetName() == "" {
				requestInfo, ok := request.RequestInfoFrom(ctx)
				if ok && requestInfo.Name != "" {
					objectMeta.SetName(requestInfo.Name)
				}
			}
		} else {
			return nil, false, err
		}

		if err := rest.BeforeCreate(a, ctx, obj); err != nil {
			return nil, false, err
		}
		if createValidation != nil {
			if err := createValidation(ctx, obj.DeepCopyObject()); err != nil {
				return nil, false, err
			}
		}

		newObj, err := a.strategy.Create(ctx, obj.(types.Object))
		return newObj, true, err
	}

	if err := rest.BeforeUpdate(a, ctx, obj, existing); err != nil {
		return nil, false, err
	}

	// at this point we have a fully formed object.  It is time to call the validators that the apiserver
	// handling chain wants to enforce.
	if updateValidation != nil {
		if err := updateValidation(ctx, obj.DeepCopyObject(), existing.DeepCopyObject()); err != nil {
			return nil, false, err
		}
	}

	if status {
		newObj, err := a.strategy.(StatusUpdater).UpdateStatus(ctx, obj.(types.Object))
		return newObj, false, err
	}

	newObj, err := a.strategy.(Updater).Update(ctx, obj.(types.Object))
	return newObj, false, err
}

func (a *UpdateAdapter) qualifiedResourceFromContext(ctx context.Context) schema.GroupResource {
	if info, ok := genericapirequest.RequestInfoFrom(ctx); ok {
		return schema.GroupResource{Group: info.APIGroup, Resource: info.Resource}
	}
	return schema.GroupResource{}
}
