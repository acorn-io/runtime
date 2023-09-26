package roles

import (
	admin_acorn_io "github.com/acorn-io/runtime/pkg/apis/admin.acorn.io"
	api_acorn_io "github.com/acorn-io/runtime/pkg/apis/api.acorn.io"
	"github.com/acorn-io/runtime/pkg/awspermissions"
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

	// AWS
	AWSAdmin = "acorn:aws:admin"
)

var (
	awsRoles = map[string][]rbacv1.PolicyRule{
		AWSAdmin: {
			{
				APIGroups: []string{awspermissions.AWSAPIGroup, awspermissions.AWSRoleAPIGroup},
				Verbs:     []string{"*"},
				Resources: []string{"*"},
			},
		},
	}

	clusterRoles = map[string][]rbacv1.PolicyRule{
		ClusterView: {
			{
				Verbs: []string{"get", "list"},
				Resources: []string{
					"projects",
					"regions",
				},
			},
		},
		ClusterEdit: {
			{
				Verbs: []string{"create", "update", "delete"},
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
					"apps/info",
					"apps/icon",
					"acornimagebuilds",
					"builders",
					"devsessions",
					"images",
					"volumes",
					"containerreplicas",
					"credentials",
					"secrets",
					"services",
					"events",
				},
			},
			{
				Verbs: []string{"get", "list"},
				Resources: []string{
					"volumeclasses",
					"computeclasses",
					"regions",
					"imageallowrules",
					"imagerolauthorizations",
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
					"devsessions",
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
					"apps/ignorecleanup",
					"events",
				},
			},
			{
				Verbs: []string{"delete"},
				Resources: []string{
					"services",
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
				Verbs: []string{"create", "update", "delete", "patch", "get", "list", "watch"},
				Resources: []string{
					"projectvolumeclasses",
					"clustervolumeclasses",
					"projectcomputeclasses",
					"clustercomputeclasses",
					"imageroleauthorizations",
					"clusterimageroleauthorizations",
				},
				APIGroups: []string{admin_acorn_io.Group},
			},
			{
				Verbs: []string{"create", "update", "delete", "patch"},
				Resources: []string{
					"imageallowrules",
				},
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
			Rules: concat(View, ViewLogs, Edit, Build, Admin),
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
		// AWS Roles
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: AWSAdmin,
			},
			Rules: awsRoles[AWSAdmin],
		},
	})
}
