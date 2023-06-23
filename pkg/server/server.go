package server

import (
	"github.com/acorn-io/baaah/pkg/clientaggregator"
	"github.com/acorn-io/baaah/pkg/restconfig"
	"github.com/acorn-io/mink/pkg/server"
	adminapi "github.com/acorn-io/runtime/pkg/apis/admin.acorn.io"
	api "github.com/acorn-io/runtime/pkg/apis/api.acorn.io"
	"github.com/acorn-io/runtime/pkg/k8sclient"
	"github.com/acorn-io/runtime/pkg/openapi"
	"github.com/acorn-io/runtime/pkg/scheme"
	"github.com/acorn-io/runtime/pkg/server/registry"
	apiserver "k8s.io/apiserver/pkg/server"
	"k8s.io/apiserver/pkg/server/options"
	"k8s.io/client-go/rest"
)

type Config struct {
	Version            string
	DefaultOpts        *options.RecommendedOptions
	LocalRestConfig    *rest.Config
	IgnoreStartFailure bool
}

func apiGroups(serverConfig Config) ([]*apiserver.APIGroupInfo, error) {
	restConfig, err := restconfig.New(scheme.Scheme)
	if err != nil {
		return nil, err
	}

	c, err := k8sclient.New(restConfig)
	if err != nil {
		return nil, err
	}

	localCfg := serverConfig.LocalRestConfig
	if localCfg == nil {
		localCfg = restConfig
	} else {
		localClient, err := k8sclient.New(localCfg)
		if err != nil {
			return nil, err
		}
		aggr := clientaggregator.New(c)
		aggr.AddGroup(api.Group, localClient)
		aggr.AddGroup(adminapi.Group, localClient)
		c = aggr
	}

	return registry.APIGroups(c, restConfig, localCfg)
}

func New(cfg Config) (*server.Server, error) {
	apiGroups, err := apiGroups(cfg)
	if err != nil {
		return nil, err
	}

	return server.New(&server.Config{
		Name:                  "Acorn",
		Version:               cfg.Version,
		HTTPSListenPort:       7443,
		LongRunningVerbs:      []string{"watch", "proxy"},
		LongRunningResources:  []string{"exec", "proxy", "log", "registryport", "port", "push", "pull", "portforward"},
		OpenAPIConfig:         openapi.GetOpenAPIDefinitions,
		Scheme:                scheme.Scheme,
		CodecFactory:          &scheme.Codecs,
		APIGroups:             apiGroups,
		DefaultOptions:        cfg.DefaultOpts,
		SupportAPIAggregation: cfg.LocalRestConfig == nil,
		IgnoreStartFailure:    cfg.IgnoreStartFailure,
	})
}
