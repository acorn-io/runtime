package appdefinition

import (
	"context"
	"fmt"
	"net/http"

	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/autoupgrade"
	"github.com/acorn-io/acorn/pkg/condition"
	"github.com/acorn-io/acorn/pkg/images"
	"github.com/acorn-io/acorn/pkg/tags"
	"github.com/acorn-io/baaah/pkg/router"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func PullAppImage(transport http.RoundTripper) router.HandlerFunc {
	i := &imagePuller{
		transport: transport,
	}
	return pullAppImage(i)
}

func pullAppImage(i imageClient) router.HandlerFunc {
	return func(req router.Request, resp router.Response) error {
		appInstance := req.Object.(*v1.AppInstance)
		cond := condition.Setter(appInstance, resp, v1.AppInstanceConditionPulled)

		targetImage, unknownReason := determineTargetImage(appInstance)
		if targetImage == "" {
			if unknownReason != "" {
				cond.Unknown(unknownReason)
			} else {
				cond.Success()
			}
			return nil
		}

		resolvedImage, _, err := tags.ResolveLocal(req.Ctx, req.Client, appInstance.Namespace, targetImage)
		if err != nil {
			cond.Error(err)
			return nil
		}

		remoteDigest, err := i.remoteImageDigest(req.Ctx, req.Client, appInstance.Namespace, resolvedImage)
		if err != nil {
			cond.Error(err)
			return nil
		}

		// No pull needed if the remote digest matches the current digest
		if remoteDigest != appInstance.Status.AppImage.Digest {
			appImage, err := i.imagePullAppImage(req.Ctx, req.Client, appInstance.Namespace, resolvedImage, "")
			if err != nil {
				cond.Error(err)
				return nil
			}
			appImage.Name = targetImage
			appInstance.Status.AppImage = *appImage
		}
		appInstance.Status.AvailableAppImage = ""
		appInstance.Status.ConfirmUpgradeAppImage = ""

		cond.Success()
		return nil
	}
}

func determineTargetImage(appInstance *v1.AppInstance) (string, string) {
	_, on := autoupgrade.Mode(appInstance.Spec)
	pattern, isPattern := autoupgrade.AutoUpgradePattern(appInstance.Spec.Image)

	if on {
		if appInstance.Status.AvailableAppImage != "" || appInstance.Status.ConfirmUpgradeAppImage != "" {
			if appInstance.Status.AvailableAppImage != "" {
				// AvailableAppImage is not blank, use it and reset the other fields
				return appInstance.Status.AvailableAppImage, ""
			} else {
				// ConfirmUpgradeAppImage is not blank. Normally, we shouldn't get the desiredImage from it. That should
				// be done explicitly by the user via the apps/confirmupgrade subresource (which would set it to the
				// AvailableAppImage field). But if AppImage.ID is blank, this app has never had an image pulled. So, do the initial pull.
				if appInstance.Status.AppImage.Name == "" {
					return appInstance.Status.ConfirmUpgradeAppImage, ""
				} else {
					return "", fmt.Sprintf("confirm upgrade to %v", appInstance.Status.ConfirmUpgradeAppImage)
				}
			}
		} else {
			// Neither AvailableAppImage nor ConfirmUpgradeAppImage is set.
			if isPattern {
				if appInstance.Status.AppImage.Name == "" {
					// Need to trigger a sync since this app has never had a concrete image set
					autoupgrade.Sync()
					return "", fmt.Sprintf("waiting for image to satisfy auto-upgrade tag %v", pattern)
				} else {
					return "", ""
				}
			} else {
				if appInstance.Spec.Image == appInstance.Status.AppImage.Name {
					return "", ""
				} else {
					return appInstance.Spec.Image, ""
				}
			}
		}
	} else {
		// Auto-upgrade is off. Only need to pull if spec and status are not equal or we're trying to trigger a repull
		if appInstance.Spec.Image != appInstance.Status.AppImage.Name ||
			appInstance.Status.AvailableAppImage == appInstance.Spec.Image {
			return appInstance.Spec.Image, ""
		} else {
			return "", ""
		}
	}
}

func (i *imagePuller) remoteImageDigest(ctx context.Context, c client.Reader, namespace, image string) (string, error) {
	return images.ImageDigest(ctx, c, namespace, image, remote.WithTransport(i.transport))
}

func (i *imagePuller) imagePullAppImage(ctx context.Context, c client.Reader, namespace, image, nestedDigest string) (*v1.AppImage, error) {
	return images.PullAppImage(ctx, c, namespace, image, nestedDigest, remote.WithTransport(i.transport))
}

type imageClient interface {
	remoteImageDigest(ctx context.Context, c client.Reader, namespace, image string) (string, error)
	imagePullAppImage(ctx context.Context, c client.Reader, namespace, image, nestedDigest string) (*v1.AppImage, error)
}

type imagePuller struct {
	transport http.RoundTripper
}
