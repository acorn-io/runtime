package appdefinition

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/acorn-io/baaah/pkg/apply"
	name2 "github.com/acorn-io/baaah/pkg/name"
	"github.com/acorn-io/baaah/pkg/router"
	"github.com/acorn-io/baaah/pkg/typed"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/autoupgrade"
	"github.com/acorn-io/runtime/pkg/controller/jobs"
	"github.com/acorn-io/runtime/pkg/images"
	"github.com/acorn-io/runtime/pkg/labels"
	"github.com/acorn-io/runtime/pkg/ports"
	"github.com/acorn-io/runtime/pkg/publicname"
	"github.com/google/go-containerregistry/pkg/name"
	"golang.org/x/exp/slices"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func addAcorns(req router.Request, appInstance *v1.AppInstance, tag name.Reference, pullSecrets *PullSecrets, resp router.Response) error {
	for _, acorn := range toAcorns(appInstance, tag, pullSecrets) {
		var devSession v1.DevSessionInstance
		err := req.Get(&devSession, acorn.Namespace, acorn.Name)
		if err == nil {
			// Don't update app in dev mode
			acorn.Annotations[apply.AnnotationUpdate] = "false"
		} else if !apierrors.IsNotFound(err) {
			return err
		}
		var existingApp v1.AppInstance
		err = req.Get(&existingApp, acorn.Namespace, acorn.Name)
		if err == nil {
			if slices.Contains(existingApp.Finalizers, jobs.DestroyJobFinalizer) {
				acorn.Annotations[apply.AnnotationPrune] = "false"
			}
		} else if !apierrors.IsNotFound(err) {
			return err
		}

		if strings.Count(acorn.Labels[labels.AcornPublicName], ".") > 10 {
			return fmt.Errorf("max limit of 10 nested acorns exceeded")
		}

		resp.Objects(acorn)
	}
	return nil
}

func toAcorns(appInstance *v1.AppInstance, tag name.Reference, pullSecrets *PullSecrets) (result []*v1.AppInstance) {
	for _, entry := range typed.Sorted(appInstance.Status.AppSpec.Acorns) {
		acornName, acorn := entry.Key, entry.Value
		if ports.IsLinked(appInstance, acornName) {
			continue
		}
		result = append(result, toAcorn(appInstance, tag, pullSecrets, acornName, acorn))
	}
	for _, entry := range typed.Sorted(appInstance.Status.AppSpec.Services) {
		serviceName, service := entry.Key, entry.Value
		if ports.IsLinked(appInstance, serviceName) || service.Image == "" {
			continue
		}
		result = append(result, toAcorn(appInstance, tag, pullSecrets, serviceName, v1.Acorn{
			Labels:              service.Labels,
			Annotations:         service.Annotations,
			Image:               service.Image,
			Build:               service.Build,
			DeployArgs:          service.ServiceArgs,
			Environment:         service.Environment,
			Secrets:             service.Secrets,
			Links:               service.Links,
			AutoUpgrade:         service.AutoUpgrade,
			NotifyUpgrade:       service.NotifyUpgrade,
			AutoUpgradeInterval: service.AutoUpgradeInterval,
			Memory:              service.Memory,
			ComputeClasses:      service.ComputeClasses,
		}))
	}
	return result
}

func scopeSecrets(app *v1.AppInstance, bindings v1.SecretBindings) (result v1.SecretBindings) {
	for _, binding := range bindings {
		binding.Secret = publicname.Get(app) + "." + binding.Secret
		result = append(result, binding)
	}
	return
}

func trimPrefixComputeClass(app *v1.AppInstance, compute v1.ComputeClassMap, name string) (result v1.ComputeClassMap) {
	prefix := name + "."
	result = map[string]string{}
	for k, v := range compute {
		result[k] = v
	}

	// add default first to maintain idempotency
	for id, class := range app.Spec.ComputeClasses {
		if id == "" {
			result[""] = class
		}
	}

	for id, class := range app.Spec.ComputeClasses {
		if id == "" {
			continue
		}
		if strings.HasPrefix(id, prefix) {
			result[strings.TrimPrefix(id, prefix)] = class
		} else if id == name {
			result[""] = class
		}
	}

	return
}

func trimPrefixMemory(app *v1.AppInstance, memory v1.MemoryMap, name string) (result v1.MemoryMap) {
	prefix := name + "."
	result = map[string]*int64{}
	for k, v := range memory {
		result[k] = v
	}

	// add default first to maintain idempotency
	for id, mem := range app.Spec.Memory {
		if id == "" {
			result[""] = mem
		}
	}

	for id, mem := range app.Spec.Memory {
		if strings.HasPrefix(id, prefix) {
			result[strings.TrimPrefix(id, prefix)] = mem
		} else if id == name {
			result[""] = mem
		}
	}

	return
}

func scopeLinks(app *v1.AppInstance, bindings v1.ServiceBindings) (result v1.ServiceBindings) {
	for _, binding := range bindings {
		binding.Service = publicname.Get(app) + "." + binding.Service
		result = append(result, binding)
	}
	return
}

func toAcorn(appInstance *v1.AppInstance, tag name.Reference, pullSecrets *PullSecrets, acornName string, acorn v1.Acorn) *v1.AppInstance {
	var image string
	pattern, isPattern := autoupgrade.AutoUpgradePattern(acorn.Image)
	if isPattern {
		image = acorn.Image

		// remove the autoupgrade pattern from the end of the image for resolving the pull secret
		// the registry is all that really matters for the pull secret so this is safe to do
		pullSecrets.ForAcorn(acornName, strings.TrimSuffix(image, ":"+pattern))
	} else {
		image = images.ResolveTag(tag, acorn.Image)
		if strings.HasPrefix(acorn.Image, "sha256:") {
			image = strings.TrimPrefix(acorn.Image, "sha256:")
		}

		pullSecrets.ForAcorn(acornName, image)
	}

	labelMap := labels.Merge(appInstanceScoped(acornName, appInstance.Status.AppSpec.Labels, appInstance.Spec.Labels, acorn.Labels),
		labels.Managed(appInstance,
			labels.AcornAcornName, acornName,
			labels.AcornParentAcornName, appInstance.Name,
			labels.AcornPublicName, publicname.ForChild(appInstance, acornName)))

	publishMode := appInstance.Spec.PublishMode
	if publishMode == "" {
		publishMode = acorn.PublishMode
	}

	acornInstance := &v1.AppInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name2.SafeHashConcatName(appInstance.Name, acornName),
			Namespace: appInstance.Namespace,
			Labels:    labelMap,
			Annotations: labels.Merge(appInstanceScoped(acornName, appInstance.Status.AppSpec.Annotations, appInstance.Spec.Annotations, acorn.Annotations),
				map[string]string{labels.AcornAppGeneration: strconv.FormatInt(appInstance.Generation, 10)}),
		},
		Spec: v1.AppInstanceSpec{
			Region:                 appInstance.GetRegion(),
			Labels:                 append(acorn.Labels, appInstance.Spec.Labels...),
			Annotations:            append(acorn.Annotations, appInstance.Spec.Annotations...),
			Image:                  image,
			Volumes:                acorn.Volumes,
			Secrets:                scopeSecrets(appInstance, acorn.Secrets),
			PublishMode:            publishMode,
			Links:                  scopeLinks(appInstance, acorn.Links),
			Profiles:               acorn.Profiles,
			DeployArgs:             acorn.DeployArgs,
			Publish:                acorn.Publish,
			Stop:                   typed.Pointer(appInstance.GetStopped()),
			Environment:            append(acorn.Environment, trimEnvPrefix(appInstance.Spec.Environment, acornName)...),
			UserGrantedPermissions: trimPermPrefix(appInstance.Spec.GetGrantedPermissions(), acornName),
			AutoUpgrade:            acorn.AutoUpgrade,
			AutoUpgradeInterval:    acorn.AutoUpgradeInterval,
			NotifyUpgrade:          acorn.NotifyUpgrade,
			ComputeClasses:         trimPrefixComputeClass(appInstance, acorn.ComputeClasses, acornName),
			Memory:                 trimPrefixMemory(appInstance, acorn.Memory, acornName),
		},
	}

	// Only set the original image annotation if auto-upgrade is off. Setting the original image annotation
	// on auto-upgrade apps will cause the pattern to be shown to the user instead of the actual image, which is bad.
	if _, on := autoupgrade.Mode(acornInstance.Spec); !on {
		acornInstance.Annotations[labels.AcornOriginalImage] = acorn.GetOriginalImage()
	}

	return acornInstance
}

func trimEnvPrefix(envs []v1.NameValue, name string) (result []v1.NameValue) {
	prefix := name + "."
	for _, env := range envs {
		if strings.Contains(env.Name, ".") {
			if strings.HasPrefix(env.Name, prefix) {
				result = append(result, v1.NameValue{
					Name:  strings.TrimPrefix(env.Name, prefix),
					Value: env.Value,
				})
			}
		} else {
			result = append(result, env)
		}
	}
	return
}

func trimPermPrefix(perms []v1.Permissions, name string) (result []v1.Permissions) {
	prefix := name + "."
	for _, perm := range perms {
		if strings.HasPrefix(perm.ServiceName, prefix) {
			result = append(result, v1.Permissions{
				ServiceName: strings.TrimPrefix(perm.ServiceName, prefix),
				Rules:       perm.GetRules(),
			})
		}
	}
	return
}

func appInstanceScoped(acornName string, globalLabels map[string]string, appSpecLabels []v1.ScopedLabel, acornScopedLabels v1.ScopedLabels) map[string]string {
	labelMap := make(map[string]string)
	for _, s := range acornScopedLabels {
		if s.ResourceType == v1.LabelTypeMeta || (s.ResourceType == "" && s.ResourceName == "") {
			labelMap[s.Key] = s.Value
		}
	}

	labelMap = labels.Merge(labelMap, labels.GatherScoped(acornName, v1.LabelTypeAcorn, globalLabels, labelMap, appSpecLabels))
	return labels.ExcludeAcornKey(labelMap)
}
