package imageroleauthorizations

import (
	"context"

	adminv1 "github.com/acorn-io/runtime/pkg/apis/admin.acorn.io/v1"
	internalv1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

type Validator struct{}

func (s *Validator) Validate(ctx context.Context, obj runtime.Object) (result field.ErrorList) {
	aiar := obj.(*adminv1.ImageRoleAuthorization)
	if len(aiar.ImageSelector.NamePatterns) == 0 {
		return append(result, field.Required(field.NewPath("imageSelector", "namePatterns"), "the image selector patterns must be defined to specify which images this rule applies to"))
	}
	result = append(result, validateSignatureRules(aiar.ImageSelector.Signatures)...)
	return
}

func (s *Validator) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	return s.Validate(ctx, obj)
}

func validateSignatureRules(sigRules []internalv1.SignatureRules) (result field.ErrorList) {
	for i, rule := range sigRules {
		if len(rule.SignedBy.AnyOf) == 0 && len(rule.SignedBy.AllOf) == 0 {
			result = append(result, field.Invalid(field.NewPath("signatures").Index(i).Child("signedBy"), rule.SignedBy, "must not be empty (at least one of anyOf or allOf must be specified)"))
		}
	}

	return
}

type ClusterValidator struct{}

func (s *ClusterValidator) Validate(ctx context.Context, obj runtime.Object) (result field.ErrorList) {
	aiar := obj.(*adminv1.ClusterImageRoleAuthorization)
	if len(aiar.ImageSelector.NamePatterns) == 0 {
		return append(result, field.Required(field.NewPath("imageSelector", "namePatterns"), "the image selector patterns must be defined to specify which images this rule applies to"))
	}
	result = append(result, validateSignatureRules(aiar.ImageSelector.Signatures)...)
	return
}

func (s *ClusterValidator) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	return s.Validate(ctx, obj)
}
