package appdefinition

import (
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/awspermissions"
	"github.com/acorn-io/baaah/pkg/router"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func toServiceAccount(req router.Request, saName string, labelMap, annotations map[string]string, appInstance *v1.AppInstance) (result kclient.Object, _ error) {
	sa := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:        saName,
			Namespace:   appInstance.Status.Namespace,
			Labels:      labelMap,
			Annotations: annotations,
		},
	}
	return sa, addAWS(req, appInstance, sa)
}

func addAWS(req router.Request, appInstance *v1.AppInstance, sa *corev1.ServiceAccount) error {
	perms := v1.Permissions{
		ServiceName: sa.Name,
	}
	for _, perm := range appInstance.Spec.Permissions {
		if perm.ServiceName == sa.Name {
			perms.Rules = append(perms.Rules, perm.Rules...)
		}
	}

	annotations, err := awspermissions.AWSAnnotations(req.Ctx, req.Client, appInstance, perms, sa.Name)
	if err != nil {
		return err
	}

	sa.Annotations = labels.Merge(sa.Annotations, annotations)
	return nil
}
