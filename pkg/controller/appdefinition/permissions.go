package appdefinition

import (
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/labels"
	"github.com/acorn-io/baaah/pkg/typed"
	"github.com/rancher/wrangler/pkg/name"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func toRulesFromClusterRules(rules []v1.ClusterPolicyRule) (result []rbacv1.PolicyRule) {
	for _, rule := range rules {
		result = append(result, rule.PolicyRule)
	}
	return
}

func toRules(rules []v1.PolicyRule) (result []rbacv1.PolicyRule) {
	for _, rule := range rules {
		result = append(result, (rbacv1.PolicyRule)(rule))
	}
	return
}

func toClusterPermissions(permissions v1.Permissions, labelMap, annotations map[string]string, appInstance *v1.AppInstance) (result []kclient.Object) {
	byNamespace := map[string][]v1.ClusterPolicyRule{}

	for _, clusterRule := range permissions.ClusterRules {
		if len(clusterRule.Namespaces) == 0 {
			byNamespace[""] = append(byNamespace[""], clusterRule)
		} else {
			for _, namespace := range clusterRule.Namespaces {
				byNamespace[namespace] = append(byNamespace[namespace], clusterRule)
			}
		}
	}

	for _, entry := range typed.Sorted(byNamespace) {
		namespace := entry.Key
		rules := entry.Value
		if namespace == "" {
			name := name.SafeConcatName(permissions.ServiceName, appInstance.Name, appInstance.Namespace, appInstance.ShortID())
			result = append(result, &rbacv1.ClusterRole{
				ObjectMeta: metav1.ObjectMeta{
					Name:        name,
					Labels:      labels.Merge(labels.Managed(appInstance), labelMap),
					Annotations: annotations,
				},
				Rules: toRulesFromClusterRules(rules),
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
		} else {
			name := name.SafeConcatName(permissions.ServiceName, appInstance.Name, appInstance.Namespace, appInstance.ShortID(), namespace)
			result = append(result, toRoleAndRoleBinding(name, namespace, permissions.ServiceName, appInstance.Status.Namespace,
				toRulesFromClusterRules(rules), labelMap, annotations, appInstance)...)
		}
	}

	return
}

func toRoleAndRoleBinding(roleName, roleNamespace, serviceAccountName, serviceAccountNamespace string, rules []rbacv1.PolicyRule, labelMap, annotations map[string]string, appInstance *v1.AppInstance) (result []kclient.Object) {
	result = append(result, &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:        roleName,
			Namespace:   roleNamespace,
			Labels:      labels.Merge(labels.Managed(appInstance), labelMap),
			Annotations: annotations,
		},
		Rules: rules,
	}, &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:        roleName,
			Namespace:   roleNamespace,
			Labels:      labels.Merge(labels.Managed(appInstance), labelMap),
			Annotations: annotations,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      serviceAccountName,
				Namespace: serviceAccountNamespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     roleName,
		},
	})

	return
}

func toPermissions(permissions v1.Permissions, labelMap, annotations map[string]string, appInstance *v1.AppInstance) (result []kclient.Object) {
	if len(permissions.ClusterRules) > 0 {
		result = append(result, toClusterPermissions(permissions, labelMap, annotations, appInstance)...)
	}
	if len(permissions.Rules) > 0 {
		result = append(result, toRoleAndRoleBinding(permissions.ServiceName, appInstance.Status.Namespace, permissions.ServiceName, appInstance.Status.Namespace,
			toRules(permissions.Rules), labelMap, annotations, appInstance)...)
	}
	return result
}
