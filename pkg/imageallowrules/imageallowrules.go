package imageallowrules

import (
	"context"
	"fmt"

	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/cosign"
	"github.com/acorn-io/acorn/pkg/images"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/rancher/wrangler/pkg/merr"
	ocosign "github.com/sigstore/cosign/pkg/cosign"
	ociremote "github.com/sigstore/cosign/pkg/oci/remote"
	"github.com/sirupsen/logrus"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ErrImageNotAllowed struct {
	Rule        string
	SubruleType string
	SubrulePath string
	Image       string
}

func (e *ErrImageNotAllowed) Error() string {
	return fmt.Sprintf("image %s is not allowed by rule %s.%s.%s", e.Image, e.Rule, e.SubruleType, e.SubrulePath)
}

func (e *ErrImageNotAllowed) Is(target error) bool {
	_, ok := target.(*ErrImageNotAllowed)
	return ok
}

// CheckImageAllowed checks if the image is allowed by the ImageAllowRules on cluster and project level
func CheckImageAllowed(ctx context.Context, c client.Reader, namespace, image string, opts ...remote.Option) error {

	// Get ImageAllowRules in the same namespace as the AppInstance
	ImageAllowRulesList := &v1.ImageAllowRulesInstanceList{}
	if err := c.List(ctx, ImageAllowRulesList, &client.ListOptions{Namespace: namespace}); err != nil {
		return fmt.Errorf("failed to list ImageAllowRules: %w", err)
	}

	opts, err := images.GetAuthenticationRemoteOptions(ctx, c, namespace, opts...)
	if err != nil {
		return err
	}

	return CheckImageAgainstRules(ctx, c, namespace, image, ImageAllowRulesList.Items, opts...)
}

func CheckImageAgainstRules(ctx context.Context, c client.Reader, namespace string, image string, imageAllowRules []v1.ImageAllowRulesInstance, opts ...remote.Option) error {

	if len(imageAllowRules) == 0 {
		// No ImageAllowRules found, so allow the image
		return nil
	}

	logrus.Debugf("Checking image %s against %d rules", image, len(imageAllowRules))

	// Check if the image is allowed
	verifyOpts := cosign.VerifyOpts{
		ImageRef:           image,
		Namespace:          namespace,
		AnnotationRules:    v1.SignatureAnnotations{},
		Key:                "",
		SignatureAlgorithm: "sha256", // FIXME: make signature algorithm configurable (?)
		RegistryClientOpts: []ociremote.Option{ociremote.WithRemoteOptions(opts...)},
	}
	for _, ImageAllowRules := range imageAllowRules {

		notAllowedErr := &ErrImageNotAllowed{Rule: fmt.Sprintf("%s/%s", ImageAllowRules.Namespace, ImageAllowRules.Name), Image: image}

		// > Signatures
		notAllowedErr.SubruleType = "signatures"
		for ruleIndex, rule := range ImageAllowRules.Signatures.Rules {

			verifyOpts.AnnotationRules = rule.Annotations
			notAllowedErr.SubrulePath = fmt.Sprintf("%d", ruleIndex)

			// allOf: all signatures must pass verification
			if len(rule.SignedBy.AllOf) != 0 {
				for allOfRuleIndex, signer := range rule.SignedBy.AllOf {
					logrus.Debugf("Checking image %s against %s/%s.signatures.allOf.%d", image, ImageAllowRules.Namespace, ImageAllowRules.Name, allOfRuleIndex)
					verifyOpts.Key = signer
					err := cosign.VerifySignature(ctx, c, verifyOpts)
					if err != nil {
						if _, ok := err.(*ocosign.VerificationError); ok {
							notAllowedErr.SubrulePath += fmt.Sprintf(".allOf.%d (%v)", allOfRuleIndex, err)
							logrus.Warnf(notAllowedErr.Error())
							return notAllowedErr
						}
						return fmt.Errorf("error verifying image %s against %s/%s.signatures.allOf.%d: %w", image, ImageAllowRules.Namespace, ImageAllowRules.Name, allOfRuleIndex, err)
					}
				}
			}
			var anyOfErrs []error
			// anyOf: only one signature must pass verification
			if len(rule.SignedBy.AnyOf) != 0 {
				anyOfOK := false
				for anyOfRuleIndex, signer := range rule.SignedBy.AnyOf {
					logrus.Debugf("Checking image %s against %s/%s.signatures.anyOf.%d", image, ImageAllowRules.Namespace, ImageAllowRules.Name, anyOfRuleIndex)
					verifyOpts.Key = signer
					err := cosign.VerifySignature(ctx, c, verifyOpts)
					if err == nil {
						anyOfOK = true
						break
					} else {
						if _, ok := err.(*ocosign.VerificationError); ok {
							logrus.Debugf("image %s not allowed as per %s/%s.signatures.anyOf.%d: %v", image, ImageAllowRules.Namespace, ImageAllowRules.Name, anyOfRuleIndex, err)
						} else {
							e := fmt.Errorf("error verifying image %s against %s/%s.signatures.anyOf.%d: %w", image, ImageAllowRules.Namespace, ImageAllowRules.Name, anyOfRuleIndex, err)
							anyOfErrs = append(anyOfErrs, e)
							logrus.Errorln(e.Error())
						}
					}
				}
				if !anyOfOK {
					notAllowedErr.SubrulePath += ".anyOf"
					if len(anyOfErrs) == len(rule.SignedBy.AnyOf) {
						// we had errors for all anyOf rules (not failed verification, but actual errors)
						e := fmt.Errorf("error verifying image %s against %s/%s.signatures.anyOf.*: %w", image, ImageAllowRules.Namespace, ImageAllowRules.Name, merr.NewErrors(anyOfErrs...))
						logrus.Errorln(e.Error())
						return e
					}
					logrus.Warnf("image %s is not allowed as per %s/%s.signatures.anyOf", image, ImageAllowRules.Namespace, ImageAllowRules.Name)
					return notAllowedErr
				}
			}
		}
	}

	logrus.Debugf("image %s is allowed", image)

	return nil
}
