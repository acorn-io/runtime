package appdefinition

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/ibuildthecloud/baaah/pkg/meta"
	"github.com/ibuildthecloud/baaah/pkg/router"
	"github.com/ibuildthecloud/baaah/pkg/typed"
	v1 "github.com/ibuildthecloud/herd/pkg/apis/herd-project.io/v1"
	"github.com/ibuildthecloud/herd/pkg/labels"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

func serviceEndpoints(req router.Request, app *v1.AppInstance, containerName string) (endpoints []v1.Endpoint, _ error) {
	service := &corev1.Service{}
	err := req.Client.Get(service, PublishServiceName(app, containerName), &meta.GetOptions{
		Namespace: app.Status.Namespace,
	})
	if apierrors.IsNotFound(err) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	for _, port := range service.Spec.Ports {
		var protocol v1.PublishProtocol

		switch port.Protocol {
		case corev1.ProtocolTCP:
			protocol = v1.PublishProtocolTCP
		case corev1.ProtocolUDP:
			protocol = v1.PublishProtocolTCP
		default:
			continue
		}

		for _, ingress := range service.Status.LoadBalancer.Ingress {
			if ingress.Hostname != "" {
				endpoints = append(endpoints, v1.Endpoint{
					Target:           containerName,
					TargetPortNumber: port.TargetPort.IntVal,
					Address:          fmt.Sprintf("%s:%d", ingress.Hostname, port.Port),
					Protocol:         protocol,
				})
			} else if ingress.IP != "" {
				endpoints = append(endpoints, v1.Endpoint{
					Target:           containerName,
					TargetPortNumber: port.TargetPort.IntVal,
					Address:          fmt.Sprintf("%s:%d", ingress.IP, port.Port),
					Protocol:         protocol,
				})
			} else {
			}
		}

		if len(service.Status.LoadBalancer.Ingress) == 0 {
			endpoints = append(endpoints, v1.Endpoint{
				Target:           containerName,
				TargetPortNumber: port.TargetPort.IntVal,
				Address:          fmt.Sprintf("<pending>:%d", port.Port),
				Protocol:         protocol,
			})
		}
	}

	return
}

func ingressEndpoints(req router.Request, app *v1.AppInstance, containerName string) (endpoints []v1.Endpoint, _ error) {
	ingress := &networkingv1.Ingress{}
	err := req.Client.Get(ingress, containerName, &meta.GetOptions{
		Namespace: app.Status.Namespace,
	})
	if apierrors.IsNotFound(err) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	portnum := ingress.Annotations[labels.HerdPortNumber]
	if portnum == "" {
		return nil, nil
	}

	port, err := strconv.Atoi(portnum)
	if err != nil {
		return nil, fmt.Errorf("parsing %s for container %s on %s/%s", portnum, containerName,
			app.Namespace, app.Name)
	}

	for _, hostname := range strings.Split(ingress.Annotations[labels.HerdHostnames], ",") {
		hostname = strings.TrimSpace(hostname)
		if hostname == "" {
			continue
		}

		endpoints = append(endpoints, v1.Endpoint{
			Target:           containerName,
			TargetPortNumber: int32(port),
			Address:          hostname,
			Protocol:         v1.PublishProtocolHTTP,
		})
	}

	return
}

func AppEndpointsStatus(req router.Request, _ router.Response) error {
	var (
		app       = req.Object.(*v1.AppInstance)
		endpoints []v1.Endpoint
	)

	for _, entry := range typed.Sorted(app.Status.AppSpec.Containers) {
		containerName, _ := entry.Key, entry.Value

		ingressEndpoints, err := ingressEndpoints(req, app, containerName)
		if err != nil {
			return err
		}

		endpoints = append(endpoints, ingressEndpoints...)

		serviceEndpoints, err := serviceEndpoints(req, app, containerName)
		if err != nil {
			return err
		}

		endpoints = append(endpoints, serviceEndpoints...)
	}

	app.Status.Endpoints = endpoints
	return nil
}
