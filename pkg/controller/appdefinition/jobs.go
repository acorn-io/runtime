package appdefinition

import (
	"strings"

	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/labels"
	"github.com/acorn-io/baaah/pkg/router"
	"github.com/acorn-io/baaah/pkg/typed"
	"github.com/google/go-containerregistry/pkg/name"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func addJobs(req router.Request, appInstance *v1.AppInstance, tag name.Reference, pullSecrets *PullSecrets, resp router.Response) error {
	jobs, err := toJobs(req, appInstance, pullSecrets, tag)
	if err != nil {
		return err
	}
	resp.Objects(jobs...)
	return nil
}

func toJobs(req router.Request, appInstance *v1.AppInstance, pullSecrets *PullSecrets, tag name.Reference) (result []kclient.Object, _ error) {
	for _, entry := range typed.Sorted(appInstance.Status.AppSpec.Jobs) {
		job, err := toJob(req, appInstance, pullSecrets, tag, entry.Key, entry.Value)
		if err != nil {
			return nil, err
		}
		sa := toServiceAccount(job.GetName(), job.GetLabels(), job.GetAnnotations(), appInstance)
		if perms := v1.FindPermission(job.GetName(), appInstance.Spec.Permissions); perms.HasRules() {
			result = append(result, toPermissions(perms, job.GetLabels(), job.GetAnnotations(), appInstance)...)
		}
		result = append(result, job, sa)
	}
	return result, nil
}

func setTerminationPath(containers []corev1.Container) (result []corev1.Container) {
	for _, c := range containers {
		c.TerminationMessagePath = "/run/secrets/output"
		result = append(result, c)
	}
	return
}

func toJob(req router.Request, appInstance *v1.AppInstance, pullSecrets *PullSecrets, tag name.Reference, name string, container v1.Container) (kclient.Object, error) {
	containers, initContainers, err := toContainers(req, appInstance, tag, name, container)
	if err != nil {
		return nil, err
	}

	secretAnnotations, err := getSecretAnnotations(req, appInstance, container)
	if err != nil {
		return nil, err
	}

	baseAnnotations := labels.Merge(secretAnnotations, labels.GatherScoped(name, v1.LabelTypeJob,
		appInstance.Status.AppSpec.Annotations, container.Annotations, appInstance.Spec.Annotations))

	volumes, err := toVolumes(appInstance, container)
	if err != nil {
		return nil, err
	}

	jobSpec := batchv1.JobSpec{
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: jobLabels(appInstance, container, name,
					labels.AcornManaged, "true",
					labels.AcornJobName, name,
					labels.AcornContainerName, ""),
				Annotations: labels.Merge(podAnnotations(appInstance, name, container), baseAnnotations),
			},
			Spec: corev1.PodSpec{
				TerminationGracePeriodSeconds: &[]int64{5}[0],
				ImagePullSecrets:              pullSecrets.ForContainer(name, append(containers, initContainers...)),
				EnableServiceLinks:            new(bool),
				RestartPolicy:                 corev1.RestartPolicyNever,
				Containers:                    setTerminationPath(containers),
				InitContainers:                setTerminationPath(initContainers),
				Volumes:                       volumes,
				ServiceAccountName:            name,
			},
		},
	}

	if container.Schedule == "" {
		return &batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{
				Name:        name,
				Namespace:   appInstance.Status.Namespace,
				Labels:      jobSpec.Template.Labels,
				Annotations: labels.Merge(getDependencyAnnotations(appInstance, container.Dependencies), baseAnnotations),
			},
			Spec: jobSpec,
		}, nil
	}
	return &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   appInstance.Status.Namespace,
			Labels:      jobSpec.Template.Labels,
			Annotations: labels.Merge(getDependencyAnnotations(appInstance, container.Dependencies), baseAnnotations),
		},
		Spec: batchv1.CronJobSpec{
			Schedule: toCronJobSchedule(container.Schedule),
			JobTemplate: batchv1.JobTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: jobSpec.Template.Labels,
				},
				Spec: jobSpec,
			},
		},
	}, nil
}

func toCronJobSchedule(schedule string) string {
	switch strings.TrimSpace(schedule) {
	case "year":
	case "annually":
	case "monthly":
	case "weekly":
	case "daily":
	case "midnight":
	case "hourly":
	default:
		return schedule
	}
	return "@" + strings.TrimSpace(schedule)
}
