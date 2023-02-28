package appdefinition

import (
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/images"
	"github.com/acorn-io/acorn/pkg/labels"
	"github.com/acorn-io/acorn/pkg/ports"
	name2 "github.com/acorn-io/baaah/pkg/name"
	"github.com/acorn-io/baaah/pkg/router"
	"github.com/acorn-io/baaah/pkg/typed"
	"github.com/google/go-containerregistry/pkg/name"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func addAcorns(appInstance *v1.AppInstance, tag name.Reference, pullSecrets *PullSecrets, resp router.Response) {
	resp.Objects(toAcorns(appInstance, tag, pullSecrets)...)
}

func toAcorns(appInstance *v1.AppInstance, tag name.Reference, pullSecrets *PullSecrets) (result []kclient.Object) {
	for _, entry := range typed.Sorted(appInstance.Status.AppSpec.Acorns) {
		acornName, acorn := entry.Key, entry.Value
		if ports.IsLinked(appInstance, acornName) {
			continue
		}
		result = append(result, toAcorn(appInstance, tag, pullSecrets, acornName, acorn))
	}
	return result
}

func toAcorn(appInstance *v1.AppInstance, tag name.Reference, pullSecrets *PullSecrets, acornName string, acorn v1.Acorn) *v1.AppInstance {
	image := images.ResolveTag(tag, acorn.Image)

	// Ensure secret gets copied
	pullSecrets.ForAcorn(acornName, image)

	parentName := appInstance.Labels[labels.AcornParentAcornName]
	if parentName == "" {
		parentName = appInstance.Name
	} else {
		parentName = parentName + "." + appInstance.Name
	}
	labelMap := labels.Merge(appInstanceScoped(acornName, appInstance.Status.AppSpec.Labels, appInstance.Spec.Labels, acorn.Labels),
		labels.Managed(appInstance, labels.AcornAcornName, acornName, labels.AcornParentAcornName, parentName))

	acornInstance := &v1.AppInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name2.SafeConcatName(appInstance.Name, acornName, appInstance.ShortID()),
			Namespace:   appInstance.Namespace,
			Labels:      labelMap,
			Annotations: appInstanceScoped(acornName, appInstance.Status.AppSpec.Annotations, appInstance.Spec.Annotations, acorn.Annotations),
		},
		Spec: v1.AppInstanceSpec{
			Labels:      append(acorn.Labels, appInstance.Spec.Labels...),
			Annotations: append(acorn.Annotations, appInstance.Spec.Annotations...),
			Image:       image,
			Volumes:     acorn.Volumes,
			Secrets:     acorn.Secrets,
			PublishMode: appInstance.Spec.PublishMode,
			Links:       acorn.Links,
			Profiles:    acorn.Profiles,
			DeployArgs:  acorn.DeployArgs,
			Publish:     acorn.Publish,
			Stop:        appInstance.Spec.Stop,
			Environment: append(acorn.Environment, appInstance.Spec.Environment...),
			Permissions: appInstance.Spec.Permissions,
		},
	}

	return acornInstance
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
