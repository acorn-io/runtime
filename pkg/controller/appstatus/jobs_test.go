package appstatus

import (
	"context"
	"testing"
	"time"

	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/labels"
	"github.com/acorn-io/runtime/pkg/scheme"
	"github.com/acorn-io/z"
	cronv3 "github.com/robfig/cron/v3"
	"github.com/stretchr/testify/assert"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestReadJobsWithNoJobs(t *testing.T) {
	app := &v1.AppInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "test-app",
			Namespace:  "test-project",
			Generation: 1,
		},
		Status: v1.AppInstanceStatus{
			EmbeddedAppStatus: v1.EmbeddedAppStatus{
				Namespace: "test-namespace",
			},
		},
	}

	client := fake.NewClientBuilder().WithScheme(scheme.Scheme).Build()
	assert.NoError(t, (&appStatusRenderer{
		ctx: context.Background(),
		c:   client,
		app: app,
	}).readJobs())

	assert.Empty(t, app.Status.AppStatus.Jobs)
}

func TestReadJobsSkippedJob(t *testing.T) {
	app := &v1.AppInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "test-app",
			Namespace:  "test-project",
			Generation: 1,
		},
		Status: v1.AppInstanceStatus{
			EmbeddedAppStatus: v1.EmbeddedAppStatus{
				Namespace: "test-namespace",
				AppSpec: v1.AppSpec{
					Jobs: map[string]v1.Container{
						"test-job": {},
					},
				},
				AppStatus: v1.AppStatus{
					Jobs: map[string]v1.JobStatus{
						"test-job": {
							Skipped: true,
						},
					},
				},
			},
		},
	}

	expectedAppStatus := v1.AppStatus{
		Jobs: map[string]v1.JobStatus{
			"test-job": {
				JobName:      "test-job",
				JobNamespace: "test-namespace",
				CreationTime: z.Pointer(metav1.NewTime(time.Time{})),
				Skipped:      true,
				CommonStatus: v1.CommonStatus{
					Ready:      true,
					UpToDate:   true,
					Defined:    true,
					State:      "completed",
					ConfigHash: "f72bbc9aab1c45a4075711c05dda5814398882c2ea25282ae558c9903c340cc1",
				},
			},
		},
	}

	client := fake.NewClientBuilder().WithScheme(scheme.Scheme).Build()
	assert.NoError(t, (&appStatusRenderer{
		ctx: context.Background(),
		c:   client,
		app: app,
	}).readJobs())

	assert.Equal(t, expectedAppStatus, app.Status.AppStatus)
}

func TestReadJobNotCreated(t *testing.T) {
	app := &v1.AppInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "test-app",
			Namespace:  "test-project",
			Generation: 1,
		},
		Status: v1.AppInstanceStatus{
			EmbeddedAppStatus: v1.EmbeddedAppStatus{
				Namespace: "test-namespace",
				AppSpec: v1.AppSpec{
					Jobs: map[string]v1.Container{
						"test-job": {},
					},
				},
			},
		},
	}

	expectedAppStatus := v1.AppStatus{
		Jobs: map[string]v1.JobStatus{
			"test-job": {
				JobName:      "test-job",
				JobNamespace: "test-namespace",
				CommonStatus: v1.CommonStatus{
					UpToDate:   false,
					Defined:    false,
					ConfigHash: "f72bbc9aab1c45a4075711c05dda5814398882c2ea25282ae558c9903c340cc1",
				},
			},
		},
	}

	client := fake.NewClientBuilder().WithScheme(scheme.Scheme).Build()
	assert.NoError(t, (&appStatusRenderer{
		ctx: context.Background(),
		c:   client,
		app: app,
	}).readJobs())

	assert.Equal(t, expectedAppStatus, app.Status.AppStatus)
}

func TestReadJobNotComplete(t *testing.T) {
	app := &v1.AppInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "test-app",
			Namespace:  "test-project",
			Generation: 1,
		},
		Status: v1.AppInstanceStatus{
			EmbeddedAppStatus: v1.EmbeddedAppStatus{
				Namespace: "test-namespace",
				AppSpec: v1.AppSpec{
					Jobs: map[string]v1.Container{
						"test-job": {},
					},
				},
			},
		},
	}

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				labels.AcornAppGeneration:        "1",
				labels.AcornConfigHashAnnotation: "f72bbc9aab1c45a4075711c05dda5814398882c2ea25282ae558c9903c340cc1",
			},
			CreationTimestamp: metav1.NewTime(time.Now().Add(-10 * time.Second).Truncate(time.Second)),
			Name:              "test-job",
			Namespace:         "test-namespace",
		},
		Status: batchv1.JobStatus{
			StartTime: z.Pointer(metav1.NewTime(time.Now().Add(-5 * time.Second).Truncate(time.Second))),
			Active:    1,
		},
	}

	expectedAppStatus := v1.AppStatus{
		Jobs: map[string]v1.JobStatus{
			"test-job": {
				JobName:      "test-job",
				JobNamespace: "test-namespace",
				CreationTime: &job.CreationTimestamp,
				LastRun:      job.Status.StartTime,
				StartTime:    job.Status.StartTime,
				RunningCount: int(job.Status.Active),
				CommonStatus: v1.CommonStatus{
					UpToDate:   true,
					Defined:    true,
					ConfigHash: "f72bbc9aab1c45a4075711c05dda5814398882c2ea25282ae558c9903c340cc1",
				},
			},
		},
	}

	client := fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(job).Build()
	assert.NoError(t, (&appStatusRenderer{
		ctx: context.Background(),
		c:   client,
		app: app,
	}).readJobs())

	assert.Equal(t, expectedAppStatus, app.Status.AppStatus)
}

func TestReadJobComplete(t *testing.T) {
	app := &v1.AppInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "test-app",
			Namespace:  "test-project",
			Generation: 1,
		},
		Status: v1.AppInstanceStatus{
			EmbeddedAppStatus: v1.EmbeddedAppStatus{
				Namespace: "test-namespace",
				AppSpec: v1.AppSpec{
					Jobs: map[string]v1.Container{
						"test-job": {},
					},
				},
			},
		},
	}

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				labels.AcornAppGeneration:        "1",
				labels.AcornConfigHashAnnotation: "f72bbc9aab1c45a4075711c05dda5814398882c2ea25282ae558c9903c340cc1",
			},
			CreationTimestamp: metav1.NewTime(time.Now().Add(-10 * time.Second).Truncate(time.Second)),
			Name:              "test-job",
			Namespace:         "test-namespace",
		},
		Status: batchv1.JobStatus{
			StartTime: z.Pointer(metav1.NewTime(time.Now().Add(-5 * time.Second).Truncate(time.Second))),
			Succeeded: 1,
		},
	}

	expectedAppStatus := v1.AppStatus{
		Jobs: map[string]v1.JobStatus{
			"test-job": {
				JobName:              "test-job",
				JobNamespace:         "test-namespace",
				CreationTime:         &job.CreationTimestamp,
				LastRun:              job.Status.StartTime,
				StartTime:            job.Status.StartTime,
				CreateEventSucceeded: true,
				CommonStatus: v1.CommonStatus{
					Ready:      true,
					UpToDate:   true,
					Defined:    true,
					ConfigHash: "f72bbc9aab1c45a4075711c05dda5814398882c2ea25282ae558c9903c340cc1",
				},
			},
		},
	}

	client := fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(job).Build()
	assert.NoError(t, (&appStatusRenderer{
		ctx: context.Background(),
		c:   client,
		app: app,
	}).readJobs())

	assert.Equal(t, expectedAppStatus, app.Status.AppStatus)
}

func TestReadJobFailed(t *testing.T) {
	app := &v1.AppInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "test-app",
			Namespace:  "test-project",
			Generation: 1,
		},
		Status: v1.AppInstanceStatus{
			EmbeddedAppStatus: v1.EmbeddedAppStatus{
				Namespace: "test-namespace",
				AppSpec: v1.AppSpec{
					Jobs: map[string]v1.Container{
						"test-job": {},
					},
				},
			},
		},
	}

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				labels.AcornAppGeneration:        "1",
				labels.AcornConfigHashAnnotation: "f72bbc9aab1c45a4075711c05dda5814398882c2ea25282ae558c9903c340cc1",
			},
			CreationTimestamp: metav1.NewTime(time.Now().Add(-10 * time.Second).Truncate(time.Second)),
			Name:              "test-job",
			Namespace:         "test-namespace",
		},
		Status: batchv1.JobStatus{
			StartTime: z.Pointer(metav1.NewTime(time.Now().Add(-5 * time.Second).Truncate(time.Second))),
			Failed:    1,
		},
	}

	expectedAppStatus := v1.AppStatus{
		Jobs: map[string]v1.JobStatus{
			"test-job": {
				JobName:      "test-job",
				JobNamespace: "test-namespace",
				CreationTime: &job.CreationTimestamp,
				LastRun:      job.Status.StartTime,
				StartTime:    job.Status.StartTime,
				ErrorCount:   1,
				CommonStatus: v1.CommonStatus{
					UpToDate:   true,
					Defined:    true,
					ConfigHash: "f72bbc9aab1c45a4075711c05dda5814398882c2ea25282ae558c9903c340cc1",
				},
			},
		},
	}

	client := fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(job).Build()
	assert.NoError(t, (&appStatusRenderer{
		ctx: context.Background(),
		c:   client,
		app: app,
	}).readJobs())

	assert.Equal(t, expectedAppStatus, app.Status.AppStatus)
}

func TestReadJobsSkippedCronJobNotSkipped(t *testing.T) {
	app := &v1.AppInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "test-app",
			Namespace:  "test-project",
			Generation: 1,
		},
		Status: v1.AppInstanceStatus{
			EmbeddedAppStatus: v1.EmbeddedAppStatus{
				Namespace: "test-namespace",
				AppSpec: v1.AppSpec{
					Jobs: map[string]v1.Container{
						"test-job": {
							Schedule: "*/5 * * * *",
						},
					},
				},
				AppStatus: v1.AppStatus{
					Jobs: map[string]v1.JobStatus{
						"test-job": {
							Skipped: true,
						},
					},
				},
			},
		},
	}

	expectedAppStatus := v1.AppStatus{
		Jobs: map[string]v1.JobStatus{
			"test-job": {
				JobName:      "test-job",
				JobNamespace: "test-namespace",
				Skipped:      true,
				CommonStatus: v1.CommonStatus{
					Ready:      false,
					UpToDate:   false,
					Defined:    false,
					ConfigHash: "3fb37507a299e87036973de4a49b1d75e43bdf690191b0a78bef842cf52b79df",
				},
			},
		},
	}

	client := fake.NewClientBuilder().WithScheme(scheme.Scheme).Build()
	assert.NoError(t, (&appStatusRenderer{
		ctx: context.Background(),
		c:   client,
		app: app,
	}).readJobs())

	assert.Equal(t, expectedAppStatus, app.Status.AppStatus)
}

func TestReadJobsCronNotCreated(t *testing.T) {
	app := &v1.AppInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "test-app",
			Namespace:  "test-project",
			Generation: 1,
		},
		Status: v1.AppInstanceStatus{
			EmbeddedAppStatus: v1.EmbeddedAppStatus{
				Namespace: "test-namespace",
				AppSpec: v1.AppSpec{
					Jobs: map[string]v1.Container{
						"test-job": {
							Schedule: "*/5 * * * *",
						},
					},
				},
			},
		},
	}

	expectedAppStatus := v1.AppStatus{
		Jobs: map[string]v1.JobStatus{
			"test-job": {
				JobName:      "test-job",
				JobNamespace: "test-namespace",
				CommonStatus: v1.CommonStatus{
					UpToDate:   false,
					Defined:    false,
					ConfigHash: "3fb37507a299e87036973de4a49b1d75e43bdf690191b0a78bef842cf52b79df",
				},
			},
		},
	}

	client := fake.NewClientBuilder().WithScheme(scheme.Scheme).Build()
	assert.NoError(t, (&appStatusRenderer{
		ctx: context.Background(),
		c:   client,
		app: app,
	}).readJobs())

	assert.Equal(t, expectedAppStatus, app.Status.AppStatus)
}

func TestReadJobsNestedNotCreated(t *testing.T) {
	app := &v1.AppInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "test-app",
			Namespace:  "test-project",
			Generation: 1,
		},
		Status: v1.AppInstanceStatus{
			EmbeddedAppStatus: v1.EmbeddedAppStatus{
				Namespace: "test-namespace",
				AppSpec: v1.AppSpec{
					Jobs: map[string]v1.Container{
						"test-job": {
							Schedule: "*/5 * * * *",
						},
					},
				},
			},
		},
	}

	cronJob := &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				labels.AcornAppGeneration:        "1",
				labels.AcornConfigHashAnnotation: "3fb37507a299e87036973de4a49b1d75e43bdf690191b0a78bef842cf52b79df",
			},
			CreationTimestamp: metav1.NewTime(time.Now().Add(-10 * time.Second).Truncate(time.Second)),
			Name:              "test-job",
			Namespace:         "test-namespace",
		},
		Spec: batchv1.CronJobSpec{
			Schedule: "*/5 * * * *",
		},
	}

	schedule, err := cronv3.ParseStandard(cronJob.Spec.Schedule)
	assert.NoError(t, err)

	expectedAppStatus := v1.AppStatus{
		Jobs: map[string]v1.JobStatus{
			"test-job": {
				JobName:      "test-job",
				JobNamespace: "test-namespace",
				Schedule:     cronJob.Spec.Schedule,
				CreationTime: &cronJob.CreationTimestamp,
				NextRun:      z.Pointer(metav1.NewTime(schedule.Next(cronJob.CreationTimestamp.Time))),
				CommonStatus: v1.CommonStatus{
					Ready:      true,
					UpToDate:   true,
					Defined:    true,
					ConfigHash: "3fb37507a299e87036973de4a49b1d75e43bdf690191b0a78bef842cf52b79df",
				},
			},
		},
	}

	client := fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(cronJob).Build()
	assert.NoError(t, (&appStatusRenderer{
		ctx: context.Background(),
		c:   client,
		app: app,
	}).readJobs())

	assert.Equal(t, expectedAppStatus, app.Status.AppStatus)
}

func TestReadJobsNestedRunning(t *testing.T) {
	app := &v1.AppInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "test-app",
			Namespace:  "test-project",
			Generation: 1,
		},
		Status: v1.AppInstanceStatus{
			EmbeddedAppStatus: v1.EmbeddedAppStatus{
				Namespace: "test-namespace",
				AppSpec: v1.AppSpec{
					Jobs: map[string]v1.Container{
						"test-job": {
							Schedule: "*/5 * * * *",
						},
					},
				},
			},
		},
	}

	cronJob := &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				labels.AcornAppGeneration:        "1",
				labels.AcornConfigHashAnnotation: "3fb37507a299e87036973de4a49b1d75e43bdf690191b0a78bef842cf52b79df",
			},
			CreationTimestamp: metav1.NewTime(time.Now().Add(-10 * time.Second).Truncate(time.Second)),
			Name:              "test-job",
			Namespace:         "test-namespace",
		},
		Spec: batchv1.CronJobSpec{
			Schedule: "*/5 * * * *",
		},
		Status: batchv1.CronJobStatus{
			LastScheduleTime: z.Pointer(metav1.NewTime(time.Now().Add(-10 * time.Second).Truncate(time.Second))),
			Active: []corev1.ObjectReference{
				{
					Namespace: "test-namespace",
					Name:      "test-nested-job",
				},
			},
		},
	}

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			CreationTimestamp: metav1.NewTime(time.Now().Add(-10 * time.Second).Truncate(time.Second)),
			Name:              "test-nested-job",
			Namespace:         "test-namespace",
		},
		Status: batchv1.JobStatus{
			StartTime: z.Pointer(metav1.NewTime(time.Now().Add(-5 * time.Second).Truncate(time.Second))),
			Active:    1,
		},
	}

	schedule, err := cronv3.ParseStandard(cronJob.Spec.Schedule)
	assert.NoError(t, err)

	expectedAppStatus := v1.AppStatus{
		Jobs: map[string]v1.JobStatus{
			"test-job": {
				JobName:      "test-job",
				JobNamespace: "test-namespace",
				Schedule:     cronJob.Spec.Schedule,
				CreationTime: &cronJob.CreationTimestamp,
				LastRun:      cronJob.Status.LastScheduleTime,
				NextRun:      z.Pointer(metav1.NewTime(schedule.Next(cronJob.CreationTimestamp.Time))),
				RunningCount: 1,
				CommonStatus: v1.CommonStatus{
					Ready:      true,
					UpToDate:   true,
					Defined:    true,
					ConfigHash: "3fb37507a299e87036973de4a49b1d75e43bdf690191b0a78bef842cf52b79df",
				},
			},
		},
	}

	client := fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(cronJob, job).Build()
	assert.NoError(t, (&appStatusRenderer{
		ctx: context.Background(),
		c:   client,
		app: app,
	}).readJobs())

	assert.Equal(t, expectedAppStatus, app.Status.AppStatus)
}

func TestReadJobsNestedCompleted(t *testing.T) {
	app := &v1.AppInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "test-app",
			Namespace:  "test-project",
			Generation: 1,
		},
		Status: v1.AppInstanceStatus{
			EmbeddedAppStatus: v1.EmbeddedAppStatus{
				Namespace: "test-namespace",
				AppSpec: v1.AppSpec{
					Jobs: map[string]v1.Container{
						"test-job": {
							Schedule: "*/5 * * * *",
						},
					},
				},
			},
		},
	}

	cronJob := &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				labels.AcornAppGeneration:        "1",
				labels.AcornConfigHashAnnotation: "3fb37507a299e87036973de4a49b1d75e43bdf690191b0a78bef842cf52b79df",
			},
			CreationTimestamp: metav1.NewTime(time.Now().Add(-10 * time.Second).Truncate(time.Second)),
			Name:              "test-job",
			Namespace:         "test-namespace",
		},
		Spec: batchv1.CronJobSpec{
			Schedule: "*/5 * * * *",
		},
		Status: batchv1.CronJobStatus{
			LastScheduleTime:   z.Pointer(metav1.NewTime(time.Now().Add(-10 * time.Second).Truncate(time.Second))),
			LastSuccessfulTime: z.Pointer(metav1.NewTime(time.Now().Add(-1 * time.Second).Truncate(time.Second))),
			Active: []corev1.ObjectReference{
				{
					Namespace: "test-namespace",
					Name:      "test-nested-job",
				},
			},
		},
	}

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			CreationTimestamp: metav1.NewTime(time.Now().Add(-10 * time.Second).Truncate(time.Second)),
			Name:              "test-nested-job",
			Namespace:         "test-namespace",
		},
		Status: batchv1.JobStatus{
			StartTime: z.Pointer(metav1.NewTime(time.Now().Add(-5 * time.Second).Truncate(time.Second))),
			Succeeded: 1,
		},
	}

	schedule, err := cronv3.ParseStandard(cronJob.Spec.Schedule)
	assert.NoError(t, err)

	expectedAppStatus := v1.AppStatus{
		Jobs: map[string]v1.JobStatus{
			"test-job": {
				JobName:              "test-job",
				JobNamespace:         "test-namespace",
				Schedule:             cronJob.Spec.Schedule,
				CreationTime:         &cronJob.CreationTimestamp,
				LastRun:              cronJob.Status.LastScheduleTime,
				CompletionTime:       cronJob.Status.LastSuccessfulTime,
				NextRun:              z.Pointer(metav1.NewTime(schedule.Next(cronJob.CreationTimestamp.Time))),
				CreateEventSucceeded: true,
				CommonStatus: v1.CommonStatus{
					Ready:      true,
					UpToDate:   true,
					Defined:    true,
					ConfigHash: "3fb37507a299e87036973de4a49b1d75e43bdf690191b0a78bef842cf52b79df",
				},
			},
		},
	}

	client := fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(cronJob, job).Build()
	assert.NoError(t, (&appStatusRenderer{
		ctx: context.Background(),
		c:   client,
		app: app,
	}).readJobs())

	assert.Equal(t, expectedAppStatus, app.Status.AppStatus)
}

func TestReadJobsNestedFailed(t *testing.T) {
	app := &v1.AppInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "test-app",
			Namespace:  "test-project",
			Generation: 1,
		},
		Status: v1.AppInstanceStatus{
			EmbeddedAppStatus: v1.EmbeddedAppStatus{
				Namespace: "test-namespace",
				AppSpec: v1.AppSpec{
					Jobs: map[string]v1.Container{
						"test-job": {
							Schedule: "*/5 * * * *",
						},
					},
				},
			},
		},
	}

	cronJob := &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				labels.AcornAppGeneration:        "1",
				labels.AcornConfigHashAnnotation: "3fb37507a299e87036973de4a49b1d75e43bdf690191b0a78bef842cf52b79df",
			},
			CreationTimestamp: metav1.NewTime(time.Now().Add(-10 * time.Second).Truncate(time.Second)),
			Name:              "test-job",
			Namespace:         "test-namespace",
		},
		Spec: batchv1.CronJobSpec{
			Schedule: "*/5 * * * *",
		},
		Status: batchv1.CronJobStatus{
			LastScheduleTime: z.Pointer(metav1.NewTime(time.Now().Add(-10 * time.Second).Truncate(time.Second))),
			Active: []corev1.ObjectReference{
				{
					Namespace: "test-namespace",
					Name:      "test-nested-job",
				},
			},
		},
	}

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			CreationTimestamp: metav1.NewTime(time.Now().Add(-10 * time.Second).Truncate(time.Second)),
			Name:              "test-nested-job",
			Namespace:         "test-namespace",
		},
		Status: batchv1.JobStatus{
			StartTime: z.Pointer(metav1.NewTime(time.Now().Add(-5 * time.Second).Truncate(time.Second))),
			Failed:    1,
		},
	}

	schedule, err := cronv3.ParseStandard(cronJob.Spec.Schedule)
	assert.NoError(t, err)

	expectedAppStatus := v1.AppStatus{
		Jobs: map[string]v1.JobStatus{
			"test-job": {
				JobName:      "test-job",
				JobNamespace: "test-namespace",
				Schedule:     cronJob.Spec.Schedule,
				CreationTime: &cronJob.CreationTimestamp,
				LastRun:      cronJob.Status.LastScheduleTime,
				NextRun:      z.Pointer(metav1.NewTime(schedule.Next(cronJob.CreationTimestamp.Time))),
				ErrorCount:   1,
				CommonStatus: v1.CommonStatus{
					UpToDate:   true,
					Defined:    true,
					ConfigHash: "3fb37507a299e87036973de4a49b1d75e43bdf690191b0a78bef842cf52b79df",
				},
			},
		},
	}

	client := fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(cronJob, job).Build()
	assert.NoError(t, (&appStatusRenderer{
		ctx: context.Background(),
		c:   client,
		app: app,
	}).readJobs())

	assert.Equal(t, expectedAppStatus, app.Status.AppStatus)
}

func TestReadJobsNestedRunningAgain(t *testing.T) {
	app := &v1.AppInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "test-app",
			Namespace:  "test-project",
			Generation: 1,
		},
		Status: v1.AppInstanceStatus{
			EmbeddedAppStatus: v1.EmbeddedAppStatus{
				Namespace: "test-namespace",
				AppSpec: v1.AppSpec{
					Jobs: map[string]v1.Container{
						"test-job": {
							Schedule: "*/5 * * * *",
						},
					},
				},
			},
		},
	}

	cronJob := &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				labels.AcornAppGeneration:        "1",
				labels.AcornConfigHashAnnotation: "3fb37507a299e87036973de4a49b1d75e43bdf690191b0a78bef842cf52b79df",
			},
			CreationTimestamp: metav1.NewTime(time.Now().Add(-10 * time.Second).Truncate(time.Second)),
			Name:              "test-job",
			Namespace:         "test-namespace",
		},
		Spec: batchv1.CronJobSpec{
			Schedule: "*/5 * * * *",
		},
		Status: batchv1.CronJobStatus{
			LastScheduleTime:   z.Pointer(metav1.NewTime(time.Now().Add(-10 * time.Second).Truncate(time.Second))),
			LastSuccessfulTime: z.Pointer(metav1.NewTime(time.Now().Add(-5 * time.Minute).Truncate(time.Second))),
			Active: []corev1.ObjectReference{
				{
					Namespace: "test-namespace",
					Name:      "test-nested-job",
				},
			},
		},
	}

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			CreationTimestamp: metav1.NewTime(time.Now().Add(-10 * time.Second).Truncate(time.Second)),
			Name:              "test-nested-job",
			Namespace:         "test-namespace",
		},
		Status: batchv1.JobStatus{
			StartTime: z.Pointer(metav1.NewTime(time.Now().Add(-5 * time.Second).Truncate(time.Second))),
			Active:    1,
		},
	}

	schedule, err := cronv3.ParseStandard(cronJob.Spec.Schedule)
	assert.NoError(t, err)

	expectedAppStatus := v1.AppStatus{
		Jobs: map[string]v1.JobStatus{
			"test-job": {
				JobName:              "test-job",
				JobNamespace:         "test-namespace",
				Schedule:             cronJob.Spec.Schedule,
				CreationTime:         &cronJob.CreationTimestamp,
				CompletionTime:       cronJob.Status.LastSuccessfulTime,
				LastRun:              cronJob.Status.LastScheduleTime,
				NextRun:              z.Pointer(metav1.NewTime(schedule.Next(cronJob.CreationTimestamp.Time))),
				RunningCount:         1,
				CreateEventSucceeded: true,
				CommonStatus: v1.CommonStatus{
					Ready:      true,
					UpToDate:   true,
					Defined:    true,
					ConfigHash: "3fb37507a299e87036973de4a49b1d75e43bdf690191b0a78bef842cf52b79df",
				},
			},
		},
	}

	client := fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(cronJob, job).Build()
	assert.NoError(t, (&appStatusRenderer{
		ctx: context.Background(),
		c:   client,
		app: app,
	}).readJobs())

	assert.Equal(t, expectedAppStatus, app.Status.AppStatus)
}

func TestReadJobsJobEventPrecedenceOverCron(t *testing.T) {
	app := &v1.AppInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "test-app",
			Namespace:  "test-project",
			Generation: 1,
		},
		Status: v1.AppInstanceStatus{
			EmbeddedAppStatus: v1.EmbeddedAppStatus{
				Namespace: "test-namespace",
				AppSpec: v1.AppSpec{
					Jobs: map[string]v1.Container{
						"test-job": {
							Schedule: "*/5 * * * *",
							Events:   []string{"create"},
						},
					},
				},
			},
		},
	}

	cronJob := &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				labels.AcornAppGeneration:        "1",
				labels.AcornConfigHashAnnotation: "a898b8bc39b4f2510970ab40e0f283fec941fb197b1e3015f11e84b0b999a63f",
			},
			CreationTimestamp: metav1.NewTime(time.Now().Add(-10 * time.Second).Truncate(time.Second)),
			Name:              "test-job",
			Namespace:         "test-namespace",
		},
		Spec: batchv1.CronJobSpec{
			Schedule: "*/5 * * * *",
		},
		Status: batchv1.CronJobStatus{
			LastScheduleTime:   z.Pointer(metav1.NewTime(time.Now().Add(-10 * time.Second).Truncate(time.Second))),
			LastSuccessfulTime: z.Pointer(metav1.NewTime(time.Now().Add(-5 * time.Minute).Truncate(time.Second))),
			Active: []corev1.ObjectReference{
				{
					Namespace: "test-namespace",
					Name:      "test-nested-job",
				},
			},
		},
	}

	nestedJob := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			CreationTimestamp: metav1.NewTime(time.Now().Add(-10 * time.Second).Truncate(time.Second)),
			Name:              "test-nested-job",
			Namespace:         "test-namespace",
		},
		Status: batchv1.JobStatus{
			StartTime: z.Pointer(metav1.NewTime(time.Now().Add(-5 * time.Second).Truncate(time.Second))),
			Failed:    1,
		},
	}

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				labels.AcornAppGeneration:        "1",
				labels.AcornConfigHashAnnotation: "a898b8bc39b4f2510970ab40e0f283fec941fb197b1e3015f11e84b0b999a63f",
			},
			CreationTimestamp: metav1.NewTime(time.Now().Add(-10 * time.Second).Truncate(time.Second)),
			Name:              "test-job",
			Namespace:         "test-namespace",
		},
		Status: batchv1.JobStatus{
			StartTime: z.Pointer(metav1.NewTime(time.Now().Add(-5 * time.Second).Truncate(time.Second))),
			Active:    1,
		},
	}

	expectedAppStatus := v1.AppStatus{
		Jobs: map[string]v1.JobStatus{
			"test-job": {
				JobName:      "test-job",
				JobNamespace: "test-namespace",
				CreationTime: &job.CreationTimestamp,
				LastRun:      job.Status.StartTime,
				StartTime:    job.Status.StartTime,
				RunningCount: int(job.Status.Active),
				CommonStatus: v1.CommonStatus{
					UpToDate:   true,
					Defined:    true,
					ConfigHash: "a898b8bc39b4f2510970ab40e0f283fec941fb197b1e3015f11e84b0b999a63f",
				},
			},
		},
	}

	client := fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(cronJob, nestedJob, job).Build()
	assert.NoError(t, (&appStatusRenderer{
		ctx: context.Background(),
		c:   client,
		app: app,
	}).readJobs())

	assert.Equal(t, expectedAppStatus, app.Status.AppStatus)
}

func TestReadJobsJobEventPrecedenceOverCronWhenUpdateNeeded(t *testing.T) {
	app := &v1.AppInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "test-app",
			Namespace:  "test-project",
			Generation: 1,
		},
		Status: v1.AppInstanceStatus{
			EmbeddedAppStatus: v1.EmbeddedAppStatus{
				Namespace: "test-namespace",
				AppSpec: v1.AppSpec{
					Jobs: map[string]v1.Container{
						"test-job": {
							Schedule: "*/5 * * * *",
							Events:   []string{"create"},
						},
					},
				},
			},
		},
	}

	cronJob := &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				labels.AcornAppGeneration:        "1",
				labels.AcornConfigHashAnnotation: "a898b8bc39b4f2510970ab40e0f283fec941fb197b1e3015f11e84b0b999a63f",
			},
			CreationTimestamp: metav1.NewTime(time.Now().Add(-10 * time.Second).Truncate(time.Second)),
			Name:              "test-job",
			Namespace:         "test-namespace",
		},
		Spec: batchv1.CronJobSpec{
			Schedule: "*/5 * * * *",
		},
		Status: batchv1.CronJobStatus{
			LastScheduleTime:   z.Pointer(metav1.NewTime(time.Now().Add(-10 * time.Second).Truncate(time.Second))),
			LastSuccessfulTime: z.Pointer(metav1.NewTime(time.Now().Add(-5 * time.Minute).Truncate(time.Second))),
			Active: []corev1.ObjectReference{
				{
					Namespace: "test-namespace",
					Name:      "test-nested-job",
				},
			},
		},
	}

	nestedJob := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			CreationTimestamp: metav1.NewTime(time.Now().Add(-10 * time.Second).Truncate(time.Second)),
			Name:              "test-nested-job",
			Namespace:         "test-namespace",
		},
		Status: batchv1.JobStatus{
			StartTime: z.Pointer(metav1.NewTime(time.Now().Add(-5 * time.Second).Truncate(time.Second))),
			Failed:    1,
		},
	}

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				labels.AcornAppGeneration:        "1",
				labels.AcornConfigHashAnnotation: "3fb37507a299e87036973de4a49b1d75e43bdf690191b0a78bef842cf52b79df",
			},
			CreationTimestamp: metav1.NewTime(time.Now().Add(-10 * time.Second).Truncate(time.Second)),
			Name:              "test-job",
			Namespace:         "test-namespace",
		},
		Status: batchv1.JobStatus{
			StartTime: z.Pointer(metav1.NewTime(time.Now().Add(-5 * time.Second).Truncate(time.Second))),
			Succeeded: 1,
		},
	}

	expectedAppStatus := v1.AppStatus{
		Jobs: map[string]v1.JobStatus{
			"test-job": {
				JobName:              "test-job",
				JobNamespace:         "test-namespace",
				CreationTime:         &job.CreationTimestamp,
				LastRun:              job.Status.StartTime,
				StartTime:            job.Status.StartTime,
				CreateEventSucceeded: true,
				RunningCount:         int(job.Status.Active),
				CommonStatus: v1.CommonStatus{
					Defined:    true,
					ConfigHash: "a898b8bc39b4f2510970ab40e0f283fec941fb197b1e3015f11e84b0b999a63f",
				},
			},
		},
	}

	client := fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(cronJob, nestedJob, job).Build()
	assert.NoError(t, (&appStatusRenderer{
		ctx: context.Background(),
		c:   client,
		app: app,
	}).readJobs())

	assert.Equal(t, expectedAppStatus, app.Status.AppStatus)
}

func TestReadJobsJobEventSucceededAndCronFailed(t *testing.T) {
	app := &v1.AppInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "test-app",
			Namespace:  "test-project",
			Generation: 1,
		},
		Status: v1.AppInstanceStatus{
			EmbeddedAppStatus: v1.EmbeddedAppStatus{
				Namespace: "test-namespace",
				AppSpec: v1.AppSpec{
					Jobs: map[string]v1.Container{
						"test-job": {
							Schedule: "*/5 * * * *",
							Events:   []string{"create"},
						},
					},
				},
			},
		},
	}

	cronJob := &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				labels.AcornAppGeneration:        "1",
				labels.AcornConfigHashAnnotation: "a898b8bc39b4f2510970ab40e0f283fec941fb197b1e3015f11e84b0b999a63f",
			},
			CreationTimestamp: metav1.NewTime(time.Now().Add(-10 * time.Second).Truncate(time.Second)),
			Name:              "test-job",
			Namespace:         "test-namespace",
		},
		Spec: batchv1.CronJobSpec{
			Schedule: "*/5 * * * *",
		},
		Status: batchv1.CronJobStatus{
			LastScheduleTime: z.Pointer(metav1.NewTime(time.Now().Add(-10 * time.Second).Truncate(time.Second))),
			Active: []corev1.ObjectReference{
				{
					Namespace: "test-namespace",
					Name:      "test-nested-job",
				},
			},
		},
	}

	nestedJob := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			CreationTimestamp: metav1.NewTime(time.Now().Add(-10 * time.Second).Truncate(time.Second)),
			Name:              "test-nested-job",
			Namespace:         "test-namespace",
		},
		Status: batchv1.JobStatus{
			StartTime: z.Pointer(metav1.NewTime(time.Now().Add(-5 * time.Second).Truncate(time.Second))),
			Failed:    1,
		},
	}

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				labels.AcornAppGeneration:        "1",
				labels.AcornConfigHashAnnotation: "a898b8bc39b4f2510970ab40e0f283fec941fb197b1e3015f11e84b0b999a63f",
			},
			CreationTimestamp: metav1.NewTime(time.Now().Add(-10 * time.Second).Truncate(time.Second)),
			Name:              "test-job",
			Namespace:         "test-namespace",
		},
		Status: batchv1.JobStatus{
			StartTime: z.Pointer(metav1.NewTime(time.Now().Add(-5 * time.Second).Truncate(time.Second))),
			Succeeded: 1,
		},
	}

	schedule, err := cronv3.ParseStandard(cronJob.Spec.Schedule)
	assert.NoError(t, err)

	expectedAppStatus := v1.AppStatus{
		Jobs: map[string]v1.JobStatus{
			"test-job": {
				JobName:      "test-job",
				JobNamespace: "test-namespace",
				Schedule:     cronJob.Spec.Schedule,
				CreationTime: &cronJob.CreationTimestamp,
				LastRun:      cronJob.Status.LastScheduleTime,
				NextRun:      z.Pointer(metav1.NewTime(schedule.Next(cronJob.CreationTimestamp.Time))),
				ErrorCount:   1,
				CommonStatus: v1.CommonStatus{
					UpToDate:   true,
					Defined:    true,
					ConfigHash: "a898b8bc39b4f2510970ab40e0f283fec941fb197b1e3015f11e84b0b999a63f",
				},
			},
		},
	}

	client := fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(cronJob, nestedJob, job).Build()
	assert.NoError(t, (&appStatusRenderer{
		ctx: context.Background(),
		c:   client,
		app: app,
	}).readJobs())

	assert.Equal(t, expectedAppStatus, app.Status.AppStatus)
}

func TestReadJobsJobEventSucceededAndCronSucceeded(t *testing.T) {
	app := &v1.AppInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "test-app",
			Namespace:  "test-project",
			Generation: 1,
		},
		Status: v1.AppInstanceStatus{
			EmbeddedAppStatus: v1.EmbeddedAppStatus{
				Namespace: "test-namespace",
				AppSpec: v1.AppSpec{
					Jobs: map[string]v1.Container{
						"test-job": {
							Schedule: "*/5 * * * *",
							Events:   []string{"create"},
						},
					},
				},
			},
		},
	}

	cronJob := &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				labels.AcornAppGeneration:        "1",
				labels.AcornConfigHashAnnotation: "a898b8bc39b4f2510970ab40e0f283fec941fb197b1e3015f11e84b0b999a63f",
			},
			CreationTimestamp: metav1.NewTime(time.Now().Add(-10 * time.Second).Truncate(time.Second)),
			Name:              "test-job",
			Namespace:         "test-namespace",
		},
		Spec: batchv1.CronJobSpec{
			Schedule: "*/5 * * * *",
		},
		Status: batchv1.CronJobStatus{
			LastScheduleTime:   z.Pointer(metav1.NewTime(time.Now().Add(-10 * time.Second).Truncate(time.Second))),
			LastSuccessfulTime: z.Pointer(metav1.NewTime(time.Now().Add(-5 * time.Second).Truncate(time.Second))),
			Active: []corev1.ObjectReference{
				{
					Namespace: "test-namespace",
					Name:      "test-nested-job",
				},
			},
		},
	}

	nestedJob := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			CreationTimestamp: metav1.NewTime(time.Now().Add(-10 * time.Second).Truncate(time.Second)),
			Name:              "test-nested-job",
			Namespace:         "test-namespace",
		},
		Status: batchv1.JobStatus{
			StartTime: z.Pointer(metav1.NewTime(time.Now().Add(-5 * time.Second).Truncate(time.Second))),
			Succeeded: 1,
		},
	}

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				labels.AcornAppGeneration:        "1",
				labels.AcornConfigHashAnnotation: "a898b8bc39b4f2510970ab40e0f283fec941fb197b1e3015f11e84b0b999a63f",
			},
			CreationTimestamp: metav1.NewTime(time.Now().Add(-10 * time.Second).Truncate(time.Second)),
			Name:              "test-job",
			Namespace:         "test-namespace",
		},
		Status: batchv1.JobStatus{
			StartTime: z.Pointer(metav1.NewTime(time.Now().Add(-5 * time.Second).Truncate(time.Second))),
			Succeeded: 1,
		},
	}

	schedule, err := cronv3.ParseStandard(cronJob.Spec.Schedule)
	assert.NoError(t, err)

	expectedAppStatus := v1.AppStatus{
		Jobs: map[string]v1.JobStatus{
			"test-job": {
				JobName:              "test-job",
				JobNamespace:         "test-namespace",
				Schedule:             cronJob.Spec.Schedule,
				CreationTime:         &cronJob.CreationTimestamp,
				CompletionTime:       cronJob.Status.LastSuccessfulTime,
				LastRun:              cronJob.Status.LastScheduleTime,
				NextRun:              z.Pointer(metav1.NewTime(schedule.Next(cronJob.CreationTimestamp.Time))),
				CreateEventSucceeded: true,
				CommonStatus: v1.CommonStatus{
					UpToDate:   true,
					Defined:    true,
					Ready:      true,
					ConfigHash: "a898b8bc39b4f2510970ab40e0f283fec941fb197b1e3015f11e84b0b999a63f",
				},
			},
		},
	}

	client := fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(cronJob, nestedJob, job).Build()
	assert.NoError(t, (&appStatusRenderer{
		ctx: context.Background(),
		c:   client,
		app: app,
	}).readJobs())

	assert.Equal(t, expectedAppStatus, app.Status.AppStatus)
}
