package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/appdefinition"
	"github.com/acorn-io/acorn/pkg/encryption/nacl"
	"github.com/acorn-io/baaah/pkg/router"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	klabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/utils/strings/slices"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	ErrJobNotDone  = errors.New("job not complete")
	ErrJobNoOutput = errors.New("job has no output")
)

// GetOutputFor obj must be acorn internal v1.Secret, v1.Service, or string
func GetOutputFor(ctx context.Context, c kclient.Client, appInstance *v1.AppInstance, name, serviceName string, obj interface{}) (job *batchv1.Job, err error) {
	defer func() {
		if err != nil && !errors.Is(err, ErrJobNoOutput) && !errors.Is(err, ErrJobNotDone) {
			err = errors.Join(err, ErrJobNotDone)
		}
	}()

	job, data, err := GetOutput(ctx, c, appInstance, name)
	if err != nil {
		return nil, err
	}

	switch v := obj.(type) {
	case *string:
		*v = string(data)
	case *v1.Secret:
		if err := json.Unmarshal(data, v); err == nil && (v.Type != "" || len(v.Data) > 0) {
			return job, nil
		}
		appSpec, err := asAppSpec(data)
		if err != nil {
			return nil, err
		}
		secret := appSpec.Secrets[serviceName]
		secret.DeepCopyInto(v)
	case *v1.Service:
		appSpec, err := asAppSpec(data)
		if err != nil {
			return nil, err
		}
		svc := appSpec.Services[serviceName]
		svc.DeepCopyInto(v)
	default:
		return nil, fmt.Errorf("invalid job output type %T", v)
	}

	return job, nil
}

func asAppSpec(data []byte) (*v1.AppSpec, error) {
	appDef, err := appdefinition.NewAppDefinition(data)
	if err != nil {
		return nil, err
	}
	return appDef.AppSpec()
}

func GetOutput(ctx context.Context, c kclient.Client, appInstance *v1.AppInstance, name string) (job *batchv1.Job, data []byte, err error) {
	defer func() {
		if err == nil {
			if nacl.IsAcornEncryptedData(data) {
				data, err = nacl.DecryptNamespacedData(ctx, c, data, appInstance.Namespace)
			}
		}
	}()

	namespace := appInstance.Status.Namespace

	if val, ok := appInstance.Status.AppSpec.Jobs[name]; ok {
		if val.Schedule != "" {
			name, err = getCronJobLatestJob(ctx, c, namespace, name)
			if err != nil {
				return nil, nil, err
			}
		}
	}

	job = &batchv1.Job{}
	err = c.Get(ctx, router.Key(namespace, name), job)
	if err != nil {
		return nil, nil, err
	}

	if job.Status.Succeeded != 1 {
		return nil, nil, ErrJobNotDone
	}

	sel, err := metav1.LabelSelectorAsSelector(job.Spec.Selector)
	if err != nil {
		return nil, nil, err
	}

	pods := &corev1.PodList{}
	err = c.List(ctx, pods, &kclient.ListOptions{
		Namespace:     namespace,
		LabelSelector: sel,
	})
	if err != nil {
		return nil, nil, err
	}

	if len(pods.Items) == 0 {
		return nil, nil, apierrors.NewNotFound(schema.GroupResource{
			Resource: "pods",
		}, "")
	}

	for _, pod := range pods.Items {
		for _, status := range pod.Status.ContainerStatuses {
			if status.State.Terminated == nil || status.State.Terminated.ExitCode != 0 {
				continue
			}
			if len(status.State.Terminated.Message) > 0 {
				return job, []byte(status.State.Terminated.Message), nil
			}
		}
	}

	return nil, nil, ErrJobNoOutput
}

func getCronJobLatestJob(ctx context.Context, c kclient.Client, namespace, name string) (jobName string, err error) {
	cronJob := &batchv1.CronJob{}
	err = c.Get(ctx, router.Key(namespace, name), cronJob)
	if err != nil {
		return "", err
	}

	l := klabels.SelectorFromSet(cronJob.Spec.JobTemplate.Labels)
	if err != nil {
		return "", err
	}

	var jobsFromCron batchv1.JobList
	err = c.List(ctx, &jobsFromCron, &kclient.ListOptions{
		Namespace:     namespace,
		LabelSelector: l,
	})
	if err != nil {
		return "", err
	}

	for _, job := range jobsFromCron.Items {
		if job.Status.CompletionTime != nil && cronJob.Status.LastSuccessfulTime != nil &&
			job.Status.CompletionTime.Time == cronJob.Status.LastSuccessfulTime.Time {
			return job.Name, nil
		}
	}

	return "", ErrJobNotDone
}

// ShouldRunForEvent returns true if the job is configured to run for the given event.
func ShouldRunForEvent(eventName string, container v1.Container) bool {
	if len(container.Events) == 0 {
		return slices.Contains([]string{"create", "update"}, eventName)
	}
	return slices.Contains(container.Events, eventName)
}

// ShouldRun determines if the job should run based on the app's status.
func ShouldRun(jobName string, appInstance *v1.AppInstance) bool {
	for name, job := range appInstance.Status.AppSpec.Jobs {
		if name == jobName {
			return ShouldRunForEvent(GetEvent(jobName, appInstance), job)
		}
	}
	return false
}

// GetEvent determines the event type for the job based on the app's status.
func GetEvent(jobName string, appInstance *v1.AppInstance) string {
	if !appInstance.DeletionTimestamp.IsZero() {
		return "delete"
	}
	if appInstance.Spec.Stop != nil && *appInstance.Spec.Stop {
		return "stop"
	}
	if appInstance.Generation <= 1 || slices.Contains(appInstance.Status.AppSpec.Jobs[jobName].Events, "create") && !appInstance.Status.AppStatus.Jobs[jobName].CreateEventSucceeded {
		// Create event jobs run at least once. So, if it hasn't succeeded, run it.
		return "create"
	}
	return "update"
}
