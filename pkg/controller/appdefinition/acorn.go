package appdefinition

import (
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/labels"
	"github.com/acorn-io/acorn/pkg/ports"
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
	image := resolveTagForAcorn(tag, appInstance.Labels[labels.AcornRootNamespace], acorn.Image)

	// Ensure secret gets copied
	pullSecrets.ForAcorn(acornName, image)

	acornInstance := &v1.AppInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      acornName,
			Namespace: appInstance.Status.Namespace,
			Labels: labels.Managed(appInstance,
				labels.AcornRootNamespace, appInstance.Labels[labels.AcornRootNamespace],
				labels.AcornRootPrefix, labels.RootPrefix(appInstance.Labels, appInstance.Name),
				labels.AcornAcornName, acornName),
		},
		Spec: v1.AppInstanceSpec{
			Image:       image,
			Volumes:     acorn.Volumes,
			Secrets:     acorn.Secrets,
			PublishMode: v1.PublishModeNone,
			Links:       acorn.Links,
			Profiles:    acorn.Profiles,
			DeployArgs:  acorn.DeployArgs,
			Ports:       ports.ForAcorn(appInstance, acornName),
			Stop:        appInstance.Spec.Stop,
		},
	}

	if acorn.Permissions.HasRules() {
		acornInstance.Spec.Permissions = appInstance.Spec.Permissions
	}

	return acornInstance
}
