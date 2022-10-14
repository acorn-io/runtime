package containers

import (
	"context"
	"encoding/json"
	"strings"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/labels"
	"github.com/acorn-io/acorn/pkg/namespace"
	"github.com/acorn-io/baaah/pkg/router"
	"github.com/acorn-io/baaah/pkg/typed"
	mtypes "github.com/acorn-io/mink/pkg/types"
	"github.com/rancher/wrangler/pkg/data/convert"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	klabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apiserver/pkg/storage"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type Translator struct {
	client kclient.Client
}

func (t *Translator) FromPublicName(ctx context.Context, namespace, name string) (string, string, error) {
	for {
		prefix, suffix, ok := strings.Cut(name, ".")
		if !ok {
			break
		}
		app := &v1.AppInstance{}
		err := t.client.Get(ctx, router.Key(namespace, prefix), app)
		if err != nil {
			return namespace, name, err
		}

		name = suffix
		namespace = app.Status.Namespace
	}

	return namespace, name, nil
}

func (t *Translator) ListOpts(namespace string, opts storage.ListOptions) (string, storage.ListOptions) {
	sel := opts.Predicate.Label
	if sel == nil {
		sel = klabels.Everything()
	}
	req, _ := klabels.NewRequirement(labels.AcornManaged, selection.Equals, []string{"true"})
	sel = sel.Add(*req)

	if namespace != "" {
		req, _ := klabels.NewRequirement(labels.AcornAppNamespace, selection.Equals, []string{namespace})
		sel = sel.Add(*req)
	}
	opts.Predicate.Label = sel
	return "", opts
}

func (t *Translator) ToPublic(objs ...runtime.Object) (result []mtypes.Object) {
	for _, obj := range objs {
		for _, con := range podToContainers(obj.(*corev1.Pod)) {
			con := con
			result = append(result, &con)
		}
	}
	if len(result) == 0 && len(objs) > 0 {
		result = append(result, &apiv1.ContainerReplica{})
	}
	return
}

func (t *Translator) FromPublic(_ context.Context, obj runtime.Object) (mtypes.Object, error) {
	con := obj.(*apiv1.ContainerReplica)
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      con.Status.PodName,
			Namespace: con.Status.PodNamespace,
		},
	}, nil
}

func (t *Translator) NewPublicList() mtypes.ObjectList {
	return &apiv1.ContainerReplicaList{}
}

func (t *Translator) NewPublic() mtypes.Object {
	return &apiv1.ContainerReplica{}
}

func podToContainers(pod *corev1.Pod) (result []apiv1.ContainerReplica) {
	containerSpecData := []byte(pod.Annotations[labels.AcornContainerSpec])
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

	imageMapping := map[string]string{}
	imageMappingData := pod.Annotations[labels.AcornImageMapping]
	if len(imageMappingData) > 0 {
		err := json.Unmarshal([]byte(imageMappingData), &imageMapping)
		if err != nil {
			logrus.Errorf("failed to unmarshal image mapping for %s/%s: %s",
				pod.Namespace, pod.Name, imageMappingData)
		}
	}

	for _, sideCarName := range append([]string{""}, typed.SortedKeys(containerSpec.Sidecars)...) {
		replica := containerSpecToContainerReplicaIgnore(pod, imageMapping, containerSpec, sideCarName)
		if replica == nil {
			return nil
		}
		result = append(result, *replica)
	}

	return result
}

func containerSpecToContainerReplicaIgnore(pod *corev1.Pod, imageMapping map[string]string, containerSpec v1.Container, sidecarName string) *apiv1.ContainerReplica {
	result, err := containerSpecToContainerReplica(pod, imageMapping, containerSpec, sidecarName)
	if err != nil {
		logrus.Errorf("failed to convert container spec for %s/%s (, sidecar: [%s]): %v",
			pod.Namespace, pod.Name, sidecarName, err)
		return nil
	}
	return result
}

func containerSpecToContainerReplica(pod *corev1.Pod, imageMapping map[string]string, containerSpec v1.Container, sidecarName string) (*apiv1.ContainerReplica, error) {
	var (
		uid                 = pod.UID
		containerName       = pod.Labels[labels.AcornContainerName]
		jobName             = pod.Labels[labels.AcornJobName]
		containerStatusName = containerName
		namespace, name     = namespace.NormalizedName(pod.ObjectMeta)
	)

	if containerStatusName == "" {
		containerStatusName = jobName
	}

	if sidecarName != "" {
		containerSpec = containerSpec.Sidecars[sidecarName]
		name += "." + sidecarName
		containerStatusName = sidecarName
		uid = types.UID(string(uid) + "-" + sidecarName)
	} else {
		uid = types.UID(string(uid) + "-c")
	}

	result := &apiv1.ContainerReplica{
		ObjectMeta: pod.ObjectMeta,
	}
	if err := convert.ToObj(containerSpec, &result.Spec); err != nil {
		return nil, err
	}

	friendlyImage, ok := imageMapping[result.Spec.Image]
	if ok {
		result.Spec.Image = friendlyImage
	}

	result.Name = name
	result.Namespace = namespace
	result.OwnerReferences = nil
	result.UID = uid
	result.Spec.AppName = pod.Labels[labels.AcornAppName]
	result.Spec.JobName = jobName
	result.Spec.ContainerName = containerName
	result.Spec.SidecarName = sidecarName
	result.Labels = pod.Labels
	result.Annotations = pod.Annotations

	delete(result.Annotations, labels.AcornContainerSpec)

	containerStatus := pod.Status.ContainerStatuses
	if result.Spec.Init {
		containerStatus = pod.Status.InitContainerStatuses
	}

	for _, status := range containerStatus {
		if status.Name != containerStatusName {
			continue
		}

		result.Status = apiv1.ContainerReplicaStatus{
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
			if status.State.Terminated.ExitCode == 0 && jobName != "" {
				// Don't include message here because it will be the termination message which
				// is a secret.  We need a secure implementation that doesn't put the secret in the
				// termination message.
				result.Status.Columns.State = "stopped"
			} else {
				result.Status.Columns.State = "stopped: " + status.State.Terminated.Message
			}
		}

		if result.Spec.JobName != "" {
			result.Status.Columns.App = result.Spec.JobName
		} else {
			result.Status.Columns.App = result.Spec.AppName
		}

		break
	}

	result.Status.PodName = pod.Name
	result.Status.PodNamespace = pod.Namespace
	result.Status.Phase = pod.Status.Phase
	result.Status.PodMessage = pod.Status.Message
	result.Status.PodReason = pod.Status.Reason

	return result, nil
}
