package appstatus

import (
	"fmt"
	"strconv"

	"github.com/acorn-io/baaah/pkg/router"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/labels"
	"github.com/acorn-io/runtime/pkg/ports"
	"github.com/acorn-io/z"
	cronv3 "github.com/robfig/cron/v3"
	batchv1 "k8s.io/api/batch/v1"
	apierror "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func (a *appStatusRenderer) readJobs() error {
	existingStatus := a.app.Status.AppStatus.Jobs

	// reset state
	a.app.Status.AppStatus.Jobs = make(map[string]v1.JobStatus, len(a.app.Status.AppSpec.Jobs))

	summary, err := a.getReplicasSummary(labels.AcornJobName)
	if err != nil {
		return err
	}

	for jobName, jobDef := range a.app.Status.AppSpec.Jobs {
		hash, err := configHash(jobDef)
		if err != nil {
			return err
		}

		c := v1.JobStatus{
			CreateEventSucceeded: existingStatus[jobName].CreateEventSucceeded,
			Skipped:              existingStatus[jobName].Skipped,
			ExpressionErrors:     existingStatus[jobName].ExpressionErrors,
			Dependencies:         existingStatus[jobName].Dependencies,
			CommonStatus: v1.CommonStatus{
				ConfigHash: hash,
			},
		}
		summary := summary[jobName]

		c.Defined = ports.IsLinked(a.app, jobName)
		c.LinkOverride = ports.LinkService(a.app, jobName)
		c.TransitioningMessages = append(c.TransitioningMessages, summary.TransitioningMessages...)
		c.ErrorMessages = append(c.ErrorMessages, summary.ErrorMessages...)
		c.RunningCount = summary.RunningCount
		c.JobName = jobName
		c.JobNamespace = a.app.Status.Namespace

		if c.Skipped && jobDef.Schedule == "" {
			c.CreationTime = &a.app.CreationTimestamp
			c.State = "completed"
			c.Ready = true
			c.UpToDate = true
			c.Defined = true
			c.Dependencies = nil
			a.app.Status.AppStatus.Jobs[jobName] = c
			continue
		}

		var job batchv1.Job
		err = a.c.Get(a.ctx, router.Key(a.app.Status.Namespace, jobName), &job)
		if apierror.IsNotFound(err) || job.Status.Succeeded > 0 && jobDef.Schedule != "" && job.Annotations[labels.AcornAppGeneration] == strconv.Itoa(int(a.app.Generation)) && job.Annotations[labels.AcornConfigHashAnnotation] == hash {
			// If the job is not found, or it has succeeded and there should be an associated cronjob, then process that cronjob instead.
			var cronJob batchv1.CronJob
			err = a.c.Get(a.ctx, router.Key(a.app.Status.Namespace, jobName), &cronJob)
			if kclient.IgnoreNotFound(err) != nil {
				return err
			} else if err == nil {
				c.CreationTime = &cronJob.CreationTimestamp
				c.LastRun = cronJob.Status.LastScheduleTime
				c.CompletionTime = cronJob.Status.LastSuccessfulTime
				c.Schedule = cronJob.Spec.Schedule
				c.Defined = true
				c.UpToDate = cronJob.Annotations[labels.AcornAppGeneration] == strconv.Itoa(int(a.app.Generation)) && cronJob.Annotations[labels.AcornConfigHashAnnotation] == hash
				for _, nj := range cronJob.Status.Active {
					nestedJob := &batchv1.Job{}
					err := a.c.Get(a.ctx, router.Key(nj.Namespace, nj.Name), nestedJob)
					if err != nil {
						return err
					}
					c.RunningCount += int(nestedJob.Status.Active)
					c.ErrorCount += int(nestedJob.Status.Failed)
				}

				c.CreateEventSucceeded = cronJob.Status.LastSuccessfulTime != nil
				c.Ready = c.UpToDate && c.ErrorCount == 0

				c.NextRun, err = nextRun(c.Schedule, cronJob.CreationTimestamp, cronJob.Status.LastScheduleTime)
				if err != nil {
					return err
				}
			}
		} else if err != nil {
			return err
		} else {
			c.CreationTime = &job.CreationTimestamp
			c.CompletionTime = job.Status.CompletionTime
			c.StartTime = job.Status.StartTime
			c.LastRun = job.Status.StartTime
			c.Defined = true
			c.UpToDate = job.Annotations[labels.AcornAppGeneration] == strconv.Itoa(int(a.app.Generation)) && (c.Skipped || job.Annotations[labels.AcornConfigHashAnnotation] == hash)
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
			var err error
			c.UpToDate = true
			c.Ready, c.Defined, err = a.isServiceReady(jobName)
			if err != nil {
				return err
			}
			if c.Ready {
				c.CreateEventSucceeded = true
			}
		}

		a.app.Status.AppStatus.Jobs[jobName] = c
	}

	return nil
}

func setJobMessages(app *v1.AppInstance) {
	for jobName, c := range app.Status.AppStatus.Jobs {
		if c.RunningCount > 0 {
			c.TransitioningMessages = append(c.TransitioningMessages, "job running")
			// Move error to transitioning to make it look better
			c.TransitioningMessages = append(c.TransitioningMessages, c.ErrorMessages...)
			c.ErrorMessages = nil
		} else if c.ErrorCount > 0 && c.ErrorCount < 3 {
			c.TransitioningMessages = append(c.TransitioningMessages, fmt.Sprintf("restarting job, previous %d attempts failed to complete", c.ErrorCount))
			// Move error to transitioning to make it look better
			c.TransitioningMessages = append(c.TransitioningMessages, c.ErrorMessages...)
			c.ErrorMessages = nil
		} else if c.ErrorCount > 0 {
			c.ErrorMessages = append(c.ErrorMessages, fmt.Sprintf("%d failed attempts", c.ErrorCount))
		}

		addExpressionErrors(&c.CommonStatus, c.ExpressionErrors)

		// Not ready if we have any error messages
		if len(c.ErrorMessages) > 0 {
			c.Ready = false
		}

		if c.Ready {
			c.State = "completed"
		} else if c.UpToDate {
			if len(c.ErrorMessages) > 0 {
				c.State = "failing"
			} else if c.RunningCount > 0 {
				c.State = "running"
			} else {
				c.State = "pending"
			}
		} else if c.Defined {
			if len(c.ErrorMessages) > 0 {
				c.State = "error"
			} else {
				c.State = "updating"
			}
		} else {
			if len(c.ErrorMessages) > 0 {
				c.State = "error"
			} else {
				c.State = "pending"
			}
		}

		if !c.Ready {
			msg, blocked := isBlocked(c.Dependencies, c.ExpressionErrors)
			if blocked {
				c.State = "waiting"
			}
			c.TransitioningMessages = append(c.TransitioningMessages, msg...)
		}

		app.Status.AppStatus.Jobs[jobName] = c
	}
}

func addExpressionErrors(status *v1.CommonStatus, expressionErrors []v1.ExpressionError) {
	for _, ee := range expressionErrors {
		if !ee.IsMissingDependencyError() {
			status.ErrorMessages = append(status.ErrorMessages, ee.String())
		}
	}
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

// nextRun uses the cron expression library used by k8s to determine the next run time of a cronjob.
func nextRun(expression string, creation metav1.Time, last *metav1.Time) (*metav1.Time, error) {
	schedule, err := cronv3.ParseStandard(expression)
	if err != nil {
		return nil, err
	}

	if last == nil {
		last = &creation
	}

	return z.Pointer(metav1.NewTime(schedule.Next(last.Time))), nil
}
