package imagerules

import (
	"context"
	"errors"
	"fmt"

	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/config"
	"github.com/acorn-io/runtime/pkg/images"
	"github.com/acorn-io/runtime/pkg/imageselector"
	"github.com/acorn-io/runtime/pkg/profiles"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/sirupsen/logrus"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ErrImageNotAllowed struct {
	Image string
}

const ErrImageNotAllowedIdentifier = "not allowed by any ImageAllowRule"

func (e *ErrImageNotAllowed) Error() string {
	return fmt.Sprintf("image [%s] is %s in this project", e.Image, ErrImageNotAllowedIdentifier)
}

func (e *ErrImageNotAllowed) Is(target error) bool {
	er, ok := target.(*ErrImageNotAllowed)
	return ok && er.Image != ""
}

// CheckImageAllowed checks if the image is allowed by the ImageAllowRules on cluster and project level
func CheckImageAllowed(ctx context.Context, c client.Reader, namespace, imageName, resolvedName, digest string, opts ...remote.Option) error {
	// IAR not enabled? Allow all images.
	if enabled, err := config.GetFeature(ctx, c, profiles.FeatureImageAllowRules); err != nil {
		return err
	} else if !enabled {
		return nil
	}

	// Get ImageAllowRules in the same namespace as the AppInstance
	rulesList := &v1.ImageAllowRuleInstanceList{}
	if err := c.List(ctx, rulesList, &client.ListOptions{Namespace: namespace}); err != nil {
		return fmt.Errorf("failed to list ImageAllowRules: %w", err)
	}

	opts, err := images.GetAuthenticationRemoteOptions(ctx, c, namespace, opts...)
	if err != nil {
		return err
	}

	return CheckImageAgainstRules(ctx, c, namespace, imageName, resolvedName, digest, rulesList.Items, opts...)
}

// CheckImageAgainstRules checks if the image is allowed by the given ImageAllowRules
// If no rules are given, the image is denied.
// ! Only one single rule has to allow the image for this to pass !
//
// About image references:
// @param imageName: the image how it was called (e.g. how it was specified by the user in `acorn run`)
// @param resolvedName: the image name after resolution (e.g. resolved to an internal image ID)
// @param digest: the digest of the image
// We will use all of those to check if an image is covered by an IAR.
// We will prefer resolvedName to find signature artifacts (potentially in the internal registry)
func CheckImageAgainstRules(ctx context.Context, c client.Reader, namespace, imageName, resolvedName, digest string, imageAllowRules []v1.ImageAllowRuleInstance, opts ...remote.Option) error {
	// No rules? Deny all images.
	if len(imageAllowRules) == 0 {
		return &ErrImageNotAllowed{Image: imageName}
	}

	logrus.Debugf("Checking image %s (%s) against %d rules", imageName, digest, len(imageAllowRules))

	for _, imageAllowRule := range imageAllowRules {
		if err := imageselector.MatchImage(ctx, c, namespace, imageName, resolvedName, digest, imageAllowRule.ImageSelector, opts...); err != nil {
			if ierr := (*imageselector.ImageSelectorNoMatchError)(nil); errors.As(err, &ierr) {
				logrus.Debugf("ImageAllowRule %s/%s did not match: %v", imageAllowRule.Namespace, imageAllowRule.Name, err)
			} else {
				logrus.Errorf("Error matching ImageAllowRule %s/%s: %v", imageAllowRule.Namespace, imageAllowRule.Name, err)
			}
			continue
		}
		logrus.Debugf("Image %s (%s) is allowed by ImageAllowRule %s/%s", imageName, digest, imageAllowRule.Namespace, imageAllowRule.Name)
		return nil
	}
	return &ErrImageNotAllowed{Image: imageName}
}
