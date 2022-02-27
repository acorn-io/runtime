package appdefinition

import (
	"encoding/base64"
	"path"
	"strings"

	"github.com/ibuildthecloud/baaah/pkg/meta"
	"github.com/ibuildthecloud/baaah/pkg/router"
	"github.com/ibuildthecloud/baaah/pkg/typed"
	v1 "github.com/ibuildthecloud/herd/pkg/apis/herd-project.io/v1"
	"github.com/ibuildthecloud/herd/pkg/labels"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func DeploySpec(req router.Request, resp router.Response) error {
	appInstance := req.Object.(*v1.AppInstance)
	if appInstance.Status.Namespace == "" {
		return nil
	}
	addNamespace(appInstance, resp)
	addDeployments(appInstance, resp)
	addServices(appInstance, resp)
	addPVCs(appInstance, resp)
	return addConfigMaps(appInstance, resp)
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

func toContainers(appName, name string, container v1.Container) ([]corev1.Container, []corev1.Container) {
	var (
		containers     []corev1.Container
		initContainers []corev1.Container
	)

	containers = append(containers, toContainer(appName, name, name, container))
	for _, entry := range typed.Sorted(container.Sidecars) {
		newContainer := toContainerFromSidekick(appName, name, entry.Key, entry.Value)
		if entry.Value.Init {
			initContainers = append(initContainers, newContainer)
		} else {
			containers = append(containers, newContainer)
		}
	}
	return containers, initContainers
}

func toContainerFromSidekick(appName, name, sidecarName string, sidecar v1.Sidecar) corev1.Container {
	return toContainer(appName, name, sidecarName, v1.Container{
		Files:       sidecar.Files,
		Image:       sidecar.Image,
		Build:       sidecar.Build,
		Command:     sidecar.Command,
		Interactive: sidecar.Interactive,
		Entrypoint:  sidecar.Entrypoint,
		Environment: sidecar.Environment,
		WorkingDir:  sidecar.WorkingDir,
		Ports:       sidecar.Ports,
	})
}

func toMounts(appName, name, sidecarName string, container v1.Container) (result []corev1.VolumeMount) {
	for _, entry := range typed.Sorted(container.Files) {
		result = append(result, corev1.VolumeMount{
			Name:      "files",
			MountPath: path.Join("/", entry.Key),
			SubPath:   path.Join("/", appName, name, sidecarName, entry.Key),
		})
	}
	return
}

func toPorts(container v1.Container) []corev1.ContainerPort {
	var ports []corev1.ContainerPort
	for _, port := range container.Ports {
		protocol := corev1.ProtocolTCP
		if port.Protocol == v1.ProtocolUDP {
			protocol = corev1.ProtocolUDP
		}
		ports = append(ports, corev1.ContainerPort{
			ContainerPort: port.ContainerPort,
			Protocol:      protocol,
		})
	}
	return ports
}

func toContainer(appName, name, sidecarName string, container v1.Container) corev1.Container {
	return corev1.Container{
		Name:         sidecarName,
		Image:        container.Image,
		Command:      container.Entrypoint,
		Args:         container.Command,
		WorkingDir:   container.WorkingDir,
		Env:          toEnv(container.Environment),
		TTY:          container.Interactive,
		Stdin:        container.Interactive,
		Ports:        toPorts(container),
		VolumeMounts: toMounts(appName, name, sidecarName, container),
	}
}

func containerLabels(appInstance *v1.AppInstance, name string, kv ...string) map[string]string {
	labels := map[string]string{
		labels.HerdAppName:       appInstance.Name,
		labels.HerdAppNamespace:  appInstance.Namespace,
		labels.HerdContainerName: name,
	}
	for i := 0; i+1 < len(kv); i += 2 {
		labels[kv[i]] = kv[i+1]
	}
	return labels
}

func toDeployment(appInstance *v1.AppInstance, name string, container v1.Container) *appsv1.Deployment {
	var replicas *int32
	if appInstance.Spec.Stop != nil && *appInstance.Spec.Stop {
		replicas = new(int32)
	}
	containers, initContainers := toContainers(appInstance.Name, name, container)
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: appInstance.Status.Namespace,
			Labels:    containerLabels(appInstance, name),
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: containerLabels(appInstance, name),
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: containerLabels(appInstance, name,
						labels.HerdAppPod, "true",
					),
				},
				Spec: corev1.PodSpec{
					Containers:     containers,
					InitContainers: initContainers,
					Volumes:        toVolumes(appInstance, container),
				},
			},
		},
	}
}

func toDeployments(appInstance *v1.AppInstance) (result []meta.Object) {
	for _, entry := range typed.Sorted(appInstance.Status.AppSpec.Containers) {
		result = append(result, toDeployment(appInstance, entry.Key, entry.Value))
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

func addServices(appInstance *v1.AppInstance, resp router.Response) {
	resp.Objects(toServices(appInstance)...)
}

func toServices(appInstance *v1.AppInstance) (result []meta.Object) {
	for _, entry := range typed.Sorted(appInstance.Status.AppSpec.Containers) {
		service := toService(appInstance, entry.Key, entry.Value)
		if service != nil {
			result = append(result, service)
		}
	}
	return result
}

func toServicePort(port v1.Port) corev1.ServicePort {
	servicePort := corev1.ServicePort{
		Protocol: corev1.ProtocolTCP,
		Port:     port.Port,
		TargetPort: intstr.IntOrString{
			IntVal: port.ContainerPort,
		},
	}
	switch port.Protocol {
	case v1.ProtocolTCP:
	case v1.ProtocolUDP:
		servicePort.Protocol = corev1.ProtocolUDP
	case v1.ProtocolHTTP:
		fallthrough
	case v1.ProtocolHTTPS:
		str := strings.ToUpper(string(port.Protocol))
		servicePort.AppProtocol = &str
	}
	return servicePort
}

func toService(appInstance *v1.AppInstance, name string, container v1.Container) *corev1.Service {
	var ports []corev1.ServicePort
	for _, port := range container.Ports {
		ports = append(ports, toServicePort(port))
	}
	for _, entry := range typed.Sorted(container.Sidecars) {
		for _, port := range entry.Value.Ports {
			ports = append(ports, toServicePort(port))
		}
	}

	if len(ports) == 0 {
		return nil
	}

	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: appInstance.Status.Namespace,
			Labels:    containerLabels(appInstance, name),
		},
		Spec: corev1.ServiceSpec{
			Ports:    ports,
			Selector: containerLabels(appInstance, name),
			Type:     corev1.ServiceTypeClusterIP,
		},
	}
}

func addFileContent(data map[string][]byte, appName, name string, container v1.Container) error {
	for filePath, file := range container.Files {
		content, err := base64.StdEncoding.DecodeString(file.Content)
		if err != nil {
			return err
		}
		data[path.Join("/", appName, name, name, filePath)] = content
	}
	for sidecarName, sidecar := range container.Sidecars {
		for filePath, file := range sidecar.Files {
			content, err := base64.StdEncoding.DecodeString(file.Content)
			if err != nil {
				return err
			}
			data[path.Join("/", appName, name, sidecarName, filePath)] = content
		}
	}
	return nil
}

func addConfigMaps(appInstance *v1.AppInstance, resp router.Response) error {
	objs, err := toConfigMaps(appInstance)
	resp.Objects(objs...)
	return err
}

func toConfigMaps(appInstance *v1.AppInstance) (result []meta.Object, err error) {
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "files",
			Namespace: appInstance.Status.Namespace,
		},
		BinaryData: map[string][]byte{},
	}
	for _, entry := range typed.Sorted(appInstance.Status.AppSpec.Containers) {
		if err := addFileContent(configMap.BinaryData, appInstance.Name, entry.Key, entry.Value); err != nil {
			return nil, err
		}
	}
	if len(configMap.BinaryData) == 0 {
		return nil, nil
	}
	return []meta.Object{configMap}, nil
}
