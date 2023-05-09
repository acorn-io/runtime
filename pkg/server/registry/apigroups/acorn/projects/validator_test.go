package projects

import (
	"context"
	"testing"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/scheme"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestProjectCreateValidation(t *testing.T) {
	validator := &Validator{DefaultRegion: apiv1.LocalRegion}

	tests := []struct {
		name      string
		project   apiv1.Project
		wantError bool
	}{
		{
			name: "Create project with no region",
			project: apiv1.Project{
				Spec: apiv1.ProjectSpec{
					DefaultRegion: "",
				},
			},
		},
		{
			name: "Create project with existing region",
			project: apiv1.Project{
				Spec: apiv1.ProjectSpec{
					DefaultRegion:    "acorn-test-region",
					SupportedRegions: []string{"acorn-test-region"},
				},
			},
		},
		{
			name: "Create project with non-existent region is valid",
			project: apiv1.Project{
				Spec: apiv1.ProjectSpec{
					DefaultRegion:    "acorn-test-dne",
					SupportedRegions: []string{"acorn-test-dne"},
				},
			},
		},
		{
			name:      "Create project with default that is not supported",
			wantError: true,
			project: apiv1.Project{
				Spec: apiv1.ProjectSpec{
					DefaultRegion:    "acorn-test-region",
					SupportedRegions: []string{},
				},
			},
		},
		{
			name: "Create project with supported region that does not exist is valid",
			project: apiv1.Project{
				Spec: apiv1.ProjectSpec{
					DefaultRegion:    "acorn-test-region",
					SupportedRegions: []string{"acorn-test-region", "acorn-test-dne"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validator.Validate(context.Background(), &tt.project); !tt.wantError {
				if err != nil {
					t.Fatal(err)
				}

				// Ensure that the default region is set on status if no default region was given
				if tt.project.Spec.DefaultRegion == "" {
					assert.NotEmpty(t, tt.project.Status.DefaultRegion, "default region should be set")
				}
			} else if tt.wantError && err == nil {
				t.Fatal("expected error for test case")
			}
		})
	}
}

func TestProjectUpdateValidation(t *testing.T) {
	validator := &Validator{DefaultRegion: apiv1.LocalRegion}
	tests := []struct {
		name                   string
		newProject, oldProject apiv1.Project
		client                 kclient.Client
		wantError              bool
	}{
		{
			name:      "Update project to have default region, no supported regions",
			wantError: true,
			newProject: apiv1.Project{
				Spec: apiv1.ProjectSpec{
					DefaultRegion: "my-region",
				},
			},
		},
		{
			name: "Update project to have default region and supported region",
			oldProject: apiv1.Project{
				Status: apiv1.ProjectStatus{
					DefaultRegion: "my-region",
				},
			},
			newProject: apiv1.Project{
				Spec: apiv1.ProjectSpec{
					DefaultRegion:    "my-region",
					SupportedRegions: []string{"my-region"},
				},
			},
		},
		{
			name: "Update project to have default region and non-existent supported regions",
			oldProject: apiv1.Project{
				Status: apiv1.ProjectStatus{
					DefaultRegion: "my-region",
				},
			},
			newProject: apiv1.Project{
				Spec: apiv1.ProjectSpec{
					DefaultRegion:    "my-region",
					SupportedRegions: []string{"my-region", "dne-region"},
				},
			},
		},
		{
			name:      "Remove default region as supported region",
			wantError: true,
			newProject: apiv1.Project{
				Spec: apiv1.ProjectSpec{
					DefaultRegion:    "my-region",
					SupportedRegions: []string{"my-other-region"},
				},
			},
		},
		{
			name: "Update project remove a supported region, no apps",
			oldProject: apiv1.Project{
				Spec: apiv1.ProjectSpec{
					DefaultRegion:    "my-region",
					SupportedRegions: []string{"my-region", "my-other-region"},
				},
			},
			newProject: apiv1.Project{
				Spec: apiv1.ProjectSpec{
					DefaultRegion:    "my-region",
					SupportedRegions: []string{"my-region"},
				},
			},
			client: fake.NewClientBuilder().WithScheme(scheme.Scheme).Build(),
		},
		{
			name: "Update project remove a supported region, no apps in project",
			oldProject: apiv1.Project{
				Spec: apiv1.ProjectSpec{
					DefaultRegion:    "my-region",
					SupportedRegions: []string{"my-region", "my-other-region"},
				},
			},
			newProject: apiv1.Project{
				Spec: apiv1.ProjectSpec{
					DefaultRegion:    "my-region",
					SupportedRegions: []string{"my-region"},
				},
			},
			client: fake.NewClientBuilder().WithScheme(scheme.Scheme).WithLists(
				&apiv1.AppList{
					Items: []apiv1.App{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "my-app",
								Namespace: "other-project",
							},
							Spec: v1.AppInstanceSpec{
								Region: "my-region",
							},
						},
					},
				},
			).Build(),
		},
		{
			name: "Update project remove a supported region, no apps in removed region",
			oldProject: apiv1.Project{
				Spec: apiv1.ProjectSpec{
					DefaultRegion:    "my-region",
					SupportedRegions: []string{"my-region", "my-other-region"},
				},
			},
			newProject: apiv1.Project{
				Spec: apiv1.ProjectSpec{
					DefaultRegion:    "my-region",
					SupportedRegions: []string{"my-region"},
				},
			},
			client: fake.NewClientBuilder().WithScheme(scheme.Scheme).WithLists(
				&apiv1.AppList{
					Items: []apiv1.App{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "my-app",
							},
							Spec: v1.AppInstanceSpec{
								Region: "my-region",
							},
						},
					},
				},
			).Build(),
		},
		{
			name:      "Update project remove a supported region with apps in removed region",
			wantError: true,
			oldProject: apiv1.Project{
				Spec: apiv1.ProjectSpec{
					DefaultRegion:    "my-region",
					SupportedRegions: []string{"my-region", "my-other-region"},
				},
			},
			newProject: apiv1.Project{
				Spec: apiv1.ProjectSpec{
					DefaultRegion:    "my-region",
					SupportedRegions: []string{"my-region"},
				},
			},
			client: fake.NewClientBuilder().WithScheme(scheme.Scheme).WithLists(
				&apiv1.AppList{
					Items: []apiv1.App{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "my-app",
							},
							Spec: v1.AppInstanceSpec{
								Region: "my-other-region",
							},
						},
					},
				},
			).Build(),
		},
		{
			name:      "Update project remove a supported region with apps defaulted to removed region",
			wantError: true,
			oldProject: apiv1.Project{
				Spec: apiv1.ProjectSpec{
					DefaultRegion:    "my-region",
					SupportedRegions: []string{"my-region", "my-other-region"},
				},
			},
			newProject: apiv1.Project{
				Spec: apiv1.ProjectSpec{
					DefaultRegion:    "my-region",
					SupportedRegions: []string{"my-region"},
				},
			},
			client: fake.NewClientBuilder().WithScheme(scheme.Scheme).WithLists(
				&apiv1.AppList{
					Items: []apiv1.App{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "my-app",
							},
							Status: v1.AppInstanceStatus{
								Defaults: v1.Defaults{
									Region: "my-other-region",
								},
							},
						},
					},
				},
			).Build(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator.Client = tt.client
			err := validator.ValidateUpdate(context.Background(), &tt.newProject, &tt.oldProject)
			if !tt.wantError {
				if err != nil {
					t.Fatal(err)
				}

				// Ensure that the default region is set
				if tt.newProject.Spec.DefaultRegion == "" && len(tt.newProject.Spec.SupportedRegions) == 0 {
					assert.NotEmpty(t, tt.newProject.Status.DefaultRegion)
				}
			} else if tt.wantError && err == nil {
				t.Fatal("expected error for test case")
			}
		})
	}
}
