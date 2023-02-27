package roles

import (
	admin_acorn_io "github.com/acorn-io/acorn/pkg/apis/admin.acorn.io"
	api_acorn_io "github.com/acorn-io/acorn/pkg/apis/api.acorn.io"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	Admin       = "acorn:project:admin"
	View        = "acorn:project:view"
	ViewLogs    = "acorn:project:view-logs"
	Edit        = "acorn:project:edit"
	Build       = "acorn:project:build"
	ClusterView = "acorn:cluster:view"
	ClusterEdit = "acorn:cluster:edit"
)

var (
	clusterRoles = map[string][]rbacv1.PolicyRule{
		ClusterView: {
			{
				Verbs: []string{"get", "list"},
				Resources: []string{
					"projects",
				},
			},
		},
		ClusterEdit: {
			{
				Verbs: []string{"create", "delete"},
				Resources: []string{
					"projects",
				},
			},
		},
	}
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
				Verbs: []string{"get", "list"},
				Resources: []string{
					"volumeclasses",
					"computeclasses",
				},
			},
			{
				Verbs: []string{"get", "create"},
				Resources: []string{
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
		ViewLogs: {
			{
				Verbs: []string{"get"},
				Resources: []string{
					"apps/log",
				},
			},
		},
		Edit: {
			{
				Verbs: []string{"create", "update", "delete", "patch"},
				Resources: []string{
					"apps",
					"credentials",
					"secrets",
				},
			},
			{
				Verbs: []string{"update", "delete", "patch"},
				Resources: []string{
					"images",
				},
			},
			{
				Verbs: []string{"create"},
				Resources: []string{
					"images/tag",
					"apps/confirmupgrade",
					"apps/pullimage",
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
					"secrets/reveal",
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
		Admin: {
			{
				Verbs: []string{"*"},
				Resources: []string{
					"projectvolumeclasses",
					"clustervolumeclasses",
					"projectcomputeclasses",
					"clustercomputeclasses",
				},
				APIGroups: []string{admin_acorn_io.Group},
			},
		},
	}
)

func addAPIGroup(roles []rbacv1.ClusterRole) []rbacv1.ClusterRole {
	for i := range roles {
		for j := range roles[i].Rules {
			if len(roles[i].Rules[j].APIGroups) == 0 {
				roles[i].Rules[j].APIGroups = []string{api_acorn_io.Group}
			}
		}
	}
	return roles
}

func clusterRolesConcat(roles ...string) []rbacv1.PolicyRule {
	var result []rbacv1.PolicyRule
	for _, role := range roles {
		result = append(result, clusterRoles[role]...)
	}
	return result
}

func concat(roles ...string) []rbacv1.PolicyRule {
	var result []rbacv1.PolicyRule
	for _, role := range roles {
		result = append(result, projectRoles[role]...)
	}
	return result
}

func ClusterRoles() []rbacv1.ClusterRole {
	return addAPIGroup([]rbacv1.ClusterRole{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: ClusterView,
			},
			Rules: clusterRolesConcat(ClusterView),
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: ClusterEdit,
			},
			Rules: clusterRolesConcat(ClusterView, ClusterEdit),
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: Admin,
			},
			Rules: concat(View, ViewLogs, Edit, Build),
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: View,
			},
			Rules: projectRoles[View],
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: ViewLogs,
			},
			Rules: concat(View, ViewLogs),
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: Edit,
			},
			Rules: concat(View, ViewLogs, Edit),
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: Build,
			},
			Rules: projectRoles[Build],
		},
	})
}
