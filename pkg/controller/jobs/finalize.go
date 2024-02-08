package jobs

import (
	"sort"
	"time"

	"github.com/acorn-io/baaah/pkg/router"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/labels"
	"github.com/sirupsen/logrus"
	"golang.org/x/exp/slices"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierror "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	DestroyJobFinalizer = "jobs.acorn.io/destroy"
)

func JobPodOrphanCleanup(req router.Request, _ router.Response) error {
	pod := req.Object.(*corev1.Pod)
	// pods with "controller-uid" and "job-name" on them are created by batchv1.Job
	if pod.Labels[labels.AcornJobName] != "" &&
		pod.Labels["controller-uid"] != "" &&
		pod.Labels["job-name"] != "" &&
		len(pod.OwnerReferences) == 0 {
		return req.Client.Delete(req.Ctx, pod)
	}
	return nil
}

func JobCleanup(req router.Request, _ router.Response) error {
	job := req.Object.(*batchv1.Job)
	if job.Status.Failed == 0 || job.Spec.Selector == nil {
		return nil
	}

	pods := &corev1.PodList{}
	sel, err := metav1.LabelSelectorAsSelector(job.Spec.Selector)
	if err != nil {
		return err
	}
	err = req.List(pods, &kclient.ListOptions{
		LabelSelector: sel,
	})
	if err != nil {
		return err
	}

	sort.Slice(pods.Items, func(i, j int) bool {
		return pods.Items[i].CreationTimestamp.Before(&pods.Items[j].CreationTimestamp)
	})

	keep := 3
	if job.Status.Succeeded > 0 {
		keep = 0
	}
	if len(pods.Items) > keep {
		for _, pod := range pods.Items[:len(pods.Items)-keep] {
			if pod.Status.Phase != corev1.PodFailed {
				continue
			}

			logrus.Infof("Purging failed job %s/%s", pod.Namespace, pod.Name)
			if err = req.Client.Delete(req.Ctx, &pod); err != nil && !apierror.IsNotFound(err) {
				return err
			}
		}
	}

	return nil
}

func NeedsDestroyJobFinalization(next router.Handler) router.Handler {
	return router.HandlerFunc(func(req router.Request, resp router.Response) error {
		if req.Object == nil {
			return nil
		}

		app := req.Object.(*v1.AppInstance)

		if !app.DeletionTimestamp.IsZero() {
			// If deleting do normal finalizer stuff even if we don't need to
			return next.Handle(req, resp)
		}

		if shouldFinalize(app) {
			return next.Handle(req, resp)
		}

		return nil
	})
}

func FinalizeDestroyJob(req router.Request, resp router.Response) error {
	app := req.Object.(*v1.AppInstance)
	ns := &corev1.Namespace{}
	if err := req.Get(ns, "", app.Namespace); err != nil || !ns.DeletionTimestamp.IsZero() {
		return err
	}

	for jobName, jobDef := range app.Status.AppSpec.Jobs {
		if !slices.Contains(jobDef.Events, "delete") {
			continue
		}

		job := &batchv1.Job{}
		err := req.Get(job, app.Status.Namespace, jobName)
		if apierror.IsNotFound(err) {
			resp.DisablePrune()
			resp.RetryAfter(15 * time.Second)
			return nil
		} else if err != nil {
			return err
		}

		if done(job) {
			continue
		}

		resp.DisablePrune()
		resp.RetryAfter(15 * time.Second)
	}

	return nil
}

func done(job *batchv1.Job) bool {
	foundEnv := false
	for _, container := range job.Spec.Template.Spec.Containers {
		for _, env := range container.Env {
			if env.Name == "ACORN_EVENT" && env.Value == "delete" {
				foundEnv = true
			}
		}
	}
	if !foundEnv {
		// The job has not been updated for destroy yet
		return false
	}
	return job.Status.Succeeded > 0
}

func shouldFinalize(app *v1.AppInstance) bool {
	for _, job := range app.Status.AppSpec.Jobs {
		if slices.Contains(job.Events, "delete") {
			return true
		}
	}
	return false
}
