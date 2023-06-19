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
	project.SetDefaultRegion(v.DefaultRegion)

	if !project.HasRegion(project.Spec.DefaultRegion) {
		return append(result, field.Invalid(field.NewPath("spec", "defaultRegion"), project.Spec.DefaultRegion, "default region is not in the supported regions list"))
	}

	return nil
}

func (v *Validator) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	// Ensure that default region and supported regions are valid.
	if err := v.Validate(ctx, obj); err != nil {
		return err
	}

	// If the user is removing a supported region, ensure that there are no apps in that region.
	oldProject, newProject := old.(*apiv1.Project), obj.(*apiv1.Project)
	var removedRegions []string
	for _, region := range oldProject.Status.SupportedRegions {
		if !newProject.HasRegion(region) {
			removedRegions = append(removedRegions, region)
		}
	}

	if len(removedRegions) > 0 {
		return v.ensureNoObjectsExistInRegions(ctx, newProject.Name, newProject.Status.SupportedRegions, removedRegions, &apiv1.AppList{}, &apiv1.VolumeList{})
	}

	return nil
}

func (v *Validator) ensureNoObjectsExistInRegions(ctx context.Context, namespace string, regions, removedRegions []string, objList ...kclient.ObjectList) field.ErrorList {
	var result field.ErrorList
	for _, obj := range objList {
		var inRemovedRegion []string
		if err := v.Client.List(ctx, obj, kclient.InNamespace(namespace)); err != nil {
			return field.ErrorList{field.InternalError(field.NewPath("spec", "supportedRegions"), err)}
		}

		if err := meta.EachListItem(obj, func(object runtime.Object) error {
			regionObject, ok := object.(regionNamer)
			if ok && slices.Contains(removedRegions, regionObject.GetRegion()) {
				inRemovedRegion = append(inRemovedRegion, regionObject.GetName())
			}
			return nil
		}); err != nil {
			return field.ErrorList{field.InternalError(field.NewPath("spec", "supportedRegions"), err)}
		}

		if len(inRemovedRegion) > 0 {
			result = append(result, field.Invalid(
				field.NewPath("spec", "supportedRegions"),
				regions,
				fmt.Sprintf(
					"cannot remove regions %v while in use by the following %ss: %v",
					removedRegions,
					v.resource(obj),
					inRemovedRegion,
				),
			))
		}
	}

	return result
}

func (v *Validator) resource(obj runtime.Object) string {
	kind := obj.GetObjectKind().GroupVersionKind().Kind
	if kind == "" {
		gvks, _, _ := v.Client.Scheme().ObjectKinds(obj)
		if len(gvks) < 1 {
			// Kind unknown
			return "resource"
		}
		kind = gvks[0].Kind
	}

	return strings.TrimSuffix(strings.ToLower(kind), "list")
}
