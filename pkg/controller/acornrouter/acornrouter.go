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

	ds, err := toDaemonSet(req, appInstance)
	if err != nil {
		return err
	}
	if ds != nil {
		resp.Objects(ds)
		resp.Objects(toService(appInstance)...)
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

func isPublishable(app *v1.AppInstance, port v1.Port) bool {
	if !port.Publish {
		return false
	}
	if app.Spec.PublishAllPorts {
		return true
	}
	for _, appPort := range app.Spec.Ports {
		if appPort.ContainerPort == port.Port && protosMatch(appPort.Protocol, port.Protocol) {
			return true
		}
	}
	return false
}

func addMappings(req router.Request, app *v1.AppInstance, portMappings map[string]PortMapping, serviceName string, ports []v1.Port) error {
	for _, port := range ports {
		portKey := strconv.Itoa(int(port.Port)) + "/" + string(normalizeProto(port.Protocol))
		if _, ok := portMappings[portKey]; ok {
			continue
		}
		if isPublishable(app, port) {
			var service corev1.Service
			err := req.Client.Get(&service, serviceName, &meta.GetOptions{
				Namespace: app.Status.Namespace,
			})
			if apierrors.IsNotFound(err) {
				continue
			} else if err != nil {
				return err
			}

			if service.Spec.Type == corev1.ServiceTypeExternalName {
				if !strings.HasSuffix(service.Spec.ExternalName, system.Namespace+"."+system.ClusterDomain) {
					continue
				}
				serviceName, _, _ := strings.Cut(service.Spec.ExternalName, ".")
				err := req.Client.Get(&service, serviceName, &meta.GetOptions{
					Namespace: system.Namespace,
				})
				if apierrors.IsNotFound(err) {
					continue
				} else if err != nil {
					return err
				}
			}

			if service.Spec.ClusterIP == "" {
				continue
			}

			portMappings[portKey] = PortMapping{
				Port:    port,
				DestIPs: append([]string{service.Spec.ClusterIP}, service.Spec.ClusterIPs...),
			}
		}
	}
	return nil
}

func getPortMappings(req router.Request, app *v1.AppInstance) (map[string]PortMapping, error) {
	portMappings := map[string]PortMapping{}

	for _, entry := range typed.Sorted(app.Status.AppSpec.Containers) {
		name := entry.Key
		if len(entry.Value.Aliases) > 0 {
			name = entry.Value.Aliases[0].Name
		}
		if err := addMappings(req, app, portMappings, name, entry.Value.Ports); err != nil {
			return nil, err
		}
	}

	for _, entry := range typed.Sorted(app.Status.AppSpec.Acorns) {
		if err := addMappings(req, app, portMappings, entry.Key, entry.Value.Ports); err != nil {
			return nil, err
		}
	}

	return portMappings, nil
}

type PortMapping struct {
	DestIPs []string
	Port    v1.Port
}

func toName(app *v1.AppInstance, name, namespace string) string {
	return name2.SafeConcatName(name, namespace, string(app.UID[:12]))
}

func toDaemonSet(req router.Request, app *v1.AppInstance) (*appsv1.DaemonSet, error) {
	if len(app.Spec.Ports) == 0 && !app.Spec.PublishAllPorts {
		return nil, nil
	}

	portMappings, err := getPortMappings(req, app)
	if err != nil {
		return nil, err
	}

	if len(portMappings) == 0 {
		return nil, err
	}

	labels := map[string]string{
		labels.AcornAppName:      app.Name,
		labels.AcornAppNamespace: app.Namespace,
		labels.AcornManaged:      "true",
		labels.AcornAcornName:    app.Name,
	}
	ds := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      toName(app, app.Name, app.Namespace),
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
			Name:  fmt.Sprintf("port-%d", portSpec.Port.Port),
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
					Value: strconv.Itoa(int(portSpec.Port.Port)),
				},
				{
					Name:  "DEST_PROTO",
					Value: string(normalizeProto(portSpec.Port.Protocol)),
				},
				{
					Name:  "DEST_PORT",
					Value: strconv.Itoa(int(portSpec.Port.Port)),
				},
				{
					Name:  "DEST_IPS",
					Value: strings.Join(portSpec.DestIPs, " "),
				},
			},
			Ports: []corev1.ContainerPort{
				{
					ContainerPort: portSpec.Port.Port,
					Protocol:      corev1.Protocol(strings.ToUpper(string(normalizeProto(portSpec.Port.Protocol)))),
				},
			},
		})
	}

	return ds, nil
}
