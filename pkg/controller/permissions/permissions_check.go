package permissions

import (
	"fmt"
	"strings"

	"github.com/acorn-io/baaah/pkg/router"
	"github.com/acorn-io/baaah/pkg/uncached"
	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/tags"
	"github.com/google/go-containerregistry/pkg/name"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func CopyPromoteStagedAppImage(req router.Request, resp router.Response) error {
	app := req.Object.(*v1.AppInstance)
	if app.Status.Staged.AppImage.ID != "" &&
		app.Status.Staged.PermissionsChecked &&
		len(app.Status.Staged.PermissionsMissing) == 0 {
		app.Status.AppImage = app.Status.Staged.AppImage
	}
	return nil
}

func CheckImagePermissions(req router.Request, resp router.Response) error {
	app := req.Object.(*v1.AppInstance)
	if app.Status.Staged.AppImage.ID == "" ||
		app.Status.Staged.AppImage.Digest == app.Status.AppImage.Digest ||
		app.Status.Staged.PermissionsObservedGeneration == app.Generation {
		return nil
	}

	var (
		appImage  = app.Status.Staged.AppImage
		imageName = appImage.ID
		details   = &apiv1.ImageDetails{
			DeployArgs:    app.Spec.DeployArgs,
			Profiles:      app.Spec.GetProfiles(app.Status.GetDevMode()),
			IncludeNested: true,
		}
	)

	if !tags.IsLocalReference(imageName) {
		ref, err := name.ParseReference(imageName)
		if err != nil {
			return err
		}
		imageName = ref.Context().Digest(appImage.Digest).String()
	}

	err := req.Client.SubResource("details").Create(req.Ctx, uncached.Get(&apiv1.Image{
		ObjectMeta: metav1.ObjectMeta{
			Name:      strings.ReplaceAll(imageName, "/", "+"),
			Namespace: app.Namespace,
		},
	}), details)
	if err != nil {
		return err
	}

	if details.AppImage.Digest != appImage.Digest {
		return fmt.Errorf("failed to lookup image [%s], resolved to digest [%s] but expected [%s]", imageName,
			details.AppImage.Digest, appImage.Digest)
	}

	missing, _ := v1.GrantsAll(app.Namespace, details.GetPermissions(), app.Spec.GetPermissions())
	app.Status.Staged.PermissionsObservedGeneration = app.Generation
	app.Status.Staged.PermissionsChecked = true
	app.Status.Staged.PermissionsMissing = missing

	return nil
}
