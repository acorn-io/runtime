package permissions

import (
	"errors"
	"fmt"
	"strings"

	"github.com/acorn-io/baaah/pkg/router"
	"github.com/acorn-io/baaah/pkg/uncached"
	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/config"
	"github.com/acorn-io/runtime/pkg/imageallowrules"
	"github.com/acorn-io/runtime/pkg/profiles"
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
	} else if details.GetParseError() != "" {
		return errors.New(details.GetParseError())
	}

	if details.AppImage.Digest != appImage.Digest {
		return fmt.Errorf("failed to lookup image [%s], resolved to digest [%s] but expected [%s]", imageName,
			details.AppImage.Digest, appImage.Digest)
	}

	// If enabled, check if the Acorn images are authorized to request the defined permissions.
	if enabled, err := config.GetFeature(req.Ctx, req.Client, profiles.FeatureImageRoleAuthorizations); err != nil {
		return err
	} else if enabled {
		parentRoles, err := imageallowrules.GetAuthorizedRoles(req.Ctx, req.Client, app.Namespace, appImage.Name, appImage.Digest)
		if err != nil {
			return err
		}

		// Need to deepcopy here since otherwise we'd override the name in the original object which we still need for other authz checks
		// For IRA Checks, we use the image name as the service name
		copyWithName := func(perms []v1.Permissions, name string) []v1.Permissions {
			nperms := make([]v1.Permissions, len(perms))
			for i := range perms {
				nperms[i] = perms[i].DeepCopy().Get()
				nperms[i].ServiceName = name
			}
			return nperms
		}

		denied, _ := imageallowrules.Authorized(appImage.Name, app.Namespace, copyWithName(details.Permissions, appImage.Name), parentRoles)

		for _, img := range details.NestedImages {
			// For nested image permissions, we first check if the parent image is parentAuthorized.
			// For all the permissions that the parent image does not have, we check if the nested image is parentAuthorized.
			// If the parent image is parentAuthorized for all permissions, we spare checking the nested image, which saves some external requests.
			parentDenied, parentAuthorized := imageallowrules.Authorized(img.ImageName, app.Namespace, copyWithName(img.Permissions, img.ImageName), parentRoles)
			if !parentAuthorized {
				nestedRoles, err := imageallowrules.GetAuthorizedRoles(req.Ctx, req.Client, app.Namespace, img.ImageName, img.Digest)
				if err != nil {
					return err
				}
				nestedDenied, authorized := imageallowrules.Authorized(img.ImageName, app.Namespace, copyWithName(parentDenied, img.ImageName), nestedRoles)
				if !authorized {
					denied = append(denied, nestedDenied...)
				}
			}
		}

		app.Status.Staged.ImagePermissionsDenied = denied
	}

	// This is checking if the user granted all permissions that the app requires
	missing, _ := v1.GrantsAll(app.Namespace, details.GetPermissions(), app.Spec.GetPermissions())
	app.Status.Staged.PermissionsObservedGeneration = app.Generation
	app.Status.Staged.PermissionsChecked = true
	app.Status.Staged.PermissionsMissing = missing

	return nil
}
