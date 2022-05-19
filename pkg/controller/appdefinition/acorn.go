package appdefinition

import (
	"strings"

	v1 "github.com/acorn-io/acorn/pkg/apis/acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/labels"
	"github.com/acorn-io/baaah/pkg/meta"
	"github.com/acorn-io/baaah/pkg/router"
	"github.com/google/go-containerregistry/pkg/name"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func addAcorns(appInstance *v1.AppInstance, tag name.Reference, pullSecrets []corev1.LocalObjectReference, resp router.Response) {
	resp.Objects(toAcorns(appInstance, tag, pullSecrets)...)
}

func toAcorns(appInstance *v1.AppInstance, tag name.Reference, pullSecrets []corev1.LocalObjectReference) (result []meta.Object) {
	for acornName, acorn := range appInstance.Status.AppSpec.Acorns {
		result = append(result, toAcorn(appInstance, tag, pullSecrets, acornName, acorn))
	}
	return result
}

func toAcorn(appInstance *v1.AppInstance, tag name.Reference, pullSecrets []corev1.LocalObjectReference, acornName string, acorn v1.Acorn) *v1.AppInstance {
	var (
		pullSecretNames []string
		images          = map[string]string{}
		imagePrefix     = acornName + "."
	)

	for _, pullSecret := range pullSecrets {
		pullSecretNames = append(pullSecretNames, pullSecret.Name)
	}

	for k, v := range appInstance.Spec.Images {
		if strings.HasPrefix(k, imagePrefix) {
			images[strings.TrimPrefix(k, imagePrefix)] = v
		}
	}

	return &v1.AppInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      acornName,
			Namespace: appInstance.Status.Namespace,
			Labels: containerLabels(appInstance, acornName,
				labels.AcornContainerName, "",
				labels.AcornAcornName, acornName),
		},
		Spec: v1.AppInstanceSpec{
			ReattachSecrets:  appInstance.Spec.ReattachSecrets,
			ReattachVolumes:  appInstance.Spec.ReattachVolumes,
			Image:            resolveTag(appInstance, tag, acornName, v1.Container{Image: acorn.Image}),
			Volumes:          acorn.Volumes,
			Secrets:          acorn.Secrets,
			DeployParams:     acorn.Params,
			Images:           images,
			ImagePullSecrets: pullSecretNames,
			Ports:            acorn.Ports,
		},
	}
}
