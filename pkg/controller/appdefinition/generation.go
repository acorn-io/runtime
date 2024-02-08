package appdefinition

import (
	"github.com/acorn-io/baaah/pkg/router"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/autoupgrade"
)

func UpdateObservedFields(req router.Request, _ router.Response) error {
	app := req.Object.(*v1.AppInstance)
	app.Status.ObservedImageDigest = app.Status.AppImage.Digest
	app.Status.ObservedGeneration = app.Generation
	app.Status.ObservedAutoUpgrade = autoUpgradeEnabled(app.Spec)
	return nil
}

func autoUpgradeEnabled(appSpec v1.AppInstanceSpec) bool {
	_, enabled := autoupgrade.Mode(appSpec)
	return enabled
}
