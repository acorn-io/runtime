package apps

import (
	"context"
	"errors"
	"fmt"
	"strings"

	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/client"
	hclient "github.com/acorn-io/acorn/pkg/k8sclient"
	"github.com/acorn-io/acorn/pkg/pullsecret"
	"github.com/acorn-io/acorn/pkg/tags"
	"github.com/acorn-io/baaah/pkg/merr"
	"github.com/acorn-io/baaah/pkg/typed"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	authv1 "k8s.io/api/authorization/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apiserver/pkg/endpoints/request"
)

func (s *Strategy) checkRemoteAccess(ctx context.Context, namespace, image string) error {
	keyChain, err := pullsecret.Keychain(ctx, s.client, namespace)
	if err != nil {
		return err
	}

	ref, err := name.ParseReference(image)
	if err != nil {
		return err
	}

	_, err = remote.Head(ref, remote.WithContext(ctx), remote.WithAuthFromKeychain(keyChain))
	if err != nil {
		return fmt.Errorf("failed to pull %s: %v", image, err)
	}
	return nil
}

func (s *Strategy) check(ctx context.Context, sar *authv1.SubjectAccessReview, rule v1.PolicyRule) error {
	err := s.client.Create(ctx, sar)
	if err != nil {
		return err
	}
	if !sar.Status.Allowed {
		return &client.ErrNotAuthorized{
			Rule: (rbacv1.PolicyRule)(rule),
		}
	}
	return nil
}

func (s *Strategy) checkNonResourceRole(ctx context.Context, sar *authv1.SubjectAccessReview, rule v1.PolicyRule, namespace string) error {
	if len(rule.Verbs) == 0 {
		return fmt.Errorf("can not deploy acorn due to requesting role with empty verbs")
	}

	for _, url := range rule.NonResourceURLs {
		for _, verb := range rule.Verbs {
			sar := sar.DeepCopy()
			sar.Spec.NonResourceAttributes = &authv1.NonResourceAttributes{
				Path: url,
				Verb: verb,
			}
			if err := s.check(ctx, sar, rule); err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *Strategy) checkResourceRole(ctx context.Context, sar *authv1.SubjectAccessReview, rule v1.PolicyRule, namespace string) error {
	if len(rule.APIGroups) == 0 {
		return fmt.Errorf("can not deploy acorn due to requesting role with empty apiGroups")
	}
	if len(rule.Verbs) == 0 {
		return fmt.Errorf("can not deploy acorn due to requesting role with empty verbs")
	}
	if len(rule.Resources) == 0 {
		return fmt.Errorf("can not deploy acorn due to requesting role with empty resources")
	}
	for _, verb := range rule.Verbs {
		for _, apiGroup := range rule.APIGroups {
			for _, resource := range rule.Resources {
				resource, subResource, _ := strings.Cut(resource, "/")
				if len(rule.ResourceNames) == 0 {
					sar := sar.DeepCopy()
					sar.Spec.ResourceAttributes = &authv1.ResourceAttributes{
						Namespace:   namespace,
						Verb:        verb,
						Group:       apiGroup,
						Version:     "*",
						Resource:    resource,
						Subresource: subResource,
					}
					if err := s.check(ctx, sar, rule); err != nil {
						return err
					}
				} else {
					for _, resourceName := range rule.ResourceNames {
						sar := sar.DeepCopy()
						sar.Spec.ResourceAttributes = &authv1.ResourceAttributes{
							Namespace:   namespace,
							Verb:        verb,
							Group:       apiGroup,
							Version:     "*",
							Resource:    resource,
							Subresource: subResource,
							Name:        resourceName,
						}
						if err := s.check(ctx, sar, rule); err != nil {
							return err
						}
					}
				}
			}
		}
	}

	return nil
}

func (s *Strategy) checkRules(ctx context.Context, sar *authv1.SubjectAccessReview, rules []v1.PolicyRule, namespace string) error {
	var errs []error
	for _, rule := range rules {
		if len(rule.NonResourceURLs) > 0 {
			if err := s.checkNonResourceRole(ctx, sar, rule, namespace); err != nil {
				errs = append(errs, err)
			}
		} else {
			if err := s.checkResourceRole(ctx, sar, rule, namespace); err != nil {
				errs = append(errs, err)
			}
		}
	}
	return merr.NewErrors(errs...)
}

// checkRequestedPermsSatisfyImagePerms checks that the user requested permissions are enough to satisfy the permissions
// specified by the image's Acornfile
func (s *Strategy) checkRequestedPermsSatisfyImagePerms(perms []v1.Permissions, requestedPerms []v1.Permissions) error {
	if len(perms) == 0 {
		return nil
	}

	permsError := &client.ErrRulesNeeded{Permissions: []v1.Permissions{}}
	for _, perm := range perms {
		if len(perm.ClusterRules) == 0 && len(perm.Rules) == 0 {
			continue
		}

		if specPerms := v1.FindPermission(perm.ServiceName, requestedPerms); !specPerms.HasRules() ||
			!equality.Semantic.DeepEqual(perm.ClusterRules, specPerms.Get().ClusterRules) ||
			!equality.Semantic.DeepEqual(perm.ClusterRules, specPerms.Get().ClusterRules) {
			permsError.Permissions = append(permsError.Permissions, perm)
			continue
		}
	}

	if len(permsError.Permissions) != 0 {
		return permsError
	}
	return nil
}

// checkPermissionsForPrivilegeEscalation is an actual RBAC check to prevent privilege escalation. The user making the request must have the
// permissions that they are requesting the app gets
func (s *Strategy) checkPermissionsForPrivilegeEscalation(ctx context.Context, requestedPerms []v1.Permissions) error {
	user, ok := request.UserFrom(ctx)
	if !ok {
		return fmt.Errorf("failed to find active user to check current privileges")
	}

	var errs []error
	for _, perm := range requestedPerms {
		sar := &authv1.SubjectAccessReview{
			Spec: authv1.SubjectAccessReviewSpec{
				User:   user.GetName(),
				Groups: user.GetGroups(),
				Extra:  map[string]authv1.ExtraValue{},
				UID:    user.GetUID(),
			},
		}

		for k, v := range user.GetExtra() {
			sar.Spec.Extra[k] = v
		}

		if err := s.checkRules(ctx, sar, perm.ClusterRules, ""); err != nil {
			errs = append(errs, err)
		}

		ns, _ := request.NamespaceFrom(ctx)
		if err := s.checkRules(ctx, sar, perm.Rules, ns); err != nil {
			errs = append(errs, err)
		}
	}

	return merr.NewErrors(errs...)
}

func (s *Strategy) getPermissions(ctx context.Context, namespace, image string) (result []v1.Permissions, _ error) {
	details, err := s.clientFactory.Namespace(namespace).ImageDetails(ctx, image, nil)
	if err != nil {
		return result, err
	}

	if details.ParseError != "" {
		return result, errors.New(details.ParseError)
	}

	result = append(result, buildPermissionsFrom(details.AppSpec.Containers)...)
	result = append(result, buildPermissionsFrom(details.AppSpec.Jobs)...)

	return result, nil
}

func buildPermissionsFrom(containers map[string]v1.Container) []v1.Permissions {
	permissions := []v1.Permissions{}
	for _, entry := range typed.Sorted(containers) {
		entryPermissions := v1.Permissions{
			ServiceName:  entry.Key,
			ClusterRules: entry.Value.Permissions.Get().ClusterRules,
			Rules:        entry.Value.Permissions.Get().Rules,
		}

		for _, sidecar := range typed.Sorted(entry.Value.Sidecars) {
			entryPermissions.ClusterRules = append(entryPermissions.ClusterRules, sidecar.Value.Permissions.Get().ClusterRules...)
			entryPermissions.Rules = append(entryPermissions.Rules, sidecar.Value.Permissions.Get().Rules...)
		}

		permissions = append(permissions, entryPermissions)
	}

	return permissions
}

func (s *Strategy) resolveLocalImage(ctx context.Context, namespace, image string) (string, bool, error) {
	localImage, err := s.clientFactory.Namespace(namespace).ImageGet(ctx, image)
	if apierrors.IsNotFound(err) {
		if tags.IsLocalReference(image) {
			return "", false, err
		}

	} else if err != nil {
		return "", false, err
	} else {
		return strings.TrimPrefix(localImage.Digest, "sha256:"), true, nil
	}
	return image, false, nil
}

func (s *Strategy) createNamespace(ctx context.Context, name string) error {
	ns := &corev1.Namespace{}
	err := s.client.Get(ctx, hclient.ObjectKey{
		Name: name,
	}, ns)
	if apierrors.IsNotFound(err) {
		err := s.client.Create(ctx, &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
			},
		})
		if err != nil {
			return fmt.Errorf("unable to create namespace %s: %w", name, err)
		}
		return nil
	}
	return err
}
