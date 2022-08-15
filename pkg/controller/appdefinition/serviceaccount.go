package appdefinition

import (
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/labels"
	"github.com/rancher/wrangler/pkg/name"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func addServiceAccount(appInstance *v1.AppInstance) []kclient.Object {
	return toServiceAccount(appInstance)
}

func toRules(rules []v1.PolicyRule) (result []rbacv1.PolicyRule) {
	for _, rule := range rules {
		result = append(result, (rbacv1.PolicyRule)(rule))
	}
	return
}
func toServiceAccount(appInstance *v1.AppInstance) (result []kclient.Object) {
	if !appInstance.Spec.Permissions.HasRules() {
		return nil
	}

	result = append(result, &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "acorn",
			Namespace: appInstance.Status.Namespace,
			Labels:    labels.Managed(appInstance),
		},
	})

	if len(appInstance.Spec.Permissions.ClusterRules) > 0 {
		name := name.SafeConcatName("acorn", appInstance.Name, appInstance.Namespace, appInstance.ShortID())
		result = append(result, &rbacv1.ClusterRole{
			ObjectMeta: metav1.ObjectMeta{
				Name:   name,
				Labels: labels.Managed(appInstance),
			},
			Rules: toRules(appInstance.Spec.Permissions.ClusterRules),
		}, &rbacv1.ClusterRoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name:   name,
				Labels: labels.Managed(appInstance),
			},
			Subjects: []rbacv1.Subject{
				{
					Kind:      "ServiceAccount",
					Name:      "acorn",
					Namespace: appInstance.Status.Namespace,
				},
			},
			RoleRef: rbacv1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "ClusterRole",
				Name:     name,
			},
		})
	}

	if len(appInstance.Spec.Permissions.Rules) > 0 {
		result = append(result, &rbacv1.Role{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "acorn",
				Namespace: appInstance.Status.Namespace,
				Labels:    labels.Managed(appInstance),
			},
			Rules: toRules(appInstance.Spec.Permissions.Rules),
		}, &rbacv1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "acorn",
				Namespace: appInstance.Status.Namespace,
				Labels:    labels.Managed(appInstance),
			},
			Subjects: []rbacv1.Subject{
				{
					Kind:      "ServiceAccount",
					Name:      "acorn",
					Namespace: appInstance.Status.Namespace,
				},
			},
			RoleRef: rbacv1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "Role",
				Name:     "acorn",
			},
		})
	}

	return result
}
