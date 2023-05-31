package imageallowrules

import (
	"context"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	internalv1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

type Validator struct{}

func (s *Validator) Validate(ctx context.Context, obj runtime.Object) (result field.ErrorList) {
	aiar := obj.(*apiv1.ImageAllowRule)
	if len(aiar.Images) == 0 && len(aiar.Signatures.Rules) == 0 {
		return append(result, field.Invalid(field.NewPath(""), aiar, "at least one of scope or signatures must be specified"))
	}
	result = append(result, validateSignatureRules(ctx, aiar.Signatures)...)
	return
}

func (s *Validator) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	return s.Validate(ctx, obj)
}

func validateSignatureRules(ctx context.Context, sigRules internalv1.ImageAllowRuleSignatures) (result field.ErrorList) {
	for i, rule := range sigRules.Rules {
		if len(rule.SignedBy.AnyOf) == 0 && len(rule.SignedBy.AllOf) == 0 {
			result = append(result, field.Invalid(field.NewPath("signatures").Index(i).Child("signedBy"), rule.SignedBy, "must not be empty (at least one of anyOf or allOf must be specified)"))
		}
	}

	return
}
