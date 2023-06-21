package registry

import (
	"github.com/acorn-io/runtime/pkg/server/registry/apigroups/acorn"
	"github.com/acorn-io/runtime/pkg/server/registry/apigroups/admin"
	genericapiserver "k8s.io/apiserver/pkg/server"
	clientgo "k8s.io/client-go/rest"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var apiGroupFactories = []APIGroupFunc{
	admin.APIGroup,
	acorn.APIGroup,
}

type APIGroupFunc func(kclient.WithWatch, *clientgo.Config, *clientgo.Config) (*genericapiserver.APIGroupInfo, error)

func APIGroups(c kclient.WithWatch, cfg, localCfg *clientgo.Config) (result []*genericapiserver.APIGroupInfo, err error) {
	for _, factory := range apiGroupFactories {
		apiGroup, err := factory(c, cfg, localCfg)
		if err != nil {
			return nil, err
		}
		result = append(result, apiGroup)
	}
	return result, nil
}
