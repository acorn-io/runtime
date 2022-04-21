package appdefinition

import (
	v1 "github.com/acorn-io/acorn/pkg/apis/acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/labels"
	"github.com/acorn-io/baaah/pkg/meta"
	"github.com/acorn-io/baaah/pkg/router"
	"github.com/acorn-io/baaah/pkg/typed"
	"github.com/google/go-containerregistry/pkg/name"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func addJobs(appInstance *v1.AppInstance, tag name.Reference, pullSecrets []corev1.LocalObjectReference, resp router.Response) {
	resp.Objects(toJobs(appInstance, pullSecrets, tag)...)
}

func toJobs(appInstance *v1.AppInstance, pullSecrets []corev1.LocalObjectReference, tag name.Reference) (result []meta.Object) {
	for _, entry := range typed.Sorted(appInstance.Status.AppSpec.Jobs) {
		result = append(result, toJob(appInstance, pullSecrets, tag, entry.Key, entry.Value))
	}
	return result
}

func setTerminationPath(containers []corev1.Container) (result []corev1.Container) {
	for _, c := range containers {
		c.TerminationMessagePath = "/run/secrets/output"
		result = append(result, c)
	}
	return
}

func toJob(appInstance *v1.AppInstance, pullSecrets []corev1.LocalObjectReference, tag name.Reference, name string, container v1.Container) *batchv1.Job {
	containers, initContainers := toContainers(appInstance, tag, name, container)
	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: appInstance.Status.Namespace,
			Labels: containerLabels(appInstance, name,
				labels.AcornManaged, "true",
				labels.AcornJobName, name,
				labels.AcornContainerName, ""),
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: containerLabels(appInstance, name,
						labels.AcornManaged, "true",
						labels.AcornJobName, name,
						labels.AcornContainerName, ""),
					Annotations: podAnnotations(appInstance, name, container),
				},
				Spec: corev1.PodSpec{
					ImagePullSecrets: pullSecrets,
					RestartPolicy:    corev1.RestartPolicyNever,
					Containers:       setTerminationPath(containers),
					InitContainers:   setTerminationPath(initContainers),
					Volumes:          toVolumes(appInstance, container),
				},
			},
		},
	}
}
