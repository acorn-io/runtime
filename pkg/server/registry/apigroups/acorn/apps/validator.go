package apps

import (
	"context"
	"errors"
	"fmt"
	"strings"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	adminv1 "github.com/acorn-io/acorn/pkg/apis/internal.admin.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/autoupgrade"
	"github.com/acorn-io/acorn/pkg/client"
	apiv1config "github.com/acorn-io/acorn/pkg/config"
	"github.com/acorn-io/acorn/pkg/imageallowrules"
	"github.com/acorn-io/acorn/pkg/imagesystem"
	"github.com/acorn-io/acorn/pkg/pullsecret"
	"github.com/acorn-io/acorn/pkg/tags"
	"github.com/acorn-io/acorn/pkg/volume"
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

	if err := imagesystem.IsNotInternalRepo(ctx, s.client, params.Spec.Image); err != nil {
		result = append(result, field.Invalid(field.NewPath("spec", "image"), params.Spec.Image, err.Error()))
		return
	}

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

		imageDetails, err := s.getImageDetails(ctx, params, image)
		if err != nil {
			result = append(result, field.Invalid(field.NewPath("spec", "image"), params.Spec.Image, err.Error()))
			return
		}

		if err = s.validateRegion(ctx, params); err != nil {
			result = append(result, field.Invalid(field.NewPath("spec", "region"), params.Spec.Region, err.Error()))
			return
		}

		disableCheckImageAllowRules := false
		if params.Spec.Stop != nil && *params.Spec.Stop {
			// app was stopped, so we don't need to check image allow rules (this could prevent stopping an app if the image allow rules changed)
			disableCheckImageAllowRules = true
		}

		if !disableCheckImageAllowRules {
			if err := s.checkImageAllowed(ctx, params.Namespace, params.Spec.Image); err != nil {
				result = append(result, field.Invalid(field.NewPath("spec", "image"), params.Spec.Image, fmt.Sprintf("disallowed by imageAllowRules: %s", err.Error())))
				return
			}
		}

		workloadsFromImage, err := s.getWorkloads(imageDetails)
		if err != nil {
			result = append(result, field.Invalid(field.NewPath("spec", "image"), params.Spec.Image, err.Error()))
			return
		}

		apiv1cfg, err := apiv1config.Get(ctx, s.client)
		if err != nil {
			result = append(result, field.Invalid(field.NewPath("config"), params.Spec.Image, err.Error()))
			return
		}

		errs := s.checkScheduling(ctx, params, *apiv1cfg, workloadsFromImage, apiv1cfg.WorkloadMemoryDefault, apiv1cfg.WorkloadMemoryMaximum)
		if len(errs) != 0 {
			result = append(result, errs...)
			return
		}

		if err := volume.ValidateVolumeClasses(ctx, s.client, params.Namespace, params.Spec, imageDetails.AppSpec); err != nil {
			result = append(result, err)
			return
		}

		permsFromImage, err := s.getPermissions(imageDetails)
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
	oldParams := old.(*apiv1.App)
	if newParams.Spec.Region != oldParams.Spec.Region && newParams.Spec.Region != oldParams.Status.Defaults.Region {
		result = append(result, field.Invalid(field.NewPath("spec", "region"), newParams.Spec.Region, "cannot change region"))
		return result
	}
	return s.Validate(ctx, newParams)
}

func (s *Validator) validateRegion(ctx context.Context, app *apiv1.App) error {
	project := new(apiv1.Project)
	err := s.client.Get(ctx, kclient.ObjectKey{Name: app.Namespace}, project)
	if err != nil {
		return err
	}

	if app.Spec.Region == "" {
		app.Status.Defaults.Region = project.Spec.DefaultRegion
		if app.Status.Defaults.Region == "" {
			if project.Status.DefaultRegion == "" {
				return fmt.Errorf("no region can be determined because project default region is not set")
			}
			app.Status.Defaults.Region = project.Status.DefaultRegion
		}

		return nil
	}

	if !project.ForRegion(app.Spec.Region) {
		return fmt.Errorf("region %s is not supported for project %s", app.Spec.Region, app.Namespace)
	}

	return nil
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

func (s *Validator) checkClusterRules(ctx context.Context, sar *authv1.SubjectAccessReview, rules []v1.ClusterPolicyRule) error {
	for _, rule := range rules {
		if len(rule.Namespaces) == 0 {
			if err := s.checkRules(ctx, sar, []v1.PolicyRule{(v1.PolicyRule)(rule.PolicyRule)}, ""); err != nil {
				return err
			}
		} else {
			for _, ns := range rule.Namespaces {
				if err := s.checkRules(ctx, sar, []v1.PolicyRule{(v1.PolicyRule)(rule.PolicyRule)}, ns); err != nil {
					return err
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

		if err := s.checkClusterRules(ctx, sar, perm.ClusterRules); err != nil {
			errs = append(errs, err)
		}

		ns, _ := request.NamespaceFrom(ctx)
		if err := s.checkRules(ctx, sar, perm.Rules, ns); err != nil {
			errs = append(errs, err)
		}
	}

	return merr.NewErrors(errs...)
}

func (s *Validator) checkScheduling(ctx context.Context, params *apiv1.App, cfg apiv1.Config, workloads map[string]v1.Container, specMemDefault, specMemMaximum *int64) []*field.Error {
	var (
		memory       = params.Spec.Memory
		computeClass = params.Spec.ComputeClass
	)
	validationErrors := []*field.Error{}
	err := validateMemoryRunFlags(memory, workloads)
	if err != nil {
		validationErrors = append(validationErrors, err...)
	}

	for workload, container := range workloads {
		wc, err := adminv1.GetClassForWorkload(ctx, s.client, computeClass, container, workload, params.Namespace)
		if err != nil {
			validationErrors = append(validationErrors, field.Invalid(field.NewPath("computeclass"), "", err.Error()))
		}

		if wc != nil {
			// Parse the memory
			wcMemory, err := adminv1.ParseComputeClassMemory(wc.Memory)
			if err != nil {
				if errors.Is(err, adminv1.ErrInvalidClass) {
					validationErrors = append(validationErrors, field.Invalid(field.NewPath("spec", "memory"), wc.Memory, err.Error()))
				}
			}

			memDefault := wcMemory.Def.Value()
			specMemDefault = &memDefault
		}

		// Validate memory
		memQuantity, err := v1.ValidateMemory(
			memory, workload, container, specMemDefault, specMemMaximum)

		// Evaluate what caused the error if one exists
		if err != nil {
			path := field.NewPath("unknown")
			if errors.Is(err, v1.ErrInvalidAcornMemory) {
				path = field.NewPath("spec", "image")
			} else if errors.Is(err, v1.ErrInvalidSetMemory) {
				path = field.NewPath("spec", "memory", workload)
			} else if errors.Is(err, v1.ErrInvalidDefaultMemory) {
				path = field.NewPath("config", "workloadMemoryDefault")
			}
			validationErrors = append(validationErrors, field.Invalid(path, memQuantity.String(), err.Error()))
		}

		// Need a ComputeClass to validate it
		if wc == nil {
			continue
		}

		err = adminv1.ValidateComputeClass(*wc, memQuantity, specMemDefault)
		if err != nil {
			if errors.Is(err, adminv1.ErrInvalidClass) {
				validationErrors = append(validationErrors, field.Invalid(field.NewPath("computeclass"), wc.Name, err.Error()))
			} else if errors.Is(err, adminv1.ErrInvalidMemoryForClass) {
				validationErrors = append(validationErrors, field.Invalid(field.NewPath("memory"), memQuantity.String(), err.Error()))
			} else {
				validationErrors = append(validationErrors, field.Invalid(field.NewPath("unknown"), "", err.Error()))
			}
		}
	}
	return validationErrors
}

func validateMemoryRunFlags(memory v1.MemoryMap, workloads map[string]v1.Container) []*field.Error {
	validationErrors := []*field.Error{}
	for key := range memory {
		if key == "" {
			continue
		}
		if _, ok := workloads[key]; !ok {
			path := field.NewPath("spec", "memory")
			validationErrors = append(validationErrors, field.Invalid(path, key, v1.ErrInvalidWorkload.Error()))
		}
	}
	return validationErrors
}

func (s *Validator) getPermissions(details *client.ImageDetails) (result []v1.Permissions, _ error) {
	result = append(result, buildPermissionsFrom(details.AppSpec.Containers)...)
	result = append(result, buildPermissionsFrom(details.AppSpec.Jobs)...)

	return result, nil
}

func (s *Validator) getWorkloads(details *client.ImageDetails) (map[string]v1.Container, error) {
	result := make(map[string]v1.Container, len(details.AppSpec.Containers)+len(details.AppSpec.Jobs))
	for workload, container := range details.AppSpec.Containers {
		result[workload] = container
		for sidecarWorkload, sidecarContainer := range container.Sidecars {
			result[sidecarWorkload] = sidecarContainer
		}
	}
	for workload, container := range details.AppSpec.Jobs {
		result[workload] = container
	}

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
	localImage, err := s.clientFactory.Namespace("", namespace).ImageGet(ctx, image)
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

func (s *Validator) getImageDetails(ctx context.Context, app *apiv1.App, image string) (*client.ImageDetails, error) {
	details, err := s.clientFactory.Namespace("", app.Namespace).ImageDetails(ctx, image,
		&client.ImageDetailsOptions{
			Profiles:   app.Spec.Profiles,
			DeployArgs: app.Spec.DeployArgs})
	if err != nil {
		return nil, err
	}

	if details.ParseError != "" {
		return nil, fmt.Errorf(details.ParseError)
	}

	return details, nil
}

func (s *Validator) checkImageAllowed(ctx context.Context, namespace, image string) error {
	return imageallowrules.CheckImageAllowed(ctx, s.client, namespace, image)
}
