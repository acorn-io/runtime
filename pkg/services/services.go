package services

import (
	"context"
	"fmt"
	"net"
	"strings"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/condition"
	"github.com/acorn-io/acorn/pkg/config"
	"github.com/acorn-io/acorn/pkg/labels"
	"github.com/acorn-io/acorn/pkg/ports"
	"github.com/acorn-io/acorn/pkg/ref"
	"github.com/acorn-io/baaah/pkg/apply"
	"github.com/acorn-io/baaah/pkg/router"
	"github.com/acorn-io/baaah/pkg/typed"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func toContainerLabelsService(service *v1.ServiceInstance) (result []kclient.Object) {
	newService := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        service.Name,
			Namespace:   service.Namespace,
			Labels:      service.Spec.Labels,
			Annotations: service.Spec.Annotations,
		},
		Spec: corev1.ServiceSpec{
			Ports: ports.ToServicePorts(service.Spec.Ports),
			Type:  corev1.ServiceTypeClusterIP,
			Selector: typed.Concat(labels.ManagedByApp(service.Spec.AppNamespace, service.Spec.AppName),
				service.Spec.ContainerLabels),
		},
	}
	result = append(result, newService)
	return
}

func toContainerService(service *v1.ServiceInstance) (result []kclient.Object) {
	newService := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        service.Name,
			Namespace:   service.Namespace,
			Labels:      service.Spec.Labels,
			Annotations: service.Spec.Annotations,
		},
		Spec: corev1.ServiceSpec{
			Ports: ports.ToServicePorts(service.Spec.Ports),
			Type:  corev1.ServiceTypeClusterIP,
			Selector: labels.ManagedByApp(service.Spec.AppNamespace,
				service.Spec.AppName, labels.AcornContainerName, service.Spec.Container),
		},
	}
	result = append(result, newService)
	return
}

func toAddressService(service *v1.ServiceInstance) (result []kclient.Object) {
	newService := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        service.Name,
			Namespace:   service.Namespace,
			Labels:      service.Spec.Labels,
			Annotations: service.Spec.Annotations,
		},
		Spec: corev1.ServiceSpec{
			Ports: ports.ToServicePorts(service.Spec.Ports),
		},
	}
	ipAddr := net.ParseIP(service.Spec.Address)
	if ipAddr == nil {
		newService.Spec.Type = corev1.ServiceTypeExternalName
		newService.Spec.ExternalName = service.Spec.Address
	} else {
		newService.Spec.Type = corev1.ServiceTypeClusterIP

		endpointsAnnotations := make(map[string]string, len(newService.Annotations)+1)
		for k, v := range newService.Annotations {
			endpointsAnnotations[k] = v
		}

		// The baaah route we are on does not prune Endpoints,
		// so we need to add this annotation to override it.
		endpointsAnnotations[apply.AnnotationPrune] = "true"

		result = append(result, &corev1.Endpoints{
			ObjectMeta: metav1.ObjectMeta{
				Name:        newService.Name,
				Namespace:   newService.Namespace,
				Labels:      newService.Labels,
				Annotations: endpointsAnnotations,
			},
			Subsets: []corev1.EndpointSubset{
				{
					Addresses: []corev1.EndpointAddress{
						{
							IP: service.Spec.Address,
						},
					},
					Ports: typed.MapSlice(newService.Spec.Ports, func(t corev1.ServicePort) corev1.EndpointPort {
						return corev1.EndpointPort{
							Name:        t.Name,
							Port:        t.Port,
							Protocol:    t.Protocol,
							AppProtocol: t.AppProtocol,
						}
					}),
				},
			},
		})
	}
	result = append(result, newService)
	return
}

func toExternalService(ctx context.Context, c kclient.Client, cfg *apiv1.Config, service *v1.ServiceInstance) (result []kclient.Object, missing []string, err error) {
	return toRefService(ctx, c, cfg, service, service.Spec.AppNamespace, service.Spec.External)
}

func toAliasService(ctx context.Context, c kclient.Client, cfg *apiv1.Config, service *v1.ServiceInstance) (result []kclient.Object, missing []string, err error) {
	return toRefService(ctx, c, cfg, service, service.Namespace, service.Spec.Alias)
}

func toRefService(ctx context.Context, c kclient.Client, cfg *apiv1.Config, service *v1.ServiceInstance, refNamespace, refName string) (result []kclient.Object, missing []string, err error) {
	var (
		servicePorts  []corev1.ServicePort
		targetService = &v1.ServiceInstance{}
	)

	err = ref.Lookup(ctx, c, targetService, refNamespace, strings.Split(refName, ".")...)
	if apierrors.IsNotFound(err) {
		k8sService := &corev1.Service{}
		if err := c.Get(ctx, router.Key(refNamespace, refName), k8sService); err == nil {
			servicePorts = ports.CopyServicePorts(k8sService.Spec.Ports)
			targetService.Name = k8sService.Name
			targetService.Namespace = k8sService.Namespace
		} else {
			missing = append(missing, refName)
		}
	} else if err != nil {
		return nil, nil, err
	} else {
		servicePorts = ports.ToServicePorts(targetService.Spec.Ports)
	}

	newService := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        service.Name,
			Namespace:   service.Namespace,
			Labels:      service.Spec.Labels,
			Annotations: service.Spec.Annotations,
		},
		Spec: corev1.ServiceSpec{
			Type:         corev1.ServiceTypeExternalName,
			ExternalName: fmt.Sprintf("%s.%s.%s", targetService.Name, targetService.Namespace, cfg.InternalClusterDomain),
			Ports:        servicePorts,
		},
	}
	result = append(result, newService)
	return
}

func toDefaultService(cfg *apiv1.Config, svc *v1.ServiceInstance, service *corev1.Service) kclient.Object {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        svc.Spec.AppName,
			Namespace:   svc.Spec.AppNamespace,
			Labels:      service.Labels,
			Annotations: service.Annotations,
		},
		Spec: corev1.ServiceSpec{
			Type:         corev1.ServiceTypeExternalName,
			ExternalName: fmt.Sprintf("%s.%s.%s", service.Name, service.Namespace, cfg.InternalClusterDomain),
			Ports:        ports.CopyServicePorts(service.Spec.Ports),
		},
	}
}

func ToK8sService(req router.Request, service *v1.ServiceInstance) (result []kclient.Object, missing []string, err error) {
	cfg, err := config.Get(req.Ctx, req.Client)
	if err != nil {
		return nil, nil, err
	}

	defer func() {
		if err != nil {
			return
		}
		if service.Spec.Default {
			for _, obj := range result {
				if svc, ok := obj.(*corev1.Service); ok {
					result = append(result, toDefaultService(cfg, service, svc))
					return
				}
			}
		}
	}()

	var waiting bool

	defer func() {
		if err != nil {
			return
		}
		cond := condition.ForName(service, v1.ServiceInstanceConditionDefined)
		if waiting {
			if service.Spec.Job == "" {
				cond.Unknown("waiting to be defined")
			} else {
				cond.Unknown(fmt.Sprintf("waiting for job [%s]", service.Spec.Job))
			}
		} else {
			cond.Success()
		}
	}()

	if service.Spec.External != "" {
		return toExternalService(req.Ctx, req.Client, cfg, service)
	} else if service.Spec.Alias != "" {
		return toAliasService(req.Ctx, req.Client, cfg, service)
	} else if service.Spec.Address != "" {
		return toAddressService(service), nil, nil
	} else if service.Spec.Container != "" {
		return toContainerService(service), nil, nil
	} else if len(service.Spec.ContainerLabels) > 0 {
		return toContainerLabelsService(service), nil, nil
	}
	return
}
