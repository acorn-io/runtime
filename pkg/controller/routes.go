package controller

import (
	"net/http"

	policyv1 "k8s.io/api/policy/v1"

	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/controller/acornimagebuildinstance"
	"github.com/acorn-io/acorn/pkg/controller/appdefinition"
	"github.com/acorn-io/acorn/pkg/controller/builder"
	"github.com/acorn-io/acorn/pkg/controller/config"
	"github.com/acorn-io/acorn/pkg/controller/defaults"
	"github.com/acorn-io/acorn/pkg/controller/gc"
	"github.com/acorn-io/acorn/pkg/controller/ingress"
	"github.com/acorn-io/acorn/pkg/controller/namespace"
	"github.com/acorn-io/acorn/pkg/controller/pvc"
	"github.com/acorn-io/acorn/pkg/controller/scheduling"
	"github.com/acorn-io/acorn/pkg/controller/secrets"
	"github.com/acorn-io/acorn/pkg/controller/service"
	"github.com/acorn-io/acorn/pkg/controller/tls"
	"github.com/acorn-io/acorn/pkg/labels"
	"github.com/acorn-io/acorn/pkg/system"
	"github.com/acorn-io/acorn/pkg/volume"
	"github.com/acorn-io/baaah/pkg/router"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	storagev1 "k8s.io/api/storage/v1"
	klabels "k8s.io/apimachinery/pkg/labels"
)

var (
	managedSelector = klabels.SelectorFromSet(map[string]string{
		labels.AcornManaged: "true",
	})
)

func routes(router *router.Router, registryTransport http.RoundTripper) {
	router.OnErrorHandler = appdefinition.OnError

	router.HandleFunc(&v1.AppInstance{}, appdefinition.AssignNamespace)
	router.HandleFunc(&v1.AppInstance{}, appdefinition.CheckImageAllowedHandler(registryTransport))
	router.Type(&v1.AppInstance{}).HandlerFunc(appdefinition.PullAppImage(registryTransport))
	router.HandleFunc(&v1.AppInstance{}, appdefinition.ParseAppImage)
	router.HandleFunc(&v1.AppInstance{}, tls.ProvisionCerts) // Provision TLS certificates for port bindings with user-defined (valid) domains
	router.Type(&v1.AppInstance{}).Middleware(appdefinition.FilterLabelsAndAnnotationsConfig).HandlerFunc(namespace.AddNamespace)

	// DeploySpec will create the namespace, so ensure it runs before anything that requires a namespace
	appRouter := router.Type(&v1.AppInstance{}).Middleware(appdefinition.RequireNamespace, appdefinition.IgnoreTerminatingNamespace, appdefinition.FilterLabelsAndAnnotationsConfig)
	appRouter.HandlerFunc(defaults.Calculate)
	appRouter.HandlerFunc(scheduling.Calculate)
	appRouter = appRouter.Middleware(appdefinition.CheckStatus)
	appRouter.Middleware(appdefinition.ImagePulled, appdefinition.CheckDependencies).HandlerFunc(appdefinition.DeploySpec)
	appRouter.Middleware(appdefinition.ImagePulled).HandlerFunc(secrets.CreateSecrets)
	appRouter.HandlerFunc(appdefinition.AppStatus)
	appRouter.HandlerFunc(appdefinition.AppEndpointsStatus)
	appRouter.HandlerFunc(appdefinition.JobStatus)
	appRouter.HandlerFunc(appdefinition.VolumeStatus)
	appRouter.HandlerFunc(appdefinition.AcornStatus)
	appRouter.HandlerFunc(appdefinition.ReadyStatus)
	appRouter.HandlerFunc(appdefinition.NetworkPolicyForApp)
	appRouter.HandlerFunc(appdefinition.AddAcornProjectLabel)
	appRouter.HandlerFunc(appdefinition.UpdateGeneration)

	router.Type(&v1.AppInstance{}).HandlerFunc(appdefinition.CLIStatus)

	router.Type(&v1.ServiceInstance{}).HandlerFunc(service.RenderServices)

	router.Type(&v1.BuilderInstance{}).HandlerFunc(builder.DeployBuilder)

	router.Type(&v1.AcornImageBuildInstance{}).HandlerFunc(acornimagebuildinstance.MarkRecorded)

	router.Type(&v1.ServiceInstance{}).HandlerFunc(gc.GCOrphans)
	router.Type(&rbacv1.ClusterRole{}).Selector(managedSelector).HandlerFunc(gc.GCOrphans)
	router.Type(&rbacv1.ClusterRoleBinding{}).Selector(managedSelector).HandlerFunc(gc.GCOrphans)
	router.Type(&corev1.PersistentVolumeClaim{}).Selector(managedSelector).HandlerFunc(pvc.MarkAndSave)
	router.Type(&corev1.PersistentVolume{}).Selector(managedSelector).HandlerFunc(appdefinition.ReleaseVolume)
	router.Type(&corev1.Namespace{}).Selector(managedSelector).HandlerFunc(namespace.DeleteOrphaned)
	router.Type(&appsv1.DaemonSet{}).Namespace(system.ImagesNamespace).HandlerFunc(gc.GCOrphans)
	router.Type(&appsv1.Deployment{}).Namespace(system.ImagesNamespace).HandlerFunc(gc.GCOrphans)
	router.Type(&corev1.Service{}).Selector(managedSelector).HandlerFunc(gc.GCOrphans)
	router.Type(&policyv1.PodDisruptionBudget{}).Namespace(system.ImagesNamespace).HandlerFunc(gc.GCOrphans)
	router.Type(&corev1.Pod{}).Selector(managedSelector).HandlerFunc(gc.GCOrphans)
	router.Type(&netv1.Ingress{}).Selector(managedSelector).Namespace(system.ImagesNamespace).HandlerFunc(gc.GCOrphans)
	router.Type(&netv1.Ingress{}).Selector(managedSelector).Middleware(ingress.RequireLBs).Handler(ingress.NewDNSHandler())
	router.Type(&corev1.Secret{}).Selector(managedSelector).Middleware(tls.RequireSecretTypeTLS).HandlerFunc(tls.RenewCert) // renew (expired) TLS certificates, including the on-acorn.io wildcard cert
	router.Type(&storagev1.StorageClass{}).HandlerFunc(volume.SyncVolumeClasses)
	router.Type(&corev1.Service{}).Selector(managedSelector).HandlerFunc(appdefinition.NetworkPolicyForService)
	router.Type(&netv1.Ingress{}).Selector(managedSelector).HandlerFunc(appdefinition.NetworkPolicyForIngress)
	router.Type(&netv1.NetworkPolicy{}).Selector(managedSelector).HandlerFunc(gc.GCOrphans)

	configRouter := router.Type(&corev1.ConfigMap{}).Namespace(system.Namespace).Name(system.ConfigName)
	configRouter.Handler(config.NewDNSConfigHandler())
	configRouter.HandlerFunc(builder.DeployRegistry)
	configRouter.HandlerFunc(config.HandleAutoUpgradeInterval)
	configRouter.HandlerFunc(volume.CreateEphemeralVolumeClass)
}
