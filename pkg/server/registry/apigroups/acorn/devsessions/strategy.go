package devsessions

import (
	"context"
	"fmt"

	"github.com/acorn-io/baaah/pkg/router"
	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/server/registry/apigroups/acorn/apps"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type Validator struct {
	client       kclient.Client
	appValidator *apps.Validator
}

func NewValidator(c kclient.Client, appValidator *apps.Validator) *Validator {
	return &Validator{
		client:       c,
		appValidator: appValidator,
	}
}

func (v *Validator) Validate(ctx context.Context, obj runtime.Object) (result field.ErrorList) {
	devSession := obj.(*apiv1.DevSession)
	app := &apiv1.App{}
	if err := v.client.Get(ctx, router.Key(devSession.Namespace, devSession.Name), app); err != nil {
		result = append(result, field.Invalid(field.NewPath("metadata", "name"), devSession.Name, err.Error()))
		return
	}

	if devSession.Spec.Region != app.GetRegion() {
		result = append(result, field.Invalid(field.NewPath("spec", "region"), devSession.Spec.Region,
			fmt.Sprintf("Region on devSession [%s] and app [%s] must match", devSession.Spec.Region, app.GetRegion())))
		return
	}

	if devSession.Spec.SpecOverride == nil {
		return nil
	}

	app.Spec = *devSession.Spec.SpecOverride
	app.Status.DevSession = nil

	errs := v.appValidator.Validate(ctx, app)
	// Super important that we set the image perms from validation
	devSession.Spec.SpecOverride.ImageGrantedPermissions = app.Spec.ImageGrantedPermissions
	return errs
}

func (v *Validator) ValidateUpdate(ctx context.Context, obj, old runtime.Object) (result field.ErrorList) {
	oldObj := old.(*apiv1.DevSession)
	newObj := obj.(*apiv1.DevSession)

	if oldObj.Spec.SpecOverride == nil {
		return v.Validate(ctx, obj)
	} else if newObj.Spec.SpecOverride == nil {
		return nil
	}

	app := &apiv1.App{}
	if err := v.client.Get(ctx, router.Key(newObj.Namespace, newObj.Name), app); err != nil {
		result = append(result, field.Invalid(field.NewPath("metadata", "name"), newObj.Name, err.Error()))
		return
	}

	oldApp := app.DeepCopy()
	oldApp.Spec = *oldObj.Spec.SpecOverride
	oldApp.Status.DevSession = nil

	newApp := app.DeepCopy()
	newApp.Spec = *newObj.Spec.SpecOverride
	newApp.Status.DevSession = nil

	errs := v.appValidator.AllowNestedUpdate().ValidateUpdate(ctx, newApp, oldApp)
	// Super important that we set the image perms from validation
	newObj.Spec.SpecOverride.ImageGrantedPermissions = newApp.Spec.ImageGrantedPermissions
	return errs
}
