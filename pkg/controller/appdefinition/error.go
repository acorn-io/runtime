package appdefinition

import (
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/condition"
	"github.com/acorn-io/baaah/pkg/router"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

func ClearError(req router.Request, resp router.Response) error {
	condition.Setter(req.Object.(*v1.AppInstance), resp, v1.AppInstanceConditionController).Success()
	return nil
}

func OnError(req router.Request, resp router.Response, err error) error {
	if apierrors.IsConflict(err) {
		return err
	}
	if app, ok := req.Object.(*v1.AppInstance); ok {
		var oldApp v1.AppInstance
		updateErr := req.Get(&oldApp, app.Namespace, app.Name)
		condition.Setter(app, resp, v1.AppInstanceConditionController).Error(err)
		if router.StatusChanged(&oldApp, app) {
			updateErr = req.Client.Status().Update(req.Ctx, app)
		}
		if updateErr == nil && apierrors.IsInvalid(err) {
			return nil
		}
	}
	return err
}
