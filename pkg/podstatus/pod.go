package podstatus

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/duration"
)

var (
	podSuccessConditions = []metav1.TableRowCondition{{Type: metav1.RowCompleted, Status: metav1.ConditionTrue, Reason: string(corev1.PodSucceeded), Message: "The pod has completed successfully."}}
	podFailedConditions  = []metav1.TableRowCondition{{Type: metav1.RowCompleted, Status: metav1.ConditionTrue, Reason: string(corev1.PodFailed), Message: "The pod failed."}}
)

type Status struct {
	Restarts        int
	RestartMessage  string
	TotalContainers int
	ReadyContainers int
	Reason          string
}

func (s Status) String() string {
	buf := &strings.Builder{}
	buf.WriteString(fmt.Sprintf("READY: %d/%d, ", s.ReadyContainers, s.TotalContainers))
	if s.RestartMessage != "" {
		buf.WriteString("RESTARTS: ")
		buf.WriteString(s.RestartMessage)
		buf.WriteString(", ")
	} else if s.Restarts > 0 {
		buf.WriteString("RESTARTS: ")
		buf.WriteString(strconv.Itoa(s.Restarts))
		buf.WriteString(", ")
	}
	buf.WriteString("STATUS: ")
	buf.WriteString(s.Reason)
	return buf.String()
}

func GetStatus(pod *corev1.Pod) Status {
	restarts := 0
	totalContainers := len(pod.Spec.Containers)
	readyContainers := 0
	lastRestartDate := metav1.NewTime(time.Time{})

	reason := string(pod.Status.Phase)
	if pod.Status.Reason != "" {
		reason = pod.Status.Reason
	}

	row := metav1.TableRow{
		Object: runtime.RawExtension{Object: pod},
	}

	switch pod.Status.Phase {
	case corev1.PodSucceeded:
		row.Conditions = podSuccessConditions
	case corev1.PodFailed:
		row.Conditions = podFailedConditions
	}

	initializing := false
	for i := range pod.Status.InitContainerStatuses {
		container := pod.Status.InitContainerStatuses[i]
		restarts += int(container.RestartCount)
		if container.LastTerminationState.Terminated != nil {
			terminatedDate := container.LastTerminationState.Terminated.FinishedAt
			if lastRestartDate.Before(&terminatedDate) {
				lastRestartDate = terminatedDate
			}
		}
		switch {
		case container.State.Terminated != nil && container.State.Terminated.ExitCode == 0:
			continue
		case container.State.Terminated != nil:
			// initialization is failed
			if len(container.State.Terminated.Reason) == 0 {
				if container.State.Terminated.Signal != 0 {
					reason = fmt.Sprintf("Init:Signal:%d", container.State.Terminated.Signal)
				} else {
					reason = fmt.Sprintf("Init:ExitCode:%d", container.State.Terminated.ExitCode)
				}
			} else {
				reason = "Init:" + container.State.Terminated.Reason
			}
			initializing = true
		case container.State.Waiting != nil && len(container.State.Waiting.Reason) > 0 && container.State.Waiting.Reason != "PodInitializing":
			reason = "Init:" + container.State.Waiting.Reason
			initializing = true
		default:
			reason = fmt.Sprintf("Init:%d/%d", i, len(pod.Spec.InitContainers))
			initializing = true
		}
		break
	}
	if !initializing {
		restarts = 0
		hasRunning := false
		for i := len(pod.Status.ContainerStatuses) - 1; i >= 0; i-- {
			container := pod.Status.ContainerStatuses[i]

			restarts += int(container.RestartCount)
			if container.LastTerminationState.Terminated != nil {
				terminatedDate := container.LastTerminationState.Terminated.FinishedAt
				if lastRestartDate.Before(&terminatedDate) {
					lastRestartDate = terminatedDate
				}
			}
			if container.State.Waiting != nil && container.State.Waiting.Reason != "" {
				reason = container.State.Waiting.Reason
			} else if container.State.Terminated != nil && container.State.Terminated.Reason != "" {
				reason = container.State.Terminated.Reason
			} else if container.State.Terminated != nil && container.State.Terminated.Reason == "" {
				if container.State.Terminated.Signal != 0 {
					reason = fmt.Sprintf("Signal:%d", container.State.Terminated.Signal)
				} else {
					reason = fmt.Sprintf("ExitCode:%d", container.State.Terminated.ExitCode)
				}
			} else if container.Ready && container.State.Running != nil {
				hasRunning = true
				readyContainers++
			}
		}

		// change pod status back to "Running" if there is at least one container still reporting as "Running" status
		if reason == "Completed" && hasRunning {
			if hasPodReadyCondition(pod.Status.Conditions) {
				reason = "Running"
			} else {
				reason = "NotReady"
			}
		}
	}

	if pod.DeletionTimestamp != nil && pod.Status.Reason == "NodeLost" {
		reason = "Unknown"
	} else if pod.DeletionTimestamp != nil {
		reason = "Terminating"
	}

	restartsStr := strconv.Itoa(restarts)
	if !lastRestartDate.IsZero() {
		restartsStr = fmt.Sprintf("%d (%s ago)", restarts, translateTimestampSince(lastRestartDate))
	}

	return Status{
		Restarts:        restarts,
		RestartMessage:  restartsStr,
		TotalContainers: totalContainers,
		ReadyContainers: readyContainers,
		Reason:          reason,
	}
}

func hasPodReadyCondition(conditions []corev1.PodCondition) bool {
	for _, condition := range conditions {
		if condition.Type == corev1.PodReady && condition.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}

func translateTimestampSince(timestamp metav1.Time) string {
	if timestamp.IsZero() {
		return "<unknown>"
	}

	return duration.HumanDuration(time.Since(timestamp.Time))
}
