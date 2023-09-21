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
	"github.com/acorn-io/runtime/pkg/imagerules"
	"github.com/acorn-io/runtime/pkg/profiles"
	"github.com/acorn-io/runtime/pkg/tags"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func CopyPromoteStagedAppImage(req router.Request, resp router.Response) error {
	app := req.Object.(*v1.AppInstance)
	if app.Status.Staged.AppImage.ID != "" &&
		app.Status.Staged.PermissionsChecked &&
		len(app.Status.Staged.PermissionsMissing) == 0 &&
		len(app.Status.Staged.ImagePermissionsDenied) == 0 {
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
		authzPerms, err := imagerules.GetAuthorizedPermissions(req.Ctx, req.Client, app.Namespace, appImage.Name, appImage.Digest)
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

		replaceName := func(perms []v1.Permissions, name string) []v1.Permissions {
			for i := range perms {
				perms[i].ServiceName = name
			}
			return perms
		}

		denied, granted := v1.GrantsAll(app.Namespace, copyWithName(details.Permissions, appImage.Name), authzPerms)
		logrus.Errorf("@@@ [app=%s / image=%s] denied: %#v", app.Name, appImage.Name, denied)

		if !granted {
			// if authorized permissions do not cover all requested permissions, go through all
			// parent images and use them to try to cover the missing permissions
			for _, parent := range app.Status.Parents {
				parentAuthz, err := imagerules.GetAuthorizedPermissions(req.Ctx, req.Client, app.Namespace, parent.ImageName, parent.ImageDigest)
				if err != nil {
					return err
				}
				denied, granted = v1.GrantsAll(app.Namespace, replaceName(denied, parent.ImageName), parentAuthz)
				if granted {
					// Yay - all permissions covered now
					break
				}
				logrus.Errorf("@@@  [app=%s / image=%s] Checked parent image %s (%s) denied: %#v", app.Name, appImage.Name, parent.ImageName, parent.ImageDigest, denied)
			}
		}

		logrus.Infof("@@@  [app=%s] denied (final): %#v", app.Name, denied)
		app.Status.Staged.ImagePermissionsDenied = replaceName(denied, app.Name)
	}

	// This is checking if the user granted all permissions that the app requires
	missing, _ := v1.GrantsAll(app.Namespace, details.GetPermissions(), app.Spec.GetPermissions())
	app.Status.Staged.PermissionsObservedGeneration = app.Generation
	app.Status.Staged.PermissionsChecked = true
	app.Status.Staged.PermissionsMissing = missing

	return nil
}

// for _, nestedImage := range details.NestedImages {
// 	// For nested image permissions, we first check if the parent (current) image is authorized.
// 	// For all the permissions that the parent image does not have, we check if the nested image is authorized.
// 	// If the parent image is authorized for all permissions, we spare checking the nested image, which saves some external requests.
// 	parentDenied, parentAuthorized := v1.GrantsAll(app.Namespace, copyWithName(nestedImage.Permissions, nestedImage.ImageName), copyWithName(authzPerms, nestedImage.ImageName))
// 	if !parentAuthorized {
// 		logrus.Errorf("@@@  [%s] parentDenied (parent: appImage.Name = %s / child: %s [%s]): %#v \n --> Parent Roles: %#v", app.Name, appImage.Name, nestedImage.ImageName, nestedImage.Digest, parentDenied, authzPerms)
// 		nestedRoles, err := imagerules.GetAuthorizedPermissions(req.Ctx, req.Client, app.Namespace, nestedImage.ImageName, nestedImage.Digest)
// 		if err != nil {
// 			return err
// 		}
// 		nestedDenied, authorized := v1.GrantsAll(app.Namespace, copyWithName(parentDenied, nestedImage.ImageName), nestedRoles)
// 		if !authorized {
// 			denied = append(denied, nestedDenied...)
// 			logrus.Errorf("@@@  [%s] denied (parent: appImage.Name = %s / child: %s): %#v", app.Name, appImage.Name, nestedImage.ImageName, denied)
// 		}
// 	}
// }
