package appstatus

import (
	"fmt"
	"strconv"

	"github.com/acorn-io/baaah/pkg/router"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/labels"
	"github.com/acorn-io/runtime/pkg/ports"
	appsv1 "k8s.io/api/apps/v1"
	apierror "k8s.io/apimachinery/pkg/api/errors"
)

func (a *appStatusRenderer) readFunctions() error {
	var (
		isTransitioning bool
		existingStatus  = a.app.Status.AppStatus.Functions
	)

	// reset state
	a.app.Status.AppStatus.Functions = make(map[string]v1.ContainerStatus, len(a.app.Status.AppSpec.Functions))

	summary, err := a.getReplicasSummary(labels.AcornFunctionName)
	if err != nil {
		return err
	}

	for functionName, functionDef := range a.app.Status.AppSpec.Functions {
		var cs v1.ContainerStatus
		summary := summary[functionName]

		cs.Defined = ports.IsLinked(a.app, functionName)
		cs.LinkOverride = ports.LinkService(a.app, functionName)
		cs.ErrorMessages = append(cs.ErrorMessages, summary.ErrorMessages...)
		cs.ExpressionErrors = existingStatus[functionName].ExpressionErrors
		cs.Dependencies = existingStatus[functionName].Dependencies
		cs.TransitioningMessages = append(cs.TransitioningMessages, summary.TransitioningMessages...)
		cs.MaxReplicaRestartCount = summary.MaxReplicaRestartCount
		hash, err := configHash(functionDef)
		if err != nil {
			return err
		}
		cs.ConfigHash = hash

		dep := appsv1.Deployment{}
		err = a.c.Get(a.ctx, router.Key(a.app.Status.Namespace, functionName), &dep)
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
				cs.Ready, err = a.isDeploymentReady(&dep, labels.AcornFunctionName)
				if err != nil {
					return err
				}
			}
		}

		if cs.LinkOverride != "" {
			var err error
			cs.UpToDate = true
			cs.Ready, cs.Defined, err = a.isServiceReady(functionName)
			if err != nil {
				return err
			}
		}

		if len(cs.TransitioningMessages) > 0 {
			isTransitioning = true
		}

		a.app.Status.AppStatus.Functions[functionName] = cs
	}

	a.app.Status.AppStatus.Stopped = false
	if !isTransitioning && a.app.GetStopped() {
		allZero := true
		for _, v := range a.app.Status.AppStatus.Functions {
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

func setFunctionMessages(app *v1.AppInstance) {
	for functionName, cs := range app.Status.AppStatus.Functions {
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
				cs.Messages = append(cs.Messages, fmt.Sprintf("%d function restarts", cs.MaxReplicaRestartCount))
			}
		}

		app.Status.AppStatus.Functions[functionName] = cs
	}
}
