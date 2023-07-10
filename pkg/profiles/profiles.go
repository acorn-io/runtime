package profiles

import apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"

type profile string

const (
	Production profile = "production"
	Prod       profile = "prod"
	Default    profile = "default"
)

var profiles = map[profile]apiv1.Config{
	Production: productionProfile(),
	Default:    defaultProfile(),
}

func Get(p *string) apiv1.Config {
	if p == nil {
		return profiles[Default]
	}

	switch prof := profile(*p); prof {
	case Production, Prod:
		return profiles[Production]
	default:
		return profiles[Default]
	}
}
