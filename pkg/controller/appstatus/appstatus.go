package appstatus

import (
	"context"
	"crypto/sha256"
	"encoding/json"
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

// resetHandlerControlledFields will set fields to empty values because it is expected that handlers will append values
// to them. If the field is not reset it will grow indefinitely in a tight loop. This behavior does not
// apply to most status field built in AppStatus as they are fully calculated and reset on each run. But these
// fields are specifically fields with aggregated values.
// Additionally, some corner cases are handled in this code where fields need to be initialized in a special way
func resetHandlerControlledFields(app *v1.AppInstance) {
	appUpToDate := app.Generation == app.Status.ObservedGeneration && app.Status.AppImage.Digest == app.Status.ObservedImageDigest
	for name, status := range app.Status.AppStatus.Containers {
		// If the app is being updated, then set the containers to not ready so that the controller will run them again and the
		// dependency status will be set correctly.
		status.Ready = status.Ready && appUpToDate
		status.ExpressionErrors = nil
		status.Dependencies = nil
		app.Status.AppStatus.Containers[name] = status
	}

	for name, status := range app.Status.AppStatus.Jobs {
		status.ExpressionErrors = nil
		status.Dependencies = nil
		if !appUpToDate && jobs.ShouldRun(name, app) {
			// If a job is going to run again, then set its status to not ready so that the controller will run it again and the
			// dependency status will be set correctly.
			status.Ready = false
		}
		app.Status.AppStatus.Jobs[name] = status
	}

	for name, status := range app.Status.AppStatus.Services {
		status.Ready = status.Ready && appUpToDate
		status.ExpressionErrors = nil
		status.MissingConsumerPermissions = nil
		app.Status.AppStatus.Services[name] = status
	}

	for name, status := range app.Status.AppStatus.Secrets {
		status.Ready = status.Ready && appUpToDate
		status.Missing = false
		status.LookupErrors = nil
		status.LookupTransitioning = nil
		app.Status.AppStatus.Secrets[name] = status
	}

	for name, status := range app.Status.AppStatus.Routers {
		status.Ready = status.Ready && appUpToDate
		status.MissingTargets = nil
		app.Status.AppStatus.Routers[name] = status
	}
}

func PrepareStatus(req router.Request, _ router.Response) error {
	app := req.Object.(*v1.AppInstance)

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

	if app.Status.AppStatus.Routers == nil {
		app.Status.AppStatus.Routers = map[string]v1.RouterStatus{}
	}

	resetHandlerControlledFields(app)

	return nil
}

func GetStatus(req router.Request, _ router.Response) error {
	app := req.Object.(*v1.AppInstance)
	status, err := Get(req.Ctx, req.Client, app)
	if err != nil {
		return err
	}

	app.Status.AppStatus = status
	return nil
}

func SetStatus(req router.Request, _ router.Response) error {
	app := req.Object.(*v1.AppInstance)
	setMessages(req.Ctx, req.Client, app)

	status := app.Status.AppStatus

	setCondition(app, v1.AppInstanceConditionContainers, status.Containers)
	setCondition(app, v1.AppInstanceConditionJobs, status.Jobs)
	setCondition(app, v1.AppInstanceConditionVolumes, status.Volumes)
	setCondition(app, v1.AppInstanceConditionServices, status.Services)
	setCondition(app, v1.AppInstanceConditionSecrets, status.Secrets)
	setCondition(app, v1.AppInstanceConditionAcorns, status.Acorns)
	setCondition(app, v1.AppInstanceConditionRouters, status.Routers)

	setPermissionCondition(app)
	return nil
}

func setPermissionCondition(app *v1.AppInstance) {
	cond := condition.ForName(app, v1.AppInstanceConditionPermissions)

	if len(app.Status.Staged.PermissionsMissing) > 0 {
		cond.Error(fmt.Errorf("cannot run new image due to missing permissions: %w", &client2.ErrRulesNeeded{
			Permissions: app.Status.Staged.PermissionsMissing,
		}))
	} else if len(app.Status.Staged.ImagePermissionsDenied) > 0 {
		cond.Error(fmt.Errorf("cannot run new image due to denied permissions: %w", &client2.ErrRulesNeeded{
			Permissions: app.Status.Staged.ImagePermissionsDenied,
		}))
	} else {
		cond.Success()
	}

	cond = condition.ForName(app, v1.AppInstanceConditionConsumerPermissions)
	if len(app.Status.DeniedConsumerPermissions) > 0 {
		cond.Error(fmt.Errorf("cannot run current image due to unauthorized permissions given to it by consumed services: %w", &client2.ErrRulesNeeded{
			Permissions: app.Status.DeniedConsumerPermissions,
		}))
	} else {
		cond.Success()
	}
}

func formatMessage(name string, parts []string, sep string) string {
	if name == "" {
		if len(parts) == 1 {
			return parts[0]
		}
		return strings.Join(parts, sep)
	}
	return fmt.Sprintf("%s: %s", name, strings.Join(parts, sep))
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
		name = fmt.Sprintf("(%s: %s)", strings.TrimSuffix(conditionName, "s"), name)
		if len(status.ErrorMessages) > 0 {
			errorMessages = append(errorMessages, formatMessage(name, status.ErrorMessages, ", "))
		} else if len(status.TransitioningMessages) > 0 {
			transitioningMessages = append(transitioningMessages, formatMessage(name, status.TransitioningMessages, ", "))
		} else if !status.Defined {
			transitioningMessages = append(transitioningMessages, formatMessage(name, []string{"pending"}, ", "))
		} else if !status.UpToDate {
			transitioningMessages = append(transitioningMessages, formatMessage(name, []string{"updating"}, ", "))
		} else if !status.Ready {
			transitioningMessages = append(transitioningMessages, formatMessage(name, []string{"not ready"}, ", "))
		}
	}

	cond := condition.ForName(obj, conditionName)
	if len(errorMessages) > 0 {
		cond.Error(errors.New(formatMessage("", append(errorMessages, transitioningMessages...), "; ")))
	} else if len(transitioningMessages) > 0 {
		cond.Unknown(formatMessage("", append(errorMessages, transitioningMessages...), "; "))
	} else {
		cond.Success()
	}
}

func setMessages(ctx context.Context, c kclient.Client, app *v1.AppInstance) {
	setContainerMessages(app)
	setJobMessages(app)
	setVolumeMessages(app)
	setServiceMessages(app)
	setSecretMessages(ctx, c, app)
	setAcornMessages(app)
	setRouterMessages(app)
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

	if err := render.readRouters(); err != nil {
		return v1.AppStatus{}, err
	}

	if err := render.readEndpoints(); err != nil {
		return v1.AppStatus{}, err
	}

	return render.app.Status.AppStatus, nil
}

func configHash(c any) (string, error) {
	b, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", sha256.Sum256(b)), nil
}
