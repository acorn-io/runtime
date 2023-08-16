package containers

import (
	"strings"
	"testing"

	client2 "github.com/acorn-io/runtime/integration/client"
	"github.com/acorn-io/runtime/integration/helper"
	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/client"
	kclient "github.com/acorn-io/runtime/pkg/k8sclient"
	"github.com/acorn-io/runtime/pkg/publicname"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestJobList(t *testing.T) {
	helper.StartController(t)
	restConfig := helper.StartAPI(t)

	ctx := helper.GetCTX(t)
	lclient, err := kclient.New(restConfig)
	require.NoError(t, err)

	kclient := helper.MustReturn(kclient.Default)
	project := helper.TempProject(t, kclient)

	c, err := client.New(restConfig, "", project.Name)
	require.NoError(t, err)

	imageID := client2.NewImageWithJobs(t, project.Name)
	app, err := c.AppRun(ctx, imageID, nil)
	require.NoError(t, err)

	app = helper.WaitForObject(t, lclient.Watch, &apiv1.AppList{}, app, func(app *apiv1.App) bool {
		return app.Status.AppStatus.Jobs["job"].CompletionTime != nil
	})

	jobs, err := c.JobList(ctx, nil)
	require.NoError(t, err)

	require.Len(t, jobs, 2)
	for _, job := range jobs {
		assert.Truef(t, strings.HasPrefix(job.Name, app.Name+"."), "not prefix %s %s", job.Name, app.Name)
		assert.Equal(t, app.Namespace, job.Namespace)
	}
}

func TestJobGet(t *testing.T) {
	helper.StartController(t)
	restConfig := helper.StartAPI(t)

	ctx := helper.GetCTX(t)
	lclient, err := kclient.New(restConfig)
	require.NoError(t, err)

	kclient := helper.MustReturn(kclient.Default)
	project := helper.TempProject(t, kclient)

	c, err := client.New(restConfig, "", project.Name)
	require.NoError(t, err)

	imageID := client2.NewImageWithJobs(t, project.Name)
	app, err := c.AppRun(ctx, imageID, nil)
	require.NoError(t, err)

	helper.WaitForObject(t, lclient.Watch, &apiv1.AppList{}, app, func(app *apiv1.App) bool {
		return app.Status.AppStatus.Jobs["job"].CompletionTime != nil
	})

	jobs, err := c.JobList(ctx, nil)
	require.NoError(t, err)

	// Determine which job is the cronjob and which is the job
	require.Len(t, jobs, 2)
	jobFromList, cronjobFromList := jobs[0], jobs[1]
	if cronjobFromList.Spec.Schedule == "" {
		jobFromList = jobs[1]
		cronjobFromList = jobs[0]
	}

	// Check that the job without a schedule is correct
	job, err := c.JobGet(ctx, jobFromList.Name)
	require.NoError(t, err)

	assert.Nil(t, jobFromList.Status.NextRun)
	assert.Equal(t, jobFromList.Name, job.Name)
	assert.Equal(t, jobFromList.Namespace, job.Namespace)
	assert.Equal(t, jobFromList.UID, job.UID)

	// Check that the cronjob is correct
	cronjob, err := c.JobGet(ctx, cronjobFromList.Name)
	require.NoError(t, err)

	assert.Equal(t, cronjobFromList.Name, cronjob.Name)
	assert.Equal(t, cronjobFromList.Namespace, cronjob.Namespace)
	assert.Equal(t, cronjobFromList.UID, cronjob.UID)
}

func TestJobRestart(t *testing.T) {
	helper.StartController(t)
	restConfig := helper.StartAPI(t)

	ctx := helper.GetCTX(t)
	lclient, err := kclient.New(restConfig)
	require.NoError(t, err)

	kclient := helper.MustReturn(kclient.Default)
	project := helper.TempProject(t, kclient)

	c, err := client.New(restConfig, "", project.Name)
	require.NoError(t, err)

	imageID := client2.NewImageWithJobs(t, project.Name)
	app, err := c.AppRun(ctx, imageID, nil)
	require.NoError(t, err)

	// Wait for the Job to initially complete
	var firstCompletion *metav1.Time
	helper.WaitForObject(t, lclient.Watch, &apiv1.AppList{}, app, func(app *apiv1.App) bool {
		firstCompletion = app.Status.AppStatus.Jobs["job"].CompletionTime
		return app.Status.Namespace != "" && app.Status.AppStatus.Jobs["job"].CompletionTime != nil
	})

	require.NoError(t, c.JobRestart(ctx, publicname.ForChild(app, "job")))

	// Wait for the Job to complete again by checking for a difference in the completion time
	helper.WaitForObject(t, lclient.Watch, &apiv1.AppList{}, app, func(app *apiv1.App) bool {
		secondCompletion := app.Status.AppStatus.Jobs["job"].CompletionTime
		return app.Status.Namespace != "" && !firstCompletion.Equal(secondCompletion)
	})

	require.NoError(t, c.JobRestart(ctx, publicname.ForChild(app, "cronjob")))

	// Wait for the CronJob to complete once, which means it has been restarted since the job
	// is scheduled to never run
	helper.WaitForObject(t, lclient.Watch, &apiv1.AppList{}, app, func(app *apiv1.App) bool {
		return app.Status.AppStatus.Jobs["cronjob"].LastRun != nil
	})
}
