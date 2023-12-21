package appdefinition

import (
	"strconv"

	"github.com/acorn-io/baaah/pkg/apply"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/labels"
)

func getDependencyAnnotations(app *v1.AppInstance, containerOrJobName string, deps []v1.Dependency) map[string]string {
	result := map[string]string{}
	if app.Generation > 0 {
		result[labels.AcornAppGeneration] = strconv.Itoa(int(app.Generation))
	}
	if len(deps) == 0 {
		return result
	}

	depStatus := map[string]v1.DependencyStatus{}
	for _, dep := range deps {
		depStatus[dep.TargetName] = v1.DependencyStatus{
			Missing: true,
		}
		for container := range app.Status.AppSpec.Containers {
			if dep.TargetName == container {
				depStatus[dep.TargetName] = v1.DependencyStatus{
					Ready:          app.Status.AppStatus.Containers[dep.TargetName].Ready,
					DependencyType: v1.DependencyContainer,
				}
			}
		}
		for function := range app.Status.AppSpec.Functions {
			if dep.TargetName == function {
				depStatus[dep.TargetName] = v1.DependencyStatus{
					Ready:          app.Status.AppStatus.Functions[dep.TargetName].Ready,
					DependencyType: v1.DependencyFunction,
				}
			}
		}
		for job := range app.Status.AppSpec.Jobs {
			if dep.TargetName == job {
				depStatus[dep.TargetName] = v1.DependencyStatus{
					Ready:          app.Status.AppStatus.Jobs[dep.TargetName].Ready,
					DependencyType: v1.DependencyJob,
				}
			}
		}
		for service := range app.Status.AppSpec.Services {
			if dep.TargetName == service {
				depStatus[dep.TargetName] = v1.DependencyStatus{
					Ready:          app.Status.AppStatus.Services[dep.TargetName].Ready,
					DependencyType: v1.DependencyService,
				}
			}
		}
	}

	allReady := true
	for _, dep := range depStatus {
		if !dep.Ready {
			allReady = false
			break
		}
	}

	if !allReady || len(app.Status.DeniedConsumerPermissions) != 0 {
		result[apply.AnnotationCreate] = "false"
		if !app.GetStopped() {
			result[apply.AnnotationUpdate] = "false"
		}
	}

	if _, ok := app.Status.AppSpec.Containers[containerOrJobName]; ok {
		s := app.Status.AppStatus.Containers[containerOrJobName]
		s.Dependencies = depStatus

		if app.Status.AppStatus.Containers == nil {
			app.Status.AppStatus.Containers = map[string]v1.ContainerStatus{}
		}
		app.Status.AppStatus.Containers[containerOrJobName] = s
	} else if _, ok := app.Status.AppSpec.Functions[containerOrJobName]; ok {
		s := app.Status.AppStatus.Functions[containerOrJobName]
		s.Dependencies = depStatus

		if app.Status.AppStatus.Functions == nil {
			app.Status.AppStatus.Functions = map[string]v1.ContainerStatus{}
		}
		app.Status.AppStatus.Functions[containerOrJobName] = s
	} else if _, ok = app.Status.AppSpec.Jobs[containerOrJobName]; ok {
		s := app.Status.AppStatus.Jobs[containerOrJobName]
		s.Dependencies = depStatus

		if app.Status.AppStatus.Jobs == nil {
			app.Status.AppStatus.Jobs = map[string]v1.JobStatus{}
		}
		app.Status.AppStatus.Jobs[containerOrJobName] = s
	}

	return result
}
