package appdefinition

import (
	"github.com/ibuildthecloud/baaah/pkg/meta"
	"github.com/ibuildthecloud/baaah/pkg/router"
	"github.com/ibuildthecloud/baaah/pkg/typed"
	v1 "github.com/ibuildthecloud/herd/pkg/apis/herd-project.io/v1"
	"github.com/ibuildthecloud/herd/pkg/labels"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func addJobs(appInstance *v1.AppInstance, resp router.Response) {
	resp.Objects(toJobs(appInstance)...)
}

func toJobs(appInstance *v1.AppInstance) (result []meta.Object) {
	for _, entry := range typed.Sorted(appInstance.Status.AppSpec.Jobs) {
		result = append(result, toJob(appInstance, entry.Key, entry.Value))
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

func toJob(appInstance *v1.AppInstance, name string, container v1.Container) *batchv1.Job {
	containers, initContainers := toContainers(appInstance.Name, name, container)
	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: appInstance.Status.Namespace,
			Labels: containerLabels(appInstance, name,
				labels.HerdManaged, "true",
				labels.HerdJobName, name,
				labels.HerdContainerName, ""),
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: containerLabels(appInstance, name,
						labels.HerdManaged, "true",
						labels.HerdJobName, name,
						labels.HerdContainerName, ""),
				},
				Spec: corev1.PodSpec{
					RestartPolicy:  corev1.RestartPolicyNever,
					Containers:     setTerminationPath(containers),
					InitContainers: setTerminationPath(initContainers),
					Volumes:        toVolumes(appInstance, container),
				},
			},
		},
	}
}
