package projects

import (
	"context"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type Validator struct {
	Client        kclient.Client
	DefaultRegion string
}

func (v *Validator) Validate(ctx context.Context, obj runtime.Object) field.ErrorList {
	var result field.ErrorList
	project := obj.(*apiv1.Project)
	if project.Spec.DefaultRegion == "" && len(project.Spec.SupportedRegions) == 0 {
		// If no regions are specified, use the default region.
		project.Status.DefaultRegion = v.DefaultRegion
		return nil
	}

	// Reset the default region on status to indicate that a "real" default is set.
	project.Status.DefaultRegion = ""

	if !project.ForRegion(project.Spec.DefaultRegion) {
		return append(result, field.Invalid(field.NewPath("spec", "defaultRegion"), project.Spec.DefaultRegion, "default region is not in the supported regions list"))
	}

	return nil
}

func (v *Validator) ValidateUpdate(ctx context.Context, obj, _ runtime.Object) field.ErrorList {
	return v.Validate(ctx, obj)
}
