package build

import "github.com/ibuildthecloud/herd/pkg/flagparams"

func ParseParams(file, cwd string, args []string) (map[string]interface{}, error) {
	appDefinition, err := ResolveAndParse(file, cwd)
	if err != nil {
		return nil, err
	}

	params, err := appDefinition.BuildParams()
	if err != nil {
		return nil, err
	}

	return flagparams.New(ResolveFile(file, cwd), params).Parse(args)
}
