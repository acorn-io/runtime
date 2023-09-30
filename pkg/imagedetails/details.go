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

func ParseDetails(acornfile string, acornfileV1 bool, deployArgs map[string]any, profiles []string) (*Details, error) {
	result := &Details{
		DeployArgs: v1.NewGenericMap(deployArgs),
		Profiles:   profiles,
	}

	var (
		appDef *appdefinition.AppDefinition
		err    error
	)
	if acornfileV1 {
		appDef, err = appdefinition.NewAppDefinition([]byte(acornfile))
		if err != nil {
			return nil, err
		}
	} else {
		appDef, err = appdefinition.NewLegacyAppDefinition([]byte(acornfile))
		if err != nil {
			return nil, err
		}
	}

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
