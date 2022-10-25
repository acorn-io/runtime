package expose

import (
	"fmt"
	"strconv"
	"strings"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/config"
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
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func Acorns(req router.Request, app *v1.AppInstance) (result []kclient.Object, _ error) {
	cfg, err := config.Get(req.Ctx, req.Client)
	if err != nil {
		return nil, err
	}

	m, err := ports.NewForAcornExpose(app)
	if err != nil {
		return nil, err
	}

	for _, service := range m.ServiceNames() {
		var ports []exposedPort
		for _, port := range m.PortsForService(service) {
			ips, err := clusterIPsForService(cfg, req, app.Status.Namespace, m.Ports[port][0].ServiceName())
			if err != nil {
				return nil, err
			}
			ports = append(ports, exposedPort{
				Port:    port,
				DestIPs: ips,
			})
		}

		ds := toRouterDeployment(service, app, ports)
		dsSvc := toRouterDeploymentService(service, app, ds, ports)
		svc := toService(cfg, service, app, dsSvc)

		result = append(result, ds, dsSvc, svc)
	}

	return result, nil
}

func toAcornLabels(app *v1.AppInstance, serviceName string) map[string]string {
	return labels.Managed(app, labels.AcornServiceName, serviceName)
}

func toName(app *v1.AppInstance, serviceName string) string {
	return name2.SafeConcatName(app.Name, app.Namespace, serviceName, app.ShortID())
}

type exposedPort struct {
	Port    v1.PortDef
	DestIPs []string
}

func toRouterDeployment(serviceName string, app *v1.AppInstance, exposedPorts []exposedPort) *appsv1.Deployment {
	labels := toAcornLabels(app, serviceName)
	ds := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      toName(app, serviceName),
			Namespace: system.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
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

	containerNames := map[string]bool{}
	for _, portSpec := range exposedPorts {
		name := fmt.Sprintf("port-%d", portSpec.Port.TargetPort)
		if containerNames[name] {
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
					Value: strconv.Itoa(int(portSpec.Port.TargetPort)),
				},
				{
					Name:  "DEST_PROTO",
					Value: string(ports.NormalizeProto(portSpec.Port.Protocol)),
				},
				{
					Name:  "DEST_PORT",
					Value: strconv.Itoa(int(portSpec.Port.TargetPort)),
				},
				{
					Name:  "DEST_IPS",
					Value: strings.Join(portSpec.DestIPs, " "),
				},
			},
			Ports: []corev1.ContainerPort{
				{
					ContainerPort: portSpec.Port.TargetPort,
					Protocol:      corev1.Protocol(strings.ToUpper(string(ports.NormalizeProto(portSpec.Port.Protocol)))),
				},
			},
		})
	}

	return ds
}

func toRouterDeploymentService(serviceName string, appInstance *v1.AppInstance, ds *appsv1.Deployment, exposedPorts []exposedPort) *corev1.Service {
	var (
		labels = toAcornLabels(appInstance, serviceName)
	)

	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      ds.Name,
			Namespace: ds.Namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Ports: typed.MapSlice(exposedPorts, func(v exposedPort) corev1.ServicePort {
				return ports.ToServicePort(v.Port)
			}),
			Selector: labels,
			Type:     corev1.ServiceTypeClusterIP,
		},
	}
}

func toService(cfg *apiv1.Config, serviceName string, app *v1.AppInstance, svc *corev1.Service) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName,
			Namespace: app.Namespace,
			Labels:    toAcornLabels(app, serviceName),
		},
		Spec: corev1.ServiceSpec{
			Type:         corev1.ServiceTypeExternalName,
			ExternalName: svc.Name + "." + svc.Namespace + "." + cfg.InternalClusterDomain,
		},
	}
}

func clusterIPsForService(cfg *apiv1.Config, req router.Request, namespace, serviceName string) ([]string, error) {
	var service corev1.Service
	err := req.Get(&service, namespace, serviceName)
	if apierrors.IsNotFound(err) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	// Acorn service are a CNAMEs to another service
	if service.Spec.Type == corev1.ServiceTypeExternalName {
		if !strings.HasSuffix(service.Spec.ExternalName, system.Namespace+"."+cfg.InternalClusterDomain) {
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
