package imagerules

import (
	"context"
	"testing"

	"github.com/acorn-io/runtime/integration/helper"
	adminv1 "github.com/acorn-io/runtime/pkg/apis/admin.acorn.io/v1"
	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	internalv1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	internaladminv1 "github.com/acorn-io/runtime/pkg/apis/internal.admin.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/client"
	"github.com/acorn-io/runtime/pkg/imagerules"
	"github.com/acorn-io/runtime/pkg/imageselector"
	kclient "github.com/acorn-io/runtime/pkg/k8sclient"
	"github.com/acorn-io/runtime/pkg/profiles"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/stretchr/testify/require"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	cclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func TestImageRoleAuthorizations(t *testing.T) {
	helper.StartController(t)
	registry, close := helper.StartRegistry(t)
	defer close()

	ctx := helper.GetCTX(t)
	c, _ := helper.ClientAndProject(t)
	kclient := helper.MustReturn(kclient.Default)

	// enable image role authorizations in acorn config
	helper.EnableFeatureWithRestore(t, ctx, kclient, profiles.FeatureImageRoleAuthorizations)

	// Delete any existing rules from this project namespace
	err := kclient.DeleteAllOf(ctx, &internaladminv1.ImageRoleAuthorizationInstance{}, cclient.InNamespace(c.GetNamespace()))
	if err != nil {
		t.Fatal(err)
	}
	err = kclient.DeleteAllOf(ctx, &internaladminv1.ClusterImageRoleAuthorizationInstance{})
	if err != nil {
		t.Fatal(err)
	}

	// Build Image
	image, err := c.AcornImageBuild(ctx, "./testdata/nested-perms/Acornfile", &client.AcornImageBuildOptions{
		Cwd: "./testdata/nested-perms/",
	})
	if err != nil {
		t.Fatal(err)
	}
	id := image.ID

	tagName := registry + "/test:ci"

	err = c.ImageTag(ctx, id, tagName)
	if err != nil {
		t.Fatal(err)
	}

	progress, err := c.ImagePush(ctx, tagName, nil)
	if err != nil {
		t.Fatal(err)
	}

	for update := range progress {
		if update.Error != "" {
			t.Fatal(update.Error)
		}
	}

	details, err := c.ImageDetails(ctx, id, &client.ImageDetailsOptions{IncludeNested: true})
	if err != nil {
		t.Fatal(err)
	}
	require.Equal(t, 1, len(details.NestedImages))

	ref, err := name.ParseReference(tagName)
	if err != nil {
		t.Fatal(err)
	}

	targetDigest := ref.Context().Digest(details.AppImage.Digest)
	t.Logf("target digest: %s", targetDigest)

	require.Empty(t, details.SignatureDigest, "signature digest should be empty before signing")

	// DEPLOY

	// Integration tests don't have proper privileges so we will by pass the permission validation
	blueprint := &internalv1.AppInstance{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-",
			Namespace:    c.GetProject(),
		},
		Spec: internalv1.AppInstanceSpec{
			Image: image.ID,
			Permissions: []internalv1.Permissions{
				{
					ServiceName: "rootapp",
					Rules: []internalv1.PolicyRule{{
						PolicyRule: rbacv1.PolicyRule{
							APIGroups: []string{"api.acorn.io"},
							Verbs:     []string{"get"},
						},
					}},
				},
				{
					ServiceName: "foo.awsapp",
					Rules: []internalv1.PolicyRule{{
						PolicyRule: rbacv1.PolicyRule{
							APIGroups: []string{"aws.acorn.io"},
							Verbs:     []string{"get"},
						},
					}},
				},
			},
		},
		Status: internalv1.AppInstanceStatus{},
	}

	// --------------------------------------------------------------------
	// Helper Functions
	// --------------------------------------------------------------------
	createWaitLoop := func(ctx context.Context, appInstance *internalv1.AppInstance) *apiv1.App {
		if err := kclient.Create(ctx, appInstance); err != nil {
			t.Fatal(err)
		}

		app, err := c.AppGet(ctx, appInstance.Name)
		if err != nil {
			t.Fatal(err)
		}

		t.Logf("Creating app %s", app.Name)

		helper.WaitForObject(t, helper.Watcher(t, c), &apiv1.AppList{}, app, func(obj *apiv1.App) bool {
			return obj.Status.Staged.PermissionsChecked
		})

		app, err = c.AppGet(ctx, appInstance.Name)
		if err != nil {
			t.Fatal(err)
		}

		t.Logf("Created app %s", app.Name)

		return app
	}

	rmapp := func(ctx context.Context, app *apiv1.App) {
		_, err := c.AppDelete(ctx, app.Name)
		if err != nil {
			t.Fatal(err)
		}
	}

	// --------------------------------------------------------------------
	// Run #1 - Expect denied permissions since we have no IRAs
	// --------------------------------------------------------------------
	app := createWaitLoop(ctx, blueprint.DeepCopy())
	require.Equal(t, 2, len(app.Status.Staged.ImagePermissionsDenied), "should have 2 denied permissions with no IRA defined")
	rmapp(ctx, app)

	// --------------------------------------------------------------------
	// Run #2 - Expect denied permissions since we have an IRA but it does not cover the image
	// --------------------------------------------------------------------
	ira := &adminv1.ImageRoleAuthorization{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: c.GetNamespace(),
		},
		ImageSelector: internalv1.ImageSelector{
			NamePatterns: []string{"foobar"}, // does not cover the image
		},
		Roles: internaladminv1.RoleAuthorizations{
			Scopes: []string{"project"},
			RoleRefs: []internaladminv1.RoleRef{
				{
					Name: "acorn:aws:admin",
					Kind: "ClusterRole",
				},
			},
		},
	}

	err = kclient.Create(ctx, ira)
	if err != nil {
		t.Fatal(err)
	}

	app = createWaitLoop(ctx, blueprint.DeepCopy())
	require.Equal(t, 2, len(app.Status.Staged.ImagePermissionsDenied), "should have 2 denied permissions")
	rmapp(ctx, app)

	// --------------------------------------------------------------------
	// Run #3 - Expect denied permissions since we have an IRA but it does only cover one api group
	// --------------------------------------------------------------------
	ira.ImageSelector.NamePatterns = []string{tagName, id}
	for _, ni := range details.NestedImages {
		ira.ImageSelector.NamePatterns = append(ira.ImageSelector.NamePatterns, ni.ImageName)
		ira.ImageSelector.NamePatterns = append(ira.ImageSelector.NamePatterns, ni.Digest)
	}
	err = kclient.Update(ctx, ira)
	require.NoError(t, err, "should not error while updating IRA")

	// Ensure that the selector matches the image now
	err = imageselector.MatchImage(ctx, kclient, c.GetNamespace(), tagName, id, image.Digest, ira.ImageSelector)
	require.NoError(t, err, "should not error while matching image")

	details, err = c.ImageDetails(ctx, id, &client.ImageDetailsOptions{IncludeNested: true})
	require.NoError(t, err, "should not error while getting image details")

	ra, err := imagerules.GetAuthorizedPermissions(ctx, kclient, c.GetNamespace(), tagName, image.Digest)
	require.NoError(t, err, "should not error while getting authorized permissions")
	t.Logf("Authorized Permissions: %#v", ra)

	// replace name in granted permissions
	for i := range ra {
		ra[i].ServiceName = "foo.awsapp"
	}

	missing, granted := internalv1.GrantsAll(c.GetNamespace(), details.NestedImages[0].Permissions, ra)
	require.True(t, granted, "should have granted permissions, but have missing: %#v\nr: %#v\ng:%#v", missing, details.NestedImages[0].Permissions, ra)

	app = createWaitLoop(ctx, blueprint.DeepCopy())
	require.Equal(t, 1, len(app.Status.Staged.ImagePermissionsDenied), "should have 1 denied permissions: %#v", app.Status.Staged.ImagePermissionsDenied)
	rmapp(ctx, app)

	// --------------------------------------------------------------------

	// // create first rule

	// // update image allow rule to cover that image
	// ira.ImageSelector.NamePatterns = []string{tagName, id}

	// err = kclient.Update(ctx, ira)
	// if err != nil {
	// 	t.Fatal(err)
	// }

	// var niar apiv1.ImageAllowRule
	// err = kclient.Get(ctx, cclient.ObjectKeyFromObject(ira), &niar)
	// if err != nil {
	// 	t.Fatal(err)
	// }

	// require.Equal(t, ira.ImageSelector.NamePatterns, niar.ImageSelector.NamePatterns)

	// // try to run by tagName - expect success
	// app, err = c.AppRun(ctx, tagName, nil)
	// require.NoError(t, err, "should not error since image `%s` is covered by images scope `%+v` of IAR and there are not other rules", tagName, ira.ImageSelector.NamePatterns)

	// // remove app
	// _, err = c.AppDelete(ctx, app.Name)
	// if err != nil {
	// 	t.Fatal(err)
	// }

	// // try to run by ID - expect success
	// app, err = c.AppRun(ctx, id, nil)
	// require.NoError(t, err, "should not error since image `%s` is covered by images scope `%+v` of IAR and there are not other rules", id, ira.ImageSelector.NamePatterns)

	// // remove app
	// _, err = c.AppDelete(ctx, app.Name)
	// if err != nil {
	// 	t.Fatal(err)
	// }

	// // update iar to require a signature
	// ira.ImageSelector.Signatures = []internalv1.SignatureRules{
	// 	{
	// 		SignedBy: internalv1.SignedBy{
	// 			AllOf: []string{string(pubkeyCosign)},
	// 		},
	// 	},
	// }

	// err = kclient.Update(ctx, ira)
	// if err != nil {
	// 	t.Fatal(err)
	// }

	// // try to run - expect failure
	// _, err = c.AppRun(ctx, tagName, nil)
	// require.Error(t, err, "should error since image %s is not signed by the required key", tagName)

	// // sign image
	// nsig, err := signImage(ctx, c, targetDigest, tagName, "./testdata/cosign.key")
	// if err != nil {
	// 	t.Fatal(err)
	// }

	// // push again (this should include the signature)
	// progress, err = c.ImagePush(ctx, tagName, nil)
	// if err != nil {
	// 	t.Fatal(err)
	// }

	// for update := range progress {
	// 	if update.Error != "" {
	// 		t.Fatal(update.Error)
	// 	}
	// }

	// require.NotEmpty(t, nsig.SignatureDigest, "signature should not be empty")

	// ndetails, err := c.ImageDetails(ctx, tagName, nil)
	// if err != nil {
	// 	t.Fatal(err)
	// }

	// t.Logf("Image %s has signature %s", tagName, ndetails.SignatureDigest)
	// require.NotEmpty(t, ndetails.SignatureDigest, "signature digest should not be empty after signing")

	// // try to run by tagName - expect success
	// app, err = c.AppRun(ctx, tagName, nil)
	// require.NoError(t, err, "should not error since image %s is signed by the required key", tagName)

	// // remove app
	// _, err = c.AppDelete(ctx, app.Name)
	// if err != nil {
	// 	t.Fatal(err)
	// }

	// // try to run by ID - expect success
	// app, err = c.AppRun(ctx, id, nil)
	// require.NoError(t, err, "should not error since image %s is signed by the required key", id)

	// // remove app
	// _, err = c.AppDelete(ctx, app.Name)
	// if err != nil {
	// 	t.Fatal(err)
	// }

	// // Delete the image, so that it's only in the external registry
	// img, tags, err := c.ImageDelete(ctx, id, nil)
	// if err != nil {
	// 	t.Fatal(err)
	// }
	// require.Contains(t, tags, tagName, "tag %s should be deleted", tagName)
	// require.Equal(t, id, img.Name, "image %s should be deleted", id)

	// // try to run by tagName - expect success
	// app, err = c.AppRun(ctx, tagName, nil)
	// require.NoError(t, err, "should not error since image %s is signed by the required key", tagName)

	// // remove app
	// _, err = c.AppDelete(ctx, app.Name)
	// if err != nil {
	// 	t.Fatal(err)
	// }

	// // try to run by id - expect failure
	// _, err = c.AppRun(ctx, id, nil)
	// require.Error(t, err, "should error since image %s was deleted", id)

	// // update iar to require a signature with specific annotation
	// ira.ImageSelector.Signatures = []internalv1.SignatureRules{
	// 	{
	// 		SignedBy: internalv1.SignedBy{
	// 			AllOf: []string{string(pubkeyCosign)},
	// 		},
	// 		Annotations: internalv1.SignatureAnnotations{
	// 			Match: map[string]string{
	// 				"foo": "bar",
	// 			},
	// 		},
	// 	},
	// }

	// err = kclient.Update(ctx, ira)
	// if err != nil {
	// 	t.Fatal(err)
	// }

	// // try to run - expect failure
	// _, err = c.AppRun(ctx, tagName, nil)
	// require.Error(t, err, "should error since image is signed by the required key but does not match the required annotation")
}
