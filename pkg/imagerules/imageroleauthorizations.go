package imagerules

import (
	"context"

	adminv1 "github.com/acorn-io/runtime/pkg/apis/admin.acorn.io/v1"
	internalv1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	internaladminv1 "github.com/acorn-io/runtime/pkg/apis/internal.admin.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/images"
	"github.com/sirupsen/logrus"
	rbacv1 "k8s.io/api/rbac/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetAuthorizedRoles(ctx context.Context, c client.Reader, namespace, imageName, digest string) ([]v1.Permissions, error) {
	var authorizedPermissions []v1.Permissions

	iras := &adminv1.ImageRoleAuthorizationList{}
	if err := c.List(ctx, iras, &client.ListOptions{Namespace: namespace}); err != nil {
		return nil, err
	}

	ciras := &adminv1.ClusterImageRoleAuthorizationList{}
	if err := c.List(ctx, ciras); err != nil {
		return nil, err
	}

	// Create a single list from both IRAs and CIRAs
	for _, cira := range ciras.Items {
		iras.Items = append(iras.Items, adminv1.ImageRoleAuthorization{
			ObjectMeta:    cira.ObjectMeta,
			ImageSelector: cira.ImageSelector,
			Roles:         cira.Roles,
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
		perms, err := resolveRoleRefs(ctx, c, namespace, imageName, ira.Roles.RoleRefs)
		if err != nil {
			return nil, err
		}
		authorizedPermissions = append(authorizedPermissions, perms...)
	}

	return authorizedPermissions, nil
}

func iarFromIRA(ira adminv1.ImageRoleAuthorization) internalv1.ImageAllowRuleInstance {
	return internalv1.ImageAllowRuleInstance{
		ObjectMeta:    ira.ObjectMeta,
		ImageSelector: ira.ImageSelector,
	}
}

type genericRole struct {
	name      string
	namespace string
	rules     []rbacv1.PolicyRule
}

func resolveRoleRefs(ctx context.Context, c client.Reader, namespace, imageName string, roleRefs []internaladminv1.RoleRef) ([]v1.Permissions, error) {
	existingRoles := make(map[string]genericRole)

	var clusterRoles rbacv1.ClusterRoleList
	if err := c.List(ctx, &clusterRoles); err != nil {
		return nil, err
	}
	for _, cr := range clusterRoles.Items {
		existingRoles["cluster/"+cr.GetName()] = genericRole{
			name:  cr.GetName(),
			rules: cr.Rules,
		}
	}

	var projectroles rbacv1.RoleList
	if err := c.List(ctx, &projectroles, &client.ListOptions{Namespace: namespace}); err != nil {
		return nil, err
	}
	for _, pr := range projectroles.Items {
		existingRoles[pr.GetName()] = genericRole{
			name:      pr.GetName(),
			rules:     pr.Rules,
			namespace: namespace,
		}
	}

	var perms []v1.Permissions
	seen := make(map[string]struct{})

	for _, roleRef := range roleRefs {
		roleName := roleRef.Name
		if roleRef.Kind == "ClusterRole" {
			roleName = "cluster/" + roleName
		}
		if _, ok := seen[roleName]; ok {
			continue
		}
		seen[roleName] = struct{}{}
		if eRole, ok := existingRoles[roleName]; ok {
			perms = append(perms, permissionsFromGenericRole(eRole, imageName))
		} else {
			logrus.Warnf("RoleRef references non-existent role [%s] in namespace: [%s]", roleName, namespace)
		}
	}
	return perms, nil
}

func permissionsFromGenericRole(role genericRole, nameOverride string) v1.Permissions {
	name := role.name
	if nameOverride != "" {
		name = nameOverride
	}
	perms := v1.Permissions{
		ServiceName: name,
	}
	scope := "cluster"
	if role.namespace != "" {
		scope = "project:" + role.namespace
	}
	for _, rule := range role.rules {
		perms.Rules = append(perms.Rules, v1.PolicyRule{PolicyRule: rule, Scopes: []string{scope}})
	}
	return perms
}
