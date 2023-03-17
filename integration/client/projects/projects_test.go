package projects

import (
	"context"
	"strconv"
	"testing"

	"github.com/acorn-io/acorn/integration/helper"
	v1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	adminv1 "github.com/acorn-io/acorn/pkg/apis/internal.admin.acorn.io/v1"
	kclient "github.com/acorn-io/acorn/pkg/k8sclient"
	"github.com/stretchr/testify/assert"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestProjectCreationValidation(t *testing.T) {
	helper.StartController(t)

	ctx := helper.GetCTX(t)
	kclient := helper.MustReturn(kclient.Default)

	region := &adminv1.RegionInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name: "acorn-test-region",
		},
		Spec: adminv1.RegionInstanceSpec{},
	}
	if err := kclient.Create(ctx, region); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := kclient.Delete(context.Background(), region); err != nil && !apierrors.IsNotFound(err) {
			t.Fatal(err)
		}
	}()

	tests := []struct {
		name      string
		project   v1.Project
		wantError bool
	}{
		{
			name: "Create project with no region",
			project: v1.Project{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-project-blank",
				},
				Spec: v1.ProjectSpec{
					DefaultRegion: "",
				},
			},
		},
		{
			name: "Create project with existing region",
			project: v1.Project{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-project",
				},
				Spec: v1.ProjectSpec{
					DefaultRegion:    "acorn-test-region",
					SupportedRegions: []string{"acorn-test-region"},
				},
			},
		},
		{
			name: "Create project with non-existent region is valid",
			project: v1.Project{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-project",
				},
				Spec: v1.ProjectSpec{
					DefaultRegion:    "acorn-test-dne",
					SupportedRegions: []string{"acorn-test-dne"},
				},
			},
		},
		{
			name:      "Create project with default that is not supported",
			wantError: true,
			project: v1.Project{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-project",
				},
				Spec: v1.ProjectSpec{
					DefaultRegion:    "acorn-test-region",
					SupportedRegions: []string{},
				},
			},
		},
		{
			name: "Create project with supported region that does not exist is valid",
			project: v1.Project{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-project",
				},
				Spec: v1.ProjectSpec{
					DefaultRegion:    "acorn-test-region",
					SupportedRegions: []string{"acorn-test-region", "acorn-test-dne"},
				},
			},
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.project.Name = tt.project.Name + strconv.Itoa(i+1)
			defer func() {
				if err := kclient.Delete(ctx, &tt.project); err != nil && !apierrors.IsNotFound(err) {
					t.Fatal(err)
				}
			}()
			if err := kclient.Create(ctx, &tt.project); !tt.wantError && err != nil {
				t.Fatal(err)
			} else if tt.wantError && err == nil {
				t.Fatal("expected error for test case")
			}

		})
	}
}

func TestProjectUpdateValidation(t *testing.T) {
	helper.StartController(t)

	ctx := helper.GetCTX(t)
	kclient := helper.MustReturn(kclient.Default)

	region := &adminv1.RegionInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-region",
		},
		Spec: adminv1.RegionInstanceSpec{},
	}
	if err := kclient.Create(ctx, region); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := kclient.Delete(context.Background(), region); err != nil && !apierrors.IsNotFound(err) {
			t.Fatal(err)
		}
	}()

	project := &v1.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name: "acorn-test-project",
		},
		Spec: v1.ProjectSpec{
			DefaultRegion: "",
		},
	}
	if err := kclient.Create(ctx, project); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := kclient.Delete(context.Background(), project); err != nil && !apierrors.IsNotFound(err) {
			t.Fatal(err)
		}
	}()

	tests := []struct {
		name             string
		defaultRegion    string
		supportedRegions []string
		wantError        bool
	}{
		{
			name:          "Update project to have default region, no supported regions",
			wantError:     true,
			defaultRegion: "my-region",
		},
		{
			name:             "Update project to have default region and supported region",
			defaultRegion:    "my-region",
			supportedRegions: []string{"my-region"},
		},
		{
			name:             "Update project to have default region and non-existent supported regions",
			defaultRegion:    "my-region",
			supportedRegions: []string{"my-region", "my-other-region"},
		},
		{
			name:             "Remove default region as supported region",
			wantError:        true,
			defaultRegion:    "my-region",
			supportedRegions: []string{"my-other-region"},
		},
		{
			name: "Remove all region information",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			project.Spec.DefaultRegion = tt.defaultRegion
			project.Spec.SupportedRegions = tt.supportedRegions

			if err := kclient.Update(ctx, project); !tt.wantError {
				if err != nil {
					t.Fatal(err)
				}
				if project.Spec.DefaultRegion == "" && len(project.Spec.SupportedRegions) == 0 {
					assert.NotEmpty(t, project.Status.DefaultRegion)
				}
			} else if tt.wantError && err == nil {
				t.Fatal("expected error for test case")
			}
		})
	}
}
