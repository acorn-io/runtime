package appstatus

import (
	"context"
	"fmt"
	"strings"

	"github.com/acorn-io/baaah/pkg/router"
	"github.com/acorn-io/baaah/pkg/typed"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	client2 "github.com/acorn-io/runtime/pkg/client"
	"github.com/acorn-io/runtime/pkg/condition"
	"github.com/acorn-io/runtime/pkg/jobs"
	"github.com/pkg/errors"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type appStatusRenderer struct {
	ctx context.Context
	c   kclient.Client
	app *v1.AppInstance
}

func PrepareStatus(req router.Request, _ router.Response) error {
	app := req.Object.(*v1.AppInstance)

	for name, status := range app.Status.AppStatus.Containers {
		// If the app is being updated, then set the containers to not ready so that the controller will run them again and the
		// dependency status will be set correctly.
		status.Ready = status.Ready && app.Generation == app.Status.ObservedGeneration
		status.ExpressionErrors = nil
		status.Dependencies = nil
		app.Status.AppStatus.Containers[name] = status
	}

	for name, status := range app.Status.AppStatus.Jobs {
		status.ExpressionErrors = nil
		status.Dependencies = nil
		if app.Generation != app.Status.ObservedGeneration && jobs.ShouldRun(name, app) {
			// If a job is going to run again, then set its status to not ready so that the controller will run it again and the
			// dependency status will be set correctly.
			status.Ready = false
		}
		app.Status.AppStatus.Jobs[name] = status
	}

	for name, status := range app.Status.AppStatus.Services {
		status.ExpressionErrors = nil
		status.MissingConsumerPermissions = nil
		app.Status.AppStatus.Services[name] = status
	}

	for name, status := range app.Status.AppStatus.Secrets {
		status.LookupErrors = nil
		status.LookupTransitioning = nil
		app.Status.AppStatus.Secrets[name] = status
	}

	if app.Status.AppStatus.Containers == nil {
		app.Status.AppStatus.Containers = map[string]v1.ContainerStatus{}
	}

	if app.Status.AppStatus.Jobs == nil {
		app.Status.AppStatus.Jobs = map[string]v1.JobStatus{}
	}

	if app.Status.AppStatus.Acorns == nil {
		app.Status.AppStatus.Acorns = map[string]v1.AcornStatus{}
	}

	if app.Status.AppStatus.Services == nil {
		app.Status.AppStatus.Services = map[string]v1.ServiceStatus{}
	}

	if app.Status.AppStatus.Secrets == nil {
		app.Status.AppStatus.Secrets = map[string]v1.SecretStatus{}
	}

	return nil
}

func SetStatus(req router.Request, _ router.Response) error {
	app := req.Object.(*v1.AppInstance)
	status, err := Get(req.Ctx, req.Client, app)
	if err != nil {
		return err
	}

	setCondition(app, v1.AppInstanceConditionContainers, status.Containers)
	setCondition(app, v1.AppInstanceConditionJobs, status.Jobs)
	setCondition(app, v1.AppInstanceConditionVolumes, status.Volumes)
	setCondition(app, v1.AppInstanceConditionServices, status.Services)
	setCondition(app, v1.AppInstanceConditionSecrets, status.Secrets)
	setCondition(app, v1.AppInstanceConditionAcorns, status.Acorns)

	setPermissionCondition(app)

	app.Status.AppStatus = status
	return nil
}

func setPermissionCondition(app *v1.AppInstance) {
	cond := condition.ForName(app, v1.AppInstanceConditionPermissions)
	if len(app.Status.Staged.PermissionsMissing) > 0 {
		cond.Error(fmt.Errorf("can not run new image due to missing permissions: %w", &client2.ErrRulesNeeded{
			Permissions: app.Status.Staged.PermissionsMissing,
		}))
	} else {
		cond.Success()
	}
}

func formatMessage(name string, parts []string) string {
	if name == "" {
		if len(parts) == 1 {
			return parts[0]
		}
		return fmt.Sprintf("[%s]", strings.Join(parts, ", "))
	}
	return fmt.Sprintf("%s: [%s]", name, strings.Join(parts, ", "))
}

type commonStatusGetter interface {
	GetCommonStatus() v1.CommonStatus
}

func setCondition[T commonStatusGetter](obj kclient.Object, conditionName string, status map[string]T) {
	var (
		errorMessages         []string
		transitioningMessages []string
	)
	for _, entry := range typed.Sorted(status) {
		name, status := entry.Key, entry.Value.GetCommonStatus()
		if len(status.ErrorMessages) > 0 {
			errorMessages = append(errorMessages, formatMessage(name, status.ErrorMessages))
		} else if len(status.TransitioningMessages) > 0 {
			transitioningMessages = append(transitioningMessages, formatMessage(name, status.TransitioningMessages))
		} else if !status.Defined && obj.GetDeletionTimestamp().IsZero() {
			transitioningMessages = append(transitioningMessages, fmt.Sprintf("%s: [pending create]", name))
		} else if !status.UpToDate && obj.GetDeletionTimestamp().IsZero() {
			transitioningMessages = append(transitioningMessages, fmt.Sprintf("%s: [pending update]", name))
		} else if !status.Ready && obj.GetDeletionTimestamp().IsZero() {
			transitioningMessages = append(transitioningMessages, fmt.Sprintf("%s: [is not ready]", name))
		}
	}

	cond := condition.ForName(obj, conditionName)
	if len(errorMessages) > 0 {
		cond.Error(errors.New(formatMessage("", append(errorMessages, transitioningMessages...))))
	} else if len(transitioningMessages) > 0 {
		cond.Unknown(formatMessage("", append(errorMessages, transitioningMessages...)))
	} else {
		cond.Success()
	}
}

func Get(ctx context.Context, c kclient.Client, app *v1.AppInstance) (v1.AppStatus, error) {
	render := appStatusRenderer{
		ctx: ctx,
		c:   c,
		app: app.DeepCopy(),
	}

	if err := render.readContainers(); err != nil {
		return v1.AppStatus{}, err
	}

	if err := render.readJobs(); err != nil {
		return v1.AppStatus{}, err
	}

	if err := render.readVolumes(); err != nil {
		return v1.AppStatus{}, err
	}

	if err := render.readServices(); err != nil {
		return v1.AppStatus{}, err
	}

	if err := render.readSecrets(); err != nil {
		return v1.AppStatus{}, err
	}

	if err := render.readAcorns(); err != nil {
		return v1.AppStatus{}, err
	}

	if err := render.readRouter(); err != nil {
		return v1.AppStatus{}, err
	}

	if err := render.readEndpoints(); err != nil {
		return v1.AppStatus{}, err
	}

	return render.app.Status.AppStatus, nil
}
