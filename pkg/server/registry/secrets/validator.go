package secrets

import (
	"context"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

type Validator struct {
}

func (v *Validator) Validate(ctx context.Context, obj runtime.Object) (result field.ErrorList) {
	sec := obj.(*apiv1.Secret)
	if sec.Type != "" {
		if !v1.SecretTypes[corev1.SecretType(v1.SecretTypePrefix+sec.Type)] {
			result = append(result, field.Invalid(field.NewPath("type"), sec.Type, "Invalid secret type"))
		}
	}
	return
}

func (v *Validator) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	return v.Validate(ctx, obj)
}
