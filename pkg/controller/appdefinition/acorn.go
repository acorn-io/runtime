package appdefinition

import (
	v1 "github.com/acorn-io/acorn/pkg/apis/acorn.io/v1"
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
		if isLinked(appInstance, acornName) {
			continue
		}
		result = append(result, toAcorn(appInstance, tag, pullSecrets, acornName, acorn))
	}
	return result
}

func toNonPublishPortBinding(portDef v1.PortDef) v1.PortBinding {
	return v1.PortBinding{
		Port:       portDef.Port,
		TargetPort: portDef.InternalPort,
		Protocol:   portDef.Protocol,
	}
}

func toPublishPortBinding(portDef v1.PortDef) v1.PortBinding {
	return v1.PortBinding{
		Port:       portDef.Port,
		TargetPort: portDef.InternalPort,
		Protocol:   portDef.Protocol,
		Publish:    true,
	}
}

func toAcorn(appInstance *v1.AppInstance, tag name.Reference, pullSecrets *PullSecrets, acornName string, acorn v1.Acorn) *v1.AppInstance {
	image := resolveTag(tag, acorn.Image)

	// Ensure secret gets copied
	pullSecrets.ForAcorn(acornName, image)

	publishPorts := ports.RemapForBinding(true, acorn.Ports, appInstance.Spec.Ports, appInstance.Spec.PublishProtocols)
	ports := append(typed.MapSlice(acorn.Ports, toNonPublishPortBinding), typed.MapSlice(publishPorts, toPublishPortBinding)...)

	return &v1.AppInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      acornName,
			Namespace: appInstance.Status.Namespace,
			Labels: labels.Managed(appInstance,
				labels.AcornAcornName, acornName),
		},
		Spec: v1.AppInstanceSpec{
			Image:      image,
			Volumes:    acorn.Volumes,
			Secrets:    acorn.Secrets,
			Services:   acorn.Services,
			DeployArgs: acorn.DeployArgs,
			Ports:      ports,
		},
	}
}
