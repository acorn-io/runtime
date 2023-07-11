package controller

import (
	"net/http"

	"github.com/acorn-io/baaah/pkg/apply"
	"github.com/acorn-io/baaah/pkg/router"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/controller/acornimagebuildinstance"
	"github.com/acorn-io/runtime/pkg/controller/appdefinition"
	"github.com/acorn-io/runtime/pkg/controller/appstatus"
	"github.com/acorn-io/runtime/pkg/controller/builder"
	"github.com/acorn-io/runtime/pkg/controller/config"
	"github.com/acorn-io/runtime/pkg/controller/defaults"
	"github.com/acorn-io/runtime/pkg/controller/devsession"
	"github.com/acorn-io/runtime/pkg/controller/eventinstance"
	"github.com/acorn-io/runtime/pkg/controller/gc"
	"github.com/acorn-io/runtime/pkg/controller/images"
	"github.com/acorn-io/runtime/pkg/controller/ingress"
	"github.com/acorn-io/runtime/pkg/controller/jobs"
	"github.com/acorn-io/runtime/pkg/controller/namespace"
	"github.com/acorn-io/runtime/pkg/controller/networkpolicy"
	"github.com/acorn-io/runtime/pkg/controller/pvc"
	"github.com/acorn-io/runtime/pkg/controller/quota"
	"github.com/acorn-io/runtime/pkg/controller/scheduling"
	"github.com/acorn-io/runtime/pkg/controller/secrets"
	"github.com/acorn-io/runtime/pkg/controller/service"
	"github.com/acorn-io/runtime/pkg/controller/tls"
	"github.com/acorn-io/runtime/pkg/event"
	"github.com/acorn-io/runtime/pkg/labels"
	"github.com/acorn-io/runtime/pkg/project"
	"github.com/acorn-io/runtime/pkg/system"
	"github.com/acorn-io/runtime/pkg/volume"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	policyv1 "k8s.io/api/policy/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	storagev1 "k8s.io/api/storage/v1"
	klabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/rest"
)

var (
	managedSelector = klabels.SelectorFromSet(map[string]string{
		labels.AcornManaged: "true",
	})
)

func routes(router *router.Router, cfg *rest.Config, registryTransport http.RoundTripper, recorder event.Recorder) error {
	jobsHandler, err := jobs.NewHandler(cfg)
	if err != nil {
		return err
	}

	apply.AddValidOwnerChange("acorn-install", "acorn-controller")
	router.OnErrorHandler = appdefinition.OnError

	appRouter := router.Type(&v1.AppInstance{}).Middleware(devsession.OverlayDevSession).IncludeFinalizing()
	appRouter.HandlerFunc(appstatus.PrepareStatus)
	appRouter.HandlerFunc(appdefinition.AssignNamespace)
	appRouter.HandlerFunc(appdefinition.CheckImageAllowedHandler(registryTransport))
	appRouter.HandlerFunc(appdefinition.PullAppImage(registryTransport, recorder))
	appRouter.HandlerFunc(images.CreateImages)
	appRouter.HandlerFunc(appdefinition.ParseAppImage)
	appRouter.Middleware(appdefinition.FilterLabelsAndAnnotationsConfig).HandlerFunc(namespace.AddNamespace)
	appRouter.Middleware(jobs.NeedsDestroyJobFinalization).FinalizeFunc(jobs.DestroyJobFinalizer, jobs.FinalizeDestroyJob)

	// DeploySpec will create the namespace, so ensure it runs before anything that requires a namespace
	appHasNamespace := appRouter.Middleware(appdefinition.RequireNamespace, appdefinition.IgnoreTerminatingNamespace, appdefinition.FilterLabelsAndAnnotationsConfig)
	appHasNamespace.HandlerFunc(defaults.Calculate)
	appHasNamespace.HandlerFunc(scheduling.Calculate)
	appHasNamespace.HandlerFunc(quota.EnsureQuotaRequest)
	appHasNamespace.HandlerFunc(quota.WaitForAllocation)

	appMeetsPreconditions := appHasNamespace.Middleware(appstatus.CheckStatus)
	appMeetsPreconditions.Middleware(appdefinition.ImagePulled).HandlerFunc(appdefinition.DeploySpec)
	appMeetsPreconditions.Middleware(appdefinition.ImagePulled).HandlerFunc(secrets.CreateSecrets)
	appMeetsPreconditions.HandlerFunc(appstatus.SetStatus)
	appMeetsPreconditions.HandlerFunc(appstatus.ReadyStatus)
	appMeetsPreconditions.HandlerFunc(networkpolicy.ForApp)
	appMeetsPreconditions.HandlerFunc(appdefinition.AddAcornProjectLabel)
	appMeetsPreconditions.HandlerFunc(appdefinition.UpdateObservedFields)

	appRouter.HandlerFunc(appstatus.CLIStatus)

	projectRouter := router.Type(&v1.ProjectInstance{})
	projectRouter.HandlerFunc(project.SetProjectSupportedRegions)
	projectRouter.HandlerFunc(project.CreateNamespace)
	projectRouter.FinalizeFunc(labels.Prefix+"project-app-delete", project.EnsureAllAppsRemoved)

	router.Type(&v1.DevSessionInstance{}).HandlerFunc(devsession.ExpireDevSession)

	router.Type(&v1.ServiceInstance{}).HandlerFunc(service.RenderServices)

	router.Type(&v1.BuilderInstance{}).HandlerFunc(defaults.SetDefaultRegion)
	router.Type(&v1.BuilderInstance{}).HandlerFunc(builder.DeployBuilder)

	router.Type(&v1.AcornImageBuildInstance{}).HandlerFunc(defaults.SetDefaultRegion)
	router.Type(&v1.AcornImageBuildInstance{}).HandlerFunc(acornimagebuildinstance.MarkRecorded)

	router.Type(&v1.ServiceInstance{}).HandlerFunc(gc.GCOrphans)

	router.Type(&v1.EventInstance{}).HandlerFunc(eventinstance.GCExpired())

	router.Type(&batchv1.Job{}).Selector(managedSelector).HandlerFunc(jobs.JobCleanup)
	router.Type(&rbacv1.ClusterRole{}).Selector(managedSelector).HandlerFunc(gc.GCOrphans)
	router.Type(&rbacv1.ClusterRoleBinding{}).Selector(managedSelector).HandlerFunc(gc.GCOrphans)
	router.Type(&corev1.PersistentVolumeClaim{}).Selector(managedSelector).HandlerFunc(pvc.MarkAndSave)
	router.Type(&corev1.PersistentVolume{}).Selector(managedSelector).HandlerFunc(appdefinition.ReleaseVolume)
	router.Type(&corev1.Namespace{}).Selector(managedSelector).HandlerFunc(namespace.DeleteOrphaned)
	// This will only catch namespace deletes when the controller is running, but that's fine for now.
	router.Type(&corev1.Namespace{}).IncludeRemoved().HandlerFunc(namespace.DeleteProjectOnNamespaceDelete)
	router.Type(&appsv1.DaemonSet{}).Namespace(system.ImagesNamespace).HandlerFunc(gc.GCOrphans)
	router.Type(&appsv1.Deployment{}).Namespace(system.ImagesNamespace).HandlerFunc(gc.GCOrphans)
	router.Type(&corev1.Service{}).Selector(managedSelector).HandlerFunc(gc.GCOrphans)
	router.Type(&policyv1.PodDisruptionBudget{}).Namespace(system.ImagesNamespace).HandlerFunc(gc.GCOrphans)
	router.Type(&corev1.Pod{}).Selector(managedSelector).HandlerFunc(gc.GCOrphans)
	router.Type(&corev1.Pod{}).Selector(managedSelector).HandlerFunc(jobs.JobPodOrphanCleanup)
	router.Type(&corev1.Pod{}).Selector(managedSelector).HandlerFunc(jobsHandler.SaveJobOutput)
	router.Type(&netv1.Ingress{}).Selector(managedSelector).Namespace(system.ImagesNamespace).HandlerFunc(gc.GCOrphans)
	router.Type(&corev1.Secret{}).Selector(managedSelector).Name(system.DNSSecretName).Namespace(system.Namespace).HandlerFunc(secrets.HandleDNSSecret)
	router.Type(&netv1.Ingress{}).Selector(managedSelector).Name(system.DNSIngressName).Namespace(system.Namespace).Middleware(ingress.RequireLBs).Handler(ingress.NewDNSHandler())
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

	return nil
}
