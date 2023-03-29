package appdefinition

import (
	"errors"
	"fmt"
	"net/http"

	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/condition"
	"github.com/acorn-io/acorn/pkg/imageallowrules"
	"github.com/acorn-io/baaah/pkg/router"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/sirupsen/logrus"
)

// CheckImageAllowedHandler is a router handler that checks if the image is allowed by the image allow rules and sets a status field accordingly
func CheckImageAllowedHandler(transport http.RoundTripper) router.HandlerFunc {
	return func(req router.Request, resp router.Response) error {

		appInstance := req.Object.(*v1.AppInstance)
		cond := condition.Setter(appInstance, resp, v1.AppInstanceConditionImageAllowed)

		targetImage, unknownReason := determineTargetImage(appInstance)
		if targetImage == "" {
			if appInstance.Status.AppImage.Name != "" {
				targetImage = appInstance.Status.AppImage.Name
			} else {
				if unknownReason != "" {
					cond.Unknown(unknownReason)
				} else {
					cond.Error(fmt.Errorf("no image specified"))
				}
				return nil
			}
		}

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
