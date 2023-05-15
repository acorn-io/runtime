package appdefinition

import (
	"context"
	"fmt"
	"net/http"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/autoupgrade"
	"github.com/acorn-io/acorn/pkg/condition"
	"github.com/acorn-io/acorn/pkg/event"
	"github.com/acorn-io/acorn/pkg/images"
	"github.com/acorn-io/acorn/pkg/tags"
	"github.com/acorn-io/baaah/pkg/router"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func PullAppImage(transport http.RoundTripper, recorder event.Recorder) router.HandlerFunc {
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

		var (
			err           error
			resolvedImage string
		)
		defer func() {
			// Record the results as an event
			recordResolutionEvent(req.Ctx, recorder, req.Object, err, targetImage, resolvedImage)
		}()

		resolvedImage, _, err = tags.ResolveLocal(req.Ctx, req.Client, appInstance.Namespace, targetImage)
		if err != nil {
			cond.Error(err)
			return nil
		}

		var appImage *v1.AppImage
		appImage, err = images.PullAppImage(req.Ctx, req.Client, appInstance.Namespace, resolvedImage, "", remote.WithTransport(transport))
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

const (
	AppImageResolutionFailureEventType = "AppImageResolutionFailure"
	AppImageResolutionSuccessEventType = "AppImageResolutionSuccess"
)

// AppImageResolutionEventDetails captures additional info about App image resolution.
type AppImageResolutionEventDetails struct {
	// AppResourceVersion is the resourceVersion of the App the image is being resolved for.
	AppResourceVersion string `json:"appResourceVersion"`

	// TargetImage is the image being resolved.
	TargetImage string `json:"targetImage,omitempty"`

	// ResolvedImage is the image, post resolution.
	// +optional
	ResolvedImage string `json:"resolvedImage,omitempty"`

	// Err the error that occurred during resolution, if any.
	// +optional
	Err string `json:"err,omitempty"`
}

func recordResolutionEvent(ctx context.Context, recorder event.Recorder, obj kclient.Object, err error, targetImage, resolvedImage string) {
	// Initialize with values for a success event
	e := apiv1.Event{
		Type:        AppImageResolutionSuccessEventType,
		Severity:    v1.EventSeverityInfo,
		Description: fmt.Sprintf("Pulled %s (resolved from %s)", resolvedImage, targetImage),
		Source:      event.ObjectSource(obj),
		Observed:    metav1.Now(),
	}
	details := AppImageResolutionEventDetails{
		AppResourceVersion: obj.GetResourceVersion(),
		TargetImage:        targetImage,
		ResolvedImage:      resolvedImage,
	}

	if err != nil {
		// It's a failure, overwrite with failure event values
		e.Type = AppImageResolutionFailureEventType
		e.Severity = v1.EventSeverityWarn
		if resolvedImage == "" {
			// Failed to resolve the target image
			e.Description = fmt.Sprintf("Failed to resolve %s", targetImage)
		} else {
			// The target image was resolved, but we failed to pull the result
			e.Description = fmt.Sprintf("Failed to pull %s (resolved from %s)", resolvedImage, targetImage)
		}
	}

	if e.Details, err = v1.Mapify(details); err != nil {
		logrus.Warnf("Failed to mapify event details: %s", err.Error())
	}

	if err := recorder.Record(ctx, &e); err != nil {
		logrus.Warnf("Failed to record event: %s", err.Error())
	}
}
