package jobs

import (
	"context"

	"github.com/acorn-io/mink/pkg/stores"
	"github.com/acorn-io/mink/pkg/types"
	"github.com/acorn-io/mink/pkg/validator"
	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	kclient "github.com/acorn-io/runtime/pkg/k8sclient"
	"github.com/acorn-io/z"
	batchv1 "k8s.io/api/batch/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewRestart(c client.WithWatch) rest.Storage {
	return stores.NewBuilder(c.Scheme(), &apiv1.JobRestart{}).
		WithCreate(&restartStrategy{client: c}).
		WithValidateName(validator.NoValidation).
		Build()
}

type restartStrategy struct {
	client client.WithWatch
}

func (s *restartStrategy) New() types.Object {
	return &apiv1.JobRestart{}
}

func (s *restartStrategy) Create(ctx context.Context, obj types.Object) (types.Object, error) {
	ri, _ := request.RequestInfoFrom(ctx)

	if ri.Namespace == "" || ri.Name == "" {
		return obj, nil
	}

	// Find the job to restart by looking it up with the name and namespace from the request info.
	job := &apiv1.Job{}
	err := s.client.Get(ctx, kclient.ObjectKey{Namespace: ri.Namespace, Name: ri.Name}, job)
	if err != nil {
		return nil, err
	}

	key := kclient.ObjectKey{Namespace: job.Status.JobNamespace, Name: job.Status.JobName}

	cronToRestart := &batchv1.CronJob{}
	if err = s.client.Get(ctx, key, cronToRestart); err == nil {
		return obj, s.restartCronJob(ctx, cronToRestart)
	}

	if !apierrors.IsNotFound(err) {
		return obj, err
	}

	jobToRestart := &batchv1.Job{}
	if err = s.client.Get(ctx, key, jobToRestart); err != nil {
		return obj, err
	}

	// Delete the Job and set the propagation policy to foreground so that the dependent resources (Pods)
	// created by the Job are also deleted.
	opts := &client.DeleteOptions{PropagationPolicy: z.Pointer(metav1.DeletePropagationForeground)}
	return obj, s.client.Delete(ctx, jobToRestart, opts)
}

func (s *restartStrategy) restartCronJob(ctx context.Context, cron *batchv1.CronJob) error {
	// Find all active jobs and delete them to make way for new ones.
	for _, jobRef := range cron.Status.Active {
		job := &batchv1.Job{ObjectMeta: metav1.ObjectMeta{Name: jobRef.Name, Namespace: jobRef.Namespace}}

		// Delete the Job and set the propagation policy to foreground so that the dependent resources (Pods)
		// created by the Job are also deleted.
		opts := &client.DeleteOptions{PropagationPolicy: z.Pointer(metav1.DeletePropagationForeground)}
		if err := s.client.Delete(ctx, job, opts); err != nil && !apierrors.IsNotFound(err) {
			return err
		}
	}

	// Create a new job from the cron's template to "restart" it. This is necessary because
	// the cronjob controller will not create a new job until the next scheduled run. By creating
	// it here we can ensure that the job is restarted immediately.
	template := cron.Spec.JobTemplate
	spec := template.Spec
	spec.TTLSecondsAfterFinished = new(int32) // Want to delete the job immediately after it finishes.
	newJob := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: cron.Name + "-",
			Namespace:    cron.Namespace,
			Labels:       template.Labels,
			Annotations:  template.Annotations,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(cron, batchv1.SchemeGroupVersion.WithKind("CronJob")),
			},
		},
		Spec: spec,
	}
	if err := s.client.Create(ctx, newJob); err != nil {
		return err
	}

	cron.Status.LastScheduleTime = &metav1.Time{Time: newJob.CreationTimestamp.Time}
	return s.client.Status().Update(ctx, cron)
}
