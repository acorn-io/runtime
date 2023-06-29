package keys

import (
	"context"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

type Validator struct{}

func (s *Validator) Validate(ctx context.Context, obj runtime.Object) (result field.ErrorList) {
	pubkey := obj.(*apiv1.PublicKey)
	result = append(result, validateKey(ctx, pubkey.Key)...)
	return
}

func (s *Validator) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	return s.Validate(ctx, obj)
}

func validateKey(ctx context.Context, pubkey string) (result field.ErrorList) {
	if pubkey == "" {
		// TODO: add validation for allowed key types
		result = append(result, field.Invalid(field.NewPath("key"), pubkey, "must not be empty"))
	}

	return
}
