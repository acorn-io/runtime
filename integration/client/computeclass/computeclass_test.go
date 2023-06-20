package computeclass

import (
	"context"
	"testing"

	"github.com/acorn-io/acorn/integration/helper"
	adminapiv1 "github.com/acorn-io/acorn/pkg/apis/admin.acorn.io/v1"
	adminv1 "github.com/acorn-io/acorn/pkg/apis/internal.admin.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/client"
	kclient "github.com/acorn-io/acorn/pkg/k8sclient"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestCreatingComputeClasses(t *testing.T) {
	helper.StartController(t)
	cfg := helper.StartAPI(t)
	ns := helper.TempNamespace(t, helper.MustReturn(kclient.Default))
	kclient := helper.MustReturn(kclient.Default)
	c, err := client.New(cfg, "", ns.Name)
	if err != nil {
		t.Fatal(err)
	}

	ctx := helper.GetCTX(t)

	checks := []struct {
		name              string
		memory            adminv1.ComputeClassMemory
		cpuScaler         float64
		priorityClassName string
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
				PriorityClassName: tt.priorityClassName,
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
