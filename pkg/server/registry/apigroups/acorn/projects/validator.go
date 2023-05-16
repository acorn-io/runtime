package projects

import (
	"context"
	"fmt"
	"strings"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/utils/strings/slices"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type regionNamer interface {
	GetRegion() string
	GetName() string
}

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
		return v.ensureNoObjectsExistInRegions(ctx, newProject.Name, removedRegions, &apiv1.AppList{}, &apiv1.VolumeList{})
	}

	return nil
}

func (v *Validator) ensureNoObjectsExistInRegions(ctx context.Context, namespace string, regions []string, objList ...kclient.ObjectList) field.ErrorList {
	var result field.ErrorList
	for _, obj := range objList {
		objectsInRemoveRegions := make([]string, 0)
		if err := v.Client.List(ctx, obj, kclient.InNamespace(namespace)); err != nil {
			return field.ErrorList{field.InternalError(field.NewPath("spec", "supportedRegions"), err)}
		}

		if err := meta.EachListItem(obj, func(object runtime.Object) error {
			regionObject, ok := object.(regionNamer)
			if ok && slices.Contains(regions, regionObject.GetRegion()) {
				objectsInRemoveRegions = append(objectsInRemoveRegions, regionObject.GetName())
			}
			return nil
		}); err != nil {
			return field.ErrorList{field.InternalError(field.NewPath("spec", "supportedRegions"), err)}
		}

		if len(objectsInRemoveRegions) > 0 {
			result = append(result, field.Invalid(
				field.NewPath("spec", "supportedRegions"),
				regions,
				fmt.Sprintf(
					"cannot remove regions %v that have %s: %v",
					regions,
					strings.TrimSuffix(strings.ToLower(obj.GetObjectKind().GroupVersionKind().Kind), "list"),
					strings.Join(objectsInRemoveRegions, ", "),
				),
			))
		}
	}

	return result
}
