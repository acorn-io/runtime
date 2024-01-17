package webhook

import (
	"strings"

	"github.com/acorn-io/baaah/pkg/webhook"
	"github.com/acorn-io/runtime/pkg/system"
	"github.com/acorn-io/z"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type Handler struct {
	c kclient.Client
}

func PatchPodSpec(podSpec *corev1.PodSpec) bool {
	var (
		modified bool
		paths    = []string{
			"/etc/passwd",
			"/etc/group",
			"/etc/docker",
			"/etc/nginx",
			"/lib",
			"/bin",
			"/sbin",
			"/usr",
			"docker-entrypoint.d",
			"docker-entrypoint.sh",
			"/etc/ssl/certs/ca-certificates.crt",
			"/var/run/docker.sock",
			"/var/lib/rancher/k3s/storage",
			"/var/lib/buildkit",
		}
		mounts   []corev1.VolumeMount
		existing = map[string]bool{}
	)

	for _, container := range podSpec.Containers {
		for _, mount := range container.VolumeMounts {
			existing[mount.MountPath] = true
		}
	}

	for _, path := range paths {
		if existing[path] {
			continue
		}
		mounts = append(mounts, corev1.VolumeMount{
			Name: "acorn-local-host",
			ReadOnly: path != "/sbin" &&
				path != "/var/lib/rancher/k3s/storage" &&
				path != "/var/lib/buildkit",
			MountPath: path,
			SubPath:   strings.TrimPrefix(path, "/"),
		})
	}

	for i, container := range podSpec.Containers {
		if container.Name == "acorn-controller" {
			modified = true
			container.SecurityContext = &corev1.SecurityContext{
				RunAsUser: z.Pointer(int64(0)),
			}
		}
		if container.Image == "acorn-local" {
			modified = true
			container.Image = system.LocalImageBind
			container.ImagePullPolicy = corev1.PullIfNotPresent
			container.VolumeMounts = append(container.VolumeMounts, mounts...)
			podSpec.Containers[i] = container
		}
	}

	for i, container := range podSpec.InitContainers {
		if container.Image == "acorn-local" {
			modified = true
			container.Image = system.LocalImageBind
			container.ImagePullPolicy = corev1.PullIfNotPresent
			container.VolumeMounts = append(container.VolumeMounts, mounts...)
			podSpec.InitContainers[i] = container
		}
	}

	if modified {
		podSpec.Volumes = append(podSpec.Volumes, corev1.Volume{
			Name: "acorn-local-host",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/",
				},
			},
		})
	}

	if podSpec.NodeName == "" {
		modified = true
		podSpec.NodeName = system.LocalNode
	}

	return modified
}

func (h *Handler) Admit(resp *webhook.Response, req *webhook.Request) error {
	resp.Allowed = true

	pod := &corev1.Pod{}
	if err := req.DecodeObject(pod); err != nil {
		return err
	}

	if !PatchPodSpec(&pod.Spec) {
		return nil
	}

	if err := resp.CreatePatch(req, pod); err != nil {
		return err
	}

	logrus.Debugf("Patching %s/%s(%s): %s on %s", pod.Namespace, pod.Name, pod.GenerateName, resp.Patch, req.Object.Raw)
	return nil
}
