package controller

import (
	"context"

	"github.com/ibuildthecloud/baaah"
	"github.com/ibuildthecloud/baaah/pkg/crds"
	"github.com/ibuildthecloud/baaah/pkg/router"
	v1 "github.com/ibuildthecloud/herd/pkg/apis/herd-project.io/v1"
	"github.com/ibuildthecloud/herd/pkg/controller/appdefinition"
	"github.com/ibuildthecloud/herd/pkg/scheme"
	"k8s.io/apimachinery/pkg/runtime"
)

type Controller struct {
	Router *router.Router
	Scheme *runtime.Scheme
}

type Config struct {
	AppImageInitImage string
}

func New(c Config) (*Controller, error) {
	router, err := baaah.DefaultRouter(scheme.Scheme)
	if err != nil {
		return nil, err
	}

	routes(router, c)

	return &Controller{
		Router: router,
		Scheme: scheme.Scheme,
	}, nil
}

func (c *Controller) Start(ctx context.Context) error {
	if err := crds.Create(ctx, c.Scheme, v1.SchemeGroupVersion); err != nil {
		return err
	}
	return c.Router.Start(ctx)
}

func routes(router *router.Router, c Config) {
	router.HandleFunc(&v1.AppInstance{}, appdefinition.PullAppImage(c.AppImageInitImage))
	router.HandleFunc(&v1.AppInstance{}, appdefinition.ParseAppImage)
	router.HandleFunc(&v1.AppInstance{}, appdefinition.AssignNamespace)
	router.HandleFunc(&v1.AppInstance{}, appdefinition.DeploySpec)
}
