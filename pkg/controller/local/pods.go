package local

import (
	"github.com/acorn-io/baaah/pkg/router"
	"github.com/acorn-io/runtime/pkg/system"
	corev1 "k8s.io/api/core/v1"
)

func DeletePods(req router.Request, resp router.Response) error {
	pod := req.Object.(*corev1.Pod)
	for _, container := range pod.Spec.Containers {
		if container.Image == system.LocalImage {
			return req.Client.Delete(req.Ctx, pod)
		}
	}
	return nil
}
