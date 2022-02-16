package controller

import (
	"github.com/ibuildthecloud/baaah/pkg/router"
	v1 "github.com/ibuildthecloud/herd/pkg/apis/herd-project.io/v1"
	"github.com/ibuildthecloud/herd/pkg/controller/appdefinition"
)

func routes(router *router.Router, c Config) {
	router.HandleFunc(&v1.AppInstance{}, appdefinition.PullAppImage(c.Images.AppImageInitImage))
	router.HandleFunc(&v1.AppInstance{}, appdefinition.ParseAppImage)
	router.HandleFunc(&v1.AppInstance{}, appdefinition.AssignNamespace)
	router.HandleFunc(&v1.AppInstance{}, appdefinition.DeploySpec)
}
