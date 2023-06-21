package appdefinition

import (
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
)

func (a *AppDefinition) Args() (*v1.ParamSpec, error) {
	args, err := a.newDecoder().Args()
	if err != nil {
		return nil, err
	}
	result := &v1.ParamSpec{
		Params: nil,
	}
	for _, param := range args.Params {
		result.Params = append(result.Params, (v1.Param)(param))
	}
	for _, profile := range args.Profiles {
		result.Profiles = append(result.Profiles, (v1.Profile)(profile))
	}

	return result, nil
}
