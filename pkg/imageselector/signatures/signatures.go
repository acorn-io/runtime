package signatures

import (
	"context"
	"fmt"

	internalv1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	acornsign "github.com/acorn-io/runtime/pkg/cosign"
	"github.com/acorn-io/runtime/pkg/imageselector/signatures/annotations"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/rancher/wrangler/pkg/merr"
	"github.com/sigstore/cosign/v2/pkg/cosign"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func VerifySignatureRule(ctx context.Context, c client.Reader, namespace string, image string, rule internalv1.SignatureRules, opts ...remote.Option) error {
	// TODO(@iwilltry42): Move this out of here again or only leave default here and merge incoming?
	// ... alternatively, re-do the function signature to avoid unnecessary external calls in EnsureReferences
	verifyOpts := acornsign.VerifyOpts{
		Namespace:          namespace,
		AnnotationRules:    nil,
		Key:                "",
		SignatureAlgorithm: "sha256",
		RemoteOpts:         opts,
	}

	if err := acornsign.EnsureReferences(ctx, c, image, namespace, &verifyOpts); err != nil {
		return fmt.Errorf(".signatures: %w", err)
	}

	// We're using Kubernetes' label selector logic here, but we need to override the error handling
	// since the annotations we're matching on are less restricted than Kubernetes labels
	sel, err := annotations.GenerateSelector(rule.Annotations, annotations.DefaultAnnotationOpts)
	if err != nil {
		return fmt.Errorf("failed to parse annotation rule: %w", err)
	}
	verifyOpts.AnnotationRules = sel

	// allOf: all signatures must pass verification
	if len(rule.SignedBy.AllOf) != 0 {
		for allOfRuleIndex, signer := range rule.SignedBy.AllOf {
			verifyOpts.Key = signer
			err := acornsign.VerifySignature(ctx, verifyOpts)
			if err != nil {
				if _, ok := err.(*cosign.VerificationError); !ok {
					return fmt.Errorf(".signatures.allOf.%d: %w", allOfRuleIndex, err)
				}
				return err // failed or errored in allOf -> noping out
			}
		}
	}
	// anyOf: only one signature must pass verification
	var anyOfErrs []error
	if len(rule.SignedBy.AnyOf) != 0 {
		anyOfOK := false
		for anyOfRuleIndex, signer := range rule.SignedBy.AnyOf {
			verifyOpts.Key = signer
			err := acornsign.VerifySignature(ctx, verifyOpts)
			if err == nil {
				anyOfOK = true
				break
			} else {
				if _, ok := err.(*cosign.VerificationError); !ok {
					e := fmt.Errorf(".signatures.anyOf.%d: %w", anyOfRuleIndex, err)
					anyOfErrs = append(anyOfErrs, e)
				}
			}
		}
		if !anyOfOK {
			if len(anyOfErrs) == len(rule.SignedBy.AnyOf) {
				// we had errors for all anyOf rules (not failed verification, but actual errors)
				return fmt.Errorf(".signatures.anyOf.*: %w", merr.NewErrors(anyOfErrs...))
			}
			return fmt.Errorf(".signature.anyOf: failed") // failed or errored in all anyOf, try next IAR
		}
	}
	return nil
}
