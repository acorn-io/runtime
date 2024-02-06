package local

import (
	"github.com/acorn-io/baaah/pkg/router"
	"github.com/acorn-io/runtime/pkg/system"
	corev1 "k8s.io/api/core/v1"
)

func DeletePods(req router.Request, _ router.Response) error {
	pod := req.Object.(*corev1.Pod)
	for _, container := range pod.Spec.Containers {
		if container.Image == system.LocalImage {
			return req.Client.Delete(req.Ctx, pod)
		}
	}
	for _, container := range pod.Spec.InitContainers {
		if container.Image == system.LocalImage {
			return req.Client.Delete(req.Ctx, pod)
		}
	}

	if pod.Status.Phase == corev1.PodRunning && len(pod.Status.ContainerStatuses) > 0 {
		allUnknown := true
		for _, status := range pod.Status.ContainerStatuses {
			if status.State.Terminated == nil || status.State.Terminated.Reason != "Unknown" {
				allUnknown = false
				break
			}
		}
		if allUnknown {
			return req.Client.Delete(req.Ctx, pod)
		}
	}

	if pod.Status.Reason == "Evicted" {
		return req.Client.Delete(req.Ctx, pod)
	}

	return nil
}
