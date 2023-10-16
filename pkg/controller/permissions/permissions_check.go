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
	"github.com/acorn-io/runtime/pkg/labels"
	"github.com/acorn-io/runtime/pkg/profiles"
	"github.com/acorn-io/runtime/pkg/tags"
	"github.com/acorn-io/z"
	"github.com/google/go-containerregistry/pkg/name"
	"golang.org/x/exp/maps"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/strings/slices"
)

// CopyPromoteStagedAppImage copies the staged app image to the app image if
// - the staged app image is set
// - the permissions have been checked
// - there are no missing permissions
// - there are no image permissions denied (if ImageRoleAuthorizations are enabled)
// - the image is allowed by the image allow rules (if enabled)
func CopyPromoteStagedAppImage(req router.Request, resp router.Response) error {
	app := req.Object.(*v1.AppInstance)
	if app.Status.Staged.AppImage.ID != "" &&
		app.Status.Staged.PermissionsChecked &&
		len(app.Status.Staged.PermissionsMissing) == 0 &&
		len(app.Status.Staged.ImagePermissionsDenied) == 0 &&
		z.Dereference[bool](app.Status.Staged.ImageAllowed) {
		app.Status.AppImage = app.Status.Staged.AppImage
		app.Status.Permissions = app.Status.Staged.AppScopedPermissions
	}
	return nil
}

/*
CheckPermissions checks various things related to permissions

	a) if the image is allowed by the image allow rules (if enabled)
	b) [if ImageRoleAuthorizations are enabled] if the authorized permissions for the appImage cover the permissions granted explicitly by the user or implicitly to the image (Authorized >= Granted)
	c) if the permissions granted by the user or implicitly to the image cover the permissions requested by the app (Granted >= Requested)

	*Note*: One thing that we do not cover here is permissions consumed from a service. Since that's dynamic and we cannot know the generated permissions at this point,
	the IRA check (if enabled) will happen later, just before the actual Kubernetes resources get applied (once the service instances are ready).
*/
func CheckPermissions(req router.Request, _ router.Response) error {
	app := req.Object.(*v1.AppInstance)

	// Reset staged status fields if the respective feature is disabled
	iraEnabled, err := config.GetFeature(req.Ctx, req.Client, profiles.FeatureImageRoleAuthorizations)
	if err != nil {
		return err
	}
	if !iraEnabled {
		app.Status.Staged.ImagePermissionsDenied = nil
	}

	// Early exit
	if app.Status.Staged.AppImage.ID == "" ||
		app.Status.Staged.AppImage.Digest == app.Status.AppImage.Digest ||
		app.Status.Staged.PermissionsObservedGeneration == app.Generation {
		// IAR disabled? Allow the Image if we're not re-checking permissions
		if enabled, err := config.GetFeature(req.Ctx, req.Client, profiles.FeatureImageAllowRules); err != nil {
			return err
		} else if !enabled {
			app.Status.Staged.ImageAllowed = z.Pointer(true)
		}
		return nil
	}

	if err := checkImageAllowed(req.Ctx, req.Client, app); err != nil {
		return err
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

	err = req.Client.SubResource("details").Create(req.Ctx, uncached.Get(&apiv1.Image{
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

	// ServiceNames of the current app level (i.e. not nested Acorns/Services)
	scvnames := maps.Keys(details.AppSpec.Containers)
	scvnames = append(scvnames, maps.Keys(details.AppSpec.Jobs)...)
	scvnames = append(scvnames, maps.Keys(details.AppSpec.Services)...)

	// Only consider the scope of the current app level (i.e. not nested Acorns/Services)
	grantedPerms := app.Spec.GetGrantedPermissions()
	scopedGrantedPerms := []v1.Permissions{}
	for i, p := range grantedPerms {
		if slices.Contains(scvnames, p.ServiceName) {
			scopedGrantedPerms = append(scopedGrantedPerms, grantedPerms[i])
		}
	}

	app.Status.Staged.AppScopedPermissions = scopedGrantedPerms

	// If iraEnabled, check if the Acorn images are authorized to request the defined permissions.
	if iraEnabled {
		imageName := appImage.Name

		// E.g. for child Acorns, the appImage.Name is the image ID, but we need the original image name (with registry/repo)
		// to check for the signatures
		if oi, ok := app.GetAnnotations()[labels.AcornOriginalImage]; ok {
			imageName = oi
		}

		authzPerms, err := imagerules.GetAuthorizedPermissions(req.Ctx, req.Client, app.Namespace, imageName, appImage.Digest)
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

		var deniedPerms []v1.Permissions
		for _, p := range scopedGrantedPerms {
			if d, granted := v1.GrantsAll(app.Namespace, []v1.Permissions{p}, copyWithName(authzPerms, p.ServiceName)); !granted {
				deniedPerms = append(deniedPerms, d...)
			}
		}

		app.Status.Staged.ImagePermissionsDenied = deniedPerms
	}

	// This is checking if the user granted all permissions that the app requires
	missing, _ := v1.GrantsAll(app.Namespace, details.GetAllImagesRequestedPermissions(), app.Spec.GetGrantedPermissions()) // Check: Granted Permissions is a superset of the requested permissions -> this includes nested images
	app.Status.Staged.PermissionsObservedGeneration = app.Generation
	app.Status.Staged.PermissionsChecked = true
	app.Status.Staged.PermissionsMissing = missing

	return nil
}
