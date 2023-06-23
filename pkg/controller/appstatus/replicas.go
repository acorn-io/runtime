package appstatus

import (
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
					summary.TransitioningMessages = append(summary.TransitioningMessages, podName(&pod)+" is not scheduled to a node")
				}
			}
		}

		transition, errored := containerMessages(&pod, append(pod.Status.InitContainerStatuses, pod.Status.ContainerStatuses...))
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

func containerMessages(pod *corev1.Pod, status []corev1.ContainerStatus) (transitionMessages, errorMessages []string) {
	for _, container := range status {
		if container.State.Waiting != nil && container.State.Waiting.Reason != "" {
			if container.State.Waiting.Message == "" {
				transitionMessages = append(transitionMessages, podName(pod)+" "+
					container.State.Waiting.Reason)
			} else {
				transitionMessages = append(transitionMessages, podName(pod)+" "+
					container.State.Waiting.Reason+": "+container.State.Waiting.Message)
			}
		}
		if container.State.Terminated != nil && container.State.Terminated.ExitCode > 0 {
			errorMessages = append(errorMessages, podName(pod)+" "+container.State.Terminated.Reason+": Exit Code "+
				strconv.Itoa(int(container.State.Terminated.ExitCode)))
		}
	}
	return
}

func podName(pod *corev1.Pod) string {
	jobName := pod.Labels[labels.AcornJobName]
	if jobName != "" {
		return jobName
	}
	return pod.Labels[labels.AcornContainerName]
}
