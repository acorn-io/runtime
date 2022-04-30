package controller

import (
	v1 "github.com/acorn-io/acorn/pkg/apis/acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/controller/acornrouter"
	"github.com/acorn-io/acorn/pkg/controller/appdefinition"
	"github.com/acorn-io/acorn/pkg/controller/namespace"
	"github.com/acorn-io/acorn/pkg/controller/pvc"
	"github.com/acorn-io/acorn/pkg/labels"
	"github.com/acorn-io/baaah/pkg/router"
	corev1 "k8s.io/api/core/v1"
	klabels "k8s.io/apimachinery/pkg/labels"
)

var (
	managedSelector = klabels.SelectorFromSet(map[string]string{
		labels.AcornManaged: "true",
	})
)

func routes(router *router.Router) {
	router.HandleFunc(&v1.AppInstance{}, appdefinition.AssignNamespace)
	router.Type(&v1.AppInstance{}).Middleware(appdefinition.RequireNamespace).HandlerFunc(appdefinition.PullAppImage)
	router.HandleFunc(&v1.AppInstance{}, appdefinition.ParseAppImage)
	router.Type(&v1.AppInstance{}).Middleware(appdefinition.RequireNamespace).HandlerFunc(appdefinition.CreateSecrets)
	router.Type(&v1.AppInstance{}).Middleware(appdefinition.RequireNamespace).HandlerFunc(appdefinition.DeploySpec)
	router.Type(&v1.AppInstance{}).Middleware(appdefinition.RequireNamespace).HandlerFunc(acornrouter.AcornRouter)
	router.Type(&v1.AppInstance{}).Middleware(appdefinition.RequireNamespace).HandlerFunc(appdefinition.AppStatus)
	router.Type(&v1.AppInstance{}).Middleware(appdefinition.RequireNamespace).HandlerFunc(appdefinition.AppEndpointsStatus)
	router.Type(&v1.AppInstance{}).Middleware(appdefinition.RequireNamespace).HandlerFunc(appdefinition.JobStatus)
	router.Type(&v1.AppInstance{}).Middleware(appdefinition.RequireNamespace).HandlerFunc(appdefinition.CLIStatus)
	router.HandleFunc(&v1.AppInstance{}, appdefinition.ReleaseVolume)

	router.Type(&corev1.PersistentVolumeClaim{}).Selector(managedSelector).HandlerFunc(pvc.MarkAndSave)
	router.Type(&corev1.Namespace{}).Selector(managedSelector).HandlerFunc(namespace.DeleteOrphaned)
}
