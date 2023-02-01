package workloadclass

import (
	"context"
	"fmt"

	adminv1 "github.com/acorn-io/acorn/pkg/apis/admin.acorn.io/v1"
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	admininternalv1 "github.com/acorn-io/acorn/pkg/apis/internal.admin.acorn.io/v1"
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
	wc := obj.(*adminv1.ProjectWorkloadClass)
	if wc.Default {
		// Ensure only one default is set for ProjectWorkloadClasses
		projectWorkloadClasses := new(admininternalv1.ProjectWorkloadClassInstanceList)
		if err := s.client.List(ctx, projectWorkloadClasses, &kclient.ListOptions{Namespace: wc.Namespace}); err != nil {
			return append(result, field.InternalError(field.NewPath("spec", "default"), err))
		}

		for _, pwc := range projectWorkloadClasses.Items {
			if pwc.Default && pwc.Name != wc.Name {
				return append(result, field.Invalid(field.NewPath("spec", "default"), wc.Default, fmt.Sprintf("%s is already default for project", pwc.Name)))
			}
		}
	}

	if _, err := admininternalv1.ParseWorkloadClassMemory(wc.Memory); err != nil {
		return append(result, field.Invalid(field.NewPath("spec", "memory"), wc.Memory, err.Error()))
	}

	min, max, def := v1.Quantity(wc.Memory.Min), v1.Quantity(wc.Memory.Max), v1.Quantity(wc.Memory.Default)
	// Ensure the min, max, and default make sense.
	if compareQuantities(min, max) > 0 && min != "0" && max != "0" {
		result = append(result, field.Invalid(field.NewPath("spec", "memory", "min"), min, "minimum memory should be at most the maximum memory"))
	}
	if compareQuantities(min, def) > 0 {
		result = append(result, field.Invalid(field.NewPath("spec", "memory", "default"), def, "default memory should be at least the minimum memory"))
	}
	if compareQuantities(def, max) > 0 && max != "0" {
		result = append(result, field.Invalid(field.NewPath("spec", "memory", "default"), def, "default memory should be at most the maximum memory"))
	}

	for i, value := range wc.Memory.Values {
		valueAsQuantity := v1.Quantity(value)
		if compareQuantities(min, valueAsQuantity) > 0 && value != "0" {
			result = append(result, field.Invalid(
				field.NewPath("spec", "memory", "values", fmt.Sprint(i)),
				value,
				"allowed value should be at least minimum memory"))
		}
		if compareQuantities(valueAsQuantity, max) > 0 {
			result = append(result, field.Invalid(
				field.NewPath("spec", "memory", "values", fmt.Sprint(i)),
				value,
				"allowed value should be at most maximum memory"))
		}
	}

	return result
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
	wc := obj.(*adminv1.ClusterWorkloadClass)
	if wc.Default {
		// Ensure only one default is set for ClusterWorkloadClasses
		clusterWorkloadClasses := new(admininternalv1.ClusterWorkloadClassInstanceList)
		if err := s.client.List(ctx, clusterWorkloadClasses, &kclient.ListOptions{Namespace: wc.Namespace}); err != nil {
			return append(result, field.InternalError(field.NewPath("default"), err))
		}

		for _, pwc := range clusterWorkloadClasses.Items {
			if pwc.Default && pwc.Name != wc.Name {
				return append(result, field.Invalid(field.NewPath("spec.default"), wc.Default, fmt.Sprintf("%s is already default for project", pwc.Name)))
			}
		}
	}

	if _, err := admininternalv1.ParseWorkloadClassMemory(wc.Memory); err != nil {
		return append(result, field.Invalid(field.NewPath("spec.memory"), wc.Memory, err.Error()))
	}

	min, max, def := v1.Quantity(wc.Memory.Min), v1.Quantity(wc.Memory.Max), v1.Quantity(wc.Memory.Default)
	// Ensure the min, max, and default make sense.
	if compareQuantities(min, max) > 0 && (min != "0" || max != "0") {
		result = append(result, field.Invalid(field.NewPath("spec", "memory", "min"), min, "minimum memory should be at most the maximum memory"))
	}
	if compareQuantities(min, def) > 0 {
		result = append(result, field.Invalid(field.NewPath("spec", "memory", "default"), def, "default memory should be at least the minimum memory"))
	}
	if compareQuantities(def, max) > 0 && max != "0" {
		result = append(result, field.Invalid(field.NewPath("spec", "memory", "default"), def, "default memory should be at most the maximum memory"))
	}

	for i, value := range wc.Memory.Values {
		valueAsQuantity := v1.Quantity(value)
		if compareQuantities(min, valueAsQuantity) > 0 && value != "0" {
			result = append(result, field.Invalid(
				field.NewPath("spec", "memory", "values", fmt.Sprint(i)),
				value,
				"allowed value should be at least minimum memory"))
		}
		if compareQuantities(valueAsQuantity, max) > 0 {
			result = append(result, field.Invalid(
				field.NewPath("spec", "memory", "values", fmt.Sprint(i)),
				value,
				"allowed value should be at most maximum memory"))
		}
	}

	return result
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
