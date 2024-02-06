package secrets

import (
	"context"
	"strings"

	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

type Validator struct {
}

func (v *Validator) Validate(_ context.Context, obj runtime.Object) (result field.ErrorList) {
	sec := obj.(*apiv1.Secret)
	if sec.Type != "" {
		if !v1.SecretTypes[corev1.SecretType(v1.SecretTypePrefix+sec.Type)] && !strings.HasPrefix(sec.Type, v1.SecretTypeCredentialPrefix) {
			result = append(result, field.Invalid(field.NewPath("type"), sec.Type, "Invalid secret type"))
		}
	}
	return
}

func (v *Validator) ValidateUpdate(ctx context.Context, obj, _ runtime.Object) field.ErrorList {
	return v.Validate(ctx, obj)
}
