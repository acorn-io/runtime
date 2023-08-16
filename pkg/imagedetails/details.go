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
}

func ParseDetails(acornfile string, deployArgs map[string]any, profiles []string) (*Details, error) {
	result := &Details{
		DeployArgs: v1.NewGenericMap(deployArgs),
		Profiles:   profiles,
	}

	appDef, err := appdefinition.NewAppDefinition([]byte(acornfile))
	if err != nil {
		return nil, err
	}

	if len(deployArgs) > 0 || len(profiles) > 0 {
		appDef, deployArgs, err = appDef.WithArgs(deployArgs, profiles)
		if err != nil {
			return nil, err
		}
		result.DeployArgs = v1.NewGenericMap(deployArgs)
	}

	appSpec, err := appDef.AppSpec()
	if err != nil {
		return nil, err
	}

	paramSpec, err := appDef.Args()
	if err != nil {
		return nil, err
	}

	result.AppSpec = appSpec
	result.Params = paramSpec
	return result, nil
}
