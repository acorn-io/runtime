package server

import (
	"context"
	"fmt"
	"net"

	api "github.com/acorn-io/acorn/pkg/apis/api.acorn.io"
	kclient "github.com/acorn-io/acorn/pkg/k8sclient"
	openapi2 "github.com/acorn-io/acorn/pkg/openapi"
	"github.com/acorn-io/acorn/pkg/scheme"
	"github.com/acorn-io/acorn/pkg/server/registry"
	"github.com/acorn-io/baaah/pkg/clientaggregator"
	"github.com/acorn-io/baaah/pkg/restconfig"
	"github.com/rancher/wrangler/pkg/merr"
	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apiserver/pkg/endpoints/openapi"
	"k8s.io/apiserver/pkg/server"
	"k8s.io/apiserver/pkg/server/filters"
	"k8s.io/apiserver/pkg/server/options"
	"k8s.io/client-go/rest"
	netutils "k8s.io/utils/net"
)

type Server struct {
	Options *options.RecommendedOptions
}

type Config struct {
	server.RecommendedConfig
	DSN             string
	LocalRestConfig *rest.Config
}

func (s *Server) AddFlags(fs *pflag.FlagSet) {
	s.Options.AddFlags(fs)
}

func (s *Server) NewConfig(version string) (*Config, error) {
	if err := s.Options.SecureServing.MaybeDefaultWithSelfSignedCerts("localhost", nil, []net.IP{netutils.ParseIPSloppy("127.0.0.1")}); err != nil {
		return nil, fmt.Errorf("error creating self-signed certificates: %v", err)
	}

	serverConfig := server.NewRecommendedConfig(scheme.Codecs)
	serverConfig.OpenAPIConfig = server.DefaultOpenAPIConfig(openapi2.GetOpenAPIDefinitions, openapi.NewDefinitionNamer(scheme.Scheme))
	serverConfig.OpenAPIConfig.Info.Title = "Acorn"
	serverConfig.OpenAPIConfig.Info.Version = version
	serverConfig.LongRunningFunc = filters.BasicLongRunningRequestCheck(
		sets.NewString("watch", "proxy"),
		sets.NewString("exec", "proxy", "log", "registryport", "port", "push", "pull"),
	)

	if err := s.Options.ApplyTo(serverConfig); err != nil {
		return nil, err
	}

	return &Config{
		RecommendedConfig: *serverConfig,
	}, nil
}

func (s *Server) Run(ctx context.Context, config *Config) error {
	if errs := s.Options.Validate(); len(errs) > 0 {
		return merr.NewErrors(errs...)
	}

	server, err := config.Complete().New("acorn", server.NewEmptyDelegate())
	if err != nil {
		return err
	}

	cfg, err := restconfig.New(scheme.Scheme)
	if err != nil {
		return err
	}

	c, err := kclient.New(cfg)
	if err != nil {
		return err
	}

	localCfg := config.LocalRestConfig
	if localCfg == nil {
		localCfg = cfg
	} else {
		localClient, err := kclient.New(localCfg)
		if err != nil {
			return err
		}
		aggr := clientaggregator.New(c)
		aggr.AddGroup(api.Group, localClient)
		c = aggr
	}

	apiGroups, err := registry.APIGroups(c, cfg, localCfg, config.DSN)
	if err != nil {
		return err
	}

	if err := server.InstallAPIGroup(apiGroups); err != nil {
		return err
	}

	return server.PrepareRun().Run(ctx.Done())
}

func New() *Server {
	opts := options.NewRecommendedOptions("", nil)
	opts.Audit = nil
	opts.Etcd = nil
	opts.CoreAPI = nil
	opts.Authorization = nil
	opts.Features = nil
	opts.Admission = nil
	opts.SecureServing.BindPort = 7443
	opts.Authentication.RemoteKubeConfigFileOptional = true
	return &Server{
		Options: opts,
	}
}
