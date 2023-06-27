package appstatus

import (
	"fmt"
	"strconv"

	"github.com/acorn-io/baaah/pkg/router"
	"github.com/acorn-io/baaah/pkg/typed"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/labels"
	"github.com/acorn-io/runtime/pkg/ports"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierror "k8s.io/apimachinery/pkg/api/errors"
	klabels "k8s.io/apimachinery/pkg/labels"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func (a *appStatusRenderer) readContainers() error {
	var (
		isTransitioning bool
		existingStatus  = a.app.Status.AppStatus.Containers
	)

	// reset state
	a.app.Status.AppStatus.Containers = make(map[string]v1.ContainerStatus, len(a.app.Status.AppSpec.Containers))

	summary, err := a.getReplicasSummary(labels.AcornContainerName)
	if err != nil {
		return err
	}

	for containerName := range a.app.Status.AppSpec.Containers {
		var cs v1.ContainerStatus
		summary := summary[containerName]

		cs.Defined = ports.IsLinked(a.app, containerName)
		cs.LinkOverride = ports.LinkService(a.app, containerName)
		cs.ErrorMessages = append(cs.ErrorMessages, summary.ErrorMessages...)
		cs.ExpressionErrors = existingStatus[containerName].ExpressionErrors
		cs.TransitioningMessages = append(cs.TransitioningMessages, summary.TransitioningMessages...)
		cs.MaxReplicaRestartCount = summary.MaxReplicaRestartCount

		dep := appsv1.Deployment{}
		err := a.c.Get(a.ctx, router.Key(a.app.Status.Namespace, containerName), &dep)
		if apierror.IsNotFound(err) {
			// do nothing
		} else if err != nil {
			return err
		} else {
			cs.UpToDate = dep.Annotations[labels.AcornAppGeneration] == strconv.Itoa(int(a.app.Generation))
			cs.ReadyReplicaCount = dep.Status.ReadyReplicas
			cs.RunningReplicaCount = dep.Status.Replicas
			cs.DesiredReplicaCount = replicas(dep.Spec.Replicas)
			cs.UpToDateReplicaCount = dep.Status.UpdatedReplicas
			cs.Defined = true

			if cs.UpToDate && cs.ReadyReplicaCount == cs.DesiredReplicaCount && len(cs.ExpressionErrors) == 0 {
				cs.Ready, err = a.isDepReady(&dep)
				if err != nil {
					return err
				}
			}
		}

		if cs.LinkOverride != "" {
			var err error
			cs.UpToDate = true
			cs.Ready, cs.Defined, err = a.isServiceReady(containerName)
			if err != nil {
				return err
			}
		}

		if len(cs.TransitioningMessages) > 0 {
			isTransitioning = true
		}

		for _, entry := range typed.Sorted(cs.Dependencies) {
			depName, dep := entry.Key, entry.Value
			if !dep.Ready {
				cs.Ready = false
				msg := fmt.Sprintf("%s %s dependency is not ready", dep.DependencyType, depName)
				if dep.Missing {
					msg = fmt.Sprintf("%s %s dependency is missing", dep.DependencyType, depName)
				}
				cs.TransitioningMessages = append(cs.TransitioningMessages, msg)
			}
		}

		addExpressionErrors(&cs.CommonStatus, cs.ExpressionErrors)

		a.app.Status.AppStatus.Containers[containerName] = cs
	}

	a.app.Status.AppStatus.Stopped = false
	if !isTransitioning && a.app.GetStopped() {
		allZero := true
		for _, v := range a.app.Status.AppStatus.Containers {
			if v.DesiredReplicaCount != 0 {
				allZero = false
				break
			}
		}
		if allZero {
			a.app.Status.AppStatus.Stopped = true
		}
	}

	return nil
}

func (a *appStatusRenderer) isDepReady(dep *appsv1.Deployment) (bool, error) {
	available := false
	for _, cond := range dep.Status.Conditions {
		if cond.Type == "Available" && cond.Status == corev1.ConditionTrue {
			available = true
			break
		}
	}

	if !available {
		return false, nil
	}

	if dep.Annotations[labels.AcornAppGeneration] != strconv.Itoa(int(a.app.Generation)) ||
		dep.Status.ObservedGeneration != dep.Generation ||
		dep.Status.Replicas != dep.Status.ReadyReplicas ||
		dep.Status.Replicas != dep.Status.UpdatedReplicas {
		return false, nil
	}

	reps := &appsv1.ReplicaSetList{}
	err := a.c.List(a.ctx, reps, &kclient.ListOptions{
		LabelSelector: klabels.SelectorFromSet(map[string]string{
			labels.AcornAppName:       a.app.Name,
			labels.AcornAppNamespace:  a.app.Namespace,
			labels.AcornContainerName: dep.Labels[labels.AcornContainerName],
		}),
		Namespace: a.app.Status.Namespace,
	})
	if err != nil {
		return false, nil
	}

	for _, rep := range reps.Items {
		if rep.Annotations[labels.AcornAppGeneration] == strconv.Itoa(int(a.app.Generation)) &&
			rep.Generation == rep.Status.ObservedGeneration &&
			rep.Status.Replicas == rep.Status.ReadyReplicas &&
			rep.Status.Replicas == rep.Status.AvailableReplicas {
			return true, nil
		}
	}

	return false, nil
}

func replicas(replicas *int32) int32 {
	if replicas == nil {
		return 1
	}
	return *replicas
}
