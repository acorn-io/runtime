package computeclass

import (
	"context"
	"testing"

	"github.com/acorn-io/runtime/integration/helper"
	adminapiv1 "github.com/acorn-io/runtime/pkg/apis/admin.acorn.io/v1"
	adminv1 "github.com/acorn-io/runtime/pkg/apis/internal.admin.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/client"
	kclient "github.com/acorn-io/runtime/pkg/k8sclient"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestCreatingComputeClasses(t *testing.T) {
	helper.StartController(t)
	cfg := helper.StartAPI(t)
	project := helper.TempProject(t, helper.MustReturn(kclient.Default))
	kclient := helper.MustReturn(kclient.Default)
	c, err := client.New(cfg, "", project.Name)
	if err != nil {
		t.Fatal(err)
	}

	ctx := helper.GetCTX(t)

	checks := []struct {
		name              string
		memory            adminv1.ComputeClassMemory
		resources         corev1.ResourceRequirements
		cpuScaler         float64
		priorityClassName string
		runtimeClassName  string
		fail              bool
	}{
		{
			name: "valid-only-max",
			memory: adminv1.ComputeClassMemory{
				Max: "512Mi",
			},
			fail: false,
		},
		{
			name: "valid-custom-resources",
			memory: adminv1.ComputeClassMemory{
				Max: "512Mi",
			},
			resources: corev1.ResourceRequirements{
				Limits: corev1.ResourceList{
					"mygpu/nvidia": resource.MustParse("1"),
				},
				Requests: corev1.ResourceList{
					"mygpu/nvidia": resource.MustParse("1"),
				},
			},
			fail: false,
		},
		{
			name: "invalid-custom-resources-limits",
			resources: corev1.ResourceRequirements{
				Limits: corev1.ResourceList{
					"cpu": resource.MustParse("1"),
				},
			},
			fail: true,
		},
		{
			name: "invalid-custom-resources-requests",
			resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					"memory": resource.MustParse("1"),
				},
			},
			fail: true,
		},
		{
			name: "valid-only-min",
			memory: adminv1.ComputeClassMemory{
				Min: "512Mi",
			},
			fail: false,
		},
		{
			name: "valid-only-default",
			memory: adminv1.ComputeClassMemory{
				Default: "512Mi",
			},
			fail: false,
		},
		{
			name:              "valid-only-priority-class",
			priorityClassName: "system-cluster-critical",
			fail:              false,
		},
		{
			name:             "valid-only-runtime-class",
			runtimeClassName: "alt-runtime",
			fail:             false,
		},
		{
			name:      "valid-values",
			cpuScaler: 0.25,
			memory: adminv1.ComputeClassMemory{
				Default: "1Gi",
				Values:  []string{"1Gi", "2Gi"},
			},
		},
		{
			name: "valid-empty",
		},
		{
			name: "invalid-memory-default",
			memory: adminv1.ComputeClassMemory{
				Default: "invalid",
			},
			fail: true,
		},
		{
			name: "invalid-memory-min",
			memory: adminv1.ComputeClassMemory{
				Min: "invalid",
			},
			fail: true,
		},
		{
			name: "invalid-memory-max",
			memory: adminv1.ComputeClassMemory{
				Max: "invalid",
			},
			fail: true,
		},
		{
			name: "invalid-memory-values",
			memory: adminv1.ComputeClassMemory{
				Values: []string{"invalid"},
			},
			fail: true,
		},
		{
			name: "invalid-default-less-than-min",
			memory: adminv1.ComputeClassMemory{
				Default: "128Mi",
				Min:     "512Mi",
			},
			fail: true,
		},
		{
			name: "invalid-default-greater-than-max",
			memory: adminv1.ComputeClassMemory{
				Default: "1Gi",
				Max:     "512Mi",
			},
			fail: true,
		},
		{
			name: "invalid-min-max-swapped",
			memory: adminv1.ComputeClassMemory{
				Min: "1Gi",
				Max: "512Mi",
			},
			fail: true,
		},
		{
			name: "invalid-default-for-values",
			memory: adminv1.ComputeClassMemory{
				Default: "128Mi",
				Values:  []string{"512Mi"},
			},
			fail: true,
		},
		{
			name: "invalid-min-max-set-with-values",
			memory: adminv1.ComputeClassMemory{
				Min:    "512Mi",
				Max:    "4Gi",
				Values: []string{"2Gi", "3Gi"},
			},
			fail: true,
		},
		{
			name:      "valid-values-with-scaler",
			cpuScaler: 0.25,
			memory: adminv1.ComputeClassMemory{
				RequestScaler: 0.1,
				Default:       "1Gi",
				Values:        []string{"1Gi", "2Gi"},
			},
		},
		{
			name: "valid-scaler-upper-bound",
			memory: adminv1.ComputeClassMemory{
				RequestScaler: 1.0,
			},
		},
		{
			name: "valid-scaler-lower-bound",
			memory: adminv1.ComputeClassMemory{
				RequestScaler: 0,
			},
		},
		{
			name: "invalid-scaler-negative",
			memory: adminv1.ComputeClassMemory{
				RequestScaler: -0.1,
			},
			fail: true,
		},
		{
			name: "invalid-scaler-too-large",
			memory: adminv1.ComputeClassMemory{
				RequestScaler: 1.1,
			},
			fail: true,
		},
	}

	for _, tt := range checks {
		t.Run(tt.name, func(t *testing.T) {
			// Create a non-instanced ComputeClass to trigger Mink validation
			computeClass := adminapiv1.ProjectComputeClass{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "acorn-test-custom",
					Namespace:    c.GetNamespace(),
				},
				CPUScaler:         tt.cpuScaler,
				Memory:            tt.memory,
				Resources:         &tt.resources,
				PriorityClassName: tt.priorityClassName,
				RuntimeClassName:  tt.runtimeClassName,
			}

			// TODO - dry run
			err = kclient.Create(ctx, &computeClass)
			if err != nil && !tt.fail {
				t.Fatal("did not expect creation to fail:", err)
			} else if err == nil {
				if err := kclient.Delete(context.Background(), &computeClass); err != nil && !apierrors.IsNotFound(err) {
					t.Fatal("failed to cleanup test:", err)
				}
				if tt.fail {
					t.Fatal("expected an error to occur when creating an invalid ComputeClass but did not receive one")
				}
			}
		})
	}
}
