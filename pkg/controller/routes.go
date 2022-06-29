package controller

import (
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/controller/acornrouter"
	"github.com/acorn-io/acorn/pkg/controller/appdefinition"
	"github.com/acorn-io/acorn/pkg/controller/gc"
	"github.com/acorn-io/acorn/pkg/controller/namespace"
	"github.com/acorn-io/acorn/pkg/controller/pvc"
	"github.com/acorn-io/acorn/pkg/labels"
	"github.com/acorn-io/acorn/pkg/system"
	"github.com/acorn-io/baaah/pkg/router"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	klabels "k8s.io/apimachinery/pkg/labels"
)

var (
	managedSelector = klabels.SelectorFromSet(map[string]string{
		labels.AcornManaged: "true",
	})
)

func routes(router *router.Router) {
	router.HandleFunc(&v1.AppInstance{}, appdefinition.AssignNamespace)
	router.HandleFunc(&v1.AppInstance{}, appdefinition.PullAppImage)
	router.HandleFunc(&v1.AppInstance{}, appdefinition.ParseAppImage)

	// DeploySpec will create the namespace, so ensure it runs before anything that requires a namespace
	appRouter := router.Type(&v1.AppInstance{}).Middleware(appdefinition.RequireNamespace).Middleware(appdefinition.IgnoreTerminatingNamespace)
	appRouter.Middleware(appdefinition.CheckDependencies).HandlerFunc(appdefinition.DeploySpec)
	appRouter.HandlerFunc(appdefinition.CreateSecrets)
	appRouter.HandlerFunc(acornrouter.AcornRouter)
	appRouter.HandlerFunc(appdefinition.AppStatus)
	appRouter.HandlerFunc(appdefinition.AppEndpointsStatus)
	appRouter.HandlerFunc(appdefinition.JobStatus)
	appRouter.HandlerFunc(appdefinition.AcornStatus)
	appRouter.HandlerFunc(appdefinition.ReadyStatus)
	appRouter.HandlerFunc(appdefinition.CLIStatus)
	router.HandleFunc(&v1.AppInstance{}, appdefinition.ReleaseVolume)

	router.Type(&rbacv1.ClusterRole{}).Selector(managedSelector).HandlerFunc(gc.GCOrphans)
	router.Type(&rbacv1.ClusterRoleBinding{}).Selector(managedSelector).HandlerFunc(gc.GCOrphans)
	router.Type(&corev1.PersistentVolumeClaim{}).Selector(managedSelector).HandlerFunc(pvc.MarkAndSave)
	router.Type(&corev1.Namespace{}).Selector(managedSelector).HandlerFunc(namespace.DeleteOrphaned)
	router.Type(&appsv1.DaemonSet{}).Namespace(system.Namespace).HandlerFunc(acornrouter.GCAcornRouter)
	router.Type(&corev1.Service{}).Namespace(system.Namespace).HandlerFunc(acornrouter.GCAcornRouterService)
}
