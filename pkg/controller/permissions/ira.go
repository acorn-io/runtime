package permissions

import (
	"errors"

	"github.com/acorn-io/baaah/pkg/router"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	adminv1 "github.com/acorn-io/runtime/pkg/apis/internal.admin.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/imageselector"
	kclient "github.com/acorn-io/runtime/pkg/k8sclient"
	"github.com/acorn-io/runtime/pkg/labels"
)

// BumpImageRoleAuthorizations will bump the failing apps covered by an image role authorization such that the app will
// be re-evaluated for image permissions.
func BumpImageRoleAuthorizations(req router.Request, _ router.Response) error {
	return bumpAppsForIRA(req, req.Object.(*adminv1.ImageRoleAuthorizationInstance))
}

// BumpClusterImageRoleAuthorizations will bump the failing apps covered by a cluster image role authorization such that
// the app will be re-evaluated for image permissions.
func BumpClusterImageRoleAuthorizations(req router.Request, _ router.Response) error {
	return bumpAppsForIRA(req, (*adminv1.ImageRoleAuthorizationInstance)(req.Object.(*adminv1.ClusterImageRoleAuthorizationInstance)))
}

func bumpAppsForIRA(req router.Request, ira *adminv1.ImageRoleAuthorizationInstance) error {
	// Only name patterns should be considered for re-evaluation, not signatures.
	nameSelectorOnly := v1.ImageSelector{
		NamePatterns: ira.Spec.ImageSelector.NamePatterns,
	}

	apps := new(v1.AppInstanceList)
	if err := req.List(apps, &kclient.ListOptions{
		Namespace: ira.Namespace,
	}); err != nil {
		return err
	}

	for _, app := range apps.Items {
		// If the app's image permissions were granted, then no need to re-evaluate those permissions
		if len(app.Status.Staged.ImagePermissionsDenied) == 0 {
			continue
		}

		imageName := app.Status.AppImage.Name

		// E.g. for child Acorns, the appImage.Name is the image ID, but we need the original image name (with registry/repo)
		// to check for the signatures
		if oi, ok := app.GetAnnotations()[labels.AcornOriginalImage]; ok {
			imageName = oi
		}

		err := imageselector.MatchImage(req.Ctx, req.Client, app.Namespace, imageName, "", app.Status.AppImage.Digest, nameSelectorOnly, imageselector.MatchImageOpts{})
		if ierr := (*imageselector.NoMatchError)(nil); errors.As(err, &ierr) {
			// If this app is not covered by the image role authorization, then no need to re-evaluate permissions
			continue
		} else if err != nil {
			return err
		}

		// The app is covered by the image role authorization, so reset the observed generation so permissions are re-evaluated.
		app.Status.Staged.PermissionsObservedGeneration = -1
		if err := req.Client.Status().Update(req.Ctx, &app); err != nil {
			return err
		}
	}

	ira.Status.ObservedGeneration = ira.Generation
	return nil
}
