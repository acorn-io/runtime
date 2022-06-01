package appdefinition

import (
	"strings"

	v1 "github.com/acorn-io/acorn/pkg/apis/acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/labels"
	"github.com/acorn-io/baaah/pkg/router"
	"github.com/google/go-containerregistry/pkg/name"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func addAcorns(appInstance *v1.AppInstance, tag name.Reference, pullSecrets *PullSecrets, resp router.Response) {
	resp.Objects(toAcorns(appInstance, tag, pullSecrets)...)
}

func toAcorns(appInstance *v1.AppInstance, tag name.Reference, pullSecrets *PullSecrets) (result []kclient.Object) {
	for acornName, acorn := range appInstance.Status.AppSpec.Acorns {
		if isLinked(appInstance, acornName) {
			continue
		}
		result = append(result, toAcorn(appInstance, tag, pullSecrets, acornName, acorn))
	}
	return result
}

func toAcorn(appInstance *v1.AppInstance, tag name.Reference, pullSecrets *PullSecrets, acornName string, acorn v1.Acorn) *v1.AppInstance {
	var (
		images      = map[string]string{}
		imagePrefix = acornName + "."
	)

	for k, v := range appInstance.Spec.Images {
		if strings.HasPrefix(k, imagePrefix) {
			images[strings.TrimPrefix(k, imagePrefix)] = v
		}
	}

	image := resolveTag(appInstance, tag, acornName, v1.Container{Image: acorn.Image})
	// Ensure secret gets copied
	pullSecrets.ForAcorn(acornName, image)

	return &v1.AppInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      acornName,
			Namespace: appInstance.Status.Namespace,
			Labels: containerLabels(appInstance, acornName,
				labels.AcornContainerName, "",
				labels.AcornAcornName, acornName),
		},
		Spec: v1.AppInstanceSpec{
			ReattachSecrets: appInstance.Spec.ReattachSecrets,
			ReattachVolumes: appInstance.Spec.ReattachVolumes,
			Image:           image,
			Volumes:         acorn.Volumes,
			Secrets:         acorn.Secrets,
			DeployParams:    acorn.Params,
			Images:          images,
			Ports:           acorn.Ports,
		},
	}
}
