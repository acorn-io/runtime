package appstatus

import (
	"context"
	"fmt"
	"strings"

	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/condition"
	"github.com/acorn-io/baaah/pkg/router"
	"github.com/acorn-io/baaah/pkg/typed"
	"github.com/pkg/errors"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type appStatusRenderer struct {
	ctx context.Context
	c   kclient.Client
	app *v1.AppInstance
}

func PrepareStatus(req router.Request, resp router.Response) error {
	app := req.Object.(*v1.AppInstance)

	for name, status := range app.Status.AppStatus.Containers {
		status.ExpressionErrors = nil
		app.Status.AppStatus.Containers[name] = status
	}

	for name, status := range app.Status.AppStatus.Jobs {
		status.ExpressionErrors = nil
		app.Status.AppStatus.Jobs[name] = status
	}

	if app.Status.AppStatus.Containers == nil {
		app.Status.AppStatus.Containers = map[string]v1.ContainerStatus{}
	}

	if app.Status.AppStatus.Jobs == nil {
		app.Status.AppStatus.Jobs = map[string]v1.JobStatus{}
	}

	return nil
}

func SetStatus(req router.Request, resp router.Response) error {
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

	app.Status.AppStatus = status
	return nil
}

func formatMessage(name string, parts []string) string {
	if name == "" {
		if len(parts) == 1 {
			return parts[0]
		}
		return fmt.Sprintf("[%s]", strings.Join(parts, ","))
	}
	return fmt.Sprintf("%s: [%s]", name, strings.Join(parts, ","))
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
		} else if !status.Defined {
			transitioningMessages = append(transitioningMessages, fmt.Sprintf("%s: [pending create]", name))
		} else if !status.UpToDate {
			transitioningMessages = append(transitioningMessages, fmt.Sprintf("%s: [pending update]", name))
		} else if !status.Ready {
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
