package services

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/acorn-io/baaah/pkg/apply"
	"github.com/acorn-io/baaah/pkg/typed"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/jobs"
	"github.com/acorn-io/runtime/pkg/labels"
	ports2 "github.com/acorn-io/runtime/pkg/ports"
	"github.com/acorn-io/runtime/pkg/publicname"
	"github.com/acorn-io/runtime/pkg/secrets"
	apierror "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func publishMode(app *v1.AppInstance) v1.PublishMode {
	if app.GetStopped() {
		return v1.PublishModeNone
	}
	return app.Spec.PublishMode
}

func forDefined(ctx context.Context, c kclient.Client, interpolator *secrets.Interpolator, appInstance *v1.AppInstance) (result []kclient.Object, _ error) {
	for _, entry := range typed.Sorted(appInstance.Status.AppSpec.Services) {
		serviceName, service := entry.Key, entry.Value

		if ports2.IsLinked(appInstance, serviceName) {
			continue
		}

		if service.Image != "" {
			// Setup external service to the default service of the app
			service = v1.Service{
				Default:  service.Default,
				External: publicname.ForChild(appInstance, serviceName),
			}
		}

		annotations := map[string]string{
			labels.AcornAppGeneration:        strconv.FormatInt(appInstance.Generation, 10),
			labels.AcornConfigHashAnnotation: appInstance.Status.AppStatus.Services[serviceName].ConfigHash,
		}

		if service.GetJob() != "" {
			service = *service.DeepCopy()
			// Populate service from job output
			err := jobs.GetOutputFor(ctx, c, appInstance, service.GetJob(), serviceName, &service)
			if errors.Is(err, jobs.ErrJobNotDone) || errors.Is(err, jobs.ErrJobNoOutput) || apierror.IsNotFound(err) {
				annotations[apply.AnnotationUpdate] = "false"
				annotations[apply.AnnotationCreate] = "false"
			} else if err != nil {
				interpolator.ForService(serviceName).AddError(err)
				continue
			}
		}

		result = append(result, &v1.ServiceInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name:      serviceName,
				Namespace: appInstance.Status.Namespace,
				Labels: labels.Managed(appInstance,
					labels.AcornPublicName, publicname.ForChild(appInstance, serviceName),
					labels.AcornServiceName, serviceName),
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
				Default:         service.Default,
				External:        service.External,
				Alias:           service.Alias,
				Address:         service.Address,
				Ports:           ports2.FilterDevPorts(service.Ports, appInstance.Status.GetDevMode()),
				Container:       service.Container,
				Secrets:         asSlice(service.Secrets),
				Data:            service.Data,
				Job:             service.GetJob(),
				Consumer:        service.Consumer,
				ContainerLabels: service.ContainerLabels,
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

func serviceNames(appInstance *v1.AppInstance) sets.Set[string] {
	result := sets.New[string]()
	for k := range appInstance.Status.AppSpec.Services {
		result.Insert(k)
	}
	for k := range appInstance.Status.AppSpec.Containers {
		result.Insert(k)
	}
	for k := range appInstance.Status.AppSpec.Jobs {
		result.Insert(k)
	}
	for k := range appInstance.Status.AppSpec.Routers {
		result.Insert(k)
	}
	for k := range appInstance.Status.AppSpec.Acorns {
		result.Insert(k)
	}
	return result
}

func forRouters(appInstance *v1.AppInstance) (result []kclient.Object, err error) {
	serviceNames := serviceNames(appInstance)

	for _, entry := range typed.Sorted(appInstance.Status.AppSpec.Routers) {
		routerName, router := entry.Key, entry.Value

		if ports2.IsLinked(appInstance, routerName) {
			continue
		}

		annotations := map[string]string{}

		for _, router := range router.Routes {
			if router.TargetServiceName != "" && !serviceNames.Has(router.TargetServiceName) {
				if appInstance.Status.AppStatus.Routers == nil {
					appInstance.Status.AppStatus.Routers = map[string]v1.RouterStatus{}
				}
				status := appInstance.Status.AppStatus.Routers[routerName]
				status.MissingTargets = append(status.MissingTargets, router.TargetServiceName)
				appInstance.Status.AppStatus.Routers[routerName] = status
				annotations[apply.AnnotationCreate] = "false"
				annotations[apply.AnnotationUpdate] = "false"
			}
		}

		result = append(result, &v1.ServiceInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name:      routerName,
				Namespace: appInstance.Status.Namespace,
				Labels: labels.Managed(appInstance,
					labels.AcornPublicName, publicname.ForChild(appInstance, routerName),
					labels.AcornRouterName, routerName),
				Annotations: labels.Merge(annotations, map[string]string{
					labels.AcornAppGeneration:        strconv.FormatInt(appInstance.Generation, 10),
					labels.AcornConfigHashAnnotation: appInstance.Status.AppStatus.Routers[routerName].ConfigHash,
				}),
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

func forAcorns(appInstance *v1.AppInstance) (result []kclient.Object) {
	for _, entry := range typed.Sorted(appInstance.Status.AppSpec.Acorns) {
		acornName, acorn := entry.Key, entry.Value

		if ports2.IsLinked(appInstance, acornName) {
			continue
		}

		result = append(result, &v1.ServiceInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name:      acornName,
				Namespace: appInstance.Status.Namespace,
				Labels: labels.Managed(appInstance,
					labels.AcornPublicName, publicname.ForChild(appInstance, acornName),
					labels.AcornAcornName, acornName),
				Annotations: map[string]string{
					labels.AcornAppGeneration:        strconv.FormatInt(appInstance.Generation, 10),
					labels.AcornConfigHashAnnotation: appInstance.Status.AppStatus.Acorns[acornName].ConfigHash,
				},
			},
			Spec: v1.ServiceInstanceSpec{
				AppName:      appInstance.Name,
				AppNamespace: appInstance.Namespace,
				External:     publicname.ForChild(appInstance, acornName),
				Labels: labels.Merge(labels.Managed(appInstance, labels.AcornAcornName, acornName),
					labels.GatherScoped(acornName, v1.LabelTypeAcorn,
						appInstance.Status.AppSpec.Labels, selfScope(acorn.Labels), appInstance.Spec.Labels)),
				Annotations: labels.GatherScoped(acornName, v1.LabelTypeAcorn,
					appInstance.Status.AppSpec.Annotations, selfScope(acorn.Annotations), appInstance.Spec.Annotations),
			},
		})
	}

	return
}

func selfScope(scopedLabels v1.ScopedLabels) map[string]string {
	labelMap := make(map[string]string)
	for _, s := range scopedLabels {
		if s.ResourceType == v1.LabelTypeMeta || (s.ResourceType == "" && s.ResourceName == "") {
			labelMap[s.Key] = s.Value
		}
	}
	return labelMap
}

func forContainers(interpolator *secrets.Interpolator, appInstance *v1.AppInstance) (result []kclient.Object) {
	for _, entry := range typed.Sorted(appInstance.Status.AppSpec.Containers) {
		containerName, container := entry.Key, entry.Value

		if ports2.IsLinked(appInstance, containerName) {
			continue
		}

		ports := ports2.CollectContainerPorts(&container, appInstance.Status.GetDevMode())
		if len(ports) == 0 {
			continue
		}

		var failed bool
		interp := interpolator.ForContainer(containerName)
		for i, p := range ports {
			replacement, err := interp.Replace(p.Path)
			if err != nil {
				interp.AddError(err)
				failed = true
				continue
			}

			ports[i].Path = replacement
		}

		if failed {
			continue
		}

		result = append(result, &v1.ServiceInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name:      containerName,
				Namespace: appInstance.Status.Namespace,
				Labels: labels.Managed(appInstance,
					labels.AcornPublicName, publicname.ForChild(appInstance, containerName),
					labels.AcornContainerName, containerName),
				Annotations: map[string]string{
					labels.AcornAppGeneration:        strconv.FormatInt(appInstance.Generation, 10),
					labels.AcornConfigHashAnnotation: appInstance.Status.AppStatus.Containers[containerName].ConfigHash,
				},
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

func forJobs(interpolator *secrets.Interpolator, appInstance *v1.AppInstance) (result []kclient.Object) {
	for _, entry := range typed.Sorted(appInstance.Status.AppSpec.Jobs) {
		jobName, job := entry.Key, entry.Value

		if ports2.IsLinked(appInstance, jobName) {
			continue
		}

		ports := ports2.CollectContainerPorts(&job, appInstance.Status.GetDevMode())
		if len(ports) == 0 {
			continue
		}

		var failed bool
		interp := interpolator.ForJob(jobName)
		for i, p := range ports {
			replacement, err := interp.Replace(p.Path)
			if err != nil {
				interp.AddError(err)
				failed = true
				continue
			}

			ports[i].Path = replacement
		}

		if failed {
			continue
		}

		publish := publishMode(appInstance)
		if appInstance.Status.AppStatus.Jobs[jobName].State == "completed" {
			// Don't publish endpoints for jobs that are completed
			publish = v1.PublishModeNone
		}

		result = append(result, &v1.ServiceInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name:      jobName,
				Namespace: appInstance.Status.Namespace,
				Labels: labels.Managed(appInstance,
					labels.AcornPublicName, publicname.ForChild(appInstance, jobName),
					labels.AcornJobName, jobName),
				Annotations: map[string]string{
					labels.AcornAppGeneration:        strconv.FormatInt(appInstance.Generation, 10),
					labels.AcornConfigHashAnnotation: appInstance.Status.AppStatus.Jobs[jobName].ConfigHash,
				},
			},
			Spec: v1.ServiceInstanceSpec{
				AppName:      appInstance.Name,
				AppNamespace: appInstance.Namespace,
				PublishMode:  publish,
				Publish:      ports2.PortPublishForService(jobName, appInstance.Spec.Publish),
				Labels: labels.Merge(labels.Managed(appInstance, labels.AcornJobName, jobName),
					labels.GatherScoped(jobName, v1.LabelTypeJob,
						appInstance.Status.AppSpec.Labels, job.Labels, appInstance.Spec.Labels)),
				Annotations: labels.GatherScoped(jobName, v1.LabelTypeJob,
					appInstance.Status.AppSpec.Annotations, job.Annotations, appInstance.Spec.Annotations),
				Ports:     ports,
				Container: jobName,
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
					labels.AcornPublicName, publicname.ForChild(app, link.Target),
					labels.AcornLinkName, link.Service),
				Annotations: map[string]string{
					labels.AcornAppGeneration: strconv.FormatInt(app.Generation, 10),
				},
			},
			Spec: v1.ServiceInstanceSpec{
				AppName:      app.Name,
				AppNamespace: app.Namespace,
				PublishMode:  publishMode(app),
				Publish:      ports2.PortPublishForService(link.Target, app.Spec.Publish),
				External:     link.Service,
				Labels: labels.Managed(app,
					labels.AcornPublicName, publicname.ForChild(app, link.Target),
					labels.AcornLinkName, link.Service),
			},
		}
		result = append(result, newService)
	}

	return
}

func findDefaultServiceName(appInstance *v1.AppInstance) (string, error) {
	// I don't like the behavior. It should be more explicit and not magically pick a default if one doesn't exist.
	// But right now there's too much going on to change the behavior. Maybe we can do better in the future.
	defaultName := ""
	for _, entry := range typed.Sorted(appInstance.Status.AppSpec.Services) {
		if entry.Value.Default {
			if defaultName != "" {
				return "", fmt.Errorf("multiple default services specified [%s] and [%s]", defaultName, entry.Key)
			}
			defaultName = entry.Key
		}
	}

	if defaultName != "" {
		return defaultName, nil
	}

	containers := sets.New[string]()
	for name, container := range appInstance.Status.AppSpec.Containers {
		if len(ports2.CollectContainerPorts(&container, appInstance.Status.GetDevMode())) > 0 {
			containers.Insert(name)
		}
	}

	// just pick the first one we find now if there is only one choice
	if containers.Len()+
		len(appInstance.Status.AppSpec.Services)+
		len(appInstance.Status.AppSpec.Acorns)+
		len(appInstance.Status.AppSpec.Routers) != 1 {
		return "", nil
	}

	for _, name := range typed.SortedKeys(appInstance.Status.AppSpec.Services) {
		return name, nil
	}

	for _, name := range typed.SortedKeys(appInstance.Status.AppSpec.Acorns) {
		return name, nil
	}

	for _, name := range sets.List(containers) {
		return name, nil
	}

	for _, name := range typed.SortedKeys(appInstance.Status.AppSpec.Routers) {
		return name, nil
	}

	return "", nil
}

func ToAcornServices(ctx context.Context, c kclient.Client, interpolator *secrets.Interpolator, appInstance *v1.AppInstance) (result []kclient.Object, _ error) {
	objs, err := forDefined(ctx, c, interpolator, appInstance)
	if err != nil {
		return nil, err
	}
	result = append(result, objs...)
	result = append(result, forAcorns(appInstance)...)
	result = append(result, forContainers(interpolator, appInstance)...)
	result = append(result, forJobs(interpolator, appInstance)...)

	routers, err := forRouters(appInstance)
	if err != nil {
		return nil, err
	}
	result = append(result, routers...)
	result = append(result, forLinkedServices(appInstance)...)

	defaultName, err := findDefaultServiceName(appInstance)
	if err != nil {
		return nil, err
	}

	for _, obj := range result {
		service := obj.(*v1.ServiceInstance)
		if service.GetName() == defaultName {
			service.Spec.Default = true
		} else {
			service.Spec.Default = false
		}
	}

	result = filterForPermissionsAndAssignStatus(appInstance, result)
	return
}

// filterForPermissionsAndAssignStatus filters out services that are not granted permissions
// and updates the appInstance status with the missing consumer permissions
func filterForPermissionsAndAssignStatus(appInstance *v1.AppInstance, services []kclient.Object) (result []kclient.Object) {
	result = make([]kclient.Object, 0, len(services))
	for _, obj := range services {
		svc, ok := obj.(*v1.ServiceInstance)
		if !ok {
			continue
		}

		ungranted := getUngranted(appInstance, svc)
		if len(ungranted) > 0 {
			serviceStatus := appInstance.Status.AppStatus.Services[svc.Name]
			serviceStatus.MissingConsumerPermissions = append(serviceStatus.MissingConsumerPermissions, v1.Permissions{
				ServiceName: svc.Name,
				Rules:       ungranted,
			})

			if appInstance.Status.AppStatus.Services == nil {
				appInstance.Status.AppStatus.Services = map[string]v1.ServiceStatus{}
			}
			appInstance.Status.AppStatus.Services[svc.Name] = serviceStatus
		} else {
			result = append(result, svc)
		}
	}
	return result
}

// getUngranted returns the list of rules that are not granted to the appInstance but are requested by the serviceInstance
func getUngranted(appInstance *v1.AppInstance, service *v1.ServiceInstance) []v1.PolicyRule {
	if service.Spec.Consumer == nil || service.Spec.Consumer.Permissions == nil ||
		len(service.Spec.Consumer.Permissions.GetRules()) == 0 {
		return nil
	}

	var (
		ungranted []v1.PolicyRule
		granted   = appInstance.Spec.GetGrantedPermissions()
	)

	for _, requested := range service.Spec.Consumer.Permissions.GetRules() {
		var isGranted bool
		for _, granted := range granted {
			if granted.Grants(appInstance.Namespace, service.Name, requested) {
				isGranted = true
				break
			}
		}
		if !isGranted {
			ungranted = append(ungranted, requested)
		}
	}

	return ungranted
}
