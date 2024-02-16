package v1_test

import (
	"context"
	"testing"

	internalv1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	internaladminv1 "github.com/acorn-io/runtime/pkg/apis/internal.admin.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/scheme"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestGetDefaultComputeClass(t *testing.T) {
	ctx := context.Background()

	type args struct {
		namespace string
		client    kclient.Client
	}
	type expected struct {
		computeClassName string
		error            bool
	}
	for _, tt := range []struct {
		name     string
		args     args
		expected expected
	}{
		{
			name: "No defaults",
			args: args{
				namespace: "pcc-project",
				client: fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(
					&internalv1.ProjectInstance{
						ObjectMeta: metav1.ObjectMeta{
							Name: "pcc-project",
						},
					},
					&internaladminv1.ProjectComputeClassInstance{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "project-compute-class",
							Namespace: "pcc-project",
						},
					},
					&internaladminv1.ClusterComputeClassInstance{
						ObjectMeta: metav1.ObjectMeta{
							Name: "cluster-compute-class",
						},
					},
				).Build(),
			},
		},
		{
			name: "Default cluster compute class",
			args: args{
				namespace: "pcc-project",
				client: fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(
					&internalv1.ProjectInstance{
						ObjectMeta: metav1.ObjectMeta{
							Name: "pcc-project",
						},
					},
					&internaladminv1.ProjectComputeClassInstance{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "other-project-compute-class",
							Namespace: "pcc-project",
						},
					},
					&internaladminv1.ProjectComputeClassInstance{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "project-compute-class",
							Namespace: "pcc-project",
						},
					},
					&internaladminv1.ClusterComputeClassInstance{
						ObjectMeta: metav1.ObjectMeta{
							Name: "other-cluster-compute-class",
						},
					},
					&internaladminv1.ClusterComputeClassInstance{
						ObjectMeta: metav1.ObjectMeta{
							Name: "cluster-compute-class",
						},
						Default: true,
					},
				).Build(),
			},
			expected: expected{
				computeClassName: "cluster-compute-class",
			},
		},
		{
			name: "Project compute classes take precedence over cluster compute classes",
			args: args{
				namespace: "pcc-project",
				client: fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(
					&internalv1.ProjectInstance{
						ObjectMeta: metav1.ObjectMeta{
							Name: "pcc-project",
						},
					},
					&internaladminv1.ProjectComputeClassInstance{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "other-project-compute-class",
							Namespace: "pcc-project",
						},
					},
					&internaladminv1.ProjectComputeClassInstance{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "project-compute-class",
							Namespace: "pcc-project",
						},
						Default: true,
					},
					&internaladminv1.ClusterComputeClassInstance{
						ObjectMeta: metav1.ObjectMeta{
							Name: "other-cluster-compute-class",
						},
					},
					&internaladminv1.ClusterComputeClassInstance{
						ObjectMeta: metav1.ObjectMeta{
							Name: "cluster-compute-class",
						},
						Default: true,
					},
				).Build(),
			},
			expected: expected{
				computeClassName: "project-compute-class",
			},
		},
		{
			name: "Project specified compute class takes precedence over default project compute class",
			args: args{
				namespace: "pcc-project",
				client: fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(
					&internalv1.ProjectInstance{
						ObjectMeta: metav1.ObjectMeta{
							Name: "pcc-project",
						},
						Status: internalv1.ProjectInstanceStatus{
							DefaultComputeClass: "project-specified-default",
						},
					},
					&internaladminv1.ProjectComputeClassInstance{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "self-specified-default",
							Namespace: "pcc-project",
						},
						Default: true,
					},
					&internaladminv1.ClusterComputeClassInstance{
						ObjectMeta: metav1.ObjectMeta{
							Name: "project-specified-default",
						},
					},
				).Build(),
			},
			expected: expected{
				computeClassName: "project-specified-default",
			},
		},
		{
			name: "Project specified compute class not found",
			args: args{
				namespace: "pcc-project",
				client: fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(
					&internalv1.ProjectInstance{
						ObjectMeta: metav1.ObjectMeta{
							Name: "pcc-project",
						},
						Status: internalv1.ProjectInstanceStatus{
							DefaultComputeClass: "project-specified-default",
						},
					},
					&internaladminv1.ClusterComputeClassInstance{
						ObjectMeta: metav1.ObjectMeta{
							Name: "self-specified-default",
						},
						Default: true,
					},
				).Build(),
			},
			expected: expected{
				error: true,
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			kc := tt.args.client
			if kc == nil {
				kc = fake.NewClientBuilder().WithScheme(scheme.Scheme).Build()
			}

			actualComputeClassName, err := internaladminv1.GetDefaultComputeClassName(ctx, kc, tt.args.namespace)
			if tt.expected.error {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			require.Equal(t, tt.expected.computeClassName, actualComputeClassName)
		})
	}
}
