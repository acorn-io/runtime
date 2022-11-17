package appdefinition

import (
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/labels"
	"github.com/rancher/wrangler/pkg/name"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func toRules(rules []v1.PolicyRule) (result []rbacv1.PolicyRule) {
	for _, rule := range rules {
		result = append(result, (rbacv1.PolicyRule)(rule))
	}
	return
}

func toPermissions(permissions v1.Permissions, labelMap, annotations map[string]string, appInstance *v1.AppInstance) (result []kclient.Object) {
	if len(permissions.ClusterRules) > 0 {
		name := name.SafeConcatName(permissions.ServiceName, appInstance.Name, appInstance.Namespace, appInstance.ShortID())
		result = append(result, &rbacv1.ClusterRole{
			ObjectMeta: metav1.ObjectMeta{
				Name:        name,
				Labels:      labels.Merge(labels.Managed(appInstance), labelMap),
				Annotations: annotations,
			},
			Rules: toRules(permissions.ClusterRules),
		}, &rbacv1.ClusterRoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name:        name,
				Labels:      labels.Merge(labels.Managed(appInstance), labelMap),
				Annotations: annotations,
			},
			Subjects: []rbacv1.Subject{
				{
					Kind:      "ServiceAccount",
					Name:      permissions.ServiceName,
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

	if len(permissions.Rules) > 0 {
		result = append(result, &rbacv1.Role{
			ObjectMeta: metav1.ObjectMeta{
				Name:        permissions.ServiceName,
				Namespace:   appInstance.Status.Namespace,
				Labels:      labels.Merge(labels.Managed(appInstance), labelMap),
				Annotations: annotations,
			},
			Rules: toRules(permissions.Rules),
		}, &rbacv1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name:        permissions.ServiceName,
				Namespace:   appInstance.Status.Namespace,
				Labels:      labels.Merge(labels.Managed(appInstance), labelMap),
				Annotations: annotations,
			},
			Subjects: []rbacv1.Subject{
				{
					Kind:      "ServiceAccount",
					Name:      permissions.ServiceName,
					Namespace: appInstance.Status.Namespace,
				},
			},
			RoleRef: rbacv1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "Role",
				Name:     permissions.ServiceName,
			},
		})
	}
	return result
}
