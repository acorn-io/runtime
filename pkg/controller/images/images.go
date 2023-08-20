package images

import (
	"strings"

	"github.com/acorn-io/baaah/pkg/router"
	"github.com/acorn-io/baaah/pkg/typed"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/tags"
	"github.com/google/go-containerregistry/pkg/name"
	apierror "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func MigrateRemoteImages(req router.Request, _ router.Response) error {
	image := req.Object.(*v1.ImageInstance)
	if !image.ZZ_Remote || image.Repo == "" || image.Digest == "" {
		return nil
	}
	apps := &v1.AppInstanceList{}
	if err := req.List(apps, &kclient.ListOptions{}); err != nil {
		return err
	}

	for _, app := range apps.Items {
		if app.Status.AppImage.ID == image.Name && app.Status.AppImage.Digest == image.Digest {
			app.Status.AppImage.ID = image.Repo
			if err := req.Client.Status().Update(req.Ctx, &app); err != nil {
				return err
			}
		}
	}

	return req.Client.Delete(req.Ctx, image)
}

func CreateImages(req router.Request, _ router.Response) error {
	app := req.Object.(*v1.AppInstance)
	if app.Status.AppImage.ID == "" {
		return nil
	}

	repo, err := getRepo(req, app)
	if err != nil {
		return nil
	}

	return createNestedReferences(req, app, repo)
}

func createNestedReferences(req router.Request, app *v1.AppInstance, repo string) error {
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
				Repo:   repo,
				Digest: "sha256:" + imageName,
			})
		}
		if err != nil {
			return err
		}
	}

	return nil
}

func getRepo(req router.Request, app *v1.AppInstance) (string, error) {
	var (
		imageName = app.Status.AppImage.ID
	)

	if tags.IsLocalReference(imageName) {
		image := &v1.ImageInstance{}
		if err := req.Get(image, app.Namespace, imageName); err != nil {
			return "", err
		}
		return image.Repo, nil
	}

	ref, err := name.ParseReference(imageName, name.WithDefaultTag(""))
	if err != nil {
		return "", err
	}

	return ref.Context().String(), nil
}
