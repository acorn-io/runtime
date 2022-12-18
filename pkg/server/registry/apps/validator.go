package apps

import (
	"context"
	"errors"
	"fmt"
	"strings"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/autoupgrade"
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
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/apiserver/pkg/endpoints/request"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type Validator struct {
	client        kclient.Client
	clientFactory *client.Factory
}

func NewValidator(client kclient.Client, clientFactory *client.Factory) *Validator {
	return &Validator{
		client:        client,
		clientFactory: clientFactory,
	}
}

func (s *Validator) Validate(ctx context.Context, obj runtime.Object) (result field.ErrorList) {
	params := obj.(*apiv1.App)

	if _, isPattern := autoupgrade.AutoUpgradePattern(params.Spec.Image); !isPattern {
		image, local, err := s.resolveLocalImage(ctx, params.Namespace, params.Spec.Image)
		if err != nil {
			result = append(result, field.Invalid(field.NewPath("spec", "image"), params.Spec.Image, err.Error()))
			return
		}

		if !local {
			if err := s.checkRemoteAccess(ctx, params.Namespace, image); err != nil {
				result = append(result, field.Invalid(field.NewPath("spec", "image"), params.Spec.Image, err.Error()))
				return
			}
		}

		permsFromImage, err := s.getPermissions(ctx, image, params)
		if err != nil {
			result = append(result, field.Invalid(field.NewPath("spec", "permissions"), params.Spec.Permissions, err.Error()))
			return
		}

		if err := s.checkRequestedPermsSatisfyImagePerms(permsFromImage, params.Spec.Permissions); err != nil {
			result = append(result, field.Invalid(field.NewPath("spec", "permissions"), params.Spec.Permissions, err.Error()))
			return
		}
	}

	if err := s.checkPermissionsForPrivilegeEscalation(ctx, params.Spec.Permissions); err != nil {
		result = append(result, field.Invalid(field.NewPath("spec", "permissions"), params.Spec.Permissions, err.Error()))
	}

	return result
}

func (s *Validator) ValidateUpdate(ctx context.Context, obj, old runtime.Object) (result field.ErrorList) {
	newParams := obj.(*apiv1.App)
	return s.Validate(ctx, newParams)
}

func (s *Validator) checkRemoteAccess(ctx context.Context, namespace, image string) error {
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

func (s *Validator) check(ctx context.Context, sar *authv1.SubjectAccessReview, rule v1.PolicyRule) error {
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

func (s *Validator) checkNonResourceRole(ctx context.Context, sar *authv1.SubjectAccessReview, rule v1.PolicyRule, namespace string) error {
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

func (s *Validator) checkResourceRole(ctx context.Context, sar *authv1.SubjectAccessReview, rule v1.PolicyRule, namespace string) error {
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

func (s *Validator) checkRules(ctx context.Context, sar *authv1.SubjectAccessReview, rules []v1.PolicyRule, namespace string) error {
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
func (s *Validator) checkRequestedPermsSatisfyImagePerms(perms []v1.Permissions, requestedPerms []v1.Permissions) error {
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
func (s *Validator) checkPermissionsForPrivilegeEscalation(ctx context.Context, requestedPerms []v1.Permissions) error {
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

func (s *Validator) getPermissions(ctx context.Context, image string, app *apiv1.App) (result []v1.Permissions, _ error) {
	details, err := s.clientFactory.Namespace(app.Namespace).ImageDetails(ctx, image,
		&client.ImageDetailsOptions{
			Profiles:   app.Spec.Profiles,
			DeployArgs: app.Spec.DeployArgs})

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

func (s *Validator) resolveLocalImage(ctx context.Context, namespace, image string) (string, bool, error) {
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
