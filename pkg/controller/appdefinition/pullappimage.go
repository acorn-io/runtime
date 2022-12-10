package appdefinition

import (
	"fmt"
	"net/http"

	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/autoupgrade"
	"github.com/acorn-io/acorn/pkg/condition"
	"github.com/acorn-io/acorn/pkg/images"
	"github.com/acorn-io/acorn/pkg/tags"
	"github.com/acorn-io/baaah/pkg/router"
	"github.com/google/go-containerregistry/pkg/v1/remote"
)

func PullAppImage(transport http.RoundTripper) router.HandlerFunc {
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

		appImage, err := images.PullAppImage(req.Ctx, req.Client, appInstance.Namespace, resolvedImage, remote.WithTransport(transport))
		if err != nil {
			cond.Error(err)
			return nil
		}
		appImage.Name = targetImage
		appInstance.Status.AvailableAppImage = ""
		appInstance.Status.ConfirmUpgradeAppImage = ""
		appInstance.Status.AppImage = *appImage

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
