package appdefinition

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/condition"
	"github.com/acorn-io/acorn/pkg/config"
	"github.com/acorn-io/acorn/pkg/labels"
	"github.com/acorn-io/baaah/pkg/merr"
	"github.com/acorn-io/baaah/pkg/router"
	"github.com/acorn-io/baaah/pkg/typed"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	klabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func ReadyStatus(req router.Request, resp router.Response) error {
	app := req.Object.(*v1.AppInstance)
	app.Status.Ready = false
	cond := condition.Setter(app, resp, v1.AppInstanceConditionReady)

	var errs []error
	for _, condition := range app.Status.Conditions {
		if condition.Type == v1.AppInstanceConditionReady {
			continue
		}

		if condition.Status != metav1.ConditionTrue {
			errs = append(errs, errors.New(condition.Message))
		}
	}

	if len(errs) > 0 {
		cond.Error(merr.NewErrors(errs...))
		return nil
	}

	cond.Success()
	app.Status.Ready = app.Spec.Image == app.Status.AppImage.ID &&
		app.Status.Condition(v1.AppInstanceConditionParsed).Success &&
		app.Status.Condition(v1.AppInstanceConditionContainers).Success &&
		app.Status.Condition(v1.AppInstanceConditionJobs).Success &&
		app.Status.Condition(v1.AppInstanceConditionAcorns).Success &&
		app.Status.Condition(v1.AppInstanceConditionSecrets).Success &&
		app.Status.Condition(v1.AppInstanceConditionPulled).Success &&
		app.Status.Condition(v1.AppInstanceConditionDefined).Success
	return nil
}

func AcornStatus(req router.Request, resp router.Response) error {
	app := req.Object.(*v1.AppInstance)
	cond := condition.Setter(app, resp, v1.AppInstanceConditionAcorns)
	app.Status.AcornStatus = map[string]v1.AcornStatus{}

	var (
		failed         bool
		failedName     string
		failedMessage  string
		waiting        bool
		waitingName    string
		waitingMessage string
	)

	for _, acornName := range typed.SortedKeys(app.Status.AppSpec.Acorns) {
		appInstance := &v1.AppInstance{}
		err := req.Get(appInstance, app.Status.Namespace, acornName)
		if apierrors.IsNotFound(err) {
			continue
		} else if err != nil {
			cond.Error(err)
			return nil
		}
		app.Status.AcornStatus[acornName] = v1.AcornStatus{
			ContainerStatus: appInstance.Status.ContainerStatus,
			JobsStatus:      appInstance.Status.JobsStatus,
			AcornStatus:     appInstance.Status.AcornStatus,
			Stopped:         appInstance.Status.Stopped,
			Ready:           appInstance.Status.Ready,
		}

		for _, condition := range appInstance.Status.Conditions {
			if condition.Error {
				failed = true
				failedName = acornName
				failedMessage = condition.Message
			} else if condition.Transitioning || !condition.Success {
				waiting = true
				waitingName = acornName
				waitingMessage = condition.Message
			}
		}
	}

	switch {
	case failed:
		cond.Error(fmt.Errorf("%s: failed [%s]", failedName, failedMessage))
	case waiting:
		cond.Unknown(fmt.Sprintf("%s: waiting [%s]", waitingName, waitingMessage))
	default:
		cond.Success()
	}

	resp.Objects(app)
	return nil
}

func JobStatus(req router.Request, resp router.Response) error {
	app := req.Object.(*v1.AppInstance)
	cond := condition.Setter(app, resp, v1.AppInstanceConditionJobs)
	jobs := &batchv1.JobList{}

	err := req.List(jobs, &kclient.ListOptions{
		Namespace: app.Status.Namespace,
		LabelSelector: klabels.SelectorFromSet(map[string]string{
			labels.AcornManaged: "true",
			labels.AcornAppName: app.Name,
		}),
	})
	if err != nil {
		return err
	}

	var (
		running     bool
		runningName string
		failed      bool
		failedName  string
	)

	sort.Slice(jobs.Items, func(i, j int) bool {
		return jobs.Items[i].Name < jobs.Items[j].Name
	})
	for _, job := range jobs.Items {
		if app.Status.JobsStatus == nil {
			app.Status.JobsStatus = map[string]v1.JobStatus{}
		}

		_, messages, err := podsStatus(req, app.Status.Namespace, klabels.SelectorFromSet(map[string]string{
			labels.AcornManaged: "true",
			labels.AcornJobName: job.Name,
		}))
		if err != nil {
			return err
		}

		jobStatus := v1.JobStatus{
			Message: strings.Join(messages, "; "),
		}
		if job.Status.Active > 0 {
			jobStatus.Running = true
			running = true
			runningName = job.Name
		}
		if job.Status.Failed > 0 {
			jobStatus.Failed = true
			failed = true
			failedName = job.Name
		}
		if job.Status.Succeeded > 0 {
			jobStatus.Succeed = true
		}
		app.Status.JobsStatus[job.Name] = jobStatus
	}

	switch {
	case failed:
		cond.Error(fmt.Errorf("%s: failed [%s]", failedName, app.Status.JobsStatus[failedName].Message))
	case running:
		cond.Unknown(fmt.Sprintf("%s: running [%s]", runningName, app.Status.JobsStatus[runningName].Message))
	default:
		cond.Success()
	}

	resp.Objects(app)
	return nil
}

func podsStatus(req router.Request, namespace string, sel klabels.Selector) (bool, []string, error) {
	var (
		isTransition bool
		message      []string
		pods         = &corev1.PodList{}
	)
	err := req.List(pods, &kclient.ListOptions{
		Namespace:     namespace,
		LabelSelector: sel,
	})
	if err != nil {
		return false, nil, err
	}

	for _, pod := range pods.Items {
		for _, cond := range pod.Status.Conditions {
			if cond.Type == corev1.PodScheduled {
				if cond.Status != corev1.ConditionTrue {
					isTransition = true
					message = append(message, podName(&pod)+" is not scheduled to a node")
				}
			}
		}

		msg, transition := containerMessages(&pod, pod.Status.InitContainerStatuses)
		message = append(message, msg...)
		if transition {
			isTransition = true
		}

		msg, transition = containerMessages(&pod, pod.Status.ContainerStatuses)
		message = append(message, msg...)
		if transition {
			isTransition = true
		}
	}

	return isTransition, message, nil
}

func AppStatus(req router.Request, resp router.Response) error {
	var (
		app  = req.Object.(*v1.AppInstance)
		cond = condition.Setter(app, resp, v1.AppInstanceConditionContainers)
		deps = &appsv1.DeploymentList{}
	)

	cfg, err := config.Get(req.Ctx, req.Client)
	if err != nil {
		return err
	}

	err = req.List(deps, &kclient.ListOptions{
		Namespace: app.Status.Namespace,
		LabelSelector: klabels.SelectorFromSet(map[string]string{
			labels.AcornManaged: "true",
			labels.AcornAppName: app.Name,
		}),
	})
	if err != nil {
		return err
	}

	notJob, err := klabels.NewRequirement(labels.AcornContainerName, selection.Exists, nil)
	if err != nil {
		return err
	}

	isTransition, message, err := podsStatus(req, app.Status.Namespace, klabels.SelectorFromSet(map[string]string{
		labels.AcornManaged: "true",
		labels.AcornAppName: app.Name,
	}).Add(*notJob))
	if err != nil {
		return err
	}

	container := map[string]v1.ContainerStatus{}
	for _, dep := range deps.Items {
		status := container[dep.Labels[labels.AcornContainerName]]
		status.Ready = dep.Status.ReadyReplicas
		status.ReadyDesired = dep.Status.Replicas
		status.UpToDate = dep.Status.UpdatedReplicas
		container[dep.Labels[labels.AcornContainerName]] = status

		if status.Ready != status.ReadyDesired {
			isTransition = true
			message = append(message, dep.Labels[labels.AcornAppName]+" is not ready")
		}
	}
	app.Status.ContainerStatus = container
	app.Status.Columns.Endpoints = endpoints(cfg, app)

	if isTransition {
		sort.Strings(message)
		cond.Unknown(strings.TrimSpace(strings.Join(message, "; ")))
	} else {
		cond.Success()
	}

	if !isTransition && app.Spec.Stop != nil && *app.Spec.Stop {
		allZero := true
		for _, v := range app.Status.ContainerStatus {
			if v.ReadyDesired != 0 {
				allZero = false
				break
			}
		}
		if allZero {
			app.Status.Stopped = true
		}
	} else {
		app.Status.Stopped = false
	}

	resp.Objects(app)
	return nil
}

func containerMessages(pod *corev1.Pod, status []corev1.ContainerStatus) (message []string, isTransition bool) {
	for _, container := range status {
		if container.State.Waiting != nil && container.State.Waiting.Reason != "" {
			isTransition = true
			if container.State.Waiting.Message == "" {
				message = append(message, podName(pod)+" "+
					container.State.Waiting.Reason)
			} else {
				message = append(message, podName(pod)+" "+
					container.State.Waiting.Reason+": "+container.State.Waiting.Message)
			}
		}
		if container.State.Terminated != nil && container.State.Terminated.ExitCode > 0 {
			isTransition = true
			message = append(message, podName(pod)+" "+container.State.Terminated.Reason+": Exit Code "+
				strconv.Itoa(int(container.State.Terminated.ExitCode)))
		}
	}
	return
}

func podName(pod *corev1.Pod) string {
	jobName := pod.Labels[labels.AcornJobName]
	if jobName != "" {
		return jobName
	}
	return pod.Labels[labels.AcornContainerName]
}

func endpoints(cfg *apiv1.Config, app *v1.AppInstance) string {
	endpointTarget := map[string][]v1.Endpoint{}
	for _, endpoint := range app.Status.Endpoints {
		target := fmt.Sprintf("%s:%d", endpoint.Target, endpoint.TargetPort)
		endpointTarget[target] = append(endpointTarget[target], endpoint)
	}

	var endpointStrings []string

	for _, entry := range typed.Sorted(endpointTarget) {
		var (
			target, endpoints = entry.Key, entry.Value
			publicStrings     []string
		)

		for _, endpoint := range endpoints {
			buf := &strings.Builder{}
			switch endpoint.Protocol {
			case v1.ProtocolHTTP:
				if !strings.HasPrefix(endpoint.Address, "http") {
					if *cfg.TLSEnabled {
						buf.WriteString("https://")
					} else {
						buf.WriteString("http://")
					}
				}
			default:
				buf.WriteString(strings.ToLower(string(endpoint.Protocol)))
				buf.WriteString("://")
			}

			buf.WriteString(endpoint.Address)
			publicStrings = append(publicStrings, buf.String())
		}

		endpointStrings = append(endpointStrings,
			fmt.Sprintf("%s => %s", strings.Join(publicStrings, "|"), target))
	}

	return strings.Join(endpointStrings, ", ")
}
