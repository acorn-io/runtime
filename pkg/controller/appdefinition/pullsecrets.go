package appdefinition

import (
	"github.com/ibuildthecloud/baaah/pkg/router"
	v1 "github.com/ibuildthecloud/herd/pkg/apis/herd-project.io/v1"
	"github.com/ibuildthecloud/herd/pkg/labels"
	"github.com/ibuildthecloud/herd/pkg/pullsecret"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func pullSecrets(req router.Request, appInstance *v1.AppInstance, resp router.Response) ([]corev1.LocalObjectReference, error) {
	secrets, err := pullsecret.ForNamespace(req.Client, appInstance.Namespace, appInstance.Spec.ImagePullSecrets...)
	if err != nil {
		return nil, err
	}

	if len(secrets) == 0 {
		return nil, nil
	}

	var (
		result []corev1.LocalObjectReference
		suffix = "-" + string(appInstance.UID[:8])
	)

	for _, secret := range secrets {
		sec := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      secret.Name + suffix,
				Namespace: appInstance.Status.Namespace,
				Labels: map[string]string{
					labels.HerdAppName:      appInstance.Name,
					labels.HerdAppNamespace: appInstance.Namespace,
					labels.HerdManaged:      "true",
				},
			},
			Data: secret.Data,
			Type: secret.Type,
		}
		result = append(result, corev1.LocalObjectReference{
			Name: sec.Name,
		})
		resp.Objects(sec)
	}

	return result, nil
}
