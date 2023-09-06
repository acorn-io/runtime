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
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/rancher/wrangler/pkg/merr"
	ocosign "github.com/sigstore/cosign/v2/pkg/cosign"
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
	cfg, err := config.Get(ctx, c)
	if err != nil {
		return err
	}

	// IAR not enabled? Allow all images.
	if cfg.Features == nil || !cfg.Features[profiles.FeatureImageAllowRules] {
		return nil
	}

	// Get ImageAllowRules in the same namespace as the AppInstance
	rulesList := &v1.ImageAllowRuleInstanceList{}
	if err := c.List(ctx, rulesList, &client.ListOptions{Namespace: namespace}); err != nil {
		return fmt.Errorf("failed to list ImageAllowRules: %w", err)
	}

	opts, err = images.GetAuthenticationRemoteOptions(ctx, c, namespace, opts...)
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
		RemoteOpts:         opts,
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
		if !ImageCovered(ref, digest, imageAllowRule.Images) {
			logrus.Infof("Image %s (%s) is not covered by ImageAllowRule %s/%s: %#v", image, digest, imageAllowRule.Namespace, imageAllowRule.Name, imageAllowRule.Images)
			continue
		}

		// > Signatures
		// Any verification error or failed verification issue will skip on to the next IAR
		for _, rule := range imageAllowRule.Signatures.Rules {
			if err := VerifySignatureRule(ctx, c, namespace, image, rule, verifyOpts); err != nil {
				logrus.Errorf("Verification failed for %s against %s/%s: %v", image, imageAllowRule.Namespace, imageAllowRule.Name, err)
				continue iarLoop
			}
		}
		logrus.Debugf("Image %s (%s) is allowed by ImageAllowRule %s/%s", image, digest, imageAllowRule.Namespace, imageAllowRule.Name)
		return nil
	}
	return &ErrImageNotAllowed{Image: image}
}

func VerifySignatureRule(ctx context.Context, c client.Reader, namespace string, image string, rule v1.SignatureRules, verifyOpts cosign.VerifyOpts) error {
	if err := cosign.EnsureReferences(ctx, c, image, namespace, &verifyOpts); err != nil {
		return fmt.Errorf(".signatures: %w", err)
	}
	verifyOpts.AnnotationRules = rule.Annotations

	// allOf: all signatures must pass verification
	if len(rule.SignedBy.AllOf) != 0 {
		for allOfRuleIndex, signer := range rule.SignedBy.AllOf {
			verifyOpts.Key = signer
			err := cosign.VerifySignature(ctx, verifyOpts)
			if err != nil {
				if _, ok := err.(*ocosign.VerificationError); !ok {
					return fmt.Errorf(".signatures.allOf.%d: %w", allOfRuleIndex, err)
				}
				return err // failed or errored in allOf, try next IAR
			}
		}
	}
	// anyOf: only one signature must pass verification
	var anyOfErrs []error
	if len(rule.SignedBy.AnyOf) != 0 {
		anyOfOK := false
		for anyOfRuleIndex, signer := range rule.SignedBy.AnyOf {
			verifyOpts.Key = signer
			err := cosign.VerifySignature(ctx, verifyOpts)
			if err == nil {
				anyOfOK = true
				break
			} else {
				if _, ok := err.(*ocosign.VerificationError); !ok {
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

func ImageCovered(image name.Reference, digest string, patterns []string) bool {
	for _, pattern := range patterns {
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
		if len(parts) > 1 {
			if !strings.Contains(parts[len(parts)-1], "/") {
				tagPattern = parts[len(parts)-1] // last part is tag
			} else {
				contextPattern = pattern // : was part of the context pattern (port)
			}
		}

		if err := matchContext(contextPattern, image.Context().String()); err != nil {
			continue
		}

		if tagPattern != "" {
			if err := matchTag(tagPattern, image.Identifier()); err != nil {
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
