package permissions

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/acorn-io/baaah/pkg/router"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	imagerules "github.com/acorn-io/runtime/pkg/imagerules"
	"github.com/acorn-io/runtime/pkg/labels"
	"github.com/acorn-io/z"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/sirupsen/logrus"
)

// CheckImageAllowed is a router handler that checks if the image is allowed by the image allow rules and sets a status field accordingly
// This is only working on the currently specified image, referenced by digest, to avoid false positives (alerts) if the remote image has been updated
func CheckImageAllowed(transport http.RoundTripper) router.HandlerFunc {
	return func(req router.Request, resp router.Response) error {
		app := req.Object.(*v1.AppInstance)

		if app.Status.Staged.AppImage.ID == "" ||
			app.Status.Staged.AppImage.Digest == app.Status.AppImage.Digest ||
			app.Status.Staged.PermissionsObservedGeneration == app.Generation {
			return nil
		}

		imageName := app.Status.Staged.AppImage.Name
		imageDigest := app.Status.Staged.AppImage.Digest

		// We're only checking against the currently staged image, so if the image name or digest is empty, we can't check
		if imageName == "" || imageDigest == "" {
			return nil
		}

		if oi, ok := app.GetAnnotations()[labels.AcornOriginalImage]; ok {
			imageName = oi
		}

		ref, err := name.ParseReference(imageName, name.WithDefaultRegistry(""), name.WithDefaultTag(""))
		if err != nil {
			logrus.Error(fmt.Errorf("failed to parse image name: %w", err))
			app.Status.Staged.ImageAllowed = z.Pointer(false)
			return nil
		}

		targetImage := strings.TrimSuffix(ref.Name(), ":")

		if err := imagerules.CheckImageAllowed(req.Ctx, req.Client, app.Namespace, targetImage, imageName, imageDigest, remote.WithTransport(transport)); err != nil {
			if _, ok := err.(*imagerules.ErrImageNotAllowed); !ok {
				err = fmt.Errorf("failed to check if image is allowed: %w", err)
			}
			app.Status.Staged.ImageAllowed = z.Pointer(false)
			logrus.Errorln(err)
			return nil
		}
		app.Status.Staged.ImageAllowed = z.Pointer(true)
		return nil
	}
}
