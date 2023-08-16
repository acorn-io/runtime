package jobs

import (
	"context"

	"github.com/acorn-io/mink/pkg/strategy"
	"github.com/acorn-io/mink/pkg/types"
	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	internalapiv1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/publicname"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apiserver/pkg/storage"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Strategy struct {
	client client.WithWatch
}

func NewStrategy(c client.WithWatch) *Strategy {
	return &Strategy{client: c}
}

func (s *Strategy) NewList() types.ObjectList {
	return &apiv1.JobList{}
}

func (s *Strategy) New() types.Object {
	return &apiv1.Job{}
}

func (s *Strategy) List(ctx context.Context, namespace string, options storage.ListOptions) (types.ObjectList, error) {
	apps := &apiv1.AppList{}
	if err := s.client.List(ctx, apps, strategy.ToListOpts(namespace, options)); err != nil {
		return nil, err
	}

	acornJobs := apiv1.JobList{}
	for _, app := range apps.Items {
		for _, jobStatus := range app.Status.AppStatus.Jobs {
			acornJobs.Items = append(acornJobs.Items, jobStatusToJob(app.Namespace, app, jobStatus))
		}
	}

	return &acornJobs, nil
}

func (s *Strategy) Get(ctx context.Context, namespace, name string) (types.Object, error) {
	list, err := s.List(ctx, namespace, storage.ListOptions{})
	if err != nil {
		return nil, err
	}

	for _, job := range list.(*apiv1.JobList).Items {
		if job.Name == name {
			return &job, nil
		}
	}

	return nil, apierrors.NewNotFound(schema.GroupResource{
		Group:    apiv1.SchemeGroupVersion.Group,
		Resource: "job",
	}, name)
}

func jobStatusToJob(namespace string, app apiv1.App, jobStatus internalapiv1.JobStatus) apiv1.Job {
	creationTime := jobStatus.CreationTime
	if creationTime == nil {
		creationTime = &metav1.Time{}
	}

	return apiv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:              publicname.ForChild(&app, jobStatus.JobName),
			Namespace:         namespace,
			CreationTimestamp: *creationTime,
		},
		Spec: apiv1.JobSpec{
			JobName:  jobStatus.JobName,
			AppName:  app.Name,
			Schedule: jobStatus.Schedule,
		},
		Status: jobStatus,
	}
}
