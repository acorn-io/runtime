package client

import (
	"context"
	"strconv"
	"testing"

	"github.com/acorn-io/acorn/integration/helper"
	v1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/build"
	"github.com/acorn-io/acorn/pkg/client"
	"github.com/acorn-io/acorn/pkg/k8sclient"
	"github.com/acorn-io/acorn/pkg/labels"
	"github.com/stretchr/testify/assert"
	"golang.org/x/sync/errgroup"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func depImage(t *testing.T, c client.Client) string {
	image, err := build.Build(helper.GetCTX(t), "./testdata/dependson/acorn.cue", &build.Options{
		Client: c,
		Cwd:    "./testdata/dependson",
	})
	if err != nil {
		t.Fatal(err)
	}
	return image.ID
}

func toRevision(t *testing.T, obj kclient.Object) int {
	i, err := strconv.Atoi(obj.GetResourceVersion())
	if err != nil {
		t.Fatalf("Invalid resource version %s on %s/%s", obj.GetResourceVersion(), obj.GetNamespace(), obj.GetName())
	}
	return i
}

func TestDependsOn(t *testing.T) {
	ctx := context.Background()
	c, _ := helper.ClientAndNamespace(t)
	k8sclient := helper.MustReturn(k8sclient.Default)
	image := depImage(t, c)

	app, err := c.AppRun(ctx, image, nil)
	if err != nil {
		t.Fatal(err)
	}

	jobs := map[string]int{}
	deployments := map[string]int{}

	app = helper.WaitForObject(t, c.GetClient().Watch, &v1.AppList{}, app, func(app *v1.App) bool {
		return app.Status.Namespace != ""
	})

	eg := errgroup.Group{}

	eg.Go(func() error {
		helper.Wait(t, k8sclient.Watch, &batchv1.JobList{}, func(job *batchv1.Job) bool {
			if job.Namespace != app.Status.Namespace {
				return false
			}
			name := job.Labels[labels.AcornJobName]
			if _, ok := jobs[name]; !ok {
				jobs[name] = toRevision(t, job)
				if len(jobs) == 2 {
					return true
				}
			}
			return false
		})
		return nil
	})

	eg.Go(func() error {
		helper.Wait(t, k8sclient.Watch, &appsv1.DeploymentList{}, func(dep *appsv1.Deployment) bool {
			if dep.Namespace != app.Status.Namespace {
				return false
			}
			name := dep.Labels[labels.AcornContainerName]
			if _, ok := deployments[name]; !ok {
				deployments[name] = toRevision(t, dep)
				if len(deployments) == 3 {
					return true
				}
			}
			return false
		})
		return nil
	})

	_ = eg.Wait()

	assert.Less(t, jobs["job2"], jobs["job1"])
	assert.Less(t, jobs["job1"], deployments["one"])
	assert.Less(t, jobs["job2"], deployments["one"])
	assert.Less(t, deployments["one"], deployments["two"])
	assert.Less(t, deployments["two"], deployments["three"])
}
