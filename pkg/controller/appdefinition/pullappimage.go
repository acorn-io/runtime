package appdefinition

import (
	"context"
	"fmt"
	"net/http"

	"github.com/acorn-io/baaah/pkg/router"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/autoupgrade"
	"github.com/acorn-io/runtime/pkg/condition"
	"github.com/acorn-io/runtime/pkg/event"
	"github.com/acorn-io/runtime/pkg/images"
	"github.com/acorn-io/runtime/pkg/tags"
	imagename "github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func PullAppImage(transport http.RoundTripper, recorder event.Recorder) router.HandlerFunc {
	return pullAppImage(transport, pullClient{
		pull: images.PullAppImage,
	})
}

type pullImageFunc func(ctx context.Context, c kclient.Client, namespace, image, nestedDigest string, opts ...remote.Option) (*v1.AppImage, error)

type pullClient struct {
	pull pullImageFunc
}

func pullAppImage(transport http.RoundTripper, client pullClient) router.HandlerFunc {
	// NOTE: It is important that this logic does not interact with status.AppImage but instead
	// status.Staged.AppImage
	return func(req router.Request, resp router.Response) error {
		appInstance := req.Object.(*v1.AppInstance)
		cond := condition.Setter(appInstance, resp, v1.AppInstanceConditionPulled)

		// For migration/upgrade purposes, if status.StagedAppImage is nil, just assign it status.AppImage
		if appInstance.Status.Staged.AppImage.ID == "" {
			appInstance.Status.Staged.AppImage = appInstance.Status.AppImage
		}

		target, unknownReason := determineTargetImage(appInstance)
		if target == "" {
			if unknownReason != "" {
				cond.Unknown(unknownReason)
			} else {
				cond.Success()
			}
			return nil
		}

		var (
			_, autoUpgradeOn = autoupgrade.Mode(appInstance.Spec)
			resolved         string
		)
		// Only attempt to resolve locally if auto-upgrade is not on, or if auto-upgrade is on but we know the image is not remote.
		if !autoUpgradeOn || !images.IsImageRemote(req.Ctx, req.Client, appInstance.Namespace, target, true, remote.WithTransport(transport)) {
			var (
				isLocal bool
				err     error
			)
			resolved, isLocal, err = tags.ResolveLocal(req.Ctx, req.Client, appInstance.Namespace, target)
			if err != nil {
				cond.Error(err)
				return nil
			}

			if !isLocal {
				if autoUpgradeOn && !tags.IsLocalReference(target) {
					ref, err := imagename.ParseReference(target, imagename.WithDefaultRegistry(images.NoDefaultRegistry))
					if err != nil {
						return err
					}
					if ref.Context().RegistryStr() == images.NoDefaultRegistry {
						// Prevent this from being resolved remotely, as we should never assume Docker Hub for auto-upgrade apps
						return fmt.Errorf("no local image found for %v - if you are trying to use a remote image, specify the full registry", target)
					}
				}

				// Force pull from remote, since the only local image we found was marked remote, and there might be a newer version
				resolved = target
			}
		} else {
			resolved = target
		}

		targetImage, err := client.pull(req.Ctx, req.Client, appInstance.Namespace, resolved, "", remote.WithTransport(transport))
		if err != nil {
			cond.Error(err)
			return nil
		}
		targetImage.Name = target
		appInstance.Status.AvailableAppImage = ""
		appInstance.Status.ConfirmUpgradeAppImage = ""
		// Reset the whole object, reset all staged state
		appInstance.Status.Staged = v1.AppStatusStaged{
			AppImage: *targetImage,
		}

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
				if appInstance.Status.Staged.AppImage.Name == "" {
					return appInstance.Status.ConfirmUpgradeAppImage, ""
				} else {
					return "", fmt.Sprintf("confirm upgrade to %v", appInstance.Status.ConfirmUpgradeAppImage)
				}
			}
		} else {
			// Neither AvailableAppImage nor ConfirmUpgradeAppImage is set.
			if isPattern {
				if appInstance.Status.Staged.AppImage.Name == "" {
					// Need to trigger a sync since this app has never had a concrete image set
					autoupgrade.Sync()
					return "", fmt.Sprintf("waiting for image to satisfy auto-upgrade tag %v", pattern)
				} else {
					return "", ""
				}
			} else {
				if appInstance.Spec.Image == appInstance.Status.Staged.AppImage.Name {
					return "", ""
				} else {
					return appInstance.Spec.Image, ""
				}
			}
		}
	} else {
		// Auto-upgrade is off. Only need to pull if spec and status are not equal or we're trying to trigger a repull
		if appInstance.Spec.Image != appInstance.Status.Staged.AppImage.Name ||
			appInstance.Status.AvailableAppImage == appInstance.Spec.Image {
			return appInstance.Spec.Image, ""
		} else {
			return "", ""
		}
	}
}
