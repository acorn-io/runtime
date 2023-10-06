package imagerules

import (
	"context"
	_ "embed"
	"os"
	"strings"
	"testing"

	"github.com/acorn-io/baaah/pkg/apply"
	"github.com/acorn-io/runtime/integration/helper"
	adminv1 "github.com/acorn-io/runtime/pkg/apis/admin.acorn.io/v1"
	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	internalv1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	internaladminv1 "github.com/acorn-io/runtime/pkg/apis/internal.admin.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/awspermissions"
	"github.com/acorn-io/runtime/pkg/client"
	"github.com/acorn-io/runtime/pkg/imagerules"
	"github.com/acorn-io/runtime/pkg/imageselector"
	kclient "github.com/acorn-io/runtime/pkg/k8sclient"
	"github.com/acorn-io/runtime/pkg/profiles"
	"github.com/stretchr/testify/require"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	cclient "sigs.k8s.io/controller-runtime/pkg/client"
)

//go:embed testdata/nested-perms/Acornfile
var parentAcornfile []byte

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

	// Build nested image
	nestedImage, err := c.AcornImageBuild(ctx, "./testdata/nested-perms/nested.acorn", &client.AcornImageBuildOptions{
		Cwd: "./testdata/nested-perms/",
	})
	if err != nil {
		t.Fatal(err)
	}
	nestedImageID := nestedImage.ID

	nestedImageTagName := registry + "/test:ci-nested"

	err = c.ImageTag(ctx, nestedImageID, nestedImageTagName)
	if err != nil {
		t.Fatal(err)
	}

	progress, err := c.ImagePush(ctx, nestedImageTagName, nil)
	if err != nil {
		t.Fatal(err)
	}

	for update := range progress {
		if update.Error != "" {
			t.Fatal(update.Error)
		}
	}

	// Copy Acornfile and replace image name with the nested image
	acornfile, err := os.CreateTemp("testdata/nested-perms", "test-acornfile-*.acorn")
	if err != nil {
		t.Fatal(err)
	}

	_, err = acornfile.Write([]byte(strings.Replace(string(parentAcornfile), "%REPLACE_IMAGE%", nestedImageTagName, 1)))
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		os.Remove(acornfile.Name())
	})

	// Build Image
	image, err := c.AcornImageBuild(ctx, acornfile.Name(), &client.AcornImageBuildOptions{
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

	progress, err = c.ImagePush(ctx, tagName, nil)
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

	// DEPLOY

	// Integration tests don't have proper privileges so we will by pass the permission validation
	blueprint := &internalv1.AppInstance{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-",
			Namespace:    c.GetProject(),
		},
		Spec: internalv1.AppInstanceSpec{
			Image: tagName,
			UserGrantedPermissions: []internalv1.Permissions{
				{
					ServiceName: "rootapp",
					Rules: []internalv1.PolicyRule{{
						PolicyRule: rbacv1.PolicyRule{
							APIGroups: []string{"foo.bar.com"},
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

		helper.WaitForObject(t, helper.Watcher(t, c), &apiv1.AppList{}, app, func(obj *apiv1.App) bool {
			return obj.Status.Staged.PermissionsChecked
		})

		app, err = c.AppGet(ctx, appInstance.Name)
		if err != nil {
			t.Fatal(err)
		}

		return app
	}

	rmapp := func(ctx context.Context, app *apiv1.App) {
		_, err := c.AppDelete(ctx, app.Name)
		if err != nil {
			t.Fatal(err)
		}
	}

	// Prep: Create a role that we can use later
	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo:bar:admin",
			Namespace: c.GetNamespace(),
		},
		Rules: []rbacv1.PolicyRule{{
			APIGroups: []string{"foo.bar.com"},
			Verbs:     []string{"*"},
			Resources: []string{"*"},
		}},
	}

	err = kclient.Create(ctx, role)
	if err != nil {
		t.Fatal(err)
	}

	// --------------------------------------------------------------------
	// Run #1 - Expect denied permissions since we have no IRAs
	// --------------------------------------------------------------------
	app := createWaitLoop(ctx, blueprint.DeepCopy())
	require.Equal(t, 1, len(app.Status.Staged.ImagePermissionsDenied), "should have 1 denied permission (rootapp) with no IRA defined")
	expectedDeniedPermissionsRootapp := []internalv1.Permissions{
		{
			ServiceName: "rootapp",
			Rules: []internalv1.PolicyRule{
				{
					PolicyRule: rbacv1.PolicyRule{
						APIGroups: []string{"foo.bar.com"},
						Verbs:     []string{"get"},
					},
				},
			},
		},
	}
	require.Equal(t, expectedDeniedPermissionsRootapp, app.Status.Staged.ImagePermissionsDenied, "should have denied permissions for foo.bar.com on rootapp")
	rmapp(ctx, app)

	// --------------------------------------------------------------------
	// Run #2 - Expect denied permissions since we have an IRA but it does not cover the image
	// --------------------------------------------------------------------
	ira := &adminv1.ImageRoleAuthorization{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: c.GetNamespace(),
		},
		Spec: internaladminv1.ImageRoleAuthorizationInstanceSpec{
			ImageSelector: internalv1.ImageSelector{
				NamePatterns: []string{"foobar"}, // does not cover the image
			},
			Roles: internaladminv1.RoleAuthorizations{
				Scopes: []string{"project"},
				RoleRefs: []internaladminv1.RoleRef{
					{
						Name: "foo:bar:admin", // required by rootapp
						Kind: "Role",          // current namespace only
					},
				},
			},
		},
	}

	err = kclient.Create(ctx, ira)
	if err != nil {
		t.Fatal(err)
	}

	app = createWaitLoop(ctx, blueprint.DeepCopy())
	require.Equal(t, 1, len(app.Status.Staged.ImagePermissionsDenied), "should have 1 denied permission (rootapp)")
	rmapp(ctx, app)

	// --------------------------------------------------------------------
	// Run #3 - Expect denied permissions since we have an IRA but it does only cover one api group
	// --------------------------------------------------------------------
	ira.Spec.ImageSelector.NamePatterns = []string{tagName, nestedImageTagName}
	err = apply.Ensure(ctx, kclient, ira)
	require.NoError(t, err, "should not error while updating IRA")

	// Ensure that the selector matches the image now
	err = imageselector.MatchImage(ctx, kclient, c.GetNamespace(), tagName, id, image.Digest, ira.Spec.ImageSelector, imageselector.MatchImageOpts{})
	require.NoError(t, err, "should not error while matching image")

	details, err = c.ImageDetails(ctx, id, &client.ImageDetailsOptions{IncludeNested: true})
	require.NoError(t, err, "should not error while getting image details")

	ra, err := imagerules.GetAuthorizedPermissions(ctx, kclient, c.GetNamespace(), tagName, image.Digest)
	require.NoError(t, err, "should not error while getting authorized permissions")

	// replace name in granted permissions
	for i := range ra {
		ra[i].ServiceName = "rootapp"
	}

	missing, granted := internalv1.GrantsAll(c.GetNamespace(), details.Permissions, ra)
	require.True(t, granted, "should have granted permissions, but have missing: %#v\nr: %#v\ng:%#v", missing, details.Permissions, ra)

	app = createWaitLoop(ctx, blueprint.DeepCopy())
	require.Equal(t, 0, len(app.Status.Staged.ImagePermissionsDenied), "should have 1 denied permissions: %#v", app.Status.Staged.ImagePermissionsDenied)

	app = helper.WaitForObject(t, helper.Watcher(t, c), &apiv1.AppList{}, app, func(obj *apiv1.App) bool {
		return obj.Status.AppStatus.Containers["rootapp"].Ready
	})

	// When parent app is ready, the nested app should not, because it still has denied permissions
	nestedApp, err := c.AppGet(ctx, app.Status.AppStatus.Acorns["foo"].AcornName)
	require.NoError(t, err, "should not error while getting nested app")

	nestedApp = helper.WaitForObject(t, helper.Watcher(t, c), &apiv1.AppList{}, nestedApp, func(obj *apiv1.App) bool {
		return obj.Status.Staged.PermissionsChecked
	})

	require.Equal(t, 1, len(nestedApp.Status.Staged.ImagePermissionsDenied), "should have 1 denied permission (foo.awsapp)")
	expectedDeniedPermissionsFooAwsapp := []internalv1.Permissions{
		{
			ServiceName: "awsapp",
			Rules: []internalv1.PolicyRule{
				{
					PolicyRule: rbacv1.PolicyRule{
						APIGroups: []string{"aws.acorn.io"},
						Verbs:     []string{"get"},
					},
				},
			},
		},
	}

	require.Equal(t, expectedDeniedPermissionsFooAwsapp, nestedApp.Status.Staged.ImagePermissionsDenied, "should have denied permissions for aws.acorn.io on foo.awsapp")

	rmapp(ctx, app)

	// --------------------------------------------------------------------
	// Run #4 - Expect no denied permissions since we have an IRA that covers the image
	// --------------------------------------------------------------------

	awsRole := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      awspermissions.AWSAdminRole,
			Namespace: c.GetNamespace(),
		},
		Rules: awspermissions.AWSRoles[awspermissions.AWSAdminRole],
	}

	err = kclient.Create(ctx, awsRole)
	require.NoError(t, err, "should not error while creating aws role")

	// Add the missing api group to the IRA
	ira.Spec.Roles.RoleRefs = append(ira.Spec.Roles.RoleRefs, internaladminv1.RoleRef{
		Name: awspermissions.AWSAdminRole, // required by foo.awsapp
		Kind: "Role",
	})
	err = apply.Ensure(ctx, kclient, ira)
	require.NoError(t, err, "should not error while updating IRA")

	// Ensure that the selector matches the image now
	err = imageselector.MatchImage(ctx, kclient, c.GetNamespace(), nestedImageTagName, "", nestedImage.Digest, ira.Spec.ImageSelector, imageselector.MatchImageOpts{})
	require.NoError(t, err, "should not error while matching image")

	app = createWaitLoop(ctx, blueprint.DeepCopy())
	require.Equal(t, 0, len(app.Status.Staged.ImagePermissionsDenied), "should have 0 denied permissions")

	app = helper.WaitForObject(t, helper.Watcher(t, c), &apiv1.AppList{}, app, func(obj *apiv1.App) bool {
		return obj.Status.AppStatus.Containers["rootapp"].Ready
	})

	nestedApp, err = c.AppGet(ctx, app.Status.AppStatus.Acorns["foo"].AcornName)
	require.NoError(t, err, "should not error while getting nested app")

	nestedApp = helper.WaitForObject(t, helper.Watcher(t, c), &apiv1.AppList{}, nestedApp, func(obj *apiv1.App) bool {
		return obj.Status.Staged.PermissionsChecked
	})

	require.Equal(t, 0, len(nestedApp.Status.Staged.ImagePermissionsDenied), "should have 0 denied permissions")

	// --------------------------------------------------------------------
	// Run #5 - Now we're updating the granted permissions to include permissions that the image is not authorized to have -> expect denied permissions
	// --------------------------------------------------------------------

	rmapp(ctx, app)

	nappinstance := blueprint.DeepCopy()
	nappinstance.Spec.UserGrantedPermissions = append(app.Spec.UserGrantedPermissions, internalv1.Permissions{
		ServiceName: "rootapp",
		Rules: []internalv1.PolicyRule{{
			PolicyRule: rbacv1.PolicyRule{
				APIGroups: []string{"*"},
				Verbs:     []string{"*"},
				Resources: []string{"*"},
			},
		}},
	})

	app = createWaitLoop(ctx, nappinstance)

	require.Equal(t, 1, len(app.Status.Staged.ImagePermissionsDenied), "should have 1 denied permission (rootapp)")
}
