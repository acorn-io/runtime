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
	return pullAppImage(transport, pullClient{
		recorder: recorder,
		resolve:  tags.ResolveLocal,
		pull:     images.PullAppImage,
		now:      metav1.Now,
	})
}

type resolveImageFunc func(ctx context.Context, c kclient.Client, namespace, image string) (resolved string, isLocal bool, error error)

type pullImageFunc func(ctx context.Context, c kclient.Reader, namespace, image, nestedDigest string, opts ...remote.Option) (*v1.AppImage, error)

type pullClient struct {
	recorder event.Recorder
	resolve  resolveImageFunc
	pull     pullImageFunc
	now      func() metav1.Time
}

func pullAppImage(transport http.RoundTripper, client pullClient) router.HandlerFunc {
	return func(req router.Request, resp router.Response) error {
		appInstance := req.Object.(*v1.AppInstance)
		cond := condition.Setter(appInstance, resp, v1.AppInstanceConditionPulled)

		target, digest, unknownReason := determineTargetImage(appInstance)
		if target == "" {
			if unknownReason != "" {
				cond.Unknown(unknownReason)
			} else {
				cond.Success()
			}
			return nil
		}

		var (
			resolved string
			err      error
		)
		// If we already have the digest, we don't need to resolve the image
		if digest != "" {
			resolved = fmt.Sprintf("%s@%s", target, digest)
		} else {
			resolved, _, err = client.resolve(req.Ctx, req.Client, appInstance.Namespace, target)
			if err != nil {
				cond.Error(err)
				return nil
			}
		}

		var (
			_, autoUpgradeOn = autoupgrade.Mode(appInstance.Spec)
			previousImage    = appInstance.Status.AppImage
			targetImage      *v1.AppImage
		)
		defer func() {
			// Record the results as an event
			if err != nil {
				targetImage = &v1.AppImage{
					Name: resolved,
				}
			}
			recordPullEvent(req.Ctx, client.recorder, client.now(), req.Object, autoUpgradeOn, err, previousImage, *targetImage)
		}()

		targetImage, err = client.pull(req.Ctx, req.Client, appInstance.Namespace, resolved, "", remote.WithTransport(transport))
		if err != nil {
			cond.Error(err)
			return nil
		}
		targetImage.Name = target
		appInstance.Status.AvailableAppImage = ""
		appInstance.Status.AvailableAppImageDigest = ""
		appInstance.Status.ConfirmUpgradeAppImage = ""
		appInstance.Status.ConfirmUpgradeAppImageDigest = ""
		appInstance.Status.AppImage = *targetImage

		cond.Success()
		return nil
	}
}

func determineTargetImage(appInstance *v1.AppInstance) (string, string, string) {
	_, on := autoupgrade.Mode(appInstance.Spec)
	pattern, isPattern := autoupgrade.AutoUpgradePattern(appInstance.Spec.Image)

	if on {
		if appInstance.Status.AvailableAppImage != "" || appInstance.Status.ConfirmUpgradeAppImage != "" {
			if appInstance.Status.AvailableAppImage != "" {
				// AvailableAppImage is not blank, use it and reset the other fields
				return appInstance.Status.AvailableAppImage, appInstance.Status.AvailableAppImageDigest, ""
			} else {
				// ConfirmUpgradeAppImage is not blank. Normally, we shouldn't get the desiredImage from it. That should
				// be done explicitly by the user via the apps/confirmupgrade subresource (which would set it to the
				// AvailableAppImage field). But if AppImage.ID is blank, this app has never had an image pulled. So, do the initial pull.
				if appInstance.Status.AppImage.Name == "" {
					return appInstance.Status.ConfirmUpgradeAppImage, appInstance.Status.ConfirmUpgradeAppImageDigest, ""
				} else {
					if appInstance.Status.ConfirmUpgradeAppImageDigest != "" {
						return "", "", fmt.Sprintf("confirm upgrade to %v@%v", appInstance.Status.ConfirmUpgradeAppImage, appInstance.Status.ConfirmUpgradeAppImageDigest)
					} else {
						return "", "", fmt.Sprintf("confirm upgrade to %v", appInstance.Status.ConfirmUpgradeAppImage)
					}
				}
			}
		} else {
			// Neither AvailableAppImage nor ConfirmUpgradeAppImage is set.
			if isPattern {
				if appInstance.Status.AppImage.Name == "" {
					// Need to trigger a sync since this app has never had a concrete image set
					autoupgrade.Sync()
					return "", "", fmt.Sprintf("waiting for image to satisfy auto-upgrade tag %v", pattern)
				} else {
					return "", "", ""
				}
			} else {
				if appInstance.Spec.Image == appInstance.Status.AppImage.Name {
					return "", "", ""
				} else {
					return appInstance.Spec.Image, "", ""
				}
			}
		}
	} else {
		// Auto-upgrade is off. Only need to pull if spec and status are not equal or we're trying to trigger a repull
		if appInstance.Spec.Image != appInstance.Status.AppImage.Name ||
			appInstance.Status.AvailableAppImage == appInstance.Spec.Image {
			return appInstance.Spec.Image, "", ""
		} else {
			return "", "", ""
		}
	}
}

const (
	AppImagePullFailureEventType = "AppImagePullFailure"
	AppImagePullSuccessEventType = "AppImagePullSuccess"
)

// AppImagePullEventDetails captures additional info about an App image pull.
type AppImagePullEventDetails struct {
	// ResourceVersion is the resourceVersion of the App the image is being pulled for.
	ResourceVersion string `json:"resourceVersion"`

	// AutoUpgrade is true if the pull was triggered by an auto-upgrade, false otherwise.
	AutoUpgrade bool `json:"autoUpgrade"`

	// Previous is the App image before pulling, if any.
	// +optional
	Previous ImageSummary `json:"previous,omitempty"`

	// Target is the image being pulled.
	Target ImageSummary `json:"target"`

	// Err is an error that occurred during the pull, if any.
	// +optional
	Err string `json:"err,omitempty"`
}

type ImageSummary struct {
	Name   string `json:"name,omitempty"`
	Digest string `json:"digest,omitempty"`
	VCS    v1.VCS `json:"vcs,omitempty"`
}

func newImageSummary(appImage v1.AppImage) ImageSummary {
	return ImageSummary{
		Name:   appImage.Name,
		Digest: appImage.Digest,
		VCS:    appImage.VCS,
	}
}

func recordPullEvent(ctx context.Context, recorder event.Recorder, observed metav1.Time, obj kclient.Object, autoUpgradeOn bool, err error, previousImage, targetImage v1.AppImage) {
	// Initialize with values for a success event
	previous, target := newImageSummary(previousImage), newImageSummary(targetImage)
	e := apiv1.Event{
		Type:        AppImagePullSuccessEventType,
		Actor:       "acorn-system",
		Severity:    v1.EventSeverityInfo,
		Description: fmt.Sprintf("Pulled %s", target.Name),
		Source:      event.ObjectSource(obj),
		Observed:    observed,
	}
	e.SetNamespace(obj.GetNamespace())

	details := AppImagePullEventDetails{
		ResourceVersion: obj.GetResourceVersion(),
		AutoUpgrade:     autoUpgradeOn,
		Previous:        previous,
		Target:          target,
	}

	if err != nil {
		// It's a failure, overwrite with failure event values
		e.Type = AppImagePullFailureEventType
		e.Severity = v1.EventSeverityWarn
		e.Description = fmt.Sprintf("Failed to pull %s", target.Name)
		details.Err = err.Error()
	}

	if e.Details, err = v1.Mapify(details); err != nil {
		logrus.Warnf("Failed to mapify event details: %s", err.Error())
	}

	if err := recorder.Record(ctx, &e); err != nil {
		logrus.Warnf("Failed to record event: %s", err.Error())
	}
}
