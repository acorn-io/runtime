package imagedetails

import (
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/appdefinition"
)

type Details struct {
	DeployArgs *v1.GenericMap `json:"deployArgs,omitempty"`
	Profiles   []string       `json:"profiles,omitempty"`
	AppSpec    *v1.AppSpec    `json:"appSpec,omitempty"`
	Params     *v1.ParamSpec  `json:"params,omitempty"`
	Image      *v1.ImagesData `json:"image,omitempty"`
}

func ParseDetails(appImage *v1.AppImage, deployArgs map[string]any, profiles []string) (*Details, error) {
	result := &Details{
		DeployArgs: v1.NewGenericMap(deployArgs),
		Profiles:   profiles,
	}

	var (
		appDef *appdefinition.AppDefinition
		err    error
	)
	appDef, err = appdefinition.FromAppImage(appImage)
	if err != nil {
		return nil, err
	}

	appDef, result.Image = appDef.ClearImageData()

	if len(deployArgs) > 0 || len(profiles) > 0 {
		appDef = appDef.WithArgs(deployArgs, profiles)
		result.DeployArgs = v1.NewGenericMap(deployArgs)
	}

	appSpec, err := appDef.AppSpec()
	if err != nil {
		return nil, err
	}

	paramSpec, err := appDef.ToParamSpec()
	if err != nil {
		return nil, err
	}

	result.AppSpec = appSpec
	result.Params = paramSpec
	return result, nil
}
