package appdefinition

import (
	"strconv"
	"strings"

	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/autoupgrade"
	"github.com/acorn-io/acorn/pkg/images"
	"github.com/acorn-io/acorn/pkg/labels"
	"github.com/acorn-io/acorn/pkg/ports"
	"github.com/acorn-io/acorn/pkg/publicname"
	"github.com/acorn-io/baaah/pkg/apply"
	name2 "github.com/acorn-io/baaah/pkg/name"
	"github.com/acorn-io/baaah/pkg/router"
	"github.com/acorn-io/baaah/pkg/typed"
	"github.com/google/go-containerregistry/pkg/name"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func addAcorns(req router.Request, appInstance *v1.AppInstance, tag name.Reference, pullSecrets *PullSecrets, resp router.Response) {
	for _, acorn := range toAcorns(appInstance, tag, pullSecrets) {
		var devSession v1.DevSessionInstance
		err := req.Get(&devSession, acorn.Namespace, acorn.Name)
		if err == nil {
			// Don't update app in dev mode
			acorn.Annotations[apply.AnnotationUpdate] = "false"
		}
		resp.Objects(acorn)
	}
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

	originalImage := acorn.Image
	if acorn.Build != nil && acorn.Build.OriginalImage != "" {
		originalImage = acorn.Build.OriginalImage
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
				map[string]string{
					labels.AcornOriginalImage: originalImage,
					labels.AcornAppGeneration: strconv.FormatInt(appInstance.Generation, 10),
				}),
		},
		Spec: v1.AppInstanceSpec{
			Region:              appInstance.GetRegion(),
			Labels:              append(acorn.Labels, appInstance.Spec.Labels...),
			Annotations:         append(acorn.Annotations, appInstance.Spec.Annotations...),
			Image:               image,
			Volumes:             acorn.Volumes,
			Secrets:             scopeSecrets(appInstance, acorn.Secrets),
			PublishMode:         publishMode,
			Links:               scopeLinks(appInstance, acorn.Links),
			Profiles:            acorn.Profiles,
			DeployArgs:          acorn.DeployArgs,
			Publish:             acorn.Publish,
			Stop:                typed.Pointer(appInstance.GetStopped()),
			Environment:         append(acorn.Environment, appInstance.Spec.Environment...),
			Permissions:         trimPermPrefix(appInstance.Spec.Permissions, acornName),
			AutoUpgrade:         acorn.AutoUpgrade,
			AutoUpgradeInterval: acorn.AutoUpgradeInterval,
			NotifyUpgrade:       acorn.NotifyUpgrade,
		},
	}

	return acornInstance
}

func trimPermPrefix(perms []v1.Permissions, name string) (result []v1.Permissions) {
	for _, perm := range perms {
		prefix := name + "."
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
