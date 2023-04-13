package appdefinition

import (
	"errors"
	"fmt"
	"net/http"

	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/condition"
	"github.com/acorn-io/acorn/pkg/imageallowrules"
	"github.com/acorn-io/baaah/pkg/router"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/sirupsen/logrus"
)

// CheckImageAllowedHandler is a router handler that checks if the image is allowed by the image allow rules and sets a status field accordingly
// This is only working on the currently specified image, referenced by digest, to avoid false positives (alerts) if the remote image has been updated
func CheckImageAllowedHandler(transport http.RoundTripper) router.HandlerFunc {
	return func(req router.Request, resp router.Response) error {
		appInstance := req.Object.(*v1.AppInstance)
		cond := condition.Setter(appInstance, resp, v1.AppInstanceConditionImageAllowed)

		// We're only checking against the currently used image, so if the image name or digest is empty, we can't check
		if appInstance.Status.AppImage.Name == "" || appInstance.Status.AppImage.Digest == "" {
			cond.Unknown("")
			return nil
		}

		ref, err := name.ParseReference(appInstance.Status.AppImage.Name, name.WithDefaultRegistry(""), name.WithDefaultTag(""))
		if err != nil {
			e := fmt.Errorf("failed to parse image name: %w", err)
			logrus.Error(e)
			cond.Error(e)
			return nil
		}

		targetImage := ref.Context().Digest(appInstance.Status.AppImage.Digest).Name()

		if err := imageallowrules.CheckImageAllowed(req.Ctx, req.Client, appInstance.Namespace, targetImage, remote.WithTransport(transport)); err != nil {
			if errors.Is(err, &imageallowrules.ErrImageNotAllowed{}) {
				cond.Error(err)
				return nil
			} else {
				e := fmt.Errorf("failed to check if image is allowed: %w", err)
				logrus.Error(e)
				cond.Error(e)
				return nil
			}
		}
		cond.Success()
		return nil
	}
}
