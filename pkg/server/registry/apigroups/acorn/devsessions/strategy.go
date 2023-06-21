package devsessions

import (
	"context"

	"github.com/acorn-io/baaah/pkg/router"
	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/server/registry/apigroups/acorn/apps"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type Strategy struct {
	client       kclient.Client
	appValidator *apps.Validator
}

func NewValidator(c kclient.Client, appValidator *apps.Validator) *Strategy {
	return &Strategy{
		client:       c,
		appValidator: appValidator,
	}
}

func (s *Strategy) Validate(ctx context.Context, obj runtime.Object) (result field.ErrorList) {
	devSession := obj.(*apiv1.DevSession)
	if devSession.Spec.SpecOverride == nil {
		return nil
	}

	app := &apiv1.App{}
	if err := s.client.Get(ctx, router.Key(devSession.Namespace, devSession.Name), app); err != nil {
		result = append(result, field.Invalid(field.NewPath("metadata", "name"), devSession.Name, err.Error()))
		return
	}

	app.Spec = *devSession.Spec.SpecOverride
	app.Status.DevSession = nil
	return s.appValidator.Validate(ctx, app)
}

func (s *Strategy) ValidateUpdate(ctx context.Context, obj, old runtime.Object) (result field.ErrorList) {
	oldObj := old.(*apiv1.DevSession)
	newObj := obj.(*apiv1.DevSession)

	if oldObj.Spec.SpecOverride == nil {
		return s.Validate(ctx, obj)
	} else if newObj.Spec.SpecOverride == nil {
		return nil
	}

	app := &apiv1.App{}
	if err := s.client.Get(ctx, router.Key(newObj.Namespace, newObj.Name), app); err != nil {
		result = append(result, field.Invalid(field.NewPath("metadata", "name"), newObj.Name, err.Error()))
		return
	}

	oldApp := app.DeepCopy()
	oldApp.Spec = *oldObj.Spec.SpecOverride
	oldApp.Status.DevSession = nil

	newApp := app.DeepCopy()
	newApp.Spec = *newObj.Spec.SpecOverride
	newApp.Status.DevSession = nil

	return s.appValidator.ValidateUpdate(ctx, newApp, oldApp)
}
