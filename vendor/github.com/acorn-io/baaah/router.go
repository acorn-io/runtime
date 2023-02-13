package baaah

import (
	"github.com/acorn-io/baaah/pkg/backend"
	"github.com/acorn-io/baaah/pkg/lasso"
	"github.com/acorn-io/baaah/pkg/leader"
	"github.com/acorn-io/baaah/pkg/restconfig"
	"github.com/acorn-io/baaah/pkg/router"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
)

type Options struct {
	Backend    backend.Backend
	RESTConfig *rest.Config
	Namespace  string
	// ElectionConfig being nil represents no leader election for the router.
	ElectionConfig *leader.ElectionConfig
}

func (o *Options) complete(scheme *runtime.Scheme) (*Options, error) {
	var result Options
	if o != nil {
		result = *o
	}

	if result.Backend == nil {
		if result.RESTConfig == nil {
			var err error
			result.RESTConfig, err = restconfig.New(scheme)
			if err != nil {
				return nil, err
			}
		}

		backend, err := lasso.NewRuntimeForNamespace(o.RESTConfig, o.Namespace, scheme)
		if err != nil {
			return nil, err
		}
		result.Backend = backend.Backend
	}

	return &result, nil
}

// DefaultOptions represent the standard options for a Router.
// The default leader election uses a lease lock and a TTL of 15 seconds.
func DefaultOptions(routerName string, scheme *runtime.Scheme) (*Options, error) {
	cfg, err := restconfig.New(scheme)
	if err != nil {
		return nil, err
	}
	rt, err := lasso.NewRuntimeForNamespace(cfg, "", scheme)
	if err != nil {
		return nil, err
	}

	return &Options{
		Backend:        rt.Backend,
		RESTConfig:     cfg,
		ElectionConfig: leader.NewDefaultElectionConfig("", routerName, cfg),
	}, nil
}

// DefaultRouter The routerName is important as this name will be used to assign ownership of objects created by this
// router. Specifically the routerName is assigned to the sub-context in the apply actions. Additionally, the routerName
// will be used for the leader election lease lock.
func DefaultRouter(routerName string, scheme *runtime.Scheme) (*router.Router, error) {
	opts, err := DefaultOptions(routerName, scheme)
	if err != nil {
		return nil, err
	}
	return NewRouter(routerName, scheme, opts)
}

func NewRouter(handlerName string, scheme *runtime.Scheme, opts *Options) (*router.Router, error) {
	opts, err := opts.complete(scheme)
	if err != nil {
		return nil, err
	}
	return router.New(router.NewHandlerSet(handlerName, scheme, opts.Backend), opts.ElectionConfig), nil
}
