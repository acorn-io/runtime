package appdefinition

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/controller/appstatus"
	"github.com/acorn-io/acorn/pkg/labels"
	"github.com/acorn-io/acorn/pkg/publish"
	"github.com/acorn-io/baaah/pkg/router"
	"github.com/acorn-io/baaah/pkg/typed"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	klabels "k8s.io/apimachinery/pkg/labels"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func serviceEndpoints(req router.Request, app *v1.AppInstance) (endpoints []v1.Endpoint, _ error) {
	serviceList := &corev1.ServiceList{}
	err := req.List(serviceList, &kclient.ListOptions{
		Namespace: app.Status.Namespace,
		LabelSelector: klabels.SelectorFromSet(map[string]string{
			labels.AcornManaged:        "true",
			labels.AcornServicePublish: "true",
		}),
	})
	if err != nil {
		return nil, err
	}

	for _, service := range serviceList.Items {
		containerName := service.Labels[labels.AcornContainerName]
		if containerName == "" {
			continue
		}

		for _, port := range service.Spec.Ports {
			var protocol v1.Protocol

			switch port.Protocol {
			case corev1.ProtocolTCP:
				protocol = v1.ProtocolTCP
			case corev1.ProtocolUDP:
				protocol = v1.ProtocolUDP
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
					Address:    fmt.Sprintf("<Pending Ingress>:%d", port.Port),
					Protocol:   protocol,
					Pending:    true,
				})
			}
		}
	}

	return
}

func ingressEndpoints(req router.Request, app *v1.AppInstance) (endpoints []v1.Endpoint, _ error) {
	ingressList := &networkingv1.IngressList{}
	err := req.List(ingressList, &kclient.ListOptions{
		Namespace: app.Status.Namespace,
		LabelSelector: klabels.SelectorFromSet(map[string]string{
			labels.AcornManaged: "true",
		}),
	})
	if err != nil {
		return nil, err
	}

	for _, ingress := range ingressList.Items {
		targetStr := ingress.Annotations[labels.AcornTargets]
		if targetStr == "" {
			return nil, err
		}

		targets := map[string]publish.Target{}
		if err := json.Unmarshal([]byte(targetStr), &targets); err != nil {
			return nil, err
		}

		for _, entry := range typed.Sorted(targets) {
			hostname, target := entry.Key, entry.Value
			hostnameOverride := ingress.Annotations[labels.AcornPublishURL]
			if hostnameOverride != "" {
				hostname = hostnameOverride
			}

			endpoints = append(endpoints, v1.Endpoint{
				Target:     target.Service,
				TargetPort: target.Port,
				Address:    hostname,
				Protocol:   v1.ProtocolHTTP,
				Pending:    len(ingress.Status.LoadBalancer.Ingress) == 0,
			})
		}
	}

	return
}

func AppEndpointsStatus(req router.Request, _ router.Response) error {
	app := req.Object.(*v1.AppInstance)

	ingressEndpoints, err := ingressEndpoints(req, app)
	if err != nil {
		return err
	}

	serviceEndpoints, err := serviceEndpoints(req, app)
	if err != nil {
		return err
	}

	eps := append(ingressEndpoints, serviceEndpoints...)

	ingressTLSHosts, err := appstatus.IngressTLSHosts(req.Ctx, req.Client, app)
	if err != nil {
		return err
	}

	for i, ep := range eps {
		if ep.Protocol == v1.ProtocolHTTP {
			ep.PublishProtocol = v1.PublishProtocolHTTP
			if _, ok := ingressTLSHosts[strings.Split(ep.Address, ":")[0]]; ok {
				ep.PublishProtocol = v1.PublishProtocolHTTPS
			}
		} else {
			ep.PublishProtocol = v1.PublishProtocol(ep.Protocol)
		}
		eps[i] = ep
	}

	sort.Slice(eps, func(i, j int) bool {
		return eps[i].Address < eps[j].Address
	})

	app.Status.Endpoints = eps
	return nil
}
