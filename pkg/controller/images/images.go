package images

import (
	"strings"

	"github.com/acorn-io/baaah/pkg/router"
	"github.com/acorn-io/baaah/pkg/typed"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/tags"
	apierror "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func CreateImages(req router.Request, _ router.Response) error {
	app := req.Object.(*v1.AppInstance)
	if app.Status.AppImage.ID == "" || app.Status.AppImage.Digest == "" || app.Status.ObservedImageDigest == app.Status.AppImage.Digest {
		// If the image hasn't changed, then don't worry about creating it. It would have been created the first time.
		return nil
	}

	return createNestedReferences(req, app, self)
}

func createNestedReferences(req router.Request, app *v1.AppInstance, self *v1.ImageInstance) error {
	for _, imageData := range typed.Concat(app.Status.AppImage.ImageData.Acorns, app.Status.AppImage.ImageData.Images) {
		if !tags.IsLocalReference(imageData.Image) {
			continue
		}
		imageName := strings.TrimPrefix(imageData.Image, "sha256:")
		nestedImage := &v1.ImageInstance{}
		err := req.Get(nestedImage, app.Namespace, imageName)
		if err == nil {
			continue
		}

		if apierror.IsNotFound(err) {
			err = req.Client.Create(req.Ctx, &v1.ImageInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:      imageName,
					Namespace: app.Namespace,
				},
				Remote: self.Remote,
				Repo:   self.Repo,
				Digest: "sha256:" + imageName,
			})
		}
		if err != nil {
			return err
		}
	}

	return nil
}
