package appdefinition

import (
	"sort"
	"strings"

	"github.com/ibuildthecloud/baaah/pkg/meta"
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
	resp.Objects(toDeployments(appInstance)...)
}

func toEnv(env []string) (result []corev1.EnvVar) {
	for _, v := range env {
		k, v, _ := strings.Cut(v, "=")
		result = append(result, corev1.EnvVar{
			Name:  k,
			Value: v,
		})
	}
	return
}

func toContainers(name string, container v1.Container) ([]corev1.Container, []corev1.Container) {
	var (
		containers     []corev1.Container
		initContainers []corev1.Container
		sidecarNames   []string
	)

	containers = append(containers, toContainer(name, container))
	for k := range container.Sidecars {
		sidecarNames = append(sidecarNames, k)
	}
	sort.Strings(sidecarNames)

	for _, name := range sidecarNames {
		newContainer := toContainerFromSidekick(name, container.Sidecars[name])
		if container.Sidecars[name].Init {
			initContainers = append(initContainers, newContainer)
		} else {
			containers = append(containers, newContainer)
		}
	}
	return containers, initContainers
}

func toContainerFromSidekick(name string, sidecar v1.Sidecar) corev1.Container {
	return toContainer(name, v1.Container{
		Image:       sidecar.Image,
		Build:       sidecar.Build,
		Command:     sidecar.Command,
		Interactive: sidecar.Interactive,
		Entrypoint:  sidecar.Entrypoint,
		Environment: sidecar.Environment,
		WorkingDir:  sidecar.WorkingDir,
	})
}

func toContainer(name string, container v1.Container) corev1.Container {
	return corev1.Container{
		Name:       name,
		Image:      container.Image,
		Command:    container.Entrypoint,
		Args:       container.Command,
		WorkingDir: container.WorkingDir,
		Env:        toEnv(container.Environment),
		TTY:        container.Interactive,
		Stdin:      container.Interactive,
	}
}

func toDeployment(appInstance *v1.AppInstance, name string, container v1.Container) *appsv1.Deployment {
	var replicas *int32
	if appInstance.Spec.Stop != nil && *appInstance.Spec.Stop {
		replicas = new(int32)
	}
	containers, initContainers := toContainers(name, container)
	return &appsv1.Deployment{
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
					Containers:     containers,
					InitContainers: initContainers,
				},
			},
		},
	}
}

func toDeployments(appInstance *v1.AppInstance) (result []meta.Object) {
	for name, container := range appInstance.Status.AppSpec.Containers {
		result = append(result, toDeployment(appInstance, name, container))
	}
	return result
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
