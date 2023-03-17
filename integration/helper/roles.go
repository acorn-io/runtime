package helper

import (
	"context"
	"testing"

	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NamespaceClusterRoleAndBinding(t *testing.T, ctx context.Context, k8sclient client.Client, namespace string) error {
	t.Helper()

	cr := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-role-" + namespace,
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups:     []string{""},
				Resources:     []string{"namespaces"},
				Verbs:         []string{"get", "list"},
				ResourceNames: []string{namespace},
			},
		},
	}
	if err := k8sclient.Create(ctx, cr); err != nil {
		return err
	}
	t.Cleanup(func() {
		if err := k8sclient.Delete(ctx, cr); err != nil && !apierrors.IsNotFound(err) {
			t.Fatal(err)
		}
	})

	crb := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-role-binding-" + namespace,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind: "User",
				Name: "system:anonymous",
			},
		},
		RoleRef: rbacv1.RoleRef{
			Kind:     "ClusterRole",
			Name:     cr.Name,
			APIGroup: "rbac.authorization.k8s.io",
		},
	}
	if err := k8sclient.Create(ctx, crb); err != nil {
		return err
	}
	t.Cleanup(func() {
		if err := k8sclient.Delete(ctx, crb); err != nil && !apierrors.IsNotFound(err) {
			t.Fatal(err)
		}
	})

	return nil
}
