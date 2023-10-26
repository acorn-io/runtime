package appdefinition

import (
	"fmt"

	"github.com/acorn-io/baaah/pkg/router"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/appdefinition"
	"github.com/acorn-io/runtime/pkg/condition"
	"github.com/acorn-io/runtime/pkg/controller/permissions"
	"github.com/acorn-io/runtime/pkg/services"
)

func ParseAppImage(req router.Request, resp router.Response) error {
	appInstance := req.Object.(*v1.AppInstance)
	status := condition.Setter(appInstance, resp, v1.AppInstanceConditionParsed)
	appImage := appInstance.Status.AppImage
	containers := appImage.ImageData.Containers

	if appImage.Acornfile == "" {
		return nil
	}

	if _, err := services.ValidateTargetServiceName(appInstance.Spec.Publish, containers); err != nil {
		status.Error(fmt.Errorf("invalid container name for publish: %s", err.Error()))
		return nil
	}

	appDef, err := appdefinition.FromAppImage(&appImage)
	if err != nil {
		status.Error(err)
		return nil
	}

	appDef = appDef.WithArgs(appInstance.Spec.DeployArgs.GetData(), appInstance.Spec.GetProfiles(appInstance.Status.GetDevMode()))

	appSpec, err := appDef.AppSpec()
	if err != nil {
		status.Error(err)
		return nil
	}

	// Migration for AppScopedPermissions
	if len(appInstance.Status.Staged.AppScopedPermissions) == 0 &&
		appInstance.Status.Staged.PermissionsObservedGeneration == appInstance.Generation &&
		len(appInstance.Status.Staged.ImagePermissionsDenied) == 0 {
		appInstance.Status.Staged.AppScopedPermissions = permissions.GetAppScopedPermissions(appInstance, appSpec)
	}

	appInstance.Status.AppSpec = *appSpec
	status.Success()
	return nil
}
