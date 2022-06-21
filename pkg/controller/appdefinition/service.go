package appdefinition

import (
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/labels"
	"github.com/acorn-io/acorn/pkg/ports"
	"github.com/acorn-io/baaah/pkg/typed"
	name2 "github.com/rancher/wrangler/pkg/name"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func isLinked(appInstance *v1.AppInstance, serviceNames ...string) bool {
	for _, link := range appInstance.Spec.Services {
		for _, serviceName := range serviceNames {
			if serviceName == "" {
				continue
			}
			if link.Target == serviceName {
				return true
			}
		}
	}
	return false
}

func toServices(appInstance *v1.AppInstance) (result []kclient.Object) {
	result = append(result, toInAcornContainerServices(appInstance)...)
	result = append(result, toInAcornAliasServices(appInstance)...)
	result = append(result, toPublishContainerService(appInstance)...)
	result = append(result, toPublishAliasService(appInstance)...)
	return
}

func PublishServiceName(appInstance *v1.AppInstance, containerName string) string {
	// UID based name is to avoid name conflict. For example if somebody had to containers, foo and foo-publish.
	return name2.SafeConcatName(containerName, "publish", string(appInstance.UID)[:12])
}

func toPublishAliasService(appInstance *v1.AppInstance) (result []kclient.Object) {
	if appInstance.Spec.Stop != nil && *appInstance.Spec.Stop {
		// remove all publishes
		return nil
	}

	aliases := getAliasedPorts(appInstance)
	for _, entry := range typed.Sorted(aliases) {
		name, portDefs := entry.Key, entry.Value

		portDefs = ports.RemapForBinding(true, portDefs, appInstance.Spec.Ports, appInstance.Spec.PublishProtocols)
		portDefs = ports.Layer4(portDefs)
		portDefs = ports.Dedup(portDefs)
		if len(portDefs) == 0 {
			continue
		}

		result = append(result, &corev1.Service{
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
				Name:      PublishServiceName(appInstance, name),
				Namespace: appInstance.Status.Namespace,
				Labels:    labels.Managed(appInstance, labels.AcornAlias+name, "true"),
			},
			Spec: corev1.ServiceSpec{
				Ports:    typed.MapSlice(portDefs, ports.ToServicePort),
				Selector: labels.Managed(appInstance, labels.AcornAlias+name, "true"),
				Type:     corev1.ServiceTypeLoadBalancer,
			},
		})
	}

	return
}

func toPublishContainerService(appInstance *v1.AppInstance) (result []kclient.Object) {
	if appInstance.Spec.Stop != nil && *appInstance.Spec.Stop {
		// remove all publishes
		return nil
	}

	for _, entry := range typed.Sorted(appInstance.Status.AppSpec.Containers) {
		containerName, container := entry.Key, entry.Value

		if container.Alias.Name != "" {
			continue
		}

		portDefs := ports.RemapForBinding(true, ports.CollectPorts(container), appInstance.Spec.Ports, appInstance.Spec.PublishProtocols)
		portDefs = ports.Layer4(portDefs)
		portDefs = ports.Dedup(portDefs)
		if len(portDefs) == 0 {
			continue
		}

		result = append(result, &corev1.Service{
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
				Name:      PublishServiceName(appInstance, containerName),
				Namespace: appInstance.Status.Namespace,
				Labels:    containerLabels(appInstance, containerName),
			},
			Spec: corev1.ServiceSpec{
				Ports:    typed.MapSlice(portDefs, ports.ToServicePort),
				Selector: containerLabels(appInstance, containerName),
				Type:     corev1.ServiceTypeLoadBalancer,
			},
		})
	}

	return
}

func getAliasedPorts(appInstance *v1.AppInstance) map[string][]v1.PortDef {
	aliased := map[string][]v1.PortDef{}

	for _, entry := range typed.Sorted(appInstance.Status.AppSpec.Containers) {
		if isLinked(appInstance, entry.Key, entry.Value.Alias.Name) {
			continue
		}
		if entry.Value.Alias.Name == "" {
			continue
		}
		aliased[entry.Value.Alias.Name] = append(aliased[entry.Value.Alias.Name], ports.CollectPorts(entry.Value)...)
	}

	return aliased
}

func toInAcornAliasServices(appInstance *v1.AppInstance) (result []kclient.Object) {
	aliased := getAliasedPorts(appInstance)

	for _, entry := range typed.Sorted(aliased) {
		name, portDefs := entry.Key, entry.Value
		if name == "" || len(portDefs) == 0 {
			continue
		}

		portDefs = ports.Dedup(portDefs)

		result = append(result, &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: appInstance.Status.Namespace,
				Labels:    labels.Managed(appInstance, labels.AcornAlias+name, "true"),
			},
			Spec: corev1.ServiceSpec{
				Ports:    typed.MapSlice(portDefs, ports.ToServicePort),
				Selector: labels.Managed(appInstance, labels.AcornAlias+name, "true"),
				Type:     corev1.ServiceTypeClusterIP,
			},
		})
	}

	return
}

func toInAcornContainerServices(appInstance *v1.AppInstance) (result []kclient.Object) {
	for _, entry := range typed.Sorted(appInstance.Status.AppSpec.Containers) {
		containerName, container := entry.Key, entry.Value
		if isLinked(appInstance, containerName, container.Alias.Name) {
			continue
		}

		ports := typed.MapSlice(ports.Dedup(ports.CollectPorts(container)), ports.ToServicePort)
		if len(ports) == 0 {
			continue
		}

		result = append(result, &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      containerName,
				Namespace: appInstance.Status.Namespace,
				Labels:    containerLabels(appInstance, containerName),
			},
			Spec: corev1.ServiceSpec{
				Ports:    ports,
				Selector: containerLabels(appInstance, containerName),
				Type:     corev1.ServiceTypeClusterIP,
			},
		})
	}
	return
}
