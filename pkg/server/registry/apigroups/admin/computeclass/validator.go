package computeclass

import (
	"context"
	"fmt"

	adminv1 "github.com/acorn-io/acorn/pkg/apis/admin.acorn.io/v1"
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	admininternalv1 "github.com/acorn-io/acorn/pkg/apis/internal.admin.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/computeclasses"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type ProjectValidator struct {
	client kclient.Client
}

func NewProjectValidator(client kclient.Client) *ProjectValidator {
	return &ProjectValidator{
		client: client,
	}
}

func (s *ProjectValidator) Validate(ctx context.Context, obj runtime.Object) (result field.ErrorList) {
	cc := obj.(*adminv1.ProjectComputeClass)
	if cc.Default {
		// Ensure only one default is set for ProjectComputeClasses
		projectComputeClasses := new(admininternalv1.ProjectComputeClassInstanceList)
		if err := s.client.List(ctx, projectComputeClasses, &kclient.ListOptions{Namespace: cc.Namespace}); err != nil {
			return append(result, field.InternalError(field.NewPath("spec", "default"), err))
		}

		for _, pcc := range projectComputeClasses.Items {
			if pcc.Default && pcc.Name != cc.Name {
				return append(result, field.Invalid(field.NewPath("spec", "default"), cc.Default, fmt.Sprintf("%s is already default for project", pcc.Name)))
			}
		}
	}

	if _, err := computeclasses.ParseComputeClassMemory(cc.Memory); err != nil {
		return append(result, field.Invalid(field.NewPath("spec", "memory"), cc.Memory, err.Error()))
	}

	return append(result, validateMemorySpec(cc.Memory)...)
}

func (s *ProjectValidator) ValidateUpdate(ctx context.Context, newObj, oldObj runtime.Object) field.ErrorList {
	return s.Validate(ctx, newObj)
}

type ClusterValidator struct {
	client kclient.Client
}

func NewClusterValidator(client kclient.Client) *ClusterValidator {
	return &ClusterValidator{
		client: client,
	}
}

func (s *ClusterValidator) Validate(ctx context.Context, obj runtime.Object) (result field.ErrorList) {
	cc := obj.(*adminv1.ClusterComputeClass)
	if cc.Default {
		// Ensure only one default is set for ClusterComputeClasses
		clusterComputeClasses := new(admininternalv1.ClusterComputeClassInstanceList)
		if err := s.client.List(ctx, clusterComputeClasses, &kclient.ListOptions{Namespace: cc.Namespace}); err != nil {
			return append(result, field.InternalError(field.NewPath("default"), err))
		}

		for _, pcc := range clusterComputeClasses.Items {
			if pcc.Default && pcc.Name != cc.Name {
				return append(result, field.Invalid(field.NewPath("spec.default"), cc.Default, fmt.Sprintf("%s is already default for project", pcc.Name)))
			}
		}
	}

	if _, err := computeclasses.ParseComputeClassMemory(cc.Memory); err != nil {
		return append(result, field.Invalid(field.NewPath("spec.memory"), cc.Memory, err.Error()))
	}

	return append(result, validateMemorySpec(cc.Memory)...)
}

func validateMemorySpec(memory admininternalv1.ComputeClassMemory) field.ErrorList {
	errors := field.ErrorList{}
	if len(memory.Values) != 0 {
		if memory.Max != "" {
			errors = append(errors, field.Invalid(field.NewPath("spec", "memory", "max"), memory.Max, "cannot set maximum memory with values specified"))
		}
		if memory.Min != "" {
			errors = append(errors, field.Invalid(field.NewPath("spec", "memory", "min"), memory.Min, "cannot set minimum memory with values specified"))
		}
	}

	min, max, def := v1.Quantity(memory.Min), v1.Quantity(memory.Max), v1.Quantity(memory.Default)
	// Ensure the min, max, and default make sense.
	if compareQuantities(min, max) > 0 && (min != "0" || max != "0") {
		errors = append(errors, field.Invalid(field.NewPath("spec", "memory", "min"), min, "minimum memory should be at most the maximum memory"))
	}
	if compareQuantities(min, def) > 0 {
		errors = append(errors, field.Invalid(field.NewPath("spec", "memory", "default"), def, "default memory should be at least the minimum memory"))
	}
	if compareQuantities(def, max) > 0 && max != "0" {
		errors = append(errors, field.Invalid(field.NewPath("spec", "memory", "default"), def, "default memory should be at most the maximum memory"))
	}

	if len(memory.Values) == 0 {
		return errors
	}

	memoryIncluded := false
	for _, value := range memory.Values {
		valueAsQuantity := v1.Quantity(value)

		if compareQuantities(valueAsQuantity, def) == 0 {
			memoryIncluded = true
		}
	}

	if !memoryIncluded {
		errors = append(errors,
			field.Invalid(
				field.NewPath("spec", "memory", "default"), def,
				fmt.Sprintf("default memory is not included in values. current values: %v", memory.Values)),
		)
	}
	return errors
}

func (s *ClusterValidator) ValidateUpdate(ctx context.Context, newObj, _ runtime.Object) field.ErrorList {
	return s.Validate(ctx, newObj)
}

func compareQuantities(x, y v1.Quantity) int {
	if x == "" || y == "" {
		// If one or the other is empty, then nothing to compare.
		return 0
	}
	return v1.MustParseResourceQuantity(x).Cmp(*v1.MustParseResourceQuantity(y))
}
