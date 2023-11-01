package appstatus

import (
	"fmt"
	"strconv"

	"github.com/acorn-io/baaah/pkg/router"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/labels"
	"github.com/acorn-io/runtime/pkg/ports"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierror "k8s.io/apimachinery/pkg/api/errors"
	klabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/kubectl/pkg/util/deployment"
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

	for containerName, containerDef := range a.app.Status.AppSpec.Containers {
		var cs v1.ContainerStatus
		summary := summary[containerName]

		cs.Defined = ports.IsLinked(a.app, containerName)
		cs.LinkOverride = ports.LinkService(a.app, containerName)
		cs.ErrorMessages = append(cs.ErrorMessages, summary.ErrorMessages...)
		cs.ExpressionErrors = existingStatus[containerName].ExpressionErrors
		cs.Dependencies = existingStatus[containerName].Dependencies
		cs.TransitioningMessages = append(cs.TransitioningMessages, summary.TransitioningMessages...)
		cs.MaxReplicaRestartCount = summary.MaxReplicaRestartCount
		hash, err := configHash(containerDef)
		if err != nil {
			return err
		}
		cs.ConfigHash = hash

		dep := appsv1.Deployment{}
		err = a.c.Get(a.ctx, router.Key(a.app.Status.Namespace, containerName), &dep)
		if apierror.IsNotFound(err) {
			// do nothing
		} else if err != nil {
			return err
		} else {
			cs.Defined = true
			cs.UpToDate = dep.Annotations[labels.AcornAppGeneration] == strconv.Itoa(int(a.app.Generation)) && dep.Annotations[labels.AcornConfigHashAnnotation] == hash
			cs.ReadyReplicaCount = dep.Status.ReadyReplicas
			cs.RunningReplicaCount = dep.Status.Replicas
			cs.DesiredReplicaCount = replicas(dep.Spec.Replicas)
			cs.UpToDateReplicaCount = dep.Status.UpdatedReplicas

			if cs.UpToDate && cs.ReadyReplicaCount == cs.DesiredReplicaCount && cs.UpToDateReplicaCount >= cs.DesiredReplicaCount {
				cs.Ready, err = a.isDeploymentReady(&dep)
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

func setContainerMessages(app *v1.AppInstance) {
	for containerName, cs := range app.Status.AppStatus.Containers {
		addExpressionErrors(&cs.CommonStatus, cs.ExpressionErrors)

		// Not ready if we have any error messages
		if len(cs.ErrorMessages) > 0 {
			cs.Ready = false
		}

		if cs.Ready {
			if app.GetStopped() {
				cs.State = "stopped"
			} else {
				cs.State = "running"
			}
		} else if cs.UpToDate {
			if len(cs.ErrorMessages) > 0 {
				cs.State = "failing"
			} else {
				cs.State = "not ready"
			}
		} else if cs.Defined {
			if len(cs.ErrorMessages) > 0 {
				cs.State = "error"
			} else {
				cs.State = "updating"
			}
		} else {
			if len(cs.ErrorMessages) > 0 {
				cs.State = "error"
			} else {
				cs.State = "pending"
			}
		}

		if !cs.Ready {
			msg, blocked := isBlocked(cs.Dependencies, cs.ExpressionErrors)
			if blocked {
				cs.State = "waiting"
			}
			cs.TransitioningMessages = append(cs.TransitioningMessages, msg...)
		}

		// Add informative messages if all else is healthy
		if len(cs.TransitioningMessages) == 0 && len(cs.ErrorMessages) == 0 {
			if cs.RunningReplicaCount > 1 {
				cs.Messages = append(cs.Messages, fmt.Sprintf("%d running replicas", cs.RunningReplicaCount))
			}
			if cs.MaxReplicaRestartCount > 0 {
				cs.Messages = append(cs.Messages, fmt.Sprintf("%d container restarts", cs.MaxReplicaRestartCount))
			}
		}

		app.Status.AppStatus.Containers[containerName] = cs
	}
}

func (a *appStatusRenderer) isDeploymentReady(dep *appsv1.Deployment) (bool, error) {
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
		if rep.Annotations[deployment.RevisionAnnotation] == dep.Annotations[deployment.RevisionAnnotation] &&
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
