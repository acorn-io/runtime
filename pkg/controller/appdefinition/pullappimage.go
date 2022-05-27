package appdefinition

import (
	v1 "github.com/acorn-io/acorn/pkg/apis/acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/condition"
	"github.com/acorn-io/acorn/pkg/pull"
	"github.com/acorn-io/baaah/pkg/router"
)

func PullAppImage(req router.Request, resp router.Response) error {
	appInstance := req.Object.(*v1.AppInstance)
	cond := condition.Setter(appInstance, resp, v1.AppInstanceConditionPulled)

	if appInstance.Spec.Image == appInstance.Status.AppImage.ID {
		cond.Success()
		return nil
	}

	appImage, err := pull.AppImage(req.Ctx, req.Client, appInstance.Namespace, appInstance.Spec.Image)
	if err != nil {
		cond.Error(err)
		return nil
	}

	appImage.ID = appInstance.Spec.Image
	appInstance.Status.AppImage = *appImage

	cond.Success()
	return nil
}
