package apps

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/acorn-io/baaah/pkg/merr"
	"github.com/acorn-io/baaah/pkg/typed"
	"github.com/acorn-io/mink/pkg/strategy"
	"github.com/acorn-io/mink/pkg/types"
	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/appdefinition"
	"github.com/acorn-io/runtime/pkg/autoupgrade"
	"github.com/acorn-io/runtime/pkg/client"
	"github.com/acorn-io/runtime/pkg/computeclasses"
	apiv1config "github.com/acorn-io/runtime/pkg/config"
	"github.com/acorn-io/runtime/pkg/imageallowrules"
	"github.com/acorn-io/runtime/pkg/images"
	"github.com/acorn-io/runtime/pkg/imagesystem"
	"github.com/acorn-io/runtime/pkg/labels"
	"github.com/acorn-io/runtime/pkg/pullsecret"
	"github.com/acorn-io/runtime/pkg/tags"
	"github.com/acorn-io/runtime/pkg/volume"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"golang.org/x/exp/slices"
	"golang.org/x/sync/errgroup"
	authv1 "k8s.io/api/authorization/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/apiserver/pkg/endpoints/request"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type Validator struct {
	client            kclient.Client
	clientFactory     *client.Factory
	deleter           strategy.Deleter
	allowNestedUpdate bool
}

func NewValidator(client kclient.Client, clientFactory *client.Factory, deleter strategy.Deleter) *Validator {
	return &Validator{
		client:        client,
		clientFactory: clientFactory,
		deleter:       deleter,
	}
}

func (s *Validator) ValidateName(ctx context.Context, obj runtime.Object) (result field.ErrorList) {
	name := obj.(kclient.Object).GetName()
	if errs := validation.IsDNS1035Label(name); len(errs) > 0 {
		result = append(result, field.Invalid(field.NewPath("metadata", "name"), name, strings.Join(errs, ",")))
	}
	return
}

func (s *Validator) AllowNestedUpdate() *Validator {
	cp := *s
	cp.allowNestedUpdate = true
	return &cp
}

func (s *Validator) Get(ctx context.Context, namespace, name string) (types.Object, error) {
	return s.deleter.Get(ctx, namespace, name)
}

// Validate will validate the App but also populate the spec.ImageGrantedPermissions on the object.
// This is a bit odd but hard to do in a different way and not be terribly inefficient with API calls
func (s *Validator) Validate(ctx context.Context, obj runtime.Object) (result field.ErrorList) {
	app := obj.(*apiv1.App)

	if err := s.validateName(app); err != nil {
		result = append(result, field.Invalid(field.NewPath("metadata", "name"), app.Name, err.Error()))
		return
	}

	project := new(v1.ProjectInstance)
	if err := s.client.Get(ctx, kclient.ObjectKey{Name: app.Namespace}, project); err != nil {
		result = append(result, field.Invalid(field.NewPath("spec", "images"), app.Spec.Image, err.Error()))
		return
	}

	if err := s.validateRegion(app, project); err != nil {
		result = append(result, field.Invalid(field.NewPath("spec", "region"), app.Spec.Region, err.Error()))
		return
	}

	if err := imagesystem.IsNotInternalRepo(ctx, s.client, app.Namespace, app.Spec.Image); err != nil {
		result = append(result, field.Invalid(field.NewPath("spec", "image"), app.Spec.Image, err.Error()))
		return
	}

	var (
		imageGrantedPerms []v1.Permissions
		checkImage        = app.Spec.Image
	)

	tagPattern, isPattern := autoupgrade.AutoUpgradePattern(app.Spec.Image)
	if isPattern {
		if latestImage, found, err := autoupgrade.FindLatestTagForImageWithPattern(ctx, s.client, "", app.Namespace, app.Spec.Image, tagPattern); err != nil {
			result = append(result, field.Invalid(field.NewPath("spec", "image"), app.Spec.Image, err.Error()))
			return
		} else if found {
			checkImage = latestImage
		} else {
			checkImage = ""
		}
	}

	if checkImage == "" {
		app.Spec.ImageGrantedPermissions = imageGrantedPerms
	} else {
		image, local, err := s.resolveLocalImage(ctx, app.Namespace, checkImage)
		if err != nil {
			result = append(result, field.Invalid(field.NewPath("spec", "image"), app.Spec.Image, err.Error()))
			return
		}

		if !local {
			if _, autoUpgradeOn := autoupgrade.Mode(app.Spec); autoUpgradeOn {
				// Make sure there is a registry specified here
				// If there isn't one, return an error in order to avoid checking Docker Hub implicitly
				ref, err := name.ParseReference(checkImage, name.WithDefaultRegistry(images.NoDefaultRegistry))
				if err != nil {
					result = append(result, field.InternalError(field.NewPath("spec", "image"), err))
					return
				}

				if ref.Context().RegistryStr() == images.NoDefaultRegistry {
					result = append(result, field.Invalid(field.NewPath("spec", "image"), app.Spec.Image,
						fmt.Sprintf("could not find local image for %v - if you are trying to use a remote image, specify the full registry", app.Spec.Image)))
					return
				}
			}

			if err := s.checkRemoteAccess(ctx, app.Namespace, image); err != nil {
				result = append(result, field.Invalid(field.NewPath("spec", "image"), app.Spec.Image, err.Error()))
				return
			}
		}

		imageDetails, err := s.getImageDetails(ctx, app.Namespace, app.Spec.Profiles, app.Spec.DeployArgs, image, "")
		if err != nil {
			result = append(result, field.Invalid(field.NewPath("spec", "image"), app.Spec.Image, err.Error()))
			return
		}

		disableCheckImageAllowRules := false
		if app.Spec.Stop != nil && *app.Spec.Stop {
			// app was stopped, so we don't need to check image allow rules (this could prevent stopping an app if the image allow rules changed)
			disableCheckImageAllowRules = true
		}

		if !disableCheckImageAllowRules {
			if err := s.checkImageAllowed(ctx, app.Namespace, checkImage); err != nil {
				result = append(result, field.Forbidden(field.NewPath("spec", "image"), fmt.Sprintf("%s not allowed to run: %s", app.Spec.Image, err.Error())))
				return
			}
		}

		workloadsFromImage, err := s.getWorkloads(imageDetails)
		if err != nil {
			result = append(result, field.Invalid(field.NewPath("spec", "image"), app.Spec.Image, err.Error()))
			return
		}

		apiv1cfg, err := apiv1config.Get(ctx, s.client)
		if err != nil {
			result = append(result, field.Invalid(field.NewPath("config"), app.Spec.Image, err.Error()))
			return
		}

		errs := s.checkScheduling(ctx, app, project, workloadsFromImage, apiv1cfg.WorkloadMemoryDefault, apiv1cfg.WorkloadMemoryMaximum)
		if len(errs) != 0 {
			result = append(result, errs...)
			return
		}

		if err := validateVolumeClasses(ctx, s.client, app.Namespace, app.Spec, imageDetails.AppSpec, project); err != nil {
			result = append(result, err)
			return
		}

		var imageRequestedPerms []v1.Permissions
		imageRequestedPerms, imageGrantedPerms, err = s.getPermissions(ctx, "", app.Namespace, image, imageDetails)
		if err != nil {
			result = append(result, field.Invalid(field.NewPath("spec", "permissions"), app.Spec.Permissions, err.Error()))
			return
		}

		if err := s.checkRequestedPermsSatisfyImagePerms(app.Namespace, imageRequestedPerms, app.Spec.Permissions, imageGrantedPerms); err != nil {
			result = append(result, field.Invalid(field.NewPath("spec", "permissions"), app.Spec.Permissions, err.Error()))
			return
		}

		app.Spec.ImageGrantedPermissions = imageGrantedPerms
	}

	if err := s.checkPermissionsForPrivilegeEscalation(ctx, app.Spec.Permissions); err != nil {
		result = append(result, field.Invalid(field.NewPath("spec", "permissions"), app.Spec.Permissions, err.Error()))
	}

	return result
}

func (s *Validator) ValidateUpdate(ctx context.Context, obj, old runtime.Object) (result field.ErrorList) {
	newParams := obj.(*apiv1.App)
	oldParams := old.(*apiv1.App)

	if !s.allowNestedUpdate {
		if len(strings.Split(newParams.Name, ".")) == 2 && newParams.Name == oldParams.Name && newParams.Labels[labels.AcornParentAcornName] != "" {
			result = append(result, field.Invalid(field.NewPath("metadata", "name"), newParams.Name, "To update a nested Acorn or a service, update the parent Acorn instead."))
			return result
		}
	}

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

	return nil
}

func (s *Validator) validateRegion(app *apiv1.App, project *v1.ProjectInstance) error {
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

func (s *Validator) checkSARs(ctx context.Context, sars []sarRequest) error {
	eg, ctx := errgroup.WithContext(ctx)
	eg.SetLimit(5)
	for _, sar := range sars {
		sar := sar
		eg.Go(func() error {
			return s.check(ctx, &sar.SAR, sar.Rule)
		})
	}

	return eg.Wait()
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

type sarRequest struct {
	SAR  authv1.SubjectAccessReview
	Rule v1.PolicyRule
}

func (s *Validator) getSARNonResourceRole(sar *authv1.SubjectAccessReview, rule v1.PolicyRule) (result []sarRequest, _ error) {
	if len(rule.Verbs) == 0 {
		return nil, fmt.Errorf("can not deploy acorn due to requesting role with empty verbs")
	}

	if len(rule.APIGroups) != 0 {
		return nil, fmt.Errorf("can not deploy acorn due to requesting role nonResourceURLs %v and non-empty apiGroups set %v", rule.NonResourceURLs, rule.APIGroups)
	}

	for _, url := range rule.NonResourceURLs {
		for _, verb := range rule.Verbs {
			sar := sar.DeepCopy()
			sar.Spec.NonResourceAttributes = &authv1.NonResourceAttributes{
				Path: url,
				Verb: verb,
			}
			result = append(result, sarRequest{
				SAR:  *sar,
				Rule: rule,
			})
		}
	}

	return result, nil
}

func (s *Validator) getSARResourceRole(sar *authv1.SubjectAccessReview, rule v1.PolicyRule, namespace string) (result []sarRequest, _ error) {
	if len(rule.APIGroups) == 0 {
		return nil, fmt.Errorf("can not deploy acorn due to requesting role with empty apiGroups")
	}
	if len(rule.Verbs) == 0 {
		return nil, fmt.Errorf("can not deploy acorn due to requesting role with empty verbs")
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
					result = append(result, sarRequest{
						SAR:  *sar,
						Rule: rule,
					})
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
						result = append(result, sarRequest{
							SAR:  *sar,
							Rule: rule,
						})
					}
				}
			}
		}
	}

	return result, nil
}

func (s *Validator) getSARs(sar *authv1.SubjectAccessReview, rules []v1.PolicyRule, currentNamespace string) (result []sarRequest, _ error) {
	var errs []error
	for _, rule := range rules {
		if len(rule.NonResourceURLs) > 0 {
			requests, err := s.getSARNonResourceRole(sar, rule)
			if err == nil {
				result = append(result, requests...)
			} else {
				errs = append(errs, err)
			}
		} else {
			for _, namespace := range rule.ResolveNamespaces(currentNamespace) {
				requests, err := s.getSARResourceRole(sar, rule, namespace)
				if err == nil {
					result = append(result, requests...)
				} else {
					errs = append(errs, err)
				}
			}
		}
	}
	return result, merr.NewErrors(errs...)
}

// checkRequestedPermsSatisfyImagePerms checks that the user requested permissions are enough to satisfy the permissions
// specified by the image's Acornfile
func (s *Validator) checkRequestedPermsSatisfyImagePerms(currentNamespace string, requestedPerms []v1.Permissions, grantedUserPerms, grantedImagePerms []v1.Permissions) error {
	grantedPermsByService := v1.GroupByServiceName(append(grantedUserPerms, grantedImagePerms...))

	for serviceName, requestedPerm := range v1.GroupByServiceName(requestedPerms) {
		if _, granted := v1.Grants(grantedPermsByService[serviceName], currentNamespace, requestedPerm); !granted {
			return &client.ErrRulesNeeded{
				Permissions: requestedPerms,
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

	var (
		sars []sarRequest
	)
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
		requests, err := s.getSARs(sar, perm.GetRules(), ns)
		if err != nil {
			return err
		}
		sars = append(sars, requests...)
	}

	return s.checkSARs(ctx, sars)
}

// checkScheduling must use apiv1.ComputeCLass to validate the scheduling instead of the Instance counterparts.
func (s *Validator) checkScheduling(ctx context.Context, params *apiv1.App, project *v1.ProjectInstance, workloads map[string]v1.Container, specMemDefault, specMemMaximum *int64) []*field.Error {
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

func validateVolumeClasses(ctx context.Context, c kclient.Client, namespace string, appInstanceSpec v1.AppInstanceSpec, appSpec *v1.AppSpec, project *v1.ProjectInstance) *field.Error {
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

func (s *Validator) imageGrants(ctx context.Context, details client.ImageDetails, perms []v1.Permissions) (bool, error) {
	sar := &authv1.SubjectAccessReview{
		Spec: authv1.SubjectAccessReviewSpec{
			User: "image://" + details.AppImage.Name,
			Extra: map[string]authv1.ExtraValue{
				"digest": {
					details.AppImage.Digest,
				},
			},
		},
	}

	ns, _ := request.NamespaceFrom(ctx)
	var sars []sarRequest
	for _, perm := range perms {
		newSARs, err := s.getSARs(sar, perm.GetRules(), ns)
		if err != nil {
			return false, err
		}
		sars = append(sars, newSARs...)
	}

	err := s.checkSARs(ctx, sars)
	if ruleErr := (*client.ErrRulesNeeded)(nil); errors.As(err, &ruleErr) {
		return false, nil
	}
	if ruleErr := (*client.ErrNotAuthorized)(nil); errors.As(err, &ruleErr) {
		return false, nil
	}

	return false, err
}

func (s *Validator) getPermissions(ctx context.Context, servicePrefix, namespace, image string, details *client.ImageDetails) (result, granted []v1.Permissions, _ error) {
	var imagePermissions []v1.Permissions
	imagePermissions = append(imagePermissions, buildPermissionsFrom(servicePrefix, details.AppSpec.Containers)...)
	imagePermissions = append(imagePermissions, buildPermissionsFrom(servicePrefix, details.AppSpec.Jobs)...)

	if isGranted, err := s.imageGrants(ctx, *details, imagePermissions); err != nil {
		return nil, nil, err
	} else if isGranted {
		granted = append(granted, imagePermissions...)
	} else {
		result = append(result, imagePermissions...)
	}

	subResults, subGranted, err := s.buildNestedImagePermissions(ctx, servicePrefix, details.AppSpec, namespace, image, details.AppImage.ImageData)
	if err != nil {
		return nil, nil, err
	}

	result = append(result, subResults...)
	granted = append(granted, subGranted...)

	return result, granted, nil
}

func (s *Validator) getImageRequestedPermissions(ctx context.Context, servicePrefix string, profiles []string, args map[string]any, namespace, image, nestedName, nestedDigest string) (imageRequested, imageGranted []v1.Permissions, _ error) {
	details, err := s.getImageDetails(ctx, namespace, profiles, args, image, nestedDigest)
	if err != nil {
		return nil, nil, err
	}
	if nestedName == "" {
		details.AppImage.Name = image
	} else {
		details.AppImage.Name = nestedName
	}
	return s.getPermissions(ctx, servicePrefix, namespace, image, details)
}

func (s *Validator) mergeInRules(servicePrefix string, existing []v1.Permissions, newRules map[string]v1.Permissions) (result []v1.Permissions) {
	result = append(result, existing...)
outerLoop:
	for _, entry := range typed.Sorted(newRules) {
		serviceName, permissions := entry.Key, entry.Value
		if !permissions.HasRules() {
			continue
		}

		newServiceName := servicePrefix + serviceName
		for i, existingRule := range result {
			if existingRule.ServiceName == newServiceName {
				existingRule.Rules = append(existingRule.Rules, permissions.GetRules()...)
				result[i] = existingRule
				continue outerLoop
			}
		}

		result = append(result, v1.Permissions{
			ServiceName: newServiceName,
			Rules:       permissions.GetRules(),
		})
	}
	return
}

func (s *Validator) buildNestedImagePermissions(ctx context.Context, servicePrefix string, app *v1.AppSpec, namespace, image string, imageData v1.ImagesData) (imageRequested, imageGranted []v1.Permissions, err error) {
	for _, entry := range typed.Sorted(app.Acorns) {
		var (
			acornName, acorn                   = entry.Key, entry.Value
			subImageRequested, subImageGranted []v1.Permissions
		)

		acornImage, ok := appdefinition.GetImageReferenceForServiceName(acornName, app, imageData)
		if !ok {
			return nil, nil, fmt.Errorf("failed to find image information for nested acorn [%s]", acornName)
		}

		if tags.IsImageDigest(acornImage) {
			subImageRequested, subImageGranted, err = s.getImageRequestedPermissions(ctx, servicePrefix+entry.Key+".", acorn.Profiles, acorn.DeployArgs, namespace, image, acorn.GetOriginalImage(), acornImage)
			if err != nil {
				return nil, nil, err
			}
		} else {
			subImageRequested, subImageGranted, err = s.getImageRequestedPermissions(ctx, servicePrefix+entry.Key+".", acorn.Profiles, acorn.DeployArgs, namespace, acornImage, "", "")
			if err != nil {
				return nil, nil, err
			}
		}

		imageGranted = append(imageGranted, subImageGranted...)

		imageRequested = append(imageRequested, subImageRequested...)
		imageRequested = s.mergeInRules(servicePrefix+acornName+".", imageRequested, acorn.Permissions)
	}

	for _, entry := range typed.Sorted(app.Services) {
		var (
			serviceName, service  = entry.Key, entry.Value
			subResult, subGranted []v1.Permissions
		)

		acornImage, ok := appdefinition.GetImageReferenceForServiceName(serviceName, app, imageData)
		if !ok {
			// not a service acorn
			continue
		}

		if tags.IsImageDigest(acornImage) {
			subResult, subGranted, err = s.getImageRequestedPermissions(ctx, servicePrefix+entry.Key+".", nil, service.ServiceArgs, namespace, image, service.GetOriginalImage(), acornImage)
			if err != nil {
				return nil, nil, err
			}
		} else {
			subResult, subGranted, err = s.getImageRequestedPermissions(ctx, servicePrefix+entry.Key+".", nil, service.ServiceArgs, namespace, acornImage, "", "")
			if err != nil {
				return nil, nil, err
			}
		}

		imageGranted = append(imageGranted, subGranted...)

		imageRequested = append(imageRequested, subResult...)
		imageRequested = s.mergeInRules(servicePrefix+serviceName+".", imageRequested, service.Permissions)
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
	return imageallowrules.CheckImageAllowed(ctx, s.client, namespace, image, digest)
}
