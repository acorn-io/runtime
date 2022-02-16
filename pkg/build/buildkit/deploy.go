package buildkit

import (
	"context"

	"github.com/ibuildthecloud/herd/pkg/system"
	"github.com/ibuildthecloud/herd/pkg/waiter"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetBuildkitPod(ctx context.Context, client client.WithWatch) (int, *corev1.Pod, error) {
	if err := applyObjects(ctx); err != nil {
		return 0, nil, err
	}

	port, err := getRegistryPort(ctx, client)
	if err != nil {
		return 0, nil, err
	}

	var (
		depWatcher = waiter.New[*appsv1.Deployment](client)
		podWatcher = waiter.New[*corev1.Pod](client)
	)

	deployment, err := depWatcher.ByName(ctx, system.Namespace, system.BuildKitName, func(dep *appsv1.Deployment) bool {
		for _, cond := range dep.Status.Conditions {
			if cond.Type == appsv1.DeploymentAvailable && cond.Status == corev1.ConditionTrue {
				return true
			}
		}
		return false
	})
	if err != nil {
		return 0, nil, err
	}

	sel, err := metav1.LabelSelectorAsSelector(deployment.Spec.Selector)
	if err != nil {
		return 0, nil, err
	}

	pod, err := podWatcher.BySelector(ctx, system.Namespace, sel, func(pod *corev1.Pod) bool {
		for _, cond := range pod.Status.Conditions {
			if cond.Type == corev1.PodReady && cond.Status == corev1.ConditionTrue {
				return true
			}
		}
		return false
	})
	return port, pod, err
}
