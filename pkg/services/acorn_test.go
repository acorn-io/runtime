package services

import (
	"testing"

	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/hexops/autogold/v2"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func Test_filterForPermissionsAndAssignStatus(t *testing.T) {
	app := &v1.AppInstance{
		Spec: v1.AppInstanceSpec{
			GrantedPermissions: []v1.Permissions{
				{
					ServiceName: "allow",
					Rules: []v1.PolicyRule{
						{
							PolicyRule: rbacv1.PolicyRule{
								Verbs:     []string{"*"},
								APIGroups: []string{"allow"},
								Resources: []string{"*"},
							},
						},
					},
				},
			},
		},
	}
	serviceAllow := &v1.ServiceInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "allow",
			Namespace: "namespace",
		},
		Spec: v1.ServiceInstanceSpec{
			Consumer: &v1.ServiceConsumer{
				Permissions: &v1.Permissions{
					Rules: []v1.PolicyRule{
						{
							PolicyRule: rbacv1.PolicyRule{
								Verbs:     []string{"*"},
								APIGroups: []string{"allow"},
								Resources: []string{"*"},
							},
						},
					},
				},
			},
		},
	}
	serviceReject := &v1.ServiceInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "reject",
			Namespace: "namespace",
		},
		Spec: v1.ServiceInstanceSpec{
			Consumer: &v1.ServiceConsumer{
				Permissions: &v1.Permissions{
					Rules: []v1.PolicyRule{
						{
							PolicyRule: rbacv1.PolicyRule{
								Verbs:     []string{"*"},
								APIGroups: []string{"*"},
								Resources: []string{"*"},
							},
						},
					},
				},
			},
		},
	}

	allowed := filterForPermissionsAndAssignStatus(app, []kclient.Object{serviceAllow, serviceReject})

	autogold.Expect(&v1.AppInstance{
		Spec: v1.AppInstanceSpec{
			GrantedPermissions: []v1.Permissions{{
				ServiceName: "allow",
				Rules: []v1.PolicyRule{{
					PolicyRule: rbacv1.PolicyRule{
						Verbs: []string{
							"*",
						},
						APIGroups: []string{"allow"},
						Resources: []string{"*"},
					},
				}},
			}},
		},
		Status: v1.AppInstanceStatus{
			EmbeddedAppStatus: v1.EmbeddedAppStatus{
				AppStatus: v1.AppStatus{
					Services: map[string]v1.ServiceStatus{
						"reject": {
							MissingConsumerPermissions: []v1.Permissions{{
								ServiceName: "reject",
								Rules: []v1.PolicyRule{{PolicyRule: rbacv1.PolicyRule{
									Verbs:     []string{"*"},
									APIGroups: []string{"*"},
									Resources: []string{"*"},
								}}},
							}}}},
				},
			},
		}}).Equal(t, app)

	autogold.Expect([]kclient.Object{
		serviceAllow,
	}).Equal(t, allowed)
}
