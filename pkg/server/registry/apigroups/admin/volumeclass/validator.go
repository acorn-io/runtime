package volumeclass

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

var validAccessModes = map[v1.AccessMode]struct{}{
	v1.AccessModeReadWriteOnce: {},
	v1.AccessModeReadWriteMany: {},
	v1.AccessModeReadOnlyMany:  {},
}

type ProjectValidator struct {
	client kclient.Client
}

func NewProjectValidator(client kclient.Client) *ProjectValidator {
	return &ProjectValidator{
		client: client,
	}
}

func (s *ProjectValidator) Validate(ctx context.Context, obj runtime.Object) (result field.ErrorList) {
	vc := obj.(*adminv1.ProjectVolumeClass)
	if vc.Default && !vc.Inactive {
		// Ensure only one default is set for ProjectVolumeClasses
		projectVolumeClasses := new(admininternalv1.ProjectVolumeClassInstanceList)
		if err := s.client.List(ctx, projectVolumeClasses, &kclient.ListOptions{Namespace: vc.Namespace}); err != nil {
			return append(result, field.InternalError(field.NewPath("default"), err))
		}

		for _, pvc := range projectVolumeClasses.Items {
			if pvc.Default && !pvc.Inactive && pvc.Name != vc.Name {
				return append(result, field.Invalid(field.NewPath("default"), vc.Default, fmt.Sprintf("%s is already default for project", pvc.Name)))
			}
		}
	}

	if err := validateVolumeClass((*admininternalv1.ProjectVolumeClassInstance)(vc)); err != nil {
		return err
	}

	return
}

func (s *ProjectValidator) ValidateUpdate(ctx context.Context, newObj, oldObj runtime.Object) field.ErrorList {
	newStorageClassName := newObj.(*adminv1.ProjectVolumeClass).StorageClassName
	if newStorageClassName != oldObj.(*adminv1.ProjectVolumeClass).StorageClassName {
		return []*field.Error{field.Invalid(field.NewPath("storageClassName"), newStorageClassName, "storageClassName cannot be changed")}
	}

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
	vc := obj.(*adminv1.ClusterVolumeClass)
	if vc.Default && !vc.Inactive {
		// Ensure only one default is set for ClusterVolumeClasses
		clusterVolumeClasses := new(admininternalv1.ClusterVolumeClassInstanceList)
		if err := s.client.List(ctx, clusterVolumeClasses, &kclient.ListOptions{Namespace: vc.Namespace}); err != nil {
			return append(result, field.InternalError(field.NewPath("default"), err))
		}

		for _, cvc := range clusterVolumeClasses.Items {
			if cvc.Default && !cvc.Inactive && cvc.Name != vc.Name {
				return append(result, field.Invalid(field.NewPath("default"), vc.Default, fmt.Sprintf("%s is already default for cluster", cvc.Name)))
			}
		}
	}

	if err := validateVolumeClass((*admininternalv1.ProjectVolumeClassInstance)(vc)); err != nil {
		return err
	}

	return
}

func (s *ClusterValidator) ValidateUpdate(ctx context.Context, newObj, oldObj runtime.Object) field.ErrorList {
	newStorageClassName := newObj.(*adminv1.ClusterVolumeClass).StorageClassName
	if newStorageClassName != oldObj.(*adminv1.ClusterVolumeClass).StorageClassName {
		return []*field.Error{field.Invalid(field.NewPath("storageClassName"), newStorageClassName, "storageClassName cannot be changed")}
	}
	return s.Validate(ctx, newObj)
}

func validateVolumeClass(class *admininternalv1.ProjectVolumeClassInstance) (result field.ErrorList) {
	// Ensure the min, max, and default make sense.
	if compareQuantities(class.Size.Min, class.Size.Max) > 0 {
		result = append(result, field.Invalid(field.NewPath("size", "min"), class.Size.Min, "min size should be at most max size"))
	}
	if compareQuantities(class.Size.Min, class.Size.Default) > 0 {
		result = append(result, field.Invalid(field.NewPath("size", "default"), class.Size.Default, "default size should be at least min size"))
	}
	if compareQuantities(class.Size.Default, class.Size.Max) > 0 {
		result = append(result, field.Invalid(field.NewPath("size", "default"), class.Size.Default, "default size should be at most max size"))
	}

	// Ensure the allowedAccessModes are valid.
	for i, am := range class.AllowedAccessModes {
		if _, ok := validAccessModes[am]; !ok {
			result = append(result, field.Invalid(field.NewPath("allowedAccessModes").Index(i), am, "invalid access mode"))
		}
	}

	return
}

func compareQuantities(x, y v1.Quantity) int {
	if x == "" || y == "" {
		// If one or the other is empty, then nothing to compare.
		return 0
	}
	return v1.MustParseResourceQuantity(x).Cmp(*v1.MustParseResourceQuantity(y))
}
