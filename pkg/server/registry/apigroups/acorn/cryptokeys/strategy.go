package cryptokeys

import (
	"context"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

type Strategy struct {
}

func (s *Strategy) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {
	s.PrepareForCreate(ctx, obj)
}

func (s *Strategy) PrepareForCreate(ctx context.Context, obj runtime.Object) {
	//cred := obj.(*apiv1.CryptoKey)
	// TODO: to PEM format
}

func (s *Strategy) Validate(ctx context.Context, obj runtime.Object) (result field.ErrorList) {
	params := obj.(*apiv1.CryptoKey)
	if err := CryptoKeyValidate(ctx, params.Key); err != nil {
		result = append(result, field.Forbidden(field.NewPath("key"), err.Error()))
	}
	return result
}
func (s *Strategy) ValidateUpdate(ctx context.Context, obj, old runtime.Object) (result field.ErrorList) {
	params := obj.(*apiv1.CryptoKey)
	return s.Validate(ctx, params)
}

// CryptoKeyValidate checks whether the given key is in a supported format
func CryptoKeyValidate(ctx context.Context, key string) error {
	// TODO: validate key

	return nil
}
