package appdefinition

import (
	"github.com/acorn-io/baaah/pkg/router"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/autoupgrade"
	"github.com/acorn-io/z"
)

func UpdateObservedFields(req router.Request, resp router.Response) error {
	app := req.Object.(*v1.AppInstance)
	app.Status.ObservedImageDigest = app.Status.AppImage.Digest
	app.Status.ObservedGeneration = app.Generation
	app.Status.ObservedAutoUpgrade = impliedAutoUpgrade(app.Spec)
	return nil
}

func impliedAutoUpgrade(appSpec v1.AppInstanceSpec) bool {
	au := appSpec.AutoUpgrade != nil && *appSpec.AutoUpgrade
	if !au && autoupgrade.ImpliedAutoUpgrade(
		appSpec.Image,
		appSpec.AutoUpgradeInterval,
		z.Dereference(appSpec.NotifyUpgrade),
	) {
		au = true
	}
	return au
}
