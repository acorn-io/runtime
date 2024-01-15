package controller

import (
	"context"
	"time"

	"github.com/acorn-io/baaah"
	"github.com/acorn-io/baaah/pkg/apply"
	"github.com/acorn-io/baaah/pkg/restconfig"
	"github.com/acorn-io/baaah/pkg/router"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	adminv1 "github.com/acorn-io/runtime/pkg/apis/internal.admin.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/autoupgrade"
	"github.com/acorn-io/runtime/pkg/config"
	"github.com/acorn-io/runtime/pkg/crds"
	"github.com/acorn-io/runtime/pkg/dns"
	"github.com/acorn-io/runtime/pkg/event"
	"github.com/acorn-io/runtime/pkg/imagemetadatacache"
	"github.com/acorn-io/runtime/pkg/imagesystem"
	"github.com/acorn-io/runtime/pkg/k8sclient"
	"github.com/acorn-io/runtime/pkg/logserver"
	"github.com/acorn-io/runtime/pkg/scheme"
	"github.com/acorn-io/runtime/pkg/system"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	// Enabled logrus logging in baaah
	_ "github.com/acorn-io/baaah/pkg/logrus"
)

var (
	dnsRenewPeriodHours = 24 * time.Hour
)

type Controller struct {
	Router *router.Router
	client client.Client
	Scheme *runtime.Scheme
	apply  apply.Apply
}

func New(ctx context.Context) (*Controller, error) {
	if err := crds.Create(ctx, scheme.Scheme, v1.SchemeGroupVersion, adminv1.SchemeGroupVersion); err != nil {
		return nil, err
	}

	opts, err := baaah.DefaultOptions("acorn-controller", scheme.Scheme)
	if err != nil {
		return nil, err
	}

	if system.IsLocal() {
		opts.ElectionConfig = nil
	}

	router, err := baaah.NewRouter("acorn-controller", opts)
	if err != nil {
		return nil, err
	}

	cfg, err := restconfig.New(scheme.Scheme)
	if err != nil {
		return nil, err
	}

	client, err := k8sclient.New(cfg)
	if err != nil {
		return nil, err
	}

	apply := apply.New(client)

	registryTransport, err := imagesystem.NewAPIBasedTransport(client, cfg)
	if err != nil {
		return nil, err
	}

	err = routes(router, cfg, registryTransport, event.NewRecorder(client))
	if err != nil {
		return nil, err
	}

	return &Controller{
		Router: router,
		client: client,
		Scheme: scheme.Scheme,
		apply:  apply,
	}, nil
}

func (c *Controller) Start(ctx context.Context) error {
	if err := c.initData(ctx); err != nil {
		return err
	}

	go func() {
		var success bool
		for i := 0; i < 6000; i++ {
			// This will error until the cache is primed
			if _, err := config.Get(ctx, c.Router.Backend()); err == nil {
				success = true
				break
			} else {
				time.Sleep(time.Millisecond * 100)
			}
		}
		if !success {
			panic("couldn't initialize client cache")
		}
		dnsInit := dns.NewDaemon(c.Router.Backend())
		go wait.UntilWithContext(ctx, dnsInit.RenewAndSync, dnsRenewPeriodHours)

		autoupgrade.StartSync(ctx, c.Router.Backend())
	}()

	logserver.StartServerWithDefaults()

	// Every 5 minutes, delete EventInstances until only the most recent 1000 remain.
	// Use c.Router.Backend() to ensure we hit the cache when possible.
	// Note: the cache will only be populated for EventInstances if a handler for EventInstances has been registered.
	go event.Truncate(ctx, c.Router.Backend(), 5*time.Minute, 1000)

	go imagemetadatacache.Purge(ctx, c.client)

	return c.Router.Start(ctx)
}
