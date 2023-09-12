package imageallowrules

import (
	"context"

	adminv1 "github.com/acorn-io/runtime/pkg/apis/admin.acorn.io/v1"
	internalv1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	internaladminv1 "github.com/acorn-io/runtime/pkg/apis/internal.admin.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/images"
	"github.com/acorn-io/runtime/pkg/roles"
	rbacv1 "k8s.io/api/rbac/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetAuthorizedRoles(ctx context.Context, c client.Reader, namespace, imageName, digest string) ([]rbacv1.ClusterRole, error) {
	var authorizedRoles []rbacv1.ClusterRole

	iras := &adminv1.ImageRoleAuthorizationList{}
	if err := c.List(ctx, iras, &client.ListOptions{Namespace: namespace}); err != nil {
		return nil, err
	}

	ciras := &adminv1.ClusterImageRoleAuthorizationList{}
	if err := c.List(ctx, ciras); err != nil {
		return nil, err
	}

	for _, cira := range ciras.Items {
		iras.Items = append(iras.Items, adminv1.ImageRoleAuthorization{
			ObjectMeta: cira.ObjectMeta,
			Images:     cira.Images,
			Signatures: cira.Signatures,
			RoleRefs:   cira.RoleRefs,
		})
	}

	if len(iras.Items) == 0 {
		return nil, nil
	}

	remoteOpts, err := images.GetAuthenticationRemoteOptions(ctx, c, namespace)
	if err != nil {
		return nil, err
	}

	for _, ira := range iras.Items {
		// TODO: We should come up with a more generic variation of the function below, at last to clarify the log messages - i.e. do not misuse IARs, but rather create a generic type
		// that works for both IARs and IRAs (and IRBs in Manager)
		if err := CheckImageAgainstRules(ctx, c, namespace, imageName, "", digest, []internalv1.ImageAllowRuleInstance{iarFromIRA(ira)}, remoteOpts...); err != nil {
			if _, ok := err.(*ErrImageNotAllowed); !ok {
				return nil, err
			}
			continue
		}
		authorizedRoles = append(authorizedRoles, rolesFromRoleRefs(ira.RoleRefs)...)
	}

	return authorizedRoles, nil
}

func iarFromIRA(ira adminv1.ImageRoleAuthorization) internalv1.ImageAllowRuleInstance {
	return internalv1.ImageAllowRuleInstance{
		ObjectMeta: ira.ObjectMeta,
		Images:     ira.Images,
		Signatures: ira.Signatures,
	}
}

func rolesFromRoleRefs(roleRefs []internaladminv1.RoleRef) []rbacv1.ClusterRole {
	clusterRoles := roles.ClusterRoles()
	var roles []rbacv1.ClusterRole
	visited := map[string]bool{}
	for _, roleRef := range roleRefs {
		if _, ok := visited[roleRef.RoleName]; ok {
			continue
		}
		for _, clusterRole := range clusterRoles {
			if roleRef.RoleName == clusterRole.Name {
				roles = append(roles, clusterRole)
			}
		}
	}
	return roles
}

func PermissionsFromClusterRole(serviceName string, role rbacv1.ClusterRole) v1.Permissions {
	perms := v1.Permissions{
		ServiceName: serviceName,
	}
	for _, rule := range role.Rules {
		perms.Rules = append(perms.Rules, v1.PolicyRule{PolicyRule: rule})
	}
	return perms
}

func Authorized(imageName, namespace string, requestedPerms []v1.Permissions, authorizedRoles []rbacv1.ClusterRole) ([]v1.Permissions, bool) {
	var authorizedPermissions []v1.Permissions

	for _, clusterrole := range authorizedRoles {
		authorizedPermissions = append(authorizedPermissions, PermissionsFromClusterRole(imageName, clusterrole))
	}

	return v1.GrantsAll(namespace, requestedPerms, authorizedPermissions)
}
