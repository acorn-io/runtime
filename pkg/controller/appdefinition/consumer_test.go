package appdefinition

import (
	"context"
	"testing"

	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/scheme"
	"github.com/hexops/autogold/v2"
	"github.com/stretchr/testify/assert"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func Test_augmentContainerWithConsumerInfo(t *testing.T) {
	var c kclient.Client = fake.NewClientBuilder().
		WithScheme(scheme.Scheme).WithObjects(&v1.ServiceInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "svc",
			Namespace: "app-namespace",
		},
		Spec: v1.ServiceInstanceSpec{
			Consumer: &v1.ServiceConsumer{
				Permissions: &v1.Permissions{
					Rules: []v1.PolicyRule{
						{
							PolicyRule: rbacv1.PolicyRule{
								Verbs: []string{"*"},
							},
							Scopes: []string{"project"},
						},
					},
				},
				Files: map[string]v1.File{
					"new": {
						Secret: v1.SecretReference{
							Name: "secretfile",
						},
					},
					"existing": {
						Content: "value",
					},
				},
				Environment: []v1.EnvVar{
					{
						Name: "new",
						Secret: v1.SecretReference{
							Name: "secretenv",
						},
					},
					{
						Name:  "existing",
						Value: "value",
					},
				},
			},
		},
	}).Build()

	result, err := augmentContainerWithConsumerInfo(context.Background(), c, "app-namespace", v1.Container{
		Files: map[string]v1.File{
			"existing": {
				Mode: "existing",
			},
		},
		Environment: []v1.EnvVar{
			{
				Name:  "existing",
				Value: "existing",
			},
		},
		Dependencies: []v1.Dependency{
			{
				TargetName: "svc",
			},
			{
				TargetName: "missing",
			},
		},
		Permissions: &v1.Permissions{
			Rules: []v1.PolicyRule{
				{
					PolicyRule: rbacv1.PolicyRule{
						Verbs: []string{"verb"},
					},
				},
			},
		},
	})

	assert.Nil(t, err)
	autogold.Expect(v1.Container{
		Files: v1.Files{
			"existing": v1.File{Mode: "existing"},
			"new":      v1.File{Secret: v1.SecretReference{Name: "svc.secretfile"}},
		},
		Environment: v1.EnvVars{
			v1.EnvVar{
				Name:  "existing",
				Value: "existing",
			},
			v1.EnvVar{
				Name:   "new",
				Secret: v1.SecretReference{Name: "svc.secretenv"},
			},
		},
		Dependencies: v1.Dependencies{
			v1.Dependency{TargetName: "svc"},
			v1.Dependency{TargetName: "missing"},
		},
		Permissions: &v1.Permissions{Rules: []v1.PolicyRule{
			{PolicyRule: rbacv1.PolicyRule{Verbs: []string{
				"verb",
			}}},
		}},
	}).Equal(t, result)
}
