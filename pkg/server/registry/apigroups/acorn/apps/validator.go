package apps

import (
	"context"
	"errors"
	"fmt"
	"strings"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/appdefinition"
	"github.com/acorn-io/acorn/pkg/autoupgrade"
	"github.com/acorn-io/acorn/pkg/client"
	"github.com/acorn-io/acorn/pkg/computeclasses"
	apiv1config "github.com/acorn-io/acorn/pkg/config"
	"github.com/acorn-io/acorn/pkg/imageallowrules"
	"github.com/acorn-io/acorn/pkg/imagesystem"
	"github.com/acorn-io/acorn/pkg/pullsecret"
	"github.com/acorn-io/acorn/pkg/tags"
	"github.com/acorn-io/acorn/pkg/volume"
	"github.com/acorn-io/baaah/pkg/merr"
	"github.com/acorn-io/baaah/pkg/typed"
	"github.com/acorn-io/mink/pkg/strategy"
	"github.com/acorn-io/mink/pkg/types"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"golang.org/x/exp/slices"
	authv1 "k8s.io/api/authorization/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/apiserver/pkg/endpoints/request"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type Validator struct {
	client        kclient.Client
	clientFactory *client.Factory
	deleter       strategy.Deleter
}

func NewValidator(client kclient.Client, clientFactory *client.Factory, deleter strategy.Deleter) *Validator {
	return &Validator{
		client:        client,
		clientFactory: clientFactory,
		deleter:       deleter,
	}
}

func (s *Validator) Get(ctx context.Context, namespace, name string) (types.Object, error) {
	return s.deleter.Get(ctx, namespace, name)
}

func (s *Validator) Validate(ctx context.Context, obj runtime.Object) (result field.ErrorList) {
	params := obj.(*apiv1.App)

	if err := s.validateName(params); err != nil {
		result = append(result, field.Invalid(field.NewPath("metadata", "name"), params.Name, err.Error()))
		return
	}

	project := new(apiv1.Project)
	if err := s.client.Get(ctx, kclient.ObjectKey{Name: params.Namespace}, project); err != nil {
		result = append(result, field.Invalid(field.NewPath("spec", "images"), params.Spec.Image, err.Error()))
		return
	}

	if err := s.validateRegion(params, project); err != nil {
		result = append(result, field.Invalid(field.NewPath("spec", "region"), params.Spec.Region, err.Error()))
		return
	}

	if err := imagesystem.IsNotInternalRepo(ctx, s.client, params.Namespace, params.Spec.Image); err != nil {
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

		imageDetails, err := s.getImageDetails(ctx, params.Namespace, params.Spec.Profiles, params.Spec.DeployArgs, image, "")
		if err != nil {
			result = append(result, field.Invalid(field.NewPath("spec", "image"), params.Spec.Image, err.Error()))
			return
		}

		disableCheckImageAllowRules := false
		if params.Spec.Stop != nil && *params.Spec.Stop {
			// app was stopped, so we don't need to check image allow rules (this could prevent stopping an app if the image allow rules changed)
			disableCheckImageAllowRules = true
		}

		if !disableCheckImageAllowRules {
			if err := s.checkImageAllowed(ctx, params.Namespace, params.Spec.Image); err != nil {
				result = append(result, field.Invalid(field.NewPath("spec", "image"), params.Spec.Image, err.Error()))
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

		errs := s.checkScheduling(ctx, params, project, workloadsFromImage, apiv1cfg.WorkloadMemoryDefault, apiv1cfg.WorkloadMemoryMaximum)
		if len(errs) != 0 {
			result = append(result, errs...)
			return
		}

		if err := validateVolumeClasses(ctx, s.client, params.Namespace, params.Spec, imageDetails.AppSpec, project); err != nil {
			result = append(result, err)
			return
		}

		permsFromImage, err := s.getPermissions(ctx, "", params.Namespace, image, imageDetails)
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

	if oldParams.Status.GetDevMode() {
		result = append(result, field.Invalid(field.NewPath("status", "devSession"), "", "app is locked by dev session"))
		return result
	}

	if newParams.Spec.Region != oldParams.Spec.Region && newParams.Spec.Region != oldParams.Status.Defaults.Region {
		result = append(result, field.Invalid(field.NewPath("spec", "region"), newParams.Spec.Region, "cannot change region"))
		return result
	}

	return s.Validate(ctx, newParams)
}

func (s *Validator) validateName(app *apiv1.App) error {
	if app.Name == "" {
		return fmt.Errorf("name is required")
	}

	errs := validation.IsDNS1035Label(app.Name)
	if len(errs) > 0 {
		return fmt.Errorf(strings.Join(errs, ": "))
	}

	return nil
}

func (s *Validator) validateRegion(app *apiv1.App, project *apiv1.Project) error {
	if app.Spec.Region == "" {
		if project.Spec.DefaultRegion == "" && project.Status.DefaultRegion == "" {
			return fmt.Errorf("no region can be determined because project default region is not set")
		}

		// Region default will be calculated later
		return nil
	}

	if !project.HasRegion(app.Spec.Region) {
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
			Rule: rule,
		}
	}
	return nil
}

func (s *Validator) checkNonResourceRole(ctx context.Context, sar *authv1.SubjectAccessReview, rule v1.PolicyRule) error {
	if len(rule.Verbs) == 0 {
		return fmt.Errorf("can not deploy acorn due to requesting role with empty verbs")
	}

	if len(rule.APIGroups) != 0 {
		return fmt.Errorf("can not deploy acorn due to requesting role nonResourceURLs %v and non-empty apiGroups set %v", rule.NonResourceURLs, rule.APIGroups)
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
		rule.Resources = []string{"*"}
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

func (s *Validator) checkRules(ctx context.Context, sar *authv1.SubjectAccessReview, rules []v1.PolicyRule, currentNamespace string) error {
	var errs []error
	for _, rule := range rules {
		if len(rule.NonResourceURLs) > 0 {
			if err := s.checkNonResourceRole(ctx, sar, rule); err != nil {
				errs = append(errs, err)
			}
		} else {
			for _, namespace := range rule.ResolveNamespaces(currentNamespace) {
				if err := s.checkResourceRole(ctx, sar, rule, namespace); err != nil {
					errs = append(errs, err)
				}
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

	for _, perm := range perms {
		if len(perm.GetRules()) == 0 {
			continue
		}

		if specPerms := v1.FindPermission(perm.ServiceName, requestedPerms); !specPerms.HasRules() ||
			!equality.Semantic.DeepEqual(perm.GetRules(), specPerms.Get().GetRules()) {
			// If any perm doesn't match then return all perms
			return &client.ErrRulesNeeded{
				Permissions: perms,
			}
		}
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

		ns, _ := request.NamespaceFrom(ctx)
		if err := s.checkRules(ctx, sar, perm.GetRules(), ns); err != nil {
			errs = append(errs, err)
		}
	}

	return merr.NewErrors(errs...)
}

// checkScheduling must use apiv1.ComputeCLass to validate the scheduling instead of the Instance counterparts.
func (s *Validator) checkScheduling(ctx context.Context, params *apiv1.App, project *apiv1.Project, workloads map[string]v1.Container, specMemDefault, specMemMaximum *int64) []*field.Error {
	var (
		memory        = params.Spec.Memory
		computeClass  = params.Spec.ComputeClasses
		defaultRegion = project.GetRegion()
	)

	var validationErrors []*field.Error
	computeClasses := new(apiv1.ComputeClassList)
	if err := s.client.List(ctx, computeClasses, kclient.InNamespace(params.Namespace)); err != nil {
		return append(validationErrors, field.Invalid(field.NewPath("spec", "image"), params.Spec.Image, fmt.Sprintf("error listing compute classes: %v", err)))
	}

	err := validateMemoryRunFlags(memory, workloads)
	if err != nil {
		validationErrors = append(validationErrors, err...)
	}

	for workload, container := range workloads {
		cc, err := getClassForWorkload(computeClasses, computeClass, container, workload)
		if err != nil {
			validationErrors = append(validationErrors, field.NotFound(field.NewPath("computeclass"), err.Error()))
		}

		if cc != nil {
			if !slices.Contains(cc.SupportedRegions, params.Spec.Region) && !slices.Contains(cc.SupportedRegions, defaultRegion) {
				validationErrors = append(validationErrors, field.Invalid(field.NewPath("computeclass"), "", fmt.Sprintf("computeclass %s does not support region %s", cc.Name, params.Spec.Region)))
				continue
			}
			// Parse the memory
			wcMemory, err := computeclasses.ParseComputeClassMemory(cc.Memory)
			if err != nil {
				if errors.Is(err, computeclasses.ErrInvalidClass) {
					validationErrors = append(validationErrors, field.Invalid(field.NewPath("spec", "memory"), cc.Memory, err.Error()))
				}
			}

			memDefault := wcMemory.Def.Value()
			specMemDefault = &memDefault
		}

		// Validate memory
		memQuantity, err := v1.ValidateMemory(memory, workload, container, specMemDefault, specMemMaximum)

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
		if cc == nil {
			continue
		}

		err = computeclasses.Validate(*cc, memQuantity, specMemDefault)
		if err != nil {
			if errors.Is(err, computeclasses.ErrInvalidClass) {
				validationErrors = append(validationErrors, field.Invalid(field.NewPath("computeclass"), cc.Name, err.Error()))
			} else if errors.Is(err, computeclasses.ErrInvalidMemoryForClass) {
				validationErrors = append(validationErrors, field.Invalid(field.NewPath("memory"), memQuantity.String(), err.Error()))
			} else {
				validationErrors = append(validationErrors, field.Invalid(field.NewPath("unknown"), "", err.Error()))
			}
		}
	}
	return validationErrors
}

func getClassForWorkload(computeClassList *apiv1.ComputeClassList, computeClasses v1.ComputeClassMap, container v1.Container, workload string) (*apiv1.ComputeClass, error) {
	ccName := computeclasses.GetComputeClassNameForWorkload(workload, container, computeClasses)

	for _, cc := range computeClassList.Items {
		if cc.Name == ccName || (ccName == "" && cc.Default) {
			return &cc, nil
		}
	}

	if ccName == "" {
		// No default compute class
		return nil, nil
	}

	return nil, fmt.Errorf("computeclass %s not found", ccName)
}

func validateMemoryRunFlags(memory v1.MemoryMap, workloads map[string]v1.Container) []*field.Error {
	var validationErrors []*field.Error
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

func validateVolumeClasses(ctx context.Context, c kclient.Client, namespace string, appInstanceSpec v1.AppInstanceSpec, appSpec *v1.AppSpec, project *apiv1.Project) *field.Error {
	if len(appInstanceSpec.Volumes) == 0 && len(appSpec.Volumes) == 0 {
		return nil
	}

	var (
		defaultRegion      = project.GetRegion()
		volumeClassList    = new(apiv1.VolumeClassList)
		defaultVolumeClass *apiv1.VolumeClass
	)
	if err := c.List(ctx, volumeClassList, kclient.InNamespace(namespace)); err != nil {
		return field.Invalid(field.NewPath("spec", "image"), appInstanceSpec.Image, fmt.Sprintf("error checking volume classes: %v", err))
	}

	volumeClasses := make(map[string]apiv1.VolumeClass, len(volumeClassList.Items))
	for _, volumeClass := range volumeClassList.Items {
		if volumeClass.Default {
			defaultVolumeClass = volumeClass.DeepCopy()
		}
		volumeClasses[volumeClass.Name] = volumeClass
	}

	volumeBindings := make(map[string]v1.VolumeBinding)
	for i, vol := range appInstanceSpec.Volumes {
		if volClass, ok := volumeClasses[vol.Class]; vol.Class != "" && (!ok || volClass.Inactive || (volClass.SupportedRegions != nil && !slices.Contains(volClass.SupportedRegions, defaultRegion) && !slices.Contains(volClass.SupportedRegions, appInstanceSpec.Region))) {
			return field.Invalid(field.NewPath("spec", "volumes").Index(i), vol.Class, "not a valid volume class")
		}
		volumeBindings[vol.Target] = vol
	}

	var volClass apiv1.VolumeClass
	for volName, vol := range appSpec.Volumes {
		calculatedVolumeRequest := volume.CopyVolumeDefaults(vol, volumeBindings[volName], v1.VolumeDefault{})
		if calculatedVolumeRequest.Class != "" {
			volClass = volumeClasses[calculatedVolumeRequest.Class]
		} else if defaultVolumeClass != nil {
			volClass = *defaultVolumeClass
		} else {
			return field.Invalid(field.NewPath("spec", "image"), appInstanceSpec.Image, fmt.Sprintf("no volume class found for %s", volName))
		}
		if volClass.Inactive || (!slices.Contains(volClass.SupportedRegions, defaultRegion) && !slices.Contains(volClass.SupportedRegions, appInstanceSpec.Region)) {
			return field.Invalid(field.NewPath("spec", "image"), appInstanceSpec.Image, fmt.Sprintf("%s is not a valid volume class", volClass.Name))
		}

		if calculatedVolumeRequest.Size != "" {
			q := v1.MustParseResourceQuantity(calculatedVolumeRequest.Size)
			if volClass.Size.Min != "" && q.Cmp(*v1.MustParseResourceQuantity(volClass.Size.Min)) < 0 {
				return field.Invalid(field.NewPath("spec", "volumes", volName, "size"), q.String(), fmt.Sprintf("less than volume class %s minimum of %v", calculatedVolumeRequest.Class, volClass.Size.Min))
			}
			if volClass.Size.Max != "" && q.Cmp(*v1.MustParseResourceQuantity(volClass.Size.Max)) > 0 {
				return field.Invalid(field.NewPath("spec", "volumes", volName, "size"), q.String(), fmt.Sprintf("greater than volume class %s maximum of %v", calculatedVolumeRequest.Class, volClass.Size.Max))
			}
		}
		if volClass.AllowedAccessModes != nil {
			for _, am := range calculatedVolumeRequest.AccessModes {
				if !slices.Contains(volClass.AllowedAccessModes, am) {
					return field.Invalid(field.NewPath("spec", "volumes", volName, "accessModes"), am, fmt.Sprintf("not an allowed access mode of %v", calculatedVolumeRequest.Class))
				}
			}
		}
	}

	return nil
}

func (s *Validator) getPermissions(ctx context.Context, servicePrefix, namespace, image string, details *client.ImageDetails) (result []v1.Permissions, _ error) {
	result = append(result, buildPermissionsFrom(servicePrefix, details.AppSpec.Containers)...)
	result = append(result, buildPermissionsFrom(servicePrefix, details.AppSpec.Jobs)...)

	subResults, err := s.buildNestedPermissions(ctx, servicePrefix, details.AppSpec, namespace, image, details.AppImage.ImageData)
	if err != nil {
		return nil, err
	}

	result = append(result, subResults...)

	return result, nil
}

func (s *Validator) getImagePermissions(ctx context.Context, servicePrefix string, profiles []string, args map[string]any, namespace, image, nestedDigest string) (result []v1.Permissions, _ error) {
	details, err := s.getImageDetails(ctx, namespace, profiles, args, image, nestedDigest)
	if err != nil {
		return nil, err
	}

	return s.getPermissions(ctx, servicePrefix, namespace, image, details)
}

func (s *Validator) buildNestedPermissions(ctx context.Context, servicePrefix string, app *v1.AppSpec, namespace, image string, imageData v1.ImagesData) (result []v1.Permissions, err error) {
	for _, entry := range typed.Sorted(app.Acorns) {
		var (
			acornName, acorn = entry.Key, entry.Value
			subResult        []v1.Permissions
		)

		acornImage, ok := appdefinition.GetImageReferenceForServiceName(acornName, app, imageData)
		if !ok {
			return nil, fmt.Errorf("failed to find image information for nested acorn [%s]", acornName)
		}

		if tags.IsImageDigest(acornImage) {
			subResult, err = s.getImagePermissions(ctx, servicePrefix+entry.Key+".", acorn.Profiles, acorn.DeployArgs, namespace, image, acornImage)
			if err != nil {
				return nil, err
			}
		} else {
			subResult, err = s.getImagePermissions(ctx, servicePrefix+entry.Key+".", acorn.Profiles, acorn.DeployArgs, namespace, acornImage, "")
			if err != nil {
				return nil, err
			}
		}

		result = append(result, subResult...)
	}

	for _, entry := range typed.Sorted(app.Services) {
		var (
			serviceName, service = entry.Key, entry.Value
			subResult            []v1.Permissions
		)

		if service.Image == "" && service.Build == nil {
			continue
		}

		acornImage, ok := appdefinition.GetImageReferenceForServiceName(serviceName, app, imageData)
		if !ok {
			return nil, fmt.Errorf("failed to find image information for service [%s]", serviceName)
		}

		if tags.IsImageDigest(acornImage) {
			subResult, err = s.getImagePermissions(ctx, servicePrefix+entry.Key+".", nil, service.ServiceArgs, namespace, image, acornImage)
			if err != nil {
				return nil, err
			}
		} else {
			subResult, err = s.getImagePermissions(ctx, servicePrefix+entry.Key+".", nil, service.ServiceArgs, namespace, acornImage, "")
			if err != nil {
				return nil, err
			}
		}

		result = append(result, subResult...)
	}

	return
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

func buildPermissionsFrom(servicePrefix string, containers map[string]v1.Container) []v1.Permissions {
	var permissions []v1.Permissions
	for _, entry := range typed.Sorted(containers) {
		entryPermissions := v1.Permissions{
			ServiceName: servicePrefix + entry.Key,
			Rules:       entry.Value.Permissions.Get().GetRules(),
		}

		for _, sidecar := range typed.Sorted(entry.Value.Sidecars) {
			entryPermissions.Rules = append(entryPermissions.Rules, sidecar.Value.Permissions.Get().GetRules()...)
		}

		if len(entryPermissions.GetRules()) > 0 {
			permissions = append(permissions, entryPermissions)
		}
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

func (s *Validator) getImageDetails(ctx context.Context, namespace string, profiles []string, args map[string]any, image, nested string) (*client.ImageDetails, error) {
	details, err := s.clientFactory.Namespace("", namespace).ImageDetails(ctx, image,
		&client.ImageDetailsOptions{
			NestedDigest: nested,
			Profiles:     profiles,
			DeployArgs:   args})
	if err != nil {
		return nil, err
	}

	if details.ParseError != "" {
		return nil, fmt.Errorf(details.ParseError)
	}

	return details, nil
}

func (s *Validator) checkImageAllowed(ctx context.Context, namespace, image string) error {
	digest, _, err := s.resolveLocalImage(ctx, namespace, image)
	if err != nil {
		return err
	}
	err = imageallowrules.CheckImageAllowed(ctx, s.client, namespace, image, digest)
	return err
}
