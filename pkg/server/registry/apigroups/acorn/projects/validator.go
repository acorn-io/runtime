package projects

import (
	"context"
	"fmt"
	"strings"

	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
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
	Client kclient.Client
}

func (v *Validator) Validate(_ context.Context, obj runtime.Object) field.ErrorList {
	var result field.ErrorList
	project := obj.(*apiv1.Project)

	if project.Spec.DefaultRegion != "" && !slices.Contains(project.Spec.SupportedRegions, project.Spec.DefaultRegion) && !slices.Contains(project.Spec.SupportedRegions, apiv1.AllRegions) {
		return append(result, field.Invalid(field.NewPath("spec", "defaultRegion"), project.Spec.DefaultRegion, "default region is not in the supported regions list"))
	}

	return nil
}

func (v *Validator) ValidateUpdate(ctx context.Context, newObj, _ runtime.Object) field.ErrorList {
	// Ensure that default region and supported regions are valid.
	if err := v.Validate(ctx, newObj); err != nil {
		return err
	}

	newProject := newObj.(*apiv1.Project)
	// If there are no supported regions given by the user (and the above validate call passed) or the user explicitly
	// allowed all regions, then the project supports all regions.
	if len(newProject.Spec.SupportedRegions) == 0 || slices.Contains(newProject.Spec.SupportedRegions, apiv1.AllRegions) {
		return nil
	}

	// If the user is removing a supported region, ensure that there are no apps in that region.
	return EnsureNoObjectsExistOutsideOfRegions(ctx, v.Client, newProject.Name, newProject.Spec.SupportedRegions, &apiv1.AppList{}, &apiv1.VolumeList{})
}

func EnsureNoObjectsExistOutsideOfRegions(ctx context.Context, client kclient.Client, namespace string, regions []string, objList ...kclient.ObjectList) field.ErrorList {
	var result field.ErrorList
	for _, obj := range objList {
		var removedRegions, inRemovedRegion []string
		if err := client.List(ctx, obj, kclient.InNamespace(namespace)); err != nil {
			return field.ErrorList{field.InternalError(field.NewPath("spec", "supportedRegions"), err)}
		}

		if err := meta.EachListItem(obj, func(object runtime.Object) error {
			regionObject, ok := object.(regionNamer)
			if ok && !slices.Contains(regions, regionObject.GetRegion()) {
				removedRegions = append(removedRegions, regionObject.GetRegion())
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
					resource(client, obj),
					inRemovedRegion,
				),
			))
		}
	}

	return result
}

func resource(client kclient.Client, obj runtime.Object) string {
	kind := obj.GetObjectKind().GroupVersionKind().Kind
	if kind == "" {
		gvks, _, _ := client.Scheme().ObjectKinds(obj)
		if len(gvks) < 1 {
			// Kind unknown
			return "resource"
		}
		kind = gvks[0].Kind
	}

	return strings.TrimSuffix(strings.ToLower(kind), "list")
}
