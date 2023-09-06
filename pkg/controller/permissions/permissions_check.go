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
	"github.com/acorn-io/runtime/pkg/images"
	"github.com/acorn-io/runtime/pkg/tags"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
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

	cfg, err := config.Get(req.Ctx, req.Client)
	if err != nil {
		return err
	}

	if len(cfg.RestrictedAPIGroups) > 0 {
		logrus.Infof("checking %d restricted API groups for image [%s]", len(cfg.RestrictedAPIGroups), appImage.ID)
		requestedAPIGroups := map[string]sets.Set[string]{}

		ref, err := images.GetImageReference(req.Ctx, req.Client, app.Namespace, imageName)
		if err != nil {
			return err
		}

		parentRef := ref.Context().Digest(appImage.Digest).String()
		imgsList := []string{parentRef}
		for _, perm := range details.Permissions {
			for _, rule := range perm.Rules {
				for _, g := range rule.APIGroups {
					if _, ok := requestedAPIGroups[g]; !ok {
						requestedAPIGroups[g] = sets.New[string](parentRef)
					}
				}
			}
		}

		for _, nested := range details.NestedImages {
			ref, err := images.GetImageReference(req.Ctx, req.Client, app.Namespace, nested.ImageRef)
			if err != nil {
				return err
			}
			nestedRef := ref.Context().Digest(nested.Digest).String()
			imgsList = append(imgsList, nestedRef)
			for _, perm := range nested.Permissions {
				for _, rule := range perm.Rules {
					for _, g := range rule.APIGroups {
						if _, ok := requestedAPIGroups[g]; !ok {
							requestedAPIGroups[g] = sets.New[string](nestedRef)
						} else {
							requestedAPIGroups[g].Insert(nestedRef)
						}
					}
				}
			}
		}
		iars := []v1.ImageAllowRuleInstance{
			{
				ObjectMeta: metav1.ObjectMeta{Name: "RAGCheck"},
				Images:     imgsList,
				Signatures: v1.ImageAllowRuleSignatures{Rules: []v1.SignatureRules{{SignedBy: v1.SignedBy{AllOf: []string{}}}}},
			},
		}
		setAuthority := func(iars []v1.ImageAllowRuleInstance, authority string) {
			for i := range iars {
				iars[i].Signatures.Rules[0].SignedBy.AllOf = []string{authority}
			}
		}
		for restrictedGroup, requiredAuthority := range cfg.RestrictedAPIGroups {
			if imgIDs, ok := requestedAPIGroups[restrictedGroup]; ok {
				logrus.Infof("restricted API group [%s] used by images [%s]: Checking parent image %s against %s", restrictedGroup, sets.List(imgIDs), parentRef, requiredAuthority)
				setAuthority(iars, requiredAuthority)
				if err := imageallowrules.CheckImageAgainstRules(req.Ctx, req.Client, app.Namespace, parentRef, details.AppImage.Digest, iars, nil); err == nil {
					logrus.Infof("Parent image [%s] authorized to use [%s] by [%s]", parentRef, restrictedGroup, requiredAuthority)
					continue
				}
				logrus.Infof("Parent image [%s] not authorized to use [%s] by [%s]: Checking individual images...", parentRef, restrictedGroup, requiredAuthority)
				for _, imageID := range sets.List[string](imgIDs) {
					logrus.Infof("restricted API group [%s] used by image [%s]: Checking against %s", restrictedGroup, imageID, requiredAuthority)
					if err := imageallowrules.CheckImageAgainstRules(req.Ctx, req.Client, app.Namespace, parentRef, details.AppImage.Digest, iars, nil); err != nil {
						return fmt.Errorf("image [%s] not authorized to use [%s] by [%s]", imageID, restrictedGroup, requiredAuthority)
					}
				}
			} else {
				logrus.Infof("restricted API group [%s] not used by image [%s]", restrictedGroup, parentRef)
			}
		}
	}

	missing, _ := v1.GrantsAll(app.Namespace, details.GetPermissions(), app.Spec.GetPermissions())
	app.Status.Staged.PermissionsObservedGeneration = app.Generation
	app.Status.Staged.PermissionsChecked = true
	app.Status.Staged.PermissionsMissing = missing

	return nil
}
