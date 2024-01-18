package apps

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/acorn-io/baaah/pkg/merr"
	"github.com/acorn-io/baaah/pkg/typed"
	"github.com/acorn-io/mink/pkg/strategy"
	"github.com/acorn-io/mink/pkg/types"
	"github.com/acorn-io/namegenerator"
	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/autoupgrade"
	"github.com/acorn-io/runtime/pkg/client"
	"github.com/acorn-io/runtime/pkg/computeclasses"
	apiv1config "github.com/acorn-io/runtime/pkg/config"
	"github.com/acorn-io/runtime/pkg/imagerules"
	"github.com/acorn-io/runtime/pkg/images"
	"github.com/acorn-io/runtime/pkg/imagesystem"
	"github.com/acorn-io/runtime/pkg/labels"
	"github.com/acorn-io/runtime/pkg/pullsecret"
	"github.com/acorn-io/runtime/pkg/tags"
	"github.com/acorn-io/runtime/pkg/volume"
	"github.com/acorn-io/z"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"golang.org/x/exp/slices"
	"golang.org/x/sync/errgroup"
	authv1 "k8s.io/api/authorization/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/apiserver/pkg/endpoints/request"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	nameGenerator = namegenerator.NewNameGenerator(time.Now().UnixNano())
)

type Validator struct {
	client            kclient.Client
	clientFactory     *client.Factory
	getter            strategy.Getter
	allowNestedUpdate bool
	transport         http.RoundTripper
	rbac              *RBACValidator
}

type RBACValidator struct {
	client kclient.Client
}

func NewRBACValidator(client kclient.Client) *RBACValidator {
	return &RBACValidator{
		client: client,
	}
}

func NewValidator(client kclient.Client, clientFactory *client.Factory, deleter strategy.Getter, transport http.RoundTripper) *Validator {
	return &Validator{
		client:        client,
		clientFactory: clientFactory,
		getter:        deleter,
		transport:     transport,
		rbac:          NewRBACValidator(client),
	}
}

func (s *Validator) PrepareForCreate(ctx context.Context, obj runtime.Object) {
	r := obj.(types.Object)
	if r.GetName() == "" && r.GetGenerateName() == "" {
		r.SetName(nameGenerator.Generate())
	}
}

func (s *Validator) ValidateName(_ context.Context, obj runtime.Object) (result field.ErrorList) {
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
	return s.getter.Get(ctx, namespace, name)
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

		imageDetails, err := s.getImageDetails(ctx, app.Namespace, app.Spec.GetProfiles(app.Status.GetDevMode()), app.Spec.DeployArgs.GetData(), image)
		if err != nil {
			result = append(result, field.Invalid(field.NewPath("spec", "image"), app.Spec.Image, err.Error()))
			return
		}

		if !z.Dereference(app.Spec.Stop) {
			// App was stopped, so we don't need to check image allow rules or compute classes
			// (this could prevent stopping an app if either of these have changed)
			if err := imagerules.CheckImageAllowed(ctx, s.client, app.Namespace, checkImage, image, imageDetails.AppImage.Digest, remote.WithTransport(s.transport)); err != nil {
				result = append(result, field.Forbidden(field.NewPath("spec", "image"), fmt.Sprintf("%s not allowed to run: %s", app.Spec.Image, err.Error())))
				return
			}

			workloadsFromImage, err := s.getWorkloads(imageDetails)
			if err != nil {
				result = append(result, field.Invalid(field.NewPath("spec", "image"), app.Spec.Image, err.Error()))
				return
			}

			var apiv1cfg *apiv1.Config
			apiv1cfg, err = apiv1config.Get(ctx, s.client)
			if err != nil {
				result = append(result, field.Invalid(field.NewPath("config"), app.Spec.Image, err.Error()))
				return
			}

			errs := s.checkScheduling(ctx,
				app,
				project,
				workloadsFromImage,
				apiv1cfg.WorkloadMemoryDefault,
				apiv1cfg.WorkloadMemoryMaximum)
			if len(errs) != 0 {
				result = append(result, errs...)
				return
			}
		}

		if err := validateVolumeClasses(ctx, s.client, app.Namespace, app.Spec, imageDetails.AppSpec, project); err != nil {
			result = append(result, err)
			return
		}

		var imageRejectedPerms []v1.Permissions
		imageGrantedPerms, imageRejectedPerms, err = s.imageGrants(ctx, imageDetails, checkImage)
		if err != nil {
			result = append(result, field.Invalid(field.NewPath("spec", "permissions"), app.Spec.GrantedPermissions, err.Error()))
			return
		}

		if err := s.checkRequestedPermsSatisfyImagePerms(app.Namespace, imageRejectedPerms, app.Spec.GrantedPermissions); err != nil {
			result = append(result, field.Invalid(field.NewPath("spec", "permissions"), app.Spec.GrantedPermissions, err.Error()))
			return
		}

		app.Spec.ImageGrantedPermissions = imageGrantedPerms
	}

	if _, rejected, err := s.rbac.CheckPermissionsForPrivilegeEscalation(ctx, app.Spec.GrantedPermissions); err != nil {
		result = append(result, field.Invalid(field.NewPath("spec", "permissions"), app.Spec.GrantedPermissions, err.Error()))
	} else if len(rejected) > 0 {
		result = append(result, field.Invalid(field.NewPath("spec", "permissions"), app.Spec.GrantedPermissions, z.Pointer(client.ErrNotAuthorized{
			Permissions: rejected,
		}).Error()))
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

func sarToPolicyRules(sars []sarRequest) []v1.PolicyRule {
	result := make([]v1.PolicyRule, 0, len(sars))
	sort.Slice(sars, func(i, j int) bool {
		return sars[i].Order < sars[j].Order
	})
	for _, sar := range sars {
		result = append(result, sar.Rule)
	}
	return result
}

func (s *RBACValidator) checkSARs(ctx context.Context, sars []sarRequest) (granted, rejected []v1.Permissions, _ error) {
	var (
		lock        sync.Mutex
		grantedMap  = map[string][]sarRequest{}
		rejectedMap = map[string][]sarRequest{}
	)

	eg, ctx := errgroup.WithContext(ctx)
	eg.SetLimit(5)
	for i, sar := range sars {
		sar := sar
		sar.Order = i
		eg.Go(func() error {
			ok, err := s.check(ctx, &sar.SAR)
			if err != nil {
				return err
			}
			lock.Lock()
			defer lock.Unlock()
			if ok {
				grantedMap[sar.ServiceName] = append(grantedMap[sar.ServiceName], sar)
			} else {
				rejectedMap[sar.ServiceName] = append(rejectedMap[sar.ServiceName], sar)
			}
			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return nil, nil, err
	}

	for _, key := range typed.SortedKeys(grantedMap) {
		granted = append(granted, v1.Permissions{
			ServiceName: key,
			Rules:       sarToPolicyRules(grantedMap[key]),
		})
	}

	for _, key := range typed.SortedKeys(rejectedMap) {
		rejected = append(rejected, v1.Permissions{
			ServiceName: key,
			Rules:       sarToPolicyRules(rejectedMap[key]),
		})
	}

	return granted, rejected, nil
}

func (s *RBACValidator) check(ctx context.Context, sar *authv1.SubjectAccessReview) (bool, error) {
	err := s.client.Create(ctx, sar)
	if err != nil {
		return false, err
	}
	return sar.Status.Allowed, nil
}

type sarRequest struct {
	SAR         authv1.SubjectAccessReview
	ServiceName string
	Rule        v1.PolicyRule
	Order       int
}

func (s *RBACValidator) getSARNonResourceRole(sar *authv1.SubjectAccessReview, serviceName string, rule v1.PolicyRule) (result []sarRequest, _ error) {
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
				ServiceName: serviceName,
				SAR:         *sar,
				Rule: v1.PolicyRule{
					PolicyRule: rbacv1.PolicyRule{
						Verbs:           []string{verb},
						NonResourceURLs: []string{url},
					},
					Scopes: []string{"cluster"},
				},
			})
		}
	}

	return result, nil
}

func toScope(namespace, currentNamespace string) []string {
	if namespace == currentNamespace {
		return []string{"project"}
	}
	if namespace == "" {
		return []string{"account"}
	}
	return []string{"project:" + namespace}
}

func (s *RBACValidator) getSARResourceRole(sar *authv1.SubjectAccessReview, serviceName string, rule v1.PolicyRule, namespace, currentNamespace string) (result []sarRequest, _ error) {
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
			for _, resourceWithSubResource := range rule.Resources {
				resource, subResource, _ := strings.Cut(resourceWithSubResource, "/")
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
						ServiceName: serviceName,
						SAR:         *sar,
						Rule: v1.PolicyRule{
							PolicyRule: rbacv1.PolicyRule{
								Verbs:     []string{verb},
								APIGroups: []string{apiGroup},
								Resources: []string{resourceWithSubResource},
							},
							Scopes: toScope(namespace, currentNamespace),
						},
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
							ServiceName: serviceName,
							SAR:         *sar,
							Rule: v1.PolicyRule{
								PolicyRule: rbacv1.PolicyRule{
									Verbs:         []string{verb},
									APIGroups:     []string{apiGroup},
									Resources:     []string{resourceWithSubResource},
									ResourceNames: []string{resourceName},
								},
								Scopes: toScope(namespace, currentNamespace),
							},
						})
					}
				}
			}
		}
	}

	return result, nil
}

func (s *RBACValidator) getSARs(sar *authv1.SubjectAccessReview, perm v1.Permissions, currentNamespace string) (result []sarRequest, _ error) {
	var errs []error
	for _, rule := range perm.GetRules() {
		if len(rule.NonResourceURLs) > 0 {
			requests, err := s.getSARNonResourceRole(sar, perm.ServiceName, rule)
			if err == nil {
				result = append(result, requests...)
			} else {
				errs = append(errs, err)
			}
		} else {
			for _, namespace := range rule.ResolveNamespaces(currentNamespace) {
				requests, err := s.getSARResourceRole(sar, perm.ServiceName, rule, namespace, currentNamespace)
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
func (s *Validator) checkRequestedPermsSatisfyImagePerms(currentNamespace string, requestedPerms, grantedUserPerms []v1.Permissions) error {
	missing, ok := v1.GrantsAll(currentNamespace, requestedPerms, grantedUserPerms)
	if !ok {
		return &client.ErrRulesNeeded{
			Missing:     missing,
			Permissions: requestedPerms,
		}
	}
	return nil
}

// CheckPermissionsForPrivilegeEscalation is an actual RBAC check to prevent privilege escalation. The user making the request must have the
// permissions that they are requesting the app gets
func (s *RBACValidator) CheckPermissionsForPrivilegeEscalation(ctx context.Context, requestedPerms []v1.Permissions) (granted, rejected []v1.Permissions, _ error) {
	user, ok := request.UserFrom(ctx)
	if !ok {
		return nil, nil, fmt.Errorf("failed to find active user to check current privileges")
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
		requests, err := s.getSARs(sar, perm, ns)
		if err != nil {
			return nil, nil, err
		}
		sars = append(sars, requests...)
	}

	return s.checkSARs(ctx, sars)
}

// checkScheduling must use apiv1.ComputeClass to validate the scheduling instead of the Instance counterparts.
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
			wcMemory, err := computeclasses.ParseComputeClassMemoryAPI(cc.Memory)
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

func (s *Validator) imageSAR(ns, imageName, imageDigest string, perms []v1.Permissions) (result []sarRequest, _ error) {
	sar := &authv1.SubjectAccessReview{
		Spec: authv1.SubjectAccessReviewSpec{
			User: "image://" + imageName,
			Extra: map[string]authv1.ExtraValue{
				"digest": {
					imageDigest,
				},
			},
		},
	}

	for _, perm := range perms {
		newSARs, err := s.rbac.getSARs(sar, perm, ns)
		if err != nil {
			return nil, err
		}
		result = append(result, newSARs...)
	}
	return
}

func (s *Validator) imageGrants(ctx context.Context, details *apiv1.ImageDetails, imageNameOverride string) (granted, rejected []v1.Permissions, _ error) {
	defer func() {
		granted = v1.SimplifySet(granted)
		rejected = v1.SimplifySet(rejected)
	}()

	ns, _ := request.NamespaceFrom(ctx)
	var sars []sarRequest

	imageName := details.AppImage.Name
	if imageNameOverride != "" {
		imageName = imageNameOverride
	}

	newSars, err := s.imageSAR(ns, imageName, details.AppImage.Digest, details.Permissions)
	if err != nil {
		return nil, nil, err
	}
	sars = append(sars, newSars...)

	for _, nested := range details.NestedImages {
		newSars, err := s.imageSAR(ns, nested.ImageName, nested.Digest, nested.Permissions)
		if err != nil {
			return nil, nil, err
		}
		sars = append(sars, newSars...)
	}

	return s.rbac.checkSARs(ctx, sars)
}

func (s *Validator) getWorkloads(details *apiv1.ImageDetails) (map[string]v1.Container, error) {
	result := make(map[string]v1.Container, len(details.AppSpec.Containers)+len(details.AppSpec.Jobs))
	for workload, container := range details.AppSpec.Containers {
		result[workload] = container
		for sidecarWorkload, sidecarContainer := range container.Sidecars {
			result[sidecarWorkload] = sidecarContainer
		}
	}
	for workload, function := range details.AppSpec.Functions {
		result[workload] = function
		for sidecarWorkload, sidecarContainer := range function.Sidecars {
			result[sidecarWorkload] = sidecarContainer
		}
	}
	for workload, container := range details.AppSpec.Jobs {
		result[workload] = container
	}

	return result, nil
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

func (s *Validator) getImageDetails(ctx context.Context, namespace string, profiles []string, args map[string]any, image string) (*apiv1.ImageDetails, error) {
	details := &apiv1.ImageDetails{
		DeployArgs:    v1.NewGenericMap(args),
		Profiles:      profiles,
		IncludeNested: true,
	}
	err := s.client.SubResource("details").Create(ctx, &apiv1.Image{
		ObjectMeta: metav1.ObjectMeta{
			Name:      strings.ReplaceAll(image, "/", "+"),
			Namespace: namespace,
		},
	}, details)
	if err != nil {
		return nil, err
	}

	if details.GetParseError() != "" {
		return nil, errors.New(details.GetParseError())
	}

	return details, nil
}
