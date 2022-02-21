package appdefinition

import (
	"github.com/ibuildthecloud/baaah/pkg/router"
	v1 "github.com/ibuildthecloud/herd/pkg/apis/herd-project.io/v1"
	"github.com/ibuildthecloud/herd/pkg/labels"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func DeploySpec(req router.Request, resp router.Response) error {
	appInstance := req.Object.(*v1.AppInstance)
	if appInstance.Status.Namespace == "" {
		return nil
	}
	addNamespace(appInstance, resp)
	addDeployments(appInstance, resp)
	return nil
}

func addDeployments(appInstance *v1.AppInstance, resp router.Response) {
	var replicas *int32
	if appInstance.Spec.Stop != nil && *appInstance.Spec.Stop {
		replicas = new(int32)
	}
	for name, container := range appInstance.Status.AppSpec.Containers {
		resp.Objects(&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: appInstance.Status.Namespace,
				Labels: map[string]string{
					labels.HerdAppName:       appInstance.Name,
					labels.HerdAppNamespace:  appInstance.Namespace,
					labels.HerdContainerName: name,
				},
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: replicas,
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						labels.HerdAppName:       appInstance.Name,
						labels.HerdAppNamespace:  appInstance.Namespace,
						labels.HerdContainerName: name,
					},
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							labels.HerdAppName:       appInstance.Name,
							labels.HerdAppNamespace:  appInstance.Namespace,
							labels.HerdContainerName: name,
							labels.HerdAppPod:        "true",
						},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  name,
								Image: container.Image,
							},
						},
					},
				},
			},
		})
	}
}

func addNamespace(appInstance *v1.AppInstance, resp router.Response) {
	resp.Objects(&corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: appInstance.Status.Namespace,
			Labels: map[string]string{
				labels.HerdAppName:      appInstance.Name,
				labels.HerdAppNamespace: appInstance.Namespace,
			},
		},
	})
}
