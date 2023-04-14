package appdefinition

import (
	"context"
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
	"github.com/acorn-io/acorn/pkg/ports"
	"github.com/acorn-io/acorn/pkg/volume"
	"github.com/acorn-io/baaah/pkg/merr"
	"github.com/acorn-io/baaah/pkg/router"
	"github.com/acorn-io/baaah/pkg/typed"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	klabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/utils/strings/slices"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func CheckStatus(h router.Handler) router.Handler {
	return router.HandlerFunc(func(req router.Request, resp router.Response) error {
		appInstance := req.Object.(*v1.AppInstance)
		conditionsToCheck := []string{
			v1.AppInstanceConditionDefaults,
			v1.AppInstanceConditionScheduling,
		}

		for _, cond := range conditionsToCheck {
			if !appInstance.Status.Condition(cond).Success {
				resp.DisablePrune()
				return nil
			}
		}

		return h.Handle(req, resp)
	})
}

func ReadyStatus(req router.Request, resp router.Response) error {
	app := req.Object.(*v1.AppInstance)
	app.Status.Ready = false
	cond := condition.Setter(app, resp, v1.AppInstanceConditionReady)

	var (
		errs          []error
		transitioning = sets.NewString()
	)
	for _, condition := range app.Status.Conditions {
		if condition.Type == v1.AppInstanceConditionReady {
			continue
		}

		if condition.Status == metav1.ConditionFalse {
			errs = append(errs, errors.New(condition.Message))
		} else if condition.Status == metav1.ConditionUnknown && condition.Message != "" {
			transitioning.Insert(condition.Message)
		}
	}

	if len(errs) > 0 {
		cond.Error(merr.NewErrors(errs...))
		return nil
	}

	if transitioning.Len() > 0 {
		cond.Unknown(strings.Join(transitioning.List(), ", "))
		return nil
	}

	ready := true
	for _, v := range app.Status.ContainerStatus {
		if !v.Created || (v.Ready < v.ReadyDesired) {
			ready = false
		}
	}
	for _, v := range app.Status.JobsStatus {
		if !v.Succeed {
			ready = false
		}
	}
	for _, v := range app.Status.AcornStatus {
		if !v.Ready {
			ready = false
		}
	}

	cond.Success()
	app.Status.Ready = ready && app.Status.AppImage.Name != "" &&
		app.Generation == app.Status.ObservedGeneration &&
		app.Status.Condition(v1.AppInstanceConditionParsed).Success &&
		app.Status.Condition(v1.AppInstanceConditionContainers).Success &&
		app.Status.Condition(v1.AppInstanceConditionJobs).Success &&
		app.Status.Condition(v1.AppInstanceConditionAcorns).Success &&
		app.Status.Condition(v1.AppInstanceConditionSecrets).Success &&
		app.Status.Condition(v1.AppInstanceConditionPulled).Success &&
		app.Status.Condition(v1.AppInstanceConditionController).Success &&
		app.Status.Condition(v1.AppInstanceConditionDefined).Success &&
		app.Status.Condition(v1.AppInstanceConditionVolumes).Success
	return nil
}

func AcornStatus(req router.Request, resp router.Response) error {
	app := req.Object.(*v1.AppInstance)
	cond := condition.Setter(app, resp, v1.AppInstanceConditionAcorns)
	app.Status.AcornStatus = map[string]v1.AcornStatus{}

	for acornName := range app.Status.AppSpec.Acorns {
		app.Status.AcornStatus[acornName] = v1.AcornStatus{}
	}

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

	app.Status.JobsStatus = map[string]v1.JobStatus{}
	for jobName := range app.Status.AppSpec.Jobs {
		app.Status.JobsStatus[jobName] = v1.JobStatus{}
	}

	var (
		running     bool
		runningName string
		failed      bool
		failedName  string
	)

	sort.Slice(jobs.Items, func(i, j int) bool {
		return jobs.Items[i].CreationTimestamp.Before(&jobs.Items[j].CreationTimestamp)
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

		messageSet := sets.NewString()
		for _, message := range messages {
			messageSet.Insert(message...)
		}
		jobStatus := v1.JobStatus{
			Message: strings.Join(messageSet.List(), "; "),
		}
		if job.Status.Active > 0 {
			jobStatus.Running = true
			running = true
			runningName = job.Name
		}
		if job.Status.Succeeded > 0 {
			jobStatus.Succeed = true
		} else if job.Status.Failed > 0 {
			jobStatus.Failed = true
			failed = true
			failedName = job.Name
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

func VolumeStatus(req router.Request, resp router.Response) error {
	var (
		app  = req.Object.(*v1.AppInstance)
		cond = condition.Setter(app, resp, v1.AppInstanceConditionVolumes)
		pvcs = new(corev1.PersistentVolumeClaimList)

		messages, errMessages []string
		err                   error
	)

	defer func() {
		if err != nil {
			cond.Error(err)
			return
		}
		if len(errMessages) > 0 {
			cond.Error(fmt.Errorf(strings.Join(errMessages, "; ")))
		} else if len(messages) > 0 {
			cond.Unknown(strings.Join(messages, "; "))
		} else {
			cond.Success()
		}
	}()

	if err = req.List(pvcs, &kclient.ListOptions{
		Namespace: app.Status.Namespace,
		LabelSelector: klabels.SelectorFromSet(map[string]string{
			labels.AcornManaged: "true",
			labels.AcornAppName: app.Name,
		}),
	}); err != nil {
		return err
	} else if len(pvcs.Items) == 0 {
		return nil
	}

	storageClassNames, err := volume.GetVolumeClassNames(req.Ctx, req.Client, app.Namespace, true)
	if err != nil {
		return err
	}

	sort.Slice(pvcs.Items, func(i, j int) bool {
		return pvcs.Items[i].CreationTimestamp.Before(&pvcs.Items[j].CreationTimestamp)
	})

	for _, pvc := range pvcs.Items {
		if pvc.Annotations[labels.AcornAppGeneration] != strconv.Itoa(int(app.Generation)) {
			messages = append(messages, fmt.Sprintf("volume %s is not ready", pvc.Name))
		}

		switch pvc.Status.Phase {
		case corev1.ClaimBound:
			// No message if the PVC is in phase bound.
		default:
			if pvc.Spec.StorageClassName != nil && *pvc.Spec.StorageClassName != "" && !slices.Contains(storageClassNames, *pvc.Spec.StorageClassName) {
				errMessages = append(errMessages, fmt.Sprintf("volume class %s for volume %s doesn't exist", *pvc.Spec.StorageClassName, pvc.Labels[labels.AcornVolumeName]))
			}
			messages = append(messages, fmt.Sprintf("waiting for volume %s to provision and bind", pvc.Labels[labels.AcornVolumeName]))
		}
	}

	resp.Objects(app)
	return nil
}

func podsStatus(req router.Request, namespace string, sel klabels.Selector) (bool, map[string][]string, error) {
	var (
		isTransition bool
		messages     = map[string][]string{}
		pods         = &corev1.PodList{}
	)
	err := req.List(pods, &kclient.ListOptions{
		Namespace:     namespace,
		LabelSelector: sel,
	})
	if err != nil {
		return false, nil, err
	}

	sort.Slice(pods.Items, func(i, j int) bool {
		return pods.Items[i].CreationTimestamp.Before(&pods.Items[j].CreationTimestamp)
	})

	for _, pod := range pods.Items {
		var message []string
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
		messages[pod.Labels[labels.AcornContainerName]] = message
	}

	return isTransition, messages, nil
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

	isTransition, podMessages, err := podsStatus(req, app.Status.Namespace, klabels.SelectorFromSet(map[string]string{
		labels.AcornManaged: "true",
		labels.AcornAppName: app.Name,
	}).Add(*notJob))
	if err != nil {
		return err
	}

	var messages []string

	container := map[string]v1.ContainerStatus{}
	for dep := range app.Status.AppSpec.Containers {
		container[dep] = v1.ContainerStatus{
			Created: ports.IsLinked(app, dep),
		}
	}

	for _, dep := range deps.Items {
		containerName := dep.Labels[labels.AcornContainerName]
		if containerName == "" {
			continue
		}

		status := container[containerName]
		status.Ready = dep.Status.ReadyReplicas
		status.ReadyDesired = dep.Status.Replicas
		status.UpToDate = dep.Status.UpdatedReplicas
		status.Created = true
		container[containerName] = status

		if podMessage := podMessages[containerName]; len(podMessage) > 0 {
			messages = append(messages, podMessage...)
		} else if dep.Annotations[labels.AcornAppGeneration] != strconv.Itoa(int(app.Generation)) {
			isTransition = true
			messages = append(messages, containerName+" pending update")
		} else if status.Ready != status.ReadyDesired {
			isTransition = true
			messages = append(messages, containerName+" is not ready")
		}
	}

	for name, status := range container {
		if !status.Created {
			messages = append(messages, name+" pending create")
		}
	}
	app.Status.ContainerStatus = container
	app.Status.Columns.Endpoints, err = endpoints(req, cfg, app)
	if err != nil {
		return err
	}

	if isTransition {
		// dedup, sort
		messages := sets.NewString(messages...).List()
		cond.Unknown(strings.TrimSpace(strings.Join(messages, "; ")))
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

func IngressTLSHosts(ctx context.Context, client kclient.Client, app *v1.AppInstance) (map[string]interface{}, error) {
	ingresses := &networkingv1.IngressList{}
	err := client.List(ctx, ingresses, &kclient.ListOptions{
		Namespace: app.Status.Namespace,
		LabelSelector: klabels.SelectorFromSet(map[string]string{
			labels.AcornManaged: "true",
			labels.AcornAppName: app.Name,
		}),
	})
	if err != nil {
		return nil, err
	}

	ingressTLSHosts := map[string]interface{}{}
	for _, ingress := range ingresses.Items {
		if ingress.Spec.TLS != nil {
			for _, tls := range ingress.Spec.TLS {
				for _, host := range tls.Hosts {
					ingressTLSHosts[host] = nil
				}
			}
		}
	}

	return ingressTLSHosts, nil
}

func endpoints(req router.Request, cfg *apiv1.Config, app *v1.AppInstance) (string, error) {
	endpointTarget := map[string][]v1.Endpoint{}
	for _, endpoint := range app.Status.Endpoints {
		target := fmt.Sprintf("%s:%d", endpoint.Target, endpoint.TargetPort)
		endpointTarget[target] = append(endpointTarget[target], endpoint)
	}

	ingressTLSHosts, err := IngressTLSHosts(req.Ctx, req.Client, app)
	if err != nil {
		return "", err
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
					var host string
					a, b, ok := strings.Cut(endpoint.Address, "://")
					if ok {
						host, _, _ = strings.Cut(b, ":")
					} else {
						host, _, _ = strings.Cut(a, ":")
					}
					if _, ok := ingressTLSHosts[host]; ok {
						buf.WriteString("https://")
					} else {
						buf.WriteString("http://")
					}
				}
			default:
				buf.WriteString(strings.ToLower(string(endpoint.Protocol)))
				buf.WriteString("://")
			}

			if endpoint.Pending {
				if endpoint.Protocol == "http" {
					buf.WriteString("<Pending Ingress>")
				} else {
					buf.WriteString("<Pending Load Balancer>")
				}
			} else {
				buf.WriteString(endpoint.Address)
			}
			publicStrings = append(publicStrings, buf.String())
		}

		endpointStrings = append(endpointStrings,
			fmt.Sprintf("%s => %s", strings.Join(publicStrings, " | "), target))
	}

	return strings.Join(endpointStrings, ", "), nil
}
