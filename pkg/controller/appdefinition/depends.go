package appdefinition

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/acorn-io/baaah/pkg/apply"
	"github.com/acorn-io/baaah/pkg/typed"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/labels"
	"k8s.io/apimachinery/pkg/util/sets"
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

	consumerPermsOk := len(app.Status.DeniedConsumerPermissions) == 0

	if !allReady || !consumerPermsOk {
		result[apply.AnnotationCreate] = "false"
		if !app.GetStopped() {
			result[apply.AnnotationUpdate] = "false"
		}
	}

	if _, ok := app.Status.AppSpec.Containers[containerOrJobName]; ok {
		s := app.Status.AppStatus.Containers[containerOrJobName]
		s.Dependencies = depStatus

		if !s.Ready {
			msg, blocked := isBlocked(s.Dependencies, s.ExpressionErrors)
			if blocked {
				s.State = "waiting"
			}
			s.TransitioningMessages = append(s.TransitioningMessages, msg...)
		}

		if app.Status.AppStatus.Containers == nil {
			app.Status.AppStatus.Containers = map[string]v1.ContainerStatus{}
		}
		app.Status.AppStatus.Containers[containerOrJobName] = s
	} else if _, ok = app.Status.AppSpec.Jobs[containerOrJobName]; ok {
		s := app.Status.AppStatus.Jobs[containerOrJobName]
		s.Dependencies = depStatus

		if !s.Ready {
			msg, blocked := isBlocked(s.Dependencies, s.ExpressionErrors)
			if blocked {
				s.State = "waiting"
			}
			s.TransitioningMessages = append(s.TransitioningMessages, msg...)
		}

		if app.Status.AppStatus.Jobs == nil {
			app.Status.AppStatus.Jobs = map[string]v1.JobStatus{}
		}
		app.Status.AppStatus.Jobs[containerOrJobName] = s
	}

	return result
}

func isBlocked(dependencies map[string]v1.DependencyStatus, expressionErrors []v1.ExpressionError) (result []string, _ bool) {
	groupedByTypeName := map[string][]string{}

	for depName, dep := range dependencies {
		var key string
		if dep.Missing {
			key = string(dep.DependencyType) + " to be created"
		} else if !dep.Ready {
			key = string(dep.DependencyType) + " to be ready"
		} else {
			continue
		}
		groupedByTypeName[key] = append(groupedByTypeName[key], depName)
	}

	for _, exprError := range expressionErrors {
		if exprError.DependencyNotFound != nil && exprError.DependencyNotFound.SubKey == "" {
			key := string(exprError.DependencyNotFound.DependencyType) + " to be created"
			groupedByTypeName[key] = append(groupedByTypeName[key], exprError.DependencyNotFound.Name)
		}
	}

	for _, key := range typed.SortedKeys(groupedByTypeName) {
		values := sets.New(groupedByTypeName[key]...).UnsortedList()
		slices.Sort(values)
		msg := fmt.Sprintf("waiting for %s [%s]", key, strings.Join(values, ", "))
		result = append(result, msg)
	}

	return result, len(result) > 0
}
