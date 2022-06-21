package appdefinition

import (
	"fmt"
	"strconv"
	"strings"

	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/labels"
	"github.com/acorn-io/baaah/pkg/router"
	"github.com/acorn-io/baaah/pkg/typed"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

func serviceEndpoints(req router.Request, app *v1.AppInstance, containerName string) (endpoints []v1.Endpoint, _ error) {
	service := &corev1.Service{}
	err := req.Get(service, app.Status.Namespace, PublishServiceName(app, containerName))
	if apierrors.IsNotFound(err) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	for _, port := range service.Spec.Ports {
		var protocol v1.Protocol

		switch port.Protocol {
		case corev1.ProtocolTCP:
			protocol = v1.ProtocolTCP
		case corev1.ProtocolUDP:
			protocol = v1.ProtocolTCP
		default:
			continue
		}

		for _, ingress := range service.Status.LoadBalancer.Ingress {
			if ingress.Hostname != "" {
				endpoints = append(endpoints, v1.Endpoint{
					Target:     containerName,
					TargetPort: port.TargetPort.IntVal,
					Address:    fmt.Sprintf("%s:%d", ingress.Hostname, port.Port),
					Protocol:   protocol,
				})
			} else if ingress.IP != "" {
				endpoints = append(endpoints, v1.Endpoint{
					Target:     containerName,
					TargetPort: port.TargetPort.IntVal,
					Address:    fmt.Sprintf("%s:%d", ingress.IP, port.Port),
					Protocol:   protocol,
				})
			}
		}

		if len(service.Status.LoadBalancer.Ingress) == 0 {
			endpoints = append(endpoints, v1.Endpoint{
				Target:     containerName,
				TargetPort: port.TargetPort.IntVal,
				Address:    fmt.Sprintf("<pending>:%d", port.Port),
				Protocol:   protocol,
			})
		}
	}

	return
}

func ingressEndpoints(req router.Request, app *v1.AppInstance, containerName string) (endpoints []v1.Endpoint, _ error) {
	ingress := &networkingv1.Ingress{}
	err := req.Get(ingress, app.Status.Namespace, containerName)
	if apierrors.IsNotFound(err) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	portnum := ingress.Annotations[labels.AcornPortNumber]
	if portnum == "" {
		return nil, nil
	}

	port, err := strconv.Atoi(portnum)
	if err != nil {
		return nil, fmt.Errorf("parsing %s for container %s on %s/%s", portnum, containerName,
			app.Namespace, app.Name)
	}

	for _, hostname := range strings.Split(ingress.Annotations[labels.AcornHostnames], ",") {
		hostname = strings.TrimSpace(hostname)
		if hostname == "" {
			continue
		}

		endpoints = append(endpoints, v1.Endpoint{
			Target:     containerName,
			TargetPort: int32(port),
			Address:    hostname,
			Protocol:   v1.ProtocolHTTP,
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
