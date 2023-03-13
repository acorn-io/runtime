package volumes

import (
	"context"
	"strings"
	"testing"

	"github.com/acorn-io/acorn/integration/helper"
	adminv1 "github.com/acorn-io/acorn/pkg/apis/admin.acorn.io/v1"
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.admin.acorn.io/v1"
	kclient "github.com/acorn-io/acorn/pkg/k8sclient"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestProjectVolumeClassCreateValidation(t *testing.T) {
	helper.StartController(t)

	ctx := helper.GetCTX(t)
	kclient := helper.MustReturn(kclient.Default)
	ns := helper.TempNamespace(t, kclient)

	volumeClass := adminv1.ProjectVolumeClass{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "acorn-test-default",
			Namespace: ns.Name,
		},
		Default: true,
	}
	if err := kclient.Create(ctx, &volumeClass); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := kclient.Delete(context.Background(), &volumeClass); err != nil && !apierrors.IsNotFound(err) {
			t.Fatal(err)
		}
	}()

	tests := []struct {
		name        string
		volumeClass adminv1.ProjectVolumeClass
		wantError   bool
	}{
		{
			name:      "Default already exists",
			wantError: true,
			volumeClass: adminv1.ProjectVolumeClass{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "new-default",
					Namespace: ns.Name,
				},
				Default: true,
			},
		},
		{
			name: "Can create inactive",
			volumeClass: adminv1.ProjectVolumeClass{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "new-inactive",
					Namespace: ns.Name,
				},
				Inactive: true,
			},
		},
		{
			name: "Can create default and inactive",
			volumeClass: adminv1.ProjectVolumeClass{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "new-inactive-default",
					Namespace: ns.Name,
				},
				Default:  true,
				Inactive: true,
			},
		},
		{
			name:      "Can't create min greater than max",
			wantError: true,
			volumeClass: adminv1.ProjectVolumeClass{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "new-inverse-limits",
					Namespace: ns.Name,
				},
				Size: v1.VolumeClassSize{
					Min: "2Gi",
					Max: "1Gi",
				},
			},
		},
		{
			name:      "Can't create min greater than default",
			wantError: true,
			volumeClass: adminv1.ProjectVolumeClass{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "new-inverse-limits",
					Namespace: ns.Name,
				},
				Size: v1.VolumeClassSize{
					Min:     "2Gi",
					Default: "1Gi",
				},
			},
		},
		{
			name:      "Can't create default greater than max",
			wantError: true,
			volumeClass: adminv1.ProjectVolumeClass{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "new-inverse-limits",
					Namespace: ns.Name,
				},
				Size: v1.VolumeClassSize{
					Default: "2Gi",
					Max:     "1Gi",
				},
			},
		},
		{
			name: "Can create limits all equal",
			volumeClass: adminv1.ProjectVolumeClass{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "new-equal-limits",
					Namespace: ns.Name,
				},
				Size: v1.VolumeClassSize{
					Min:     "5Gi",
					Default: "5Gi",
					Max:     "5Gi",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := kclient.Create(ctx, &tt.volumeClass); !tt.wantError && err != nil {
				t.Fatal(err)
			} else if tt.wantError && err == nil {
				t.Fatal("expected error for test case")
			}
			if err := kclient.Delete(ctx, &tt.volumeClass); err != nil && !apierrors.IsNotFound(err) {
				t.Fatal(err)
			}

		})
	}
}

func TestEnsureCanUpdateProjectVolumeClassDefault(t *testing.T) {
	helper.StartController(t)

	ctx := helper.GetCTX(t)
	kclient := helper.MustReturn(kclient.Default)
	ns := helper.TempNamespace(t, kclient)

	volumeClass := adminv1.ProjectVolumeClass{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "acorn-test-default",
			Namespace: ns.Name,
		},
		Default: true,
	}
	if err := kclient.Create(ctx, &volumeClass); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := kclient.Delete(context.Background(), &volumeClass); err != nil && !apierrors.IsNotFound(err) {
			t.Fatal(err)
		}
	}()

	volumeClass.Inactive = true
	if err := kclient.Update(ctx, &volumeClass); err != nil {
		t.Fatal(err)
	}

	volumeClass.Inactive = false
	volumeClass.Default = false
	if err := kclient.Update(ctx, &volumeClass); err != nil {
		t.Fatal(err)
	}

	volumeClass.Default = true
	if err := kclient.Update(ctx, &volumeClass); err != nil {
		t.Fatal(err)
	}
}

func TestClusterVolumeClassCreateValidation(t *testing.T) {
	helper.StartController(t)

	ctx := helper.GetCTX(t)
	kclient := helper.MustReturn(kclient.Default)

	volumeClass := adminv1.ClusterVolumeClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: "acorn-test-default",
		},
		Default: true,
	}
	// The cluster may already have a default if it was created by the controller.
	// Just make sure we have a default for the validation tests.
	if err := kclient.Create(ctx, &volumeClass); err != nil && !strings.HasSuffix(err.Error(), "already default for cluster") {
		t.Fatal(err)
	}
	defer func() {
		if err := kclient.Delete(context.Background(), &volumeClass); err != nil && !apierrors.IsNotFound(err) {
			t.Fatal(err)
		}
	}()

	tests := []struct {
		name        string
		volumeClass adminv1.ClusterVolumeClass
		wantError   bool
	}{
		{
			name:      "Default already exists",
			wantError: true,
			volumeClass: adminv1.ClusterVolumeClass{
				ObjectMeta: metav1.ObjectMeta{
					Name: "new-default",
				},
				Default: true,
			},
		},
		{
			name: "Can create inactive",
			volumeClass: adminv1.ClusterVolumeClass{
				ObjectMeta: metav1.ObjectMeta{
					Name: "new-inactive",
				},
				Inactive: true,
			},
		},
		{
			name: "Can create default and inactive",
			volumeClass: adminv1.ClusterVolumeClass{
				ObjectMeta: metav1.ObjectMeta{
					Name: "new-inactive-default",
				},
				Default:  true,
				Inactive: true,
			},
		},
		{
			name:      "Can't create min greater than max",
			wantError: true,
			volumeClass: adminv1.ClusterVolumeClass{
				ObjectMeta: metav1.ObjectMeta{
					Name: "new-inverse-limits",
				},
				Size: v1.VolumeClassSize{
					Min: "2Gi",
					Max: "1Gi",
				},
			},
		},
		{
			name:      "Can't create min greater than default",
			wantError: true,
			volumeClass: adminv1.ClusterVolumeClass{
				ObjectMeta: metav1.ObjectMeta{
					Name: "new-inverse-limits",
				},
				Size: v1.VolumeClassSize{
					Min:     "2Gi",
					Default: "1Gi",
				},
			},
		},
		{
			name:      "Can't create default greater than max",
			wantError: true,
			volumeClass: adminv1.ClusterVolumeClass{
				ObjectMeta: metav1.ObjectMeta{
					Name: "new-inverse-limits",
				},
				Size: v1.VolumeClassSize{
					Default: "2Gi",
					Max:     "1Gi",
				},
			},
		},
		{
			name: "Can create limits all equal",
			volumeClass: adminv1.ClusterVolumeClass{
				ObjectMeta: metav1.ObjectMeta{
					Name: "new-equal-limits",
				},
				Size: v1.VolumeClassSize{
					Min:     "5Gi",
					Default: "5Gi",
					Max:     "5Gi",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := kclient.Create(ctx, &tt.volumeClass); !tt.wantError && err != nil {
				t.Fatal(err)
			} else if tt.wantError && err == nil {
				t.Fatal("expected error for test case")
			}
			if err := kclient.Delete(ctx, &tt.volumeClass); err != nil && !apierrors.IsNotFound(err) {
				t.Fatal(err)
			}

		})
	}
}

func TestEnsureCanUpdateClusterVolumeClassDefault(t *testing.T) {
	helper.StartController(t)

	ctx := helper.GetCTX(t)
	kclient := helper.MustReturn(kclient.Default)

	volumeClass := adminv1.ClusterVolumeClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: "acorn-test-default",
		},
		Default: true,
	}
	// The cluster may already have a default if it was created by the controller.
	// Just make sure we have a default for the validation tests.
	if err := kclient.Create(ctx, &volumeClass); err != nil {
		if !strings.HasSuffix(err.Error(), "already default for cluster") {
			t.Fatal(err)
		}

		// There is already a default in the cluster, so get that one and test on it.
		clusterVolumeClasses := new(adminv1.ClusterVolumeClassList)
		if err = kclient.List(ctx, clusterVolumeClasses); err != nil {
			t.Fatal(err)
		}

		for _, vc := range clusterVolumeClasses.Items {
			if vc.Default {
				volumeClass = vc
			}
		}
	}
	defer func() {
		if err := kclient.Delete(context.Background(), &volumeClass); err != nil && !apierrors.IsNotFound(err) {
			t.Fatal(err)
		}
	}()

	volumeClass.Inactive = true
	if err := kclient.Update(ctx, &volumeClass); err != nil {
		t.Fatal(err)
	}

	// Get the volume class to ensure the value was persisted as expected.
	volumeClass.Inactive = false
	if err := kclient.Get(ctx, client.ObjectKey{Namespace: volumeClass.Namespace, Name: volumeClass.Name}, &volumeClass); err != nil {
		t.Fatal(err)
	} else if !volumeClass.Inactive {
		t.Fatal("inactive is false instead of true")
	}

	volumeClass.Inactive = false
	volumeClass.Default = false
	if err := kclient.Update(ctx, &volumeClass); err != nil {
		t.Fatal(err)
	}

	// Get the volume class to ensure the value was persisted as expected.
	// Note: the controller likely switched default back to true, so not testing that here.
	volumeClass.Inactive = true
	if err := kclient.Get(ctx, client.ObjectKey{Namespace: volumeClass.Namespace, Name: volumeClass.Name}, &volumeClass); err != nil {
		t.Fatal(err)
	} else if volumeClass.Inactive {
		t.Fatal("inactive is false instead of true")
	}
}

func TestCreateProjectDefaultWithExistingClusterDefault(t *testing.T) {
	helper.StartController(t)

	ctx := helper.GetCTX(t)
	kclient := helper.MustReturn(kclient.Default)
	c, ns := helper.ClientAndNamespace(t)

	clusterVolumeClass := adminv1.ClusterVolumeClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: "acorn-test-default",
		},
		Default: true,
	}
	// The cluster may already have a default if it was created by the controller.
	// Just make sure we have a default for the validation tests.
	if err := kclient.Create(ctx, &clusterVolumeClass); err != nil && !strings.HasSuffix(err.Error(), "already default for cluster") {
		t.Fatal(err)
	}
	defer func() {
		if err := kclient.Delete(context.Background(), &clusterVolumeClass); err != nil && !apierrors.IsNotFound(err) {
			t.Fatal(err)
		}
	}()

	// Get VolumeClasses and ensure there is a default
	volumeClassList, err := c.VolumeClassList(ctx)
	if err != nil {
		t.Fatal(err)
	}

	var foundDefault bool
	for _, vc := range volumeClassList {
		if vc.Default {
			clusterVolumeClass = adminv1.ClusterVolumeClass(vc)
			foundDefault = true
		}
	}

	if !foundDefault {
		t.Fatal("no default cluster volume class found")
	}

	projectVolumeClass := adminv1.ProjectVolumeClass{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "acorn-test-default",
			Namespace: ns.Name,
		},
		Default: true,
	}
	if err := kclient.Create(ctx, &projectVolumeClass); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := kclient.Delete(context.Background(), &projectVolumeClass); err != nil && !apierrors.IsNotFound(err) {
			t.Fatal(err)
		}
	}()

	// Get VolumeClasses and ensure default is true
	volumeClassList, err = c.VolumeClassList(ctx)
	if err != nil {
		t.Fatal(err)
	}

	for _, vc := range volumeClassList {
		if vc.Default && vc.Name != clusterVolumeClass.Name && vc.Namespace != clusterVolumeClass.Namespace {
			return
		}
	}

	t.Fatal("project default not found")
}
