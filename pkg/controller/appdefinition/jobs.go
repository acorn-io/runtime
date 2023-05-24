package appdefinition

import (
	"strings"

	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/labels"
	"github.com/acorn-io/acorn/pkg/publicname"
	"github.com/acorn-io/acorn/pkg/secrets"
	"github.com/acorn-io/baaah/pkg/apply"
	"github.com/acorn-io/baaah/pkg/router"
	"github.com/acorn-io/baaah/pkg/typed"
	"github.com/google/go-containerregistry/pkg/name"
	"golang.org/x/exp/slices"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func addJobs(req router.Request, appInstance *v1.AppInstance, tag name.Reference, pullSecrets *PullSecrets, interpolator *secrets.Interpolator, resp router.Response) error {
	jobs, err := toJobs(req, appInstance, pullSecrets, tag, interpolator)
	if err != nil {
		return err
	}
	resp.Objects(jobs...)
	return nil
}

func stripPruneAndUpdate(annotations map[string]string) map[string]string {
	result := map[string]string{}
	for k, v := range annotations {
		if k == apply.AnnotationPrune || k == apply.AnnotationUpdate {
			continue
		}
		result[k] = v
	}
	return result
}

func toJobs(req router.Request, appInstance *v1.AppInstance, pullSecrets *PullSecrets, tag name.Reference, interpolator *secrets.Interpolator) (result []kclient.Object, _ error) {
	for _, entry := range typed.Sorted(appInstance.Status.AppSpec.Jobs) {
		job, err := toJob(req, appInstance, pullSecrets, tag, entry.Key, entry.Value, interpolator)
		if err != nil || job == nil {
			return nil, err
		}
		sa, err := toServiceAccount(req, job.GetName(), job.GetLabels(), stripPruneAndUpdate(job.GetAnnotations()), appInstance)
		if err != nil {
			return nil, err
		}
		if perms := v1.FindPermission(job.GetName(), appInstance.Spec.Permissions); perms.HasRules() {
			result = append(result, toPermissions(perms, job.GetLabels(), stripPruneAndUpdate(job.GetAnnotations()), appInstance)...)
		}
		result = append(result, sa, job)
	}
	return result, nil
}

func setJobEventName(containers []corev1.Container, eventName string) (result []corev1.Container) {
	for _, c := range containers {
		c.Env = append(c.Env, corev1.EnvVar{
			Name:  "ACORN_EVENT",
			Value: eventName,
		})
		result = append(result, c)
	}
	return
}

func setTerminationPath(containers []corev1.Container) (result []corev1.Container) {
	for _, c := range containers {
		c.TerminationMessagePath = "/run/secrets/output"
		result = append(result, c)
	}
	return
}

func matchesJobEvent(eventName string, container v1.Container) bool {
	if len(container.Events) == 0 {
		return slices.Contains([]string{"create", "update"}, eventName)
	}
	return slices.Contains(container.Events, eventName)
}

func getJobEvent(jobName string, appInstance *v1.AppInstance) string {
	if !appInstance.DeletionTimestamp.IsZero() {
		return "delete"
	}
	if appInstance.Generation > 1 && appInstance.Status.JobsStatus[jobName].CreateEventSucceeded {
		return "update"
	}
	return "create"
}

func toJob(req router.Request, appInstance *v1.AppInstance, pullSecrets *PullSecrets, tag name.Reference, name string, container v1.Container, interpolator *secrets.Interpolator) (kclient.Object, error) {
	interpolator = interpolator.ForService(name)
	jobEventName := getJobEvent(name, appInstance)

	jobStatus := appInstance.Status.JobsStatus[name]
	if appInstance.Status.JobsStatus == nil {
		appInstance.Status.JobsStatus = make(map[string]v1.JobStatus)
	}
	jobStatus.Skipped = !matchesJobEvent(jobEventName, container)
	appInstance.Status.JobsStatus[name] = jobStatus

	if jobStatus.Skipped {
		return nil, nil
	}

	containers, initContainers := toContainers(appInstance, tag, name, container, interpolator)

	secretAnnotations, err := getSecretAnnotations(req, appInstance, container, interpolator)
	if err != nil {
		return nil, err
	}

	volumes, err := toVolumes(appInstance, container, interpolator)
	if err != nil {
		return nil, err
	}

	baseAnnotations := labels.Merge(secretAnnotations, labels.GatherScoped(name, v1.LabelTypeJob,
		appInstance.Status.AppSpec.Annotations, container.Annotations, appInstance.Spec.Annotations))

	jobSpec := batchv1.JobSpec{
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: jobLabels(appInstance, container, name,
					labels.AcornManaged, "true",
					labels.AcornAppPublicName, publicname.Get(appInstance),
					labels.AcornJobName, name,
					labels.AcornContainerName, ""),
				Annotations: labels.Merge(podAnnotations(appInstance, name, container), baseAnnotations),
			},
			Spec: corev1.PodSpec{
				Affinity:                      appInstance.Status.Scheduling[name].Affinity,
				Tolerations:                   appInstance.Status.Scheduling[name].Tolerations,
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

	interpolator.AddMissingAnnotations(baseAnnotations)

	if container.Schedule == "" {
		jobSpec.BackoffLimit = &[]int32{1000}[0]
		job := &batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{
				Name:        name,
				Namespace:   appInstance.Status.Namespace,
				Labels:      jobSpec.Template.Labels,
				Annotations: labels.Merge(getDependencyAnnotations(appInstance, container.Dependencies), baseAnnotations),
			},
			Spec: jobSpec,
		}
		job.Spec.Template.Spec.Containers = setJobEventName(setTerminationPath(containers), jobEventName)
		job.Spec.Template.Spec.InitContainers = setJobEventName(setTerminationPath(initContainers), jobEventName)
		job.Annotations[apply.AnnotationPrune] = "false"
		job.Annotations[apply.AnnotationUpdate] = "true"
		return job, nil
	}
	return &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   appInstance.Status.Namespace,
			Labels:      jobSpec.Template.Labels,
			Annotations: labels.Merge(getDependencyAnnotations(appInstance, container.Dependencies), baseAnnotations),
		},
		Spec: batchv1.CronJobSpec{
			FailedJobsHistoryLimit:     &[]int32{3}[0],
			SuccessfulJobsHistoryLimit: &[]int32{1}[0],
			ConcurrencyPolicy:          batchv1.ReplaceConcurrent,
			Schedule:                   toCronJobSchedule(container.Schedule),
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
