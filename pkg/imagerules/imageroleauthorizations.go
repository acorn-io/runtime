package imagerules

import (
	"context"
	"errors"

	adminv1 "github.com/acorn-io/runtime/pkg/apis/admin.acorn.io/v1"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	internaladminv1 "github.com/acorn-io/runtime/pkg/apis/internal.admin.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/images"
	"github.com/acorn-io/runtime/pkg/imageselector"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/sirupsen/logrus"
	rbacv1 "k8s.io/api/rbac/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetAuthorizedPermissions(ctx context.Context, c client.Reader, namespace, imageName, digest string) ([]v1.Permissions, error) {
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

	authorizedRoles, err := CheckRoleAuthorizations(ctx, c, namespace, imageName, "", digest, iras.Items, remoteOpts...)
	if err != nil {
		if _, ok := err.(*ErrImageNotAllowed); ok {
			return nil, nil
		} else {
			return nil, err
		}
	}

	return resolveAuthorizedRoles(ctx, c, namespace, imageName, authorizedRoles)
}

func CheckRoleAuthorizations(ctx context.Context, c client.Reader, namespace, imageName, resolvedName, digest string, iras []adminv1.ImageRoleAuthorization, opts ...remote.Option) ([]internaladminv1.RoleAuthorizations, error) {
	// No rules? Deny all images.
	if len(iras) == 0 {
		return nil, &ErrImageNotAllowed{Image: imageName}
	}

	logrus.Debugf("Checking image %s (%s) against %d image role authorizations", imageName, digest, len(iras))
	var authorized []internaladminv1.RoleAuthorizations

	for _, ira := range iras {
		if err := imageselector.MatchImage(ctx, c, namespace, imageName, resolvedName, digest, ira.ImageSelector, opts...); err != nil {
			if ierr := (*imageselector.ImageSelectorNoMatchError)(nil); errors.As(err, &ierr) {
				logrus.Debugf("ImageRoleAuthorization %s/%s did not match: %v", ira.Namespace, ira.Name, err)
			} else {
				logrus.Errorf("Error matching ImageRoleAuthorization %s/%s: %v", ira.Namespace, ira.Name, err)
			}
			continue
		}
		logrus.Debugf("Image %s (%s) is allowed by ImageRoleAuthorization %s/%s", imageName, digest, ira.Namespace, ira.Name)
		authorized = append(authorized, ira.Roles)
	}
	if len(authorized) == 0 {
		return authorized, &ErrImageNotAllowed{Image: imageName}
	}
	return authorized, nil
}

type genericRole struct {
	name  string
	rules []rbacv1.PolicyRule
}

func resolveAuthorizedRoles(ctx context.Context, c client.Reader, namespace, imageName string, authorizedRoles []internaladminv1.RoleAuthorizations) ([]v1.Permissions, error) {
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
			name:  pr.GetName(),
			rules: pr.Rules,
		}
	}

	var perms []v1.Permissions

	for _, ar := range authorizedRoles {
		for _, roleRef := range ar.RoleRefs {
			roleName := roleRef.Name
			if roleRef.Kind == "ClusterRole" {
				roleName = "cluster/" + roleName
			}
			if eRole, ok := existingRoles[roleName]; ok {
				perms = append(perms, resolveGenericRole(eRole, imageName, ar.Scopes))
			} else {
				logrus.Warnf("RoleRef references non-existent role [%s] in namespace: [%s]", roleName, namespace)
			}
		}
	}
	return perms, nil
}

func resolveGenericRole(role genericRole, nameOverride string, scopes []string) v1.Permissions {
	perms := v1.Permissions{
		ServiceName: role.name,
	}
	if nameOverride != "" {
		perms.ServiceName = nameOverride
	}
	for _, rule := range role.rules {
		perms.Rules = append(perms.Rules, v1.PolicyRule{PolicyRule: rule, Scopes: scopes})
	}
	return perms
}
