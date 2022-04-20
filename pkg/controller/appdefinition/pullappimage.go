package appdefinition

import (
	"github.com/ibuildthecloud/baaah/pkg/router"
	v1 "github.com/ibuildthecloud/herd/pkg/apis/herd-project.io/v1"
	"github.com/ibuildthecloud/herd/pkg/condition"
	"github.com/ibuildthecloud/herd/pkg/pull"
)

func PullAppImage(req router.Request, resp router.Response) error {
	appInstance := req.Object.(*v1.AppInstance)
	cond := condition.Setter(appInstance, resp, v1.AppInstanceConditionPulled)

	if appInstance.Spec.Image == appInstance.Status.AppImage.ID {
		cond.Success()
		return nil
	}

	appImage, err := pull.AppImage(req.Ctx, router.ToReader(req.Client), appInstance.Namespace, appInstance.Spec.Image, appInstance.Spec.ImagePullSecrets)
	if err != nil {
		cond.Error(err)
		return nil
	}

	appImage.ID = appInstance.Spec.Image
	appInstance.Status.AppImage = *appImage

	cond.Success()
	return nil
}
