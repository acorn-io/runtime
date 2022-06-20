package appdefinition

import (
	"bytes"
	"errors"

	cueerrors "cuelang.org/go/cue/errors"
	v1 "github.com/acorn-io/acorn/pkg/apis/acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/appdefinition"
	"github.com/acorn-io/acorn/pkg/condition"
	"github.com/acorn-io/baaah/pkg/router"
)

func ParseAppImage(req router.Request, resp router.Response) error {
	appInstance := req.Object.(*v1.AppInstance)
	status := condition.Setter(appInstance, resp, v1.AppInstanceConditionParsed)
	appImage := appInstance.Status.AppImage

	if appImage.Acornfile == "" {
		return nil
	}

	appDef, err := appdefinition.FromAppImage(&appImage)
	if err != nil {
		status.Error(err)
		return nil
	}

	appDef, _, err = appDef.WithDeployArgs(appInstance.Spec.DeployArgs, appInstance.Spec.Profiles)
	if err != nil {
		status.Error(err)
		return nil
	}

	appSpec, err := appDef.AppSpec()
	if err != nil {
		buf := &bytes.Buffer{}
		cueerrors.Print(buf, err, nil)
		status.Error(errors.New(buf.String()))
		return nil
	}

	appInstance.Status.AppSpec = *appSpec
	status.Success()
	return nil
}
