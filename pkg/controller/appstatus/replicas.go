package appstatus

import (
	"fmt"
	"sort"
	"strconv"

	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/labels"
	corev1 "k8s.io/api/core/v1"
	klabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func (a *appStatusRenderer) getReplicasSummary(nameLabel string) (map[string]v1.ReplicasSummary, error) {
	var (
		pods   = &corev1.PodList{}
		result = map[string]v1.ReplicasSummary{}
	)

	sel := klabels.SelectorFromSet(map[string]string{
		labels.AcornManaged: "true",
		labels.AcornAppName: a.app.Name,
	})

	hasNameLabel, err := klabels.NewRequirement(nameLabel, selection.Exists, nil)
	if err != nil {
		return nil, err
	}

	err = a.c.List(a.ctx, pods, &kclient.ListOptions{
		Namespace:     a.app.Status.Namespace,
		LabelSelector: sel.Add(*hasNameLabel),
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(pods.Items, func(i, j int) bool {
		return pods.Items[i].CreationTimestamp.Before(&pods.Items[j].CreationTimestamp)
	})

	for _, pod := range pods.Items {
		var summary v1.ReplicasSummary

		for _, cond := range pod.Status.Conditions {
			if cond.Type == corev1.PodScheduled {
				if cond.Status != corev1.ConditionTrue {
					summary.TransitioningMessages = append(summary.TransitioningMessages, "not scheduled to a node")
				}
			}
		}

		transition, errored := containerMessages(append(pod.Status.InitContainerStatuses, pod.Status.ContainerStatuses...))
		summary.TransitioningMessages = append(summary.TransitioningMessages, transition...)
		summary.ErrorMessages = append(summary.ErrorMessages, errored...)

		for _, status := range append(pod.Status.InitContainerStatuses, pod.Status.ContainerStatuses...) {
			if status.RestartCount > summary.MaxReplicaRestartCount {
				summary.MaxReplicaRestartCount = status.RestartCount
			}
		}

		result[pod.Labels[nameLabel]] = summary
	}

	return result, nil
}

func containerMessages(status []corev1.ContainerStatus) (transitionMessages, errorMessages []string) {
	for _, container := range status {
		if container.State.Waiting != nil {
			if container.LastTerminationState.Terminated != nil {
				msg := fmt.Sprintf("%s: %s", container.State.Waiting.Reason,
					terminatedMessage(container.LastTerminationState))
				errorMessages = append(errorMessages, msg)
			} else {
				suffix := ""
				if container.State.Waiting.Message != "" {
					suffix = ": " + container.State.Waiting.Message
				}
				transitionMessages = append(transitionMessages, container.State.Waiting.Reason+suffix)
			}
		}
		if container.State.Terminated != nil && container.State.Terminated.ExitCode > 0 {
			errorMessages = append(errorMessages, terminatedMessage(container.State))
		}
	}
	return
}

func terminatedMessage(containerState corev1.ContainerState) string {
	msg := containerState.Terminated.Reason + ": Exit Code " +
		strconv.Itoa(int(containerState.Terminated.ExitCode))
	if containerState.Terminated.Message != "" {
		msg += ": " + containerState.Terminated.Message
	}
	return msg
}
