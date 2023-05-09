package projects

import (
	"context"
	"fmt"
	"strings"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/utils/strings/slices"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type Validator struct {
	DefaultRegion string
	Client        kclient.Client
}

func (v *Validator) Validate(_ context.Context, obj runtime.Object) field.ErrorList {
	var result field.ErrorList
	project := obj.(*apiv1.Project)
	if project.Spec.DefaultRegion == "" && len(project.Spec.SupportedRegions) == 0 {
		// If no regions are specified, use the default region.
		project.Status.DefaultRegion = v.DefaultRegion
		return nil
	}

	// Reset the default region on status to indicate that a "real" default is set.
	project.Status.DefaultRegion = ""

	if !project.HasRegion(project.Spec.DefaultRegion) {
		return append(result, field.Invalid(field.NewPath("spec", "defaultRegion"), project.Spec.DefaultRegion, "default region is not in the supported regions list"))
	}

	return nil
}

func (v *Validator) ValidateUpdate(ctx context.Context, new, old runtime.Object) field.ErrorList {
	// Ensure that default region and supported regions are valid.
	if err := v.Validate(ctx, new); err != nil {
		return err
	}

	// If the user is removing a supported region, ensure that there are no apps in that region.
	oldProject, newProject := old.(*apiv1.Project), new.(*apiv1.Project)
	var removedRegions []string
	for _, region := range append(oldProject.Spec.SupportedRegions, oldProject.Status.DefaultRegion) {
		if !newProject.HasRegion(region) {
			removedRegions = append(removedRegions, region)
		}
	}

	if len(removedRegions) > 0 {
		var (
			appList              apiv1.AppList
			appsInRemovedRegions []string
		)
		if err := v.Client.List(ctx, &appList, kclient.InNamespace(newProject.Name)); err != nil {
			return field.ErrorList{field.InternalError(field.NewPath("spec", "supportedRegions"), err)}
		}

		for _, app := range appList.Items {
			if slices.Contains(removedRegions, app.GetRegion()) {
				appsInRemovedRegions = append(appsInRemovedRegions, app.Name)
			}
		}

		if len(appsInRemovedRegions) > 0 {
			return field.ErrorList{
				field.Invalid(
					field.NewPath("spec", "supportedRegions"),
					newProject.GetSupportedRegions(),
					fmt.Sprintf("cannot remove regions %v that have apps: %v", removedRegions, strings.Join(appsInRemovedRegions, ", ")),
				),
			}
		}
	}

	return nil
}
