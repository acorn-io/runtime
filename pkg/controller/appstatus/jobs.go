package appstatus

import (
	"fmt"
	"strconv"

	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/labels"
	"github.com/acorn-io/acorn/pkg/ports"
	"github.com/acorn-io/baaah/pkg/router"
	"github.com/acorn-io/baaah/pkg/typed"
	batchv1 "k8s.io/api/batch/v1"
	apierror "k8s.io/apimachinery/pkg/api/errors"
)

func (a *appStatusRenderer) readJobs() error {
	var (
		existingStatus = a.app.Status.AppStatus.Jobs
	)

	// reset state
	a.app.Status.AppStatus.Jobs = make(map[string]v1.JobStatus, len(a.app.Status.AppSpec.Jobs))

	summary, err := a.getReplicasSummary(labels.AcornJobName)
	if err != nil {
		return err
	}

	for jobName := range a.app.Status.AppSpec.Jobs {
		c := v1.JobStatus{
			CreateEventSucceeded: existingStatus[jobName].CreateEventSucceeded,
			Skipped:              existingStatus[jobName].Skipped,
			ExpressionErrors:     existingStatus[jobName].ExpressionErrors,
		}
		summary := summary[jobName]

		c.Defined = ports.IsLinked(a.app, jobName)
		c.LinkOverride = ports.LinkService(a.app, jobName)
		c.TransitioningMessages = append(c.TransitioningMessages, summary.TransitioningMessages...)
		c.ErrorMessages = append(c.ErrorMessages, summary.ErrorMessages...)
		c.RunningCount = summary.RunningCount

		for _, ee := range c.ExpressionErrors {
			c.ErrorMessages = append(c.ErrorMessages, ee.String())
		}

		if c.Skipped {
			c.Ready = true
			c.UpToDate = true
			c.Defined = true
			c.ErrorCount = 0
			c.RunningCount = 0
			c.Dependencies = nil
			a.app.Status.AppStatus.Jobs[jobName] = c
			continue
		}

		var job batchv1.Job
		err := a.c.Get(a.ctx, router.Key(a.app.Status.Namespace, jobName), &job)
		if apierror.IsNotFound(err) {
			var cronJob batchv1.CronJob
			err := a.c.Get(a.ctx, router.Key(a.app.Status.Namespace, jobName), &cronJob)
			if apierror.IsNotFound(err) {
				// do nothing
			} else if err != nil {
				return err
			} else {
				c.Defined = true
				c.UpToDate = cronJob.Annotations[labels.AcornAppGeneration] == strconv.Itoa(int(a.app.Generation))
				c.RunningCount = len(cronJob.Status.Active)
				if cronJob.Status.LastSuccessfulTime != nil {
					c.CreateEventSucceeded = true
					c.Ready = c.UpToDate
				}
			}
		} else if err != nil {
			return err
		} else {
			c.Defined = true
			c.UpToDate = job.Annotations[labels.AcornAppGeneration] == strconv.Itoa(int(a.app.Generation))
			if job.Status.Succeeded > 0 {
				c.CreateEventSucceeded = true
				c.Ready = c.UpToDate
			} else if job.Status.Failed > 0 {
				c.ErrorCount = int(job.Status.Failed)
			} else if job.Status.Active > 0 && c.RunningCount == 0 {
				c.RunningCount = int(job.Status.Active)
			}
		}

		if c.LinkOverride != "" {
			c.UpToDate = true
			c.Ready, c.Defined = a.isServiceReady(jobName)
			if c.Ready {
				c.CreateEventSucceeded = true
			}
		}

		for _, entry := range typed.Sorted(c.Dependencies) {
			depName, dep := entry.Key, entry.Value
			if !dep.Ready {
				c.Ready = false
				msg := fmt.Sprintf("%s %s dependency is not ready", dep.DependencyType, depName)
				if dep.Missing {
					msg = fmt.Sprintf("%s %s dependency is missing", dep.DependencyType, depName)
				}
				c.TransitioningMessages = append(c.TransitioningMessages, msg)
			}
		}

		a.app.Status.AppStatus.Jobs[jobName] = c
	}

	return nil
}

func (a *appStatusRenderer) isJobReady(jobName string) (ready bool, err error) {
	var jobDep batchv1.Job
	err = a.c.Get(a.ctx, router.Key(a.app.Status.Namespace, jobName), &jobDep)
	if apierror.IsNotFound(err) {
		return false, nil
	}
	if err != nil {
		// if err just return it as not ready
		return false, err
	}

	if jobDep.Annotations[labels.AcornAppGeneration] != strconv.Itoa(int(a.app.Generation)) ||
		jobDep.Status.Succeeded != 1 {
		return false, nil
	}

	return true, nil
}
