package client

import (
	"context"
	"encoding/json"
	"sort"
	"strings"

	"github.com/ibuildthecloud/baaah/pkg/typed"
	"github.com/ibuildthecloud/herd/pkg/apis/herd-project.io/v1"
	"github.com/ibuildthecloud/herd/pkg/labels"
	"github.com/rancher/wrangler/pkg/data/convert"
	"github.com/rancher/wrangler/pkg/merr"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	klabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func containerSpecToContainerReplicaIgnore(pod *corev1.Pod, containerSpec v1.Container, sidecarName string) *ContainerReplica {
	result, err := containerSpecToContainerReplica(pod, containerSpec, sidecarName)
	if err != nil {
		logrus.Errorf("failed to convert container spec for %s/%s (, sidecar: [%s]): %v",
			pod.Namespace, pod.Name, sidecarName, err)
		return nil
	}
	return result
}

func containerSpecToContainerReplica(pod *corev1.Pod, containerSpec v1.Container, sidecarName string) (*ContainerReplica, error) {
	var (
		name                = pod.Name
		containerName       = pod.Labels[labels.HerdContainerName]
		containerStatusName = containerName
	)

	if sidecarName != "" {
		containerSpec = containerSpec.Sidecars[sidecarName]
		name += "/" + sidecarName
		containerStatusName = sidecarName
	}

	result := &ContainerReplica{}
	if err := convert.ToObj(containerSpec, result); err != nil {
		return nil, err
	}

	result.Name = name
	result.AppName = pod.Labels[labels.HerdAppName]
	result.JobName = pod.Labels[labels.HerdJobName]
	result.ContainerName = containerName
	result.SidecarName = sidecarName
	result.Created = pod.CreationTimestamp
	result.Revision = pod.ResourceVersion
	result.Labels = pod.Labels
	result.Annotations = pod.Annotations

	delete(result.Annotations, labels.HerdContainerSpec)

	containerStatus := pod.Status.ContainerStatuses
	if result.Init {
		containerStatus = pod.Status.InitContainerStatuses
	}

	for _, status := range containerStatus {
		if status.Name != containerStatusName {
			continue
		}

		result.Status = ContainerReplicaStatus{
			PodName:              pod.Name,
			PodNamespace:         pod.Namespace,
			Phase:                pod.Status.Phase,
			PodMessage:           pod.Status.Message,
			PodReason:            pod.Status.Reason,
			State:                status.State,
			LastTerminationState: status.LastTerminationState,
			Ready:                status.Ready,
			RestartCount:         status.RestartCount,
			Image:                status.Image,
			ImageID:              status.ImageID,
			Started:              status.Started,
		}

		if status.State.Running != nil {
			if result.Status.Ready {
				result.Status.Columns.State = "running"
			} else {
				result.Status.Columns.State = "running (not ready)"
			}
		} else if status.State.Waiting != nil {
			result.Status.Columns.State = status.State.Waiting.Reason
			if status.State.Waiting.Message != "" {
				result.Status.Columns.State += ": " + status.State.Waiting.Message
			}
		} else if status.State.Terminated != nil {
			result.Status.Columns.State = "stopped: " + status.State.Terminated.Message
		}

		if result.JobName != "" {
			result.Status.Columns.App = result.JobName
		} else {
			result.Status.Columns.App = result.AppName
		}

		break
	}

	return result, nil
}

func podToContainers(pod *corev1.Pod) (result []ContainerReplica) {
	containerSpecData := []byte(pod.Annotations[labels.HerdContainerSpec])
	if len(containerSpecData) == 0 {
		return nil
	}

	containerSpec := v1.Container{}
	err := json.Unmarshal(containerSpecData, &containerSpec)
	if err != nil {
		logrus.Errorf("failed to unmarshal container spec for %s/%s: %s",
			pod.Namespace, pod.Name, containerSpecData)
		return nil
	}

	for _, sideCarName := range append([]string{""}, typed.SortedKeys(containerSpec.Sidecars)...) {
		replica := containerSpecToContainerReplicaIgnore(pod, containerSpec, sideCarName)
		if replica == nil {
			return nil
		}
		result = append(result, *replica)
	}

	return result
}

func (c *client) ContainerReplicaGet(ctx context.Context, name string) (*ContainerReplica, error) {
	podName, _, _ := strings.Cut(name, "/")

	apps, err := c.AppList(ctx)
	if err != nil {
		return nil, err
	}

	var (
		pod  = &corev1.Pod{}
		errs []error
	)

	for _, app := range apps {
		err := c.Client.Get(ctx, kclient.ObjectKey{
			Name:      podName,
			Namespace: app.Status.Namespace,
		}, pod)
		if apierrors.IsNotFound(err) {
			continue
		}
		if err != nil {
			errs = append(errs, err)
			continue
		}

		for _, container := range podToContainers(pod) {
			if container.Name == name {
				return &container, nil
			}
		}
	}

	err = merr.NewErrors(errs...)
	if err != nil {
		return nil, err
	}

	return nil, apierrors.NewNotFound(schema.GroupResource{
		Group:    "",
		Resource: "pods",
	}, podName)
}

func (c *client) ContainerReplicaList(ctx context.Context, opts *ContainerReplicaListOptions) (result []ContainerReplica, _ error) {
	var (
		apps []App
		err  error
		pods = make(chan corev1.Pod)
		eg   = errgroup.Group{}
	)

	opts = opts.complete()

	if opts.App == "" {
		apps, err = c.AppList(ctx)
		if err != nil {
			return nil, err
		}
	} else {
		app, err := c.AppGet(ctx, opts.App)
		if err != nil {
			return nil, err
		}
		apps = []App{*app}
		var ()

	}

	for _, app := range apps {
		if app.Status.Namespace != "" {
			c.containersForNS(ctx, &eg, app.Status.Namespace, pods)
		}
	}

	waitAndClose(&eg, pods, &err)
	if err != nil {
		return nil, err
	}

	for pod := range pods {
		result = append(result, podToContainers(&pod)...)
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].Created.Time == result[j].Created.Time {
			return result[i].Name < result[j].Name
		}
		return result[i].Created.After(result[j].Created.Time)
	})

	return result, nil
}

func (c *client) containersForNS(ctx context.Context, eg *errgroup.Group, namespace string, pods chan<- corev1.Pod) {
	eg.Go(func() error {
		podList := &corev1.PodList{}
		err := c.Client.List(ctx, podList, &kclient.ListOptions{
			Namespace: namespace,
			LabelSelector: klabels.SelectorFromSet(map[string]string{
				labels.HerdManaged: "true",
			}),
		})
		if err != nil {
			return err
		}

		for _, pod := range podList.Items {
			pods <- pod
		}

		return nil
	})
}
