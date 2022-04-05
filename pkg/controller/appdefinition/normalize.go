package appdefinition

import (
	"bytes"
	"errors"

	cueerrors "cuelang.org/go/cue/errors"
	"github.com/ibuildthecloud/baaah/pkg/router"
	v1 "github.com/ibuildthecloud/herd/pkg/apis/herd-project.io/v1"
	"github.com/ibuildthecloud/herd/pkg/appdefinition"
	"github.com/ibuildthecloud/herd/pkg/condition"
)

func ParseAppImage(req router.Request, resp router.Response) error {
	appInstance := req.Object.(*v1.AppInstance)
	status := condition.Setter(appInstance, resp, v1.AppInstanceConditionParsed)
	appImage := appInstance.Status.AppImage

	if appImage.Herdfile == "" {
		return nil
	}

	appDef, err := appdefinition.FromAppImage(&appImage)
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
