package imageallowrules

import (
	"context"
	"fmt"

	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/cosign"
	"github.com/acorn-io/acorn/pkg/images"
	"github.com/google/go-containerregistry/pkg/v1/remote"
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
			if rule.SignedBy.AllOf != nil {
				for allOfRuleIndex, signer := range rule.SignedBy.AllOf {
					verifyOpts.Key = signer
					err := cosign.VerifySignature(ctx, c, verifyOpts)
					if err != nil {
						if _, ok := err.(*ocosign.VerificationError); ok {
							notAllowedErr.SubrulePath += fmt.Sprintf(".allOf.%d (%v)", allOfRuleIndex, err)
							logrus.Warnf(notAllowedErr.Error())
							return notAllowedErr
						}
						return fmt.Errorf("failed to verify signature: %w", err)
					}
				}
			}

			// anyOf: only one signature must pass verification
			if rule.SignedBy.AnyOf != nil {
				anyOfOK := false
				for anyOfRuleIndex, signer := range rule.SignedBy.AnyOf {
					logrus.Debugf("Checking image %s against anyOf rule #%d in %s/%s", image, anyOfRuleIndex, ImageAllowRules.Namespace, ImageAllowRules.Name)
					verifyOpts.Key = signer
					err := cosign.VerifySignature(ctx, c, verifyOpts)
					if err == nil {
						anyOfOK = true
						break
					} else {
						if _, ok := err.(*ocosign.VerificationError); ok {
							logrus.Debugf("image %s not allowed as per anyOf rule %s/%s #%d: %v", image, ImageAllowRules.Namespace, ImageAllowRules.Name, anyOfRuleIndex, err)
						} else {
							logrus.Errorf("failed to verify %s/%s.anyOf.%d: %v", ImageAllowRules.Namespace, ImageAllowRules.Name, anyOfRuleIndex, err)
						}
					}
				}
				if !anyOfOK {
					logrus.Warnf("image %s is not allowed as per anyOf rule in %s/%s", image, ImageAllowRules.Namespace, ImageAllowRules.Name)
					notAllowedErr.SubrulePath += ".anyOf"
					return notAllowedErr
				}
			}
		}
	}

	logrus.Debugf("image %s is allowed", image)

	return nil
}
