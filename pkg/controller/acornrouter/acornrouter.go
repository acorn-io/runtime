package acornrouter

import (
	"fmt"
	"strconv"
	"strings"

	v1 "github.com/acorn-io/acorn/pkg/apis/acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/labels"
	"github.com/acorn-io/acorn/pkg/system"
	"github.com/acorn-io/baaah/pkg/meta"
	"github.com/acorn-io/baaah/pkg/router"
	"github.com/acorn-io/baaah/pkg/typed"
	name2 "github.com/rancher/wrangler/pkg/name"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func AcornRouter(req router.Request, resp router.Response) error {
	appInstance := req.Object.(*v1.AppInstance)

	for _, entry := range typed.Sorted(appInstance.Status.AppSpec.Acorns) {
		acornName, acorn := entry.Key, entry.Value
		ds, err := toDaemonSet(req, appInstance, acornName, acorn)
		if err != nil {
			return err
		}
		if ds != nil {
			resp.Objects(ds)
		}
	}

	return nil
}

func normalizeProto(proto v1.Protocol) v1.Protocol {
	switch proto {
	case v1.ProtocolHTTP:
		return v1.ProtocolTCP
	case v1.ProtocolHTTPS:
		return v1.ProtocolTCP
	}
	return proto
}

func protosMatch(left, right v1.Protocol) bool {
	return normalizeProto(left) == normalizeProto(right)
}

func findPortServiceName(srcPort v1.Port, app v1.AppInstance) string {
	for _, entry := range typed.Sorted(app.Status.AppSpec.Containers) {
		for _, port := range entry.Value.Ports {
			if port.Publish && port.Port == srcPort.ContainerPort &&
				protosMatch(srcPort.Protocol, port.Protocol) {
				return entry.Key
			}
		}
	}

	for _, entry := range typed.Sorted(app.Status.AppSpec.Acorns) {
		for _, port := range entry.Value.Ports {
			if port.Publish && port.Port == srcPort.ContainerPort &&
				protosMatch(srcPort.Protocol, port.Protocol) {
				return entry.Key
			}
		}
	}

	return ""
}

type PortMapping struct {
	DestIPs []string
	Port    v1.Port
}

func getPortMappings(req router.Request, acorn v1.Acorn, childApp v1.AppInstance) (map[int32]PortMapping, error) {
	portMappings := map[int32]PortMapping{}

	for _, port := range acorn.Ports {
		serviceName := findPortServiceName(port, childApp)
		if serviceName == "" {
			continue
		}

		var service corev1.Service
		err := req.Client.Get(&service, serviceName, &meta.GetOptions{
			Namespace: childApp.Status.Namespace,
		})
		if apierrors.IsNotFound(err) {
			continue
		} else if err != nil {
			return nil, err
		}

		if service.Spec.ClusterIP == "" {
			continue
		}

		portMappings[port.Port] = PortMapping{
			Port:    port,
			DestIPs: append([]string{service.Spec.ClusterIP}, service.Spec.ClusterIPs...),
		}
	}

	return portMappings, nil
}

func toName(app *v1.AppInstance, name, namespace string) string {
	return name2.SafeConcatName(name, namespace, string(app.UID[:12]))
}

func toDaemonSet(req router.Request, app *v1.AppInstance, acornName string, acorn v1.Acorn) (*appsv1.DaemonSet, error) {
	if len(acorn.Ports) == 0 {
		return nil, nil
	}

	var childApp v1.AppInstance

	err := req.Client.Get(&childApp, acornName, &meta.GetOptions{
		Namespace: app.Status.Namespace,
	})
	if apierrors.IsNotFound(err) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	portMappings, err := getPortMappings(req, acorn, childApp)
	if err != nil {
		return nil, err
	}

	if len(portMappings) == 0 {
		return nil, err
	}

	labels := map[string]string{
		labels.AcornAppName:      app.Name,
		labels.AcornAppNamespace: app.Namespace,
		labels.AcornAcornName:    acornName,
		labels.AcornManaged:      "true",
	}
	ds := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      toName(app, acornName, app.Status.Namespace),
			Namespace: system.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: metav1.SetAsLabelSelector(labels),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					AutomountServiceAccountToken: new(bool),
				},
			},
		},
	}

	for _, entry := range typed.Sorted(portMappings) {
		portSpec := entry.Value
		ds.Spec.Template.Spec.Containers = append(ds.Spec.Template.Spec.Containers, corev1.Container{
			Name:  fmt.Sprintf("port-%d", portSpec.Port.ContainerPort),
			Image: system.KlipperLBImage,
			SecurityContext: &corev1.SecurityContext{
				Capabilities: &corev1.Capabilities{
					Add: []corev1.Capability{
						"NET_ADMIN",
					},
				},
			},
			Env: []corev1.EnvVar{
				{
					Name:  "SRC_PORT",
					Value: strconv.Itoa(int(portSpec.Port.ContainerPort)),
				},
				{
					Name:  "DEST_PROTO",
					Value: string(normalizeProto(portSpec.Port.Protocol)),
				},
				{
					Name:  "DEST_PORT",
					Value: strconv.Itoa(int(portSpec.Port.ContainerPort)),
				},
				{
					Name:  "DEST_IPS",
					Value: strings.Join(portSpec.DestIPs, " "),
				},
			},
			Ports: []corev1.ContainerPort{
				{
					ContainerPort: portSpec.Port.ContainerPort,
					Protocol:      corev1.Protocol(strings.ToUpper(string(normalizeProto(portSpec.Port.Protocol)))),
				},
			},
		})
	}

	return ds, nil
}
