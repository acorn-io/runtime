package apps

import (
	"context"
	"errors"
	"fmt"
	"strings"

	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/client"
	"github.com/acorn-io/acorn/pkg/pullsecret"
	"github.com/acorn-io/acorn/pkg/tags"
	"github.com/acorn-io/baaah/pkg/merr"
	"github.com/acorn-io/baaah/pkg/typed"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	authv1 "k8s.io/api/authorization/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierror "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apiserver/pkg/endpoints/request"
)

func (s *Storage) checkRemotePermissions(ctx context.Context, namespace, image string) error {
	keyChain, err := pullsecret.Keychain(ctx, s.client, namespace)
	if err != nil {
		return err
	}

	ref, err := name.ParseReference(image)
	if err != nil {
		return err
	}

	_, err = remote.Image(ref, remote.WithContext(ctx), remote.WithAuthFromKeychain(keyChain))
	if err != nil {
		return fmt.Errorf("failed to pull %s: %v", image, err)
	}
	return nil
}

func (s *Storage) check(ctx context.Context, sar *authv1.SubjectAccessReview, rule v1.PolicyRule) error {
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

func (s *Storage) checkNonResourceRole(ctx context.Context, sar *authv1.SubjectAccessReview, rule v1.PolicyRule, namespace string) error {
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

func (s *Storage) checkResourceRole(ctx context.Context, sar *authv1.SubjectAccessReview, rule v1.PolicyRule, namespace string) error {
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

func (s *Storage) checkRules(ctx context.Context, sar *authv1.SubjectAccessReview, rules []v1.PolicyRule, namespace string) error {
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

func (s *Storage) compareAndCheckPermissions(ctx context.Context, perms v1.Permissions, requestedPerms *v1.Permissions) error {
	if len(perms.ClusterRules) == 0 && len(perms.Rules) == 0 {
		return nil
	}

	if !equality.Semantic.DeepEqual(perms.ClusterRules, requestedPerms.Get().ClusterRules) ||
		!equality.Semantic.DeepEqual(perms.Rules, requestedPerms.Get().Rules) {
		return &client.ErrRulesNeeded{
			Permissions: perms,
		}
	}

	user, ok := request.UserFrom(ctx)
	if !ok {
		return fmt.Errorf("failed to find active user to check current privileges")
	}

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

	var errs []error
	if err := s.checkRules(ctx, sar, perms.ClusterRules, ""); err != nil {
		errs = append(errs, err)
	}

	ns, _ := request.NamespaceFrom(ctx)
	if err := s.checkRules(ctx, sar, perms.Rules, ns); err != nil {
		errs = append(errs, err)
	}

	return merr.NewErrors(errs...)
}

func (s *Storage) getPermissions(ctx context.Context, image string) (result v1.Permissions, _ error) {
	details, err := s.imageDetails.GetDetails(ctx, image, nil, nil)
	if err != nil {
		return result, err
	}

	if details.ParseError != "" {
		return result, errors.New(details.ParseError)
	}

	for _, entry := range typed.Sorted(details.AppSpec.Containers) {
		result.ClusterRules = append(result.ClusterRules, entry.Value.Permissions.Get().ClusterRules...)
		result.Rules = append(result.Rules, entry.Value.Permissions.Get().Rules...)
		for _, sidecar := range typed.Sorted(entry.Value.Sidecars) {
			result.ClusterRules = append(result.ClusterRules, sidecar.Value.Permissions.Get().ClusterRules...)
			result.Rules = append(result.Rules, sidecar.Value.Permissions.Get().Rules...)
		}
	}

	return result, nil
}

func (s *Storage) resolveTag(ctx context.Context, namespace, image string) (string, error) {
	localImage, err := s.images.ImageGet(ctx, image)
	if apierror.IsNotFound(err) {
		if tags.IsLocalReference(image) {
			return "", err
		}
		if err := s.checkRemotePermissions(ctx, namespace, image); err != nil {
			return "", err
		}
	} else if err != nil {
		return "", err
	} else {
		return strings.TrimPrefix(localImage.Digest, "sha256:"), nil
	}
	return image, nil
}
