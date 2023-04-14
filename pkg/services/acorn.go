package services

import (
	"context"
	"errors"

	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/jobs"
	"github.com/acorn-io/acorn/pkg/labels"
	ports2 "github.com/acorn-io/acorn/pkg/ports"
	"github.com/acorn-io/baaah/pkg/apply"
	"github.com/acorn-io/baaah/pkg/typed"
	apierror "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func publishMode(app *v1.AppInstance) v1.PublishMode {
	if app.Spec.GetStopped() {
		return v1.PublishModeNone
	}
	return app.Spec.PublishMode
}

func forDefined(ctx context.Context, c kclient.Client, appInstance *v1.AppInstance) (result []kclient.Object, _ error) {
	for _, entry := range typed.Sorted(appInstance.Status.AppSpec.Services) {
		serviceName, service := entry.Key, entry.Value

		if ports2.IsLinked(appInstance, serviceName) {
			continue
		}

		// service acorn, skip because acorn will be defined
		if service.Image != "" {
			continue
		}

		annotations := map[string]string{}

		// generated service, will be defined elsewhere
		if service.GetJob() != "" {
			service = *service.DeepCopy()
			_, err := jobs.GetOutputFor(ctx, c, appInstance, service.GetJob(), serviceName, &service)
			if errors.Is(err, jobs.ErrJobNotDone) || errors.Is(err, jobs.ErrJobNoOutput) || apierror.IsNotFound(err) {
				annotations[apply.AnnotationUpdate] = "false"
				annotations[apply.AnnotationCreate] = "false"
			} else if err != nil && !apierror.IsNotFound(err) {
				return nil, err
			}
		}

		result = append(result, &v1.ServiceInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name:        serviceName,
				Namespace:   appInstance.Status.Namespace,
				Labels:      labels.Managed(appInstance, labels.AcornServiceName, serviceName),
				Annotations: annotations,
			},
			Spec: v1.ServiceInstanceSpec{
				AppName:      appInstance.Name,
				AppNamespace: appInstance.Namespace,
				PublishMode:  publishMode(appInstance),
				Publish:      ports2.PortPublishForService(serviceName, appInstance.Spec.Publish),
				Labels: labels.Merge(labels.Managed(appInstance, labels.AcornServiceName, serviceName),
					labels.GatherScoped(serviceName, v1.LabelTypeService,
						appInstance.Status.AppSpec.Labels, asMap(service.Labels), appInstance.Spec.Labels)),
				Annotations: labels.GatherScoped(serviceName, v1.LabelTypeService,
					appInstance.Status.AppSpec.Annotations, asMap(service.Annotations), appInstance.Spec.Annotations),
				Default:   service.Default,
				External:  service.External,
				Address:   service.Address,
				Ports:     service.Ports,
				Container: service.Container,
				Secrets:   asSlice(service.Secrets),
				Data:      service.Data,
				Job:       service.GetJob(),
			},
		})
	}

	return
}

func asSlice(s v1.SecretBindings) (result []string) {
	for _, s := range s {
		if s.Target != "" {
			result = append(result, s.Target)
		}
	}
	return
}

func asMap(s v1.ScopedLabels) map[string]string {
	result := map[string]string{}
	for _, s := range s {
		result[s.Key] = s.Value
	}
	return result
}

func forRouters(appInstance *v1.AppInstance) (result []kclient.Object) {
	for _, entry := range typed.Sorted(appInstance.Status.AppSpec.Routers) {
		routerName, router := entry.Key, entry.Value

		if ports2.IsLinked(appInstance, routerName) {
			continue
		}

		result = append(result, &v1.ServiceInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name:      routerName,
				Namespace: appInstance.Status.Namespace,
				Labels:    labels.Managed(appInstance, labels.AcornRouterName, routerName),
			},
			Spec: v1.ServiceInstanceSpec{
				AppName:      appInstance.Name,
				AppNamespace: appInstance.Namespace,
				PublishMode:  publishMode(appInstance),
				Publish:      ports2.PortPublishForService(routerName, appInstance.Spec.Publish),
				Routes:       router.Routes,
				Labels: labels.Merge(labels.Managed(appInstance, labels.AcornRouterName, routerName),
					labels.GatherScoped(routerName, v1.LabelTypeRouter,
						appInstance.Status.AppSpec.Labels, router.Labels, appInstance.Spec.Labels)),
				Annotations: labels.GatherScoped(routerName, v1.LabelTypeRouter,
					appInstance.Status.AppSpec.Annotations, router.Annotations, appInstance.Spec.Annotations),
				Ports: []v1.PortDef{
					ports2.RouterPortDef,
				},
				ContainerLabels: labels.Managed(appInstance, labels.AcornRouterName, routerName),
			},
		})
	}

	return
}

func forContainers(appInstance *v1.AppInstance) (result []kclient.Object) {
	for _, entry := range typed.Sorted(appInstance.Status.AppSpec.Containers) {
		containerName, container := entry.Key, entry.Value

		if ports2.IsLinked(appInstance, containerName) {
			continue
		}

		ports := ports2.CollectContainerPorts(&container)
		if len(ports) == 0 {
			continue
		}

		result = append(result, &v1.ServiceInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name:      containerName,
				Namespace: appInstance.Status.Namespace,
				Labels:    labels.Managed(appInstance, labels.AcornContainerName, containerName),
			},
			Spec: v1.ServiceInstanceSpec{
				AppName:      appInstance.Name,
				AppNamespace: appInstance.Namespace,
				PublishMode:  publishMode(appInstance),
				Publish:      ports2.PortPublishForService(containerName, appInstance.Spec.Publish),
				Labels: labels.Merge(labels.Managed(appInstance, labels.AcornContainerName, containerName),
					labels.GatherScoped(containerName, v1.LabelTypeContainer,
						appInstance.Status.AppSpec.Labels, container.Labels, appInstance.Spec.Labels)),
				Annotations: labels.GatherScoped(containerName, v1.LabelTypeContainer,
					appInstance.Status.AppSpec.Annotations, container.Annotations, appInstance.Spec.Annotations),
				Ports:     ports,
				Container: containerName,
			},
		})
	}

	return
}

func forLinkedServices(app *v1.AppInstance) (result []kclient.Object) {
	for _, link := range app.Spec.Links {
		newService := &v1.ServiceInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name:      link.Target,
				Namespace: app.Status.Namespace,
				Labels: labels.Managed(app,
					labels.AcornLinkName, link.Service),
			},
			Spec: v1.ServiceInstanceSpec{
				AppName:      app.Name,
				AppNamespace: app.Namespace,
				PublishMode:  publishMode(app),
				Publish:      ports2.PortPublishForService(link.Target, app.Spec.Publish),
				External:     link.Service,
			},
		}
		result = append(result, newService)
	}

	return
}

func ToAcornServices(ctx context.Context, c kclient.Client, appInstance *v1.AppInstance) (result []kclient.Object, _ error) {
	objs, err := forDefined(ctx, c, appInstance)
	if err != nil {
		return nil, err
	}
	result = append(result, objs...)
	result = append(result, forContainers(appInstance)...)
	result = append(result, forRouters(appInstance)...)
	// determine default before adding linked services
	if len(result) == 1 {
		result[0].(*v1.ServiceInstance).Spec.Default = true
	}
	result = append(result, forLinkedServices(appInstance)...)
	return
}
