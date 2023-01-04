package roles

import (
	api_acorn_io "github.com/acorn-io/acorn/pkg/apis/api.acorn.io"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	Admin = "acorn:project:admin"
	View  = "acorn:project:view"
	Edit  = "acorn:project:edit"
	Build = "acorn:project:build"
)

var (
	projectRoles = map[string][]rbacv1.PolicyRule{
		View: {
			{
				Verbs: []string{"get", "list", "watch"},
				Resources: []string{
					"apps",
					"acornimagebuilds",
					"builders",
					"images",
					"volumes",
					"containerreplicas",
					"credentials",
					"secrets",
				},
			},
			{
				Verbs: []string{"get"},
				Resources: []string{
					"apps/log",
					"images/details",
				},
			},
			{
				Verbs: []string{"list"},
				Resources: []string{
					"infos",
				},
			},
		},
		Edit: {
			{
				Verbs: []string{"create", "update", "delete", "patch"},
				Resources: []string{
					"apps",
					"images",
					"credentials",
					"secrets",
				},
			},
			{
				Verbs: []string{"create"},
				Resources: []string{
					"images/tag",
					"apps/confirmupgrade",
				},
			},
			{
				Verbs: []string{"delete"},
				Resources: []string{
					"volumes",
					"containerreplicas",
				},
			},
			{
				Verbs: []string{"get"},
				Resources: []string{
					"images/push",
					"images/pull",
					"containerreplicas/exec",
					"secrets/expose",
				},
			},
		},
		Build: {
			{
				Verbs: []string{"create", "delete"},
				Resources: []string{
					"builders",
					"acornimagebuilds",
				},
			},
			{
				Verbs: []string{"get"},
				Resources: []string{
					"builders/port",
				},
			},
		},
	}
)

func addAPIGroup(roles []rbacv1.ClusterRole) []rbacv1.ClusterRole {
	for i := range roles {
		for j := range roles[i].Rules {
			roles[i].Rules[j].APIGroups = []string{api_acorn_io.Group}
		}
	}
	return roles
}

func ClusterRoles() []rbacv1.ClusterRole {
	return addAPIGroup([]rbacv1.ClusterRole{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: Admin,
			},
			Rules: append(projectRoles[View], append(projectRoles[Edit], projectRoles[Build]...)...),
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: View,
			},
			Rules: projectRoles[View],
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: Edit,
			},
			Rules: append(projectRoles[View], projectRoles[Edit]...),
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: Build,
			},
			Rules: projectRoles[Build],
		},
	})
}
