package controller

import (
	"context"

	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/config"
	"github.com/acorn-io/acorn/pkg/system"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (c *Controller) initData(ctx context.Context) error {
	err := c.apply.Ensure(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: system.Namespace,
		},
	}, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: system.ImagesNamespace,
		},
	}, &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: "acorn:system:builder",
		},
		Rules: []rbacv1.PolicyRule{
			{
				Verbs:         []string{"get"},
				APIGroups:     []string{""},
				Resources:     []string{"configmaps"},
				ResourceNames: []string{"acorn-config"},
			},
			{
				Verbs:     []string{"list"},
				APIGroups: []string{""},
				Resources: []string{"nodes"},
			},
			{
				Verbs:     []string{"get", "list"},
				APIGroups: []string{""},
				Resources: []string{"secrets", "namespaces"},
			},
			{
				Verbs:     []string{"get"},
				APIGroups: []string{""},
				Resources: []string{"services"},
			},
			{
				Verbs:     []string{"get", "create", "patch"},
				APIGroups: []string{v1.SchemeGroupVersion.Group},
				Resources: []string{"imageinstances"},
			},
			{
				Verbs:     []string{"get", "list", "watch"},
				APIGroups: []string{v1.SchemeGroupVersion.Group},
				Resources: []string{"acornimagebuildinstances"},
			},
			{
				Verbs:     []string{"update"},
				APIGroups: []string{v1.SchemeGroupVersion.Group},
				Resources: []string{"acornimagebuildinstances/status"},
			},
		},
	}, &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: "acorn:system:builder",
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      "acorn-builder",
				Namespace: system.ImagesNamespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			Name: "acorn:system:builder",
			Kind: "ClusterRole",
		},
	}, &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "acorn-builder",
			Namespace: system.ImagesNamespace,
		},
	})
	if err != nil {
		return err
	}
	return config.Init(ctx, c.client)
}
