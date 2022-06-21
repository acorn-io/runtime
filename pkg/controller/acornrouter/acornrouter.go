package acornrouter

import (
	"fmt"
	"strconv"
	"strings"

	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/labels"
	"github.com/acorn-io/acorn/pkg/ports"
	"github.com/acorn-io/acorn/pkg/system"
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

	exposedPorts, err := getExposedPorts(req, appInstance)
	if err != nil {
		return err
	}

	if len(exposedPorts) == 0 {
		return nil
	}

	ds, err := toDaemonSet(appInstance, exposedPorts)
	if err != nil {
		return err
	}
	resp.Objects(ds)

	dsSvc := toDaemonSetService(appInstance, ds, exposedPorts)
	resp.Objects(dsSvc)

	svc := toService(appInstance, dsSvc)
	resp.Objects(svc)

	return nil
}

func clusterIPsForService(req router.Request, namespace, serviceName string) ([]string, error) {
	var service corev1.Service
	err := req.Get(&service, namespace, serviceName)
	if apierrors.IsNotFound(err) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	// Acorn service are a CNAMEs to another service
	if service.Spec.Type == corev1.ServiceTypeExternalName {
		if !strings.HasSuffix(service.Spec.ExternalName, system.Namespace+"."+system.ClusterDomain) {
			return nil, nil
		}
		serviceName, _, _ := strings.Cut(service.Spec.ExternalName, ".")
		err := req.Get(&service, system.Namespace, serviceName)
		if apierrors.IsNotFound(err) {
			return nil, nil
		} else if err != nil {
			return nil, err
		}
	}

	if service.Spec.ClusterIP == "" {
		return nil, nil
	}

	return append([]string{service.Spec.ClusterIP}, service.Spec.ClusterIPs...), nil
}

func mapPortToClusterIP(req router.Request, app *v1.AppInstance, portMappings map[string]ExposedPort, serviceName string, ports []v1.PortDef) error {
	for _, port := range ports {
		if !port.Expose {
			continue
		}

		portKey := strconv.Itoa(int(port.Port)) + "/" + string(port.Protocol)
		if _, ok := portMappings[portKey]; ok {
			continue
		}

		clusterIPs, err := clusterIPsForService(req, app.Status.Namespace, serviceName)
		if len(clusterIPs) == 0 || err != nil {
			return err
		}

		portMappings[portKey] = ExposedPort{
			Port:    port,
			DestIPs: clusterIPs,
		}
	}
	return nil
}

func getExposedPorts(req router.Request, app *v1.AppInstance) (result []ExposedPort, _ error) {
	portMappings := map[string]ExposedPort{}

	for _, entry := range typed.Sorted(app.Status.AppSpec.Containers) {
		name := entry.Key
		if entry.Value.Alias.Name != "" {
			name = entry.Value.Alias.Name
		}
		ports := ports.CollectPorts(entry.Value)
		if err := mapPortToClusterIP(req, app, portMappings, name, ports); err != nil {
			return nil, err
		}
	}

	for _, entry := range typed.Sorted(app.Status.AppSpec.Acorns) {
		if err := mapPortToClusterIP(req, app, portMappings, entry.Key, entry.Value.Ports); err != nil {
			return nil, err
		}
	}

	for _, entry := range typed.Sorted(portMappings) {
		result = append(result, entry.Value)
	}

	return
}

type ExposedPort struct {
	Port    v1.PortDef
	DestIPs []string
}

func toName(app *v1.AppInstance) string {
	return name2.SafeConcatName(app.Name, app.Namespace, string(app.UID[:12]))
}

func toAcornLabels(app *v1.AppInstance) map[string]string {
	return map[string]string{
		labels.AcornAppName:      app.Name,
		labels.AcornAppNamespace: app.Namespace,
		labels.AcornManaged:      "true",
		labels.AcornAcornName:    app.Name,
	}
}

func toDaemonSet(app *v1.AppInstance, exposedPorts []ExposedPort) (*appsv1.DaemonSet, error) {
	labels := toAcornLabels(app)
	ds := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      toName(app),
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
					EnableServiceLinks:           new(bool),
				},
			},
		},
	}

	for _, portSpec := range exposedPorts {
		containerNames := map[string]bool{}
		name := fmt.Sprintf("port-%d", portSpec.Port.Port)
		if _, ok := containerNames[name]; ok {
			continue
		}
		containerNames[name] = true

		ds.Spec.Template.Spec.Containers = append(ds.Spec.Template.Spec.Containers, corev1.Container{
			Name:  name,
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
					Value: string(ports.NormalizeProto(portSpec.Port.Protocol)),
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
					Protocol:      corev1.Protocol(strings.ToUpper(string(ports.NormalizeProto(portSpec.Port.Protocol)))),
				},
			},
		})
	}

	return ds, nil
}

func toDaemonSetService(appInstance *v1.AppInstance, ds *appsv1.DaemonSet, exposedPorts []ExposedPort) *corev1.Service {
	var (
		labels   = toAcornLabels(appInstance)
		portDefs = typed.MapSlice(exposedPorts, func(t ExposedPort) v1.PortDef {
			return t.Port
		})
	)

	portDefs = ports.RemapForBinding(false, portDefs, appInstance.Spec.Ports, appInstance.Spec.PublishProtocols)

	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      ds.Name,
			Namespace: ds.Namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Ports:                 typed.MapSlice(portDefs, ports.ToServicePort),
			Selector:              labels,
			Type:                  corev1.ServiceTypeClusterIP,
			InternalTrafficPolicy: &[]corev1.ServiceInternalTrafficPolicyType{corev1.ServiceInternalTrafficPolicyLocal}[0],
		},
	}
}

func toService(app *v1.AppInstance, svc *corev1.Service) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      app.Name,
			Namespace: app.Namespace,
			Labels:    toAcornLabels(app),
		},
		Spec: corev1.ServiceSpec{
			Type:         corev1.ServiceTypeExternalName,
			ExternalName: svc.Name + "." + svc.Namespace + "." + system.ClusterDomain,
		},
	}
}
