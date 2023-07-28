package imageallowrules

import (
	"context"
	"fmt"
	"strings"

	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/config"
	"github.com/acorn-io/runtime/pkg/cosign"
	"github.com/acorn-io/runtime/pkg/imagepattern"
	"github.com/acorn-io/runtime/pkg/images"
	"github.com/acorn-io/runtime/pkg/profiles"
	"github.com/acorn-io/runtime/pkg/tags"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/rancher/wrangler/pkg/merr"
	ocosign "github.com/sigstore/cosign/v2/pkg/cosign"
	ociremote "github.com/sigstore/cosign/v2/pkg/oci/remote"
	"github.com/sirupsen/logrus"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ErrImageNotAllowed struct {
	Image string
}

const ErrImageNotAllowedIdentifier = "not allowed by any ImageAllowRule"

func (e *ErrImageNotAllowed) Error() string {
	return fmt.Sprintf("image <%s> is %s in this project", e.Image, ErrImageNotAllowedIdentifier)
}

func (e *ErrImageNotAllowed) Is(target error) bool {
	er, ok := target.(*ErrImageNotAllowed)
	return ok && er.Image != ""
}

// CheckImageAllowed checks if the image is allowed by the ImageAllowRules on cluster and project level
func CheckImageAllowed(ctx context.Context, c client.Reader, namespace, image, digest string, opts ...remote.Option) error {
	// Get ImageAllowRules in the same namespace as the AppInstance
	rulesList := &v1.ImageAllowRuleInstanceList{}
	if err := c.List(ctx, rulesList, &client.ListOptions{Namespace: namespace}); err != nil {
		return fmt.Errorf("failed to list ImageAllowRules: %w", err)
	}

	opts, err := images.GetAuthenticationRemoteOptions(ctx, c, namespace, opts...)
	if err != nil {
		return err
	}

	keychain, err := images.GetAuthenticationRemoteKeychainWithLocalAuth(ctx, nil, nil, c, namespace)
	if err != nil {
		return err
	}

	return CheckImageAgainstRules(ctx, c, namespace, image, digest, rulesList.Items, keychain, opts...)
}

// CheckImageAgainstRules checks if the image is allowed by the given ImageAllowRules
// If no rules are given, the image is
// - DENIED if strict mode (deny-by-default) is enabled
// - ALLOWED if strict mode is disabled (the default)
// ! Only one single rule has to allow the image for this to pass !
func CheckImageAgainstRules(ctx context.Context, c client.Reader, namespace string, image string, digest string, imageAllowRules []v1.ImageAllowRuleInstance, keychain authn.Keychain, opts ...remote.Option) error {
	cfg, err := config.Get(ctx, c)
	if err != nil {
		return err
	}

	// IAR not enabled? Allow all images.
	if cfg.Features == nil || !cfg.Features[profiles.FeatureImageAllowRules] {
		return nil
	}

	// No rules? Deny all images.
	if len(imageAllowRules) == 0 {
		return &ErrImageNotAllowed{Image: image}
	}

	logrus.Debugf("Checking image %s (%s) against %d rules", image, digest, len(imageAllowRules))

	// Check if the image is allowed
	verifyOpts := cosign.VerifyOpts{
		Namespace:          namespace,
		AnnotationRules:    v1.SignatureAnnotations{},
		Key:                "",
		SignatureAlgorithm: "sha256", // FIXME: make signature algorithm configurable (?)
		OciRemoteOpts:      []ociremote.Option{ociremote.WithRemoteOptions(opts...)},
		CraneOpts:          []crane.Option{crane.WithContext(ctx), crane.WithAuthFromKeychain(keychain)},
	}

	ref, err := name.ParseReference(image, name.WithDefaultRegistry(""), name.WithDefaultTag(""))
	if err != nil {
		return fmt.Errorf("error parsing image reference %s: %w", image, err)
	}

	if ref.Identifier() == "" && tags.SHAPattern.MatchString(image) {
		// image is a digest and was parsed as repository-only reference
		if digest == "" {
			digest = image
		}
	} else if ref.Context().String() != "" {
		digest = ref.Context().Digest(digest).Name()
	}

iarLoop:
	for _, imageAllowRule := range imageAllowRules {
		// Check if the image is in scope of the ImageAllowRule
		if !imageCovered(ref, digest, imageAllowRule) {
			continue
		}

		// > Signatures
		// Any verification error or failed verification issue will skip on to the next IAR
		for _, rule := range imageAllowRule.Signatures.Rules {
			if err := cosign.EnsureReferences(ctx, c, image, &verifyOpts); err != nil {
				return err
			}
			verifyOpts.AnnotationRules = rule.Annotations

			// allOf: all signatures must pass verification
			if len(rule.SignedBy.AllOf) != 0 {
				for allOfRuleIndex, signer := range rule.SignedBy.AllOf {
					logrus.Debugf("Checking image %s against %s/%s.signatures.allOf.%d", image, imageAllowRule.Namespace, imageAllowRule.Name, allOfRuleIndex)
					verifyOpts.Key = signer
					err := cosign.VerifySignature(ctx, verifyOpts)
					if err != nil {
						if _, ok := err.(*ocosign.VerificationError); !ok {
							logrus.Errorf("error verifying image %s against %s/%s.signatures.allOf.%d: %v", image, imageAllowRule.Namespace, imageAllowRule.Name, allOfRuleIndex, err)
						}
						continue iarLoop // failed or errored in allOf, try next IAR
					}
				}
			}
			var anyOfErrs []error
			// anyOf: only one signature must pass verification
			if len(rule.SignedBy.AnyOf) != 0 {
				anyOfOK := false
				for anyOfRuleIndex, signer := range rule.SignedBy.AnyOf {
					logrus.Debugf("Checking image %s against %s/%s.signatures.anyOf.%d", image, imageAllowRule.Namespace, imageAllowRule.Name, anyOfRuleIndex)
					verifyOpts.Key = signer
					err := cosign.VerifySignature(ctx, verifyOpts)
					if err == nil {
						anyOfOK = true
						break
					} else {
						if _, ok := err.(*ocosign.VerificationError); ok {
							logrus.Debugf("image %s not allowed as per %s/%s.signatures.anyOf.%d: %v", image, imageAllowRule.Namespace, imageAllowRule.Name, anyOfRuleIndex, err)
						} else {
							e := fmt.Errorf("error verifying image %s against %s/%s.signatures.anyOf.%d: %w", image, imageAllowRule.Namespace, imageAllowRule.Name, anyOfRuleIndex, err)
							anyOfErrs = append(anyOfErrs, e)
							logrus.Errorln(e.Error())
						}
					}
				}
				if !anyOfOK {
					if len(anyOfErrs) == len(rule.SignedBy.AnyOf) {
						// we had errors for all anyOf rules (not failed verification, but actual errors)
						e := fmt.Errorf("error verifying image %s against %s/%s.signatures.anyOf.*: %w", image, imageAllowRule.Namespace, imageAllowRule.Name, merr.NewErrors(anyOfErrs...))
						logrus.Errorln(e.Error())
					}
					continue iarLoop // failed or errored in all anyOf, try next IAR
				}
			}
		}

		return nil
	}
	return &ErrImageNotAllowed{Image: image}
}

func imageCovered(image name.Reference, digest string, iar v1.ImageAllowRuleInstance) bool {
	for _, pattern := range iar.Images {
		// empty pattern? skip (should've been caught by IAR validation already)
		if strings.TrimSpace(pattern) == "" {
			continue
		}

		// not a pattern? must be exact match then.
		if !imagepattern.IsImagePattern(pattern) {
			if strings.TrimSuffix(image.Name(), ":") != pattern && digest != pattern {
				continue
			}
			return true
		}

		parts := strings.Split(pattern, ":")
		contextPattern := parts[0]
		tagPattern := ""
		if len(parts) > 1 && !strings.Contains(parts[len(parts)-1], "/") {
			tagPattern = parts[len(parts)-1]
		}

		if err := matchContext(contextPattern, image.Context().String()); err != nil {
			logrus.Debugf("image %s not in scope of ImageAllowRule %s/%s: %v", image, iar.Namespace, iar.Name, err)
			continue
		}

		if tagPattern != "" {
			if err := matchTag(tagPattern, image.Identifier()); err != nil {
				logrus.Debugf("image %s not in scope of ImageAllowRule %s/%s: %v", image, iar.Namespace, iar.Name, err)
				continue
			}
		}

		return true
	}
	return false
}

// matchContext matches the image context against the context pattern, similar to globbing
func matchContext(contextPattern string, imageContext string) error {
	re, _, err := imagepattern.NewMatcher(contextPattern)
	if err != nil {
		return fmt.Errorf("error parsing context pattern %s: %w", contextPattern, err)
	}

	if re.MatchString(imageContext) {
		return nil
	}

	return fmt.Errorf("image context %s does not match pattern %s (regex: `%s`)", imageContext, contextPattern, re.String())
}

// matchTag matches the image tag against the tag pattern, similar to auto-upgrade pattern
func matchTag(tagPattern string, imageTag string) error {
	re, _, err := imagepattern.NewMatcher(tagPattern)
	if err != nil {
		return fmt.Errorf("error parsing tag pattern %s: %w", tagPattern, err)
	}

	if re.MatchString(imageTag) {
		return nil
	}

	return fmt.Errorf("image tag %s does not match pattern %s (regex: `%s`)", imageTag, tagPattern, re.String())
}
