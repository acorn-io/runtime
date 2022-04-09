package controller

import (
	"github.com/ibuildthecloud/baaah/pkg/router"
	v1 "github.com/ibuildthecloud/herd/pkg/apis/herd-project.io/v1"
	"github.com/ibuildthecloud/herd/pkg/controller/appdefinition"
	"github.com/ibuildthecloud/herd/pkg/controller/namespace"
	"github.com/ibuildthecloud/herd/pkg/controller/pvc"
	"github.com/ibuildthecloud/herd/pkg/labels"
	corev1 "k8s.io/api/core/v1"
	klabels "k8s.io/apimachinery/pkg/labels"
)

var (
	managedSelector = klabels.SelectorFromSet(map[string]string{
		labels.HerdManaged: "true",
	})
)

func routes(router *router.Router, c Config) {
	router.HandleFunc(&v1.AppInstance{}, appdefinition.AssignNamespace)
	router.Type(&v1.AppInstance{}).Middleware(appdefinition.RequireNamespace).Handler(appdefinition.PullAppImage(c.Images.AppImageInitImage))
	router.HandleFunc(&v1.AppInstance{}, appdefinition.ParseAppImage)
	router.Type(&v1.AppInstance{}).Middleware(appdefinition.RequireNamespace).HandlerFunc(appdefinition.CreateSecrets)
	router.Type(&v1.AppInstance{}).Middleware(appdefinition.RequireNamespace).HandlerFunc(appdefinition.DeploySpec)
	router.Type(&v1.AppInstance{}).Middleware(appdefinition.RequireNamespace).HandlerFunc(appdefinition.AppStatus)
	router.Type(&v1.AppInstance{}).Middleware(appdefinition.RequireNamespace).HandlerFunc(appdefinition.AppEndpointsStatus)
	router.Type(&v1.AppInstance{}).Middleware(appdefinition.RequireNamespace).HandlerFunc(appdefinition.JobStatus)
	router.Type(&v1.AppInstance{}).Middleware(appdefinition.RequireNamespace).HandlerFunc(appdefinition.CLIStatus)
	router.HandleFunc(&v1.AppInstance{}, appdefinition.ReleaseVolume)

	router.Type(&corev1.PersistentVolumeClaim{}).Selector(managedSelector).HandlerFunc(pvc.MarkAndSave)
	router.Type(&corev1.Namespace{}).Selector(managedSelector).HandlerFunc(namespace.DeleteOrphaned)
}
