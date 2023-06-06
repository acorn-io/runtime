package controller

import (
	"net/http"

	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/controller/acornimagebuildinstance"
	"github.com/acorn-io/acorn/pkg/controller/appdefinition"
	"github.com/acorn-io/acorn/pkg/controller/appstatus"
	"github.com/acorn-io/acorn/pkg/controller/builder"
	"github.com/acorn-io/acorn/pkg/controller/config"
	"github.com/acorn-io/acorn/pkg/controller/defaults"
	"github.com/acorn-io/acorn/pkg/controller/devsession"
	"github.com/acorn-io/acorn/pkg/controller/eventinstance"
	"github.com/acorn-io/acorn/pkg/controller/gc"
	"github.com/acorn-io/acorn/pkg/controller/images"
	"github.com/acorn-io/acorn/pkg/controller/ingress"
	"github.com/acorn-io/acorn/pkg/controller/jobs"
	"github.com/acorn-io/acorn/pkg/controller/namespace"
	"github.com/acorn-io/acorn/pkg/controller/networkpolicy"
	"github.com/acorn-io/acorn/pkg/controller/pvc"
	"github.com/acorn-io/acorn/pkg/controller/quota"
	"github.com/acorn-io/acorn/pkg/controller/scheduling"
	"github.com/acorn-io/acorn/pkg/controller/secrets"
	"github.com/acorn-io/acorn/pkg/controller/service"
	"github.com/acorn-io/acorn/pkg/controller/tls"
	"github.com/acorn-io/acorn/pkg/event"
	"github.com/acorn-io/acorn/pkg/labels"
	"github.com/acorn-io/acorn/pkg/system"
	"github.com/acorn-io/acorn/pkg/volume"
	"github.com/acorn-io/baaah/pkg/router"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	policyv1 "k8s.io/api/policy/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	storagev1 "k8s.io/api/storage/v1"
	klabels "k8s.io/apimachinery/pkg/labels"
)

var (
	managedSelector = klabels.SelectorFromSet(map[string]string{
		labels.AcornManaged: "true",
	})
)

func routes(router *router.Router, registryTransport http.RoundTripper, recorder event.Recorder) {
	router.OnErrorHandler = appdefinition.OnError

	appRouter := router.Type(&v1.AppInstance{}).Middleware(devsession.OverlayDevSession)
	appRouter.HandlerFunc(appstatus.PrepareStatus)
	appRouter.HandlerFunc(appdefinition.AssignNamespace)
	appRouter.HandlerFunc(appdefinition.CheckImageAllowedHandler(registryTransport))
	appRouter.HandlerFunc(appdefinition.PullAppImage(registryTransport, recorder))
	appRouter.HandlerFunc(images.CreateImages)
	appRouter.HandlerFunc(appdefinition.ParseAppImage)
	appRouter.HandlerFunc(tls.ProvisionCerts) // Provision TLS certificates for port bindings with user-defined (valid) domains
	appRouter.Middleware(appdefinition.FilterLabelsAndAnnotationsConfig).HandlerFunc(namespace.AddNamespace)
	appRouter.Middleware(jobs.NeedsDestroyJobFinalization).FinalizeFunc(jobs.DestroyJobFinalizer, jobs.FinalizeDestroyJob)

	// DeploySpec will create the namespace, so ensure it runs before anything that requires a namespace
	appHasNamespace := appRouter.Middleware(appdefinition.RequireNamespace, appdefinition.IgnoreTerminatingNamespace, appdefinition.FilterLabelsAndAnnotationsConfig)
	appHasNamespace.HandlerFunc(defaults.Calculate)
	appHasNamespace.HandlerFunc(scheduling.Calculate)
	appHasNamespace.HandlerFunc(quota.EnsureQuotaRequest)
	appHasNamespace.HandlerFunc(quota.WaitForAllocation)

	appMeetsPreconditions := appHasNamespace.Middleware(appstatus.CheckStatus)
	appMeetsPreconditions.Middleware(appdefinition.ImagePulled).IncludeRemoved().HandlerFunc(appdefinition.DeploySpec)
	appMeetsPreconditions.Middleware(appdefinition.ImagePulled).HandlerFunc(secrets.CreateSecrets)
	appMeetsPreconditions.HandlerFunc(appstatus.SetStatus)
	appMeetsPreconditions.HandlerFunc(appstatus.ReadyStatus)
	appMeetsPreconditions.HandlerFunc(networkpolicy.ForApp)
	appMeetsPreconditions.HandlerFunc(appdefinition.AddAcornProjectLabel)
	appMeetsPreconditions.HandlerFunc(appdefinition.UpdateObservedFields)

	appRouter.HandlerFunc(appstatus.CLIStatus)

	router.Type(&v1.DevSessionInstance{}).HandlerFunc(devsession.ExpireDevSession)

	router.Type(&v1.ServiceInstance{}).HandlerFunc(service.RenderServices)

	router.Type(&v1.BuilderInstance{}).HandlerFunc(builder.SetRegion)
	router.Type(&v1.BuilderInstance{}).HandlerFunc(builder.DeployBuilder)

	router.Type(&v1.AcornImageBuildInstance{}).HandlerFunc(acornimagebuildinstance.SetRegion)
	router.Type(&v1.AcornImageBuildInstance{}).HandlerFunc(acornimagebuildinstance.MarkRecorded)

	router.Type(&v1.ServiceInstance{}).HandlerFunc(gc.GCOrphans)

	router.Type(&v1.EventInstance{}).HandlerFunc(eventinstance.GCExpired())

	router.Type(&batchv1.Job{}).Selector(managedSelector).HandlerFunc(jobs.JobCleanup)
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
	router.Type(&corev1.Pod{}).Selector(managedSelector).HandlerFunc(jobs.JobPodOrphanCleanup)
	router.Type(&netv1.Ingress{}).Selector(managedSelector).Namespace(system.ImagesNamespace).HandlerFunc(gc.GCOrphans)
	router.Type(&netv1.Ingress{}).Selector(managedSelector).Middleware(ingress.RequireLBs).Handler(ingress.NewDNSHandler())
	router.Type(&corev1.Secret{}).Selector(managedSelector).Middleware(tls.RequireSecretTypeTLS).HandlerFunc(tls.RenewCert) // renew (expired) TLS certificates, including the oss-acorn.io wildcard cert
	router.Type(&storagev1.StorageClass{}).HandlerFunc(volume.SyncVolumeClasses)
	router.Type(&corev1.Service{}).Selector(managedSelector).HandlerFunc(networkpolicy.ForService)
	router.Type(&netv1.Ingress{}).Selector(managedSelector).HandlerFunc(networkpolicy.ForIngress)
	router.Type(&appsv1.Deployment{}).Namespace(system.ImagesNamespace).HandlerFunc(networkpolicy.ForBuilder)
	router.Type(&netv1.NetworkPolicy{}).Selector(managedSelector).HandlerFunc(gc.GCOrphans)

	configRouter := router.Type(&corev1.ConfigMap{}).Namespace(system.Namespace).Name(system.ConfigName)
	configRouter.Handler(config.NewDNSConfigHandler())
	configRouter.HandlerFunc(builder.DeployRegistry)
	configRouter.HandlerFunc(config.HandleAutoUpgradeInterval)
	configRouter.HandlerFunc(volume.CreateEphemeralVolumeClass)
}
