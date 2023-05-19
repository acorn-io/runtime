package appdefinition

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/condition"
	"github.com/acorn-io/acorn/pkg/config"
	"github.com/acorn-io/acorn/pkg/labels"
	"github.com/acorn-io/baaah/pkg/apply"
	"github.com/acorn-io/baaah/pkg/router"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	klabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/sets"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func CheckDependencies(h router.Handler) router.Handler {
	return router.HandlerFunc(func(req router.Request, resp router.Response) error {
		app := req.Object.(*v1.AppInstance)
		if app != nil && app.Spec.GetStopped() {
			return h.Handle(req, resp)
		}

		depResp := &depCheckingResponse{
			app:     req.Object.(*v1.AppInstance),
			req:     req,
			resp:    resp,
			waiting: sets.New[string](),
		}

		if err := h.Handle(req, depResp); err != nil {
			return err
		}

		cond := condition.ForName(req.Object, v1.AppInstanceConditionDependencies)
		if depResp.waiting.Len() > 0 {
			cond.Unknown(fmt.Sprintf("waiting for ready %v", sets.List(depResp.waiting)))
		} else {
			cond.Success()
		}

		return nil
	})
}

type depCheckingResponse struct {
	app     *v1.AppInstance
	req     router.Request
	resp    router.Response
	waiting sets.Set[string]
}

func (d *depCheckingResponse) DisablePrune() {
	d.resp.DisablePrune()
}

func (d *depCheckingResponse) RetryAfter(delay time.Duration) {
	d.resp.RetryAfter(delay)
}

func (d *depCheckingResponse) Objects(objs ...kclient.Object) {
	for _, obj := range objs {
		objAnnotations := obj.GetAnnotations()
		if deps := objAnnotations[labels.AcornDepNames]; deps != "" {
			depName, ready := d.checkDeps(strings.Split(deps, ","))
			if !ready {
				objAnnotations[apply.AnnotationCreate] = "false"
				objAnnotations[apply.AnnotationUpdate] = "false"
				obj.SetAnnotations(objAnnotations)
				d.waiting.Insert(depName)
			}
		}
	}
	d.resp.Objects(objs...)
}

func (d *depCheckingResponse) isServiceReady(svc string) (ready bool, found bool) {
	return d.isServiceReadyByNamespace(d.app.Status.Namespace, svc)
}

func (d *depCheckingResponse) isServiceReadyByNamespace(namespace, svc string) (ready bool, found bool) {
	var svcDep corev1.Service
	err := d.req.Get(&svcDep, namespace, svc)
	if apierrors.IsNotFound(err) {
		return false, false
	}
	if err != nil {
		// if err just return it as not ready
		return false, true
	}

	if svcDep.Labels[labels.AcornManaged] != "true" {
		// for services we don't managed just return ready always
		return true, true
	}

	if svcDep.Spec.ExternalName != "" {
		cfg, err := config.Get(d.req.Ctx, d.req.Client)
		if err != nil {
			// if err just return it as not ready
			return false, true
		}
		if strings.HasSuffix(svcDep.Spec.ExternalName, cfg.InternalClusterDomain) {
			parts := strings.Split(svcDep.Spec.ExternalName, ".")
			if len(parts) > 2 {
				return d.isServiceReadyByNamespace(parts[1], parts[0])
			}
		}
		// for unknown external names we just assume they are always ready
		return true, true
	}

	var endpoints corev1.Endpoints
	err = d.req.Get(&endpoints, namespace, svc)
	if apierrors.IsNotFound(err) {
		return false, false
	}
	if err != nil {
		// if err just return it as not ready
		return false, true
	}

	for _, subset := range endpoints.Subsets {
		if len(subset.Addresses) > 0 && len(subset.Ports) > 0 {
			return true, true
		}
	}

	return false, true
}

func (d *depCheckingResponse) isCronJobReady(jobName string) (ready bool, found bool) {
	var jobDep batchv1.CronJob
	err := d.req.Get(&jobDep, d.app.Status.Namespace, jobName)
	if apierrors.IsNotFound(err) {
		return false, false
	}
	if err != nil {
		// if err just return it as not ready
		return false, true
	}

	if jobDep.Annotations[labels.AcornAppGeneration] != strconv.Itoa(int(d.app.Generation)) ||
		jobDep.Status.LastSuccessfulTime == nil {
		return false, true
	}

	return true, true
}

func (d *depCheckingResponse) isJobReady(jobName string) (ready bool, found bool) {
	var jobDep batchv1.Job
	err := d.req.Get(&jobDep, d.app.Status.Namespace, jobName)
	if apierrors.IsNotFound(err) {
		return false, false
	}
	if err != nil {
		// if err just return it as not ready
		return false, true
	}

	if jobDep.Annotations[labels.AcornAppGeneration] != strconv.Itoa(int(d.app.Generation)) ||
		jobDep.Status.Succeeded != 1 {
		return false, true
	}

	return true, true
}

func getDependencyAnnotations(app *v1.AppInstance, deps []v1.Dependency) map[string]string {
	result := map[string]string{}
	if app.Generation > 0 {
		result[labels.AcornAppGeneration] = strconv.Itoa(int(app.Generation))
	}
	if len(deps) > 0 {
		buf := &strings.Builder{}
		for _, dep := range deps {
			if dep.TargetName != "" {
				if buf.Len() > 0 {
					buf.WriteString(",")
				}
				buf.WriteString(dep.TargetName)
			}
		}
		result[labels.AcornDepNames] = buf.String()
	}
	return result
}

func (d *depCheckingResponse) isDepReady(depName string) (ready bool, found bool) {
	var depDep appsv1.Deployment
	err := d.req.Get(&depDep, d.app.Status.Namespace, depName)
	if apierrors.IsNotFound(err) {
		return false, false
	}
	if err != nil {
		// if err just return it as not ready
		return false, true
	}

	available := false
	for _, cond := range depDep.Status.Conditions {
		if cond.Type == "Available" && cond.Status == corev1.ConditionTrue {
			available = true
			break
		}
	}

	if !available {
		return false, true
	}

	if depDep.Annotations[labels.AcornAppGeneration] != strconv.Itoa(int(d.app.Generation)) ||
		depDep.Status.ObservedGeneration != depDep.Generation ||
		depDep.Status.Replicas != depDep.Status.ReadyReplicas ||
		depDep.Status.Replicas != depDep.Status.UpdatedReplicas {
		return false, true
	}

	reps := &appsv1.ReplicaSetList{}
	err = d.req.List(reps, &kclient.ListOptions{
		LabelSelector: klabels.SelectorFromSet(map[string]string{
			labels.AcornAppName:       d.app.Name,
			labels.AcornAppNamespace:  d.app.Namespace,
			labels.AcornContainerName: depName,
		}),
		Namespace: d.app.Status.Namespace,
	})
	if err != nil {
		return false, true
	}

	for _, rep := range reps.Items {
		if rep.Annotations[labels.AcornAppGeneration] == strconv.Itoa(int(d.app.Generation)) &&
			rep.Generation == rep.Status.ObservedGeneration &&
			rep.Status.Replicas == rep.Status.ReadyReplicas &&
			rep.Status.Replicas == rep.Status.AvailableReplicas {
			return true, true
		}
	}

	return false, true
}

type depCheck func(string) (bool, bool)

func (d *depCheckingResponse) checkDeps(deps []string) (string, bool) {
outer:
	for _, depName := range deps {
		for _, link := range d.app.Spec.Links {
			if link.Target == depName {
				continue outer
			}
		}
		for _, depCheck := range []depCheck{d.isDepReady, d.isJobReady, d.isCronJobReady, d.isServiceReady} {
			if ready, found := depCheck(depName); found && !ready {
				return depName, false
			} else if found && ready {
				continue outer
			}
		}
		return depName, false
	}

	return "", true
}
