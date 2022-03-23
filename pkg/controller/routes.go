package controller

import (
	"github.com/ibuildthecloud/baaah/pkg/router"
	v1 "github.com/ibuildthecloud/herd/pkg/apis/herd-project.io/v1"
	"github.com/ibuildthecloud/herd/pkg/controller/appdefinition"
	"github.com/ibuildthecloud/herd/pkg/controller/namespace"
	"github.com/ibuildthecloud/herd/pkg/controller/pvc"
	"github.com/ibuildthecloud/herd/pkg/labels"
	corev1 "k8s.io/api/core/v1"
)

func routes(router *router.Router, c Config) {
	router.HandleFunc(&v1.AppInstance{}, appdefinition.RequireNamespace(appdefinition.PullAppImage(c.Images.AppImageInitImage)))
	router.HandleFunc(&v1.AppInstance{}, appdefinition.ParseAppImage)
	router.HandleFunc(&v1.AppInstance{}, appdefinition.AssignNamespace)
	router.HandleFunc(&v1.AppInstance{}, appdefinition.RequireNamespace(appdefinition.DeploySpec))
	router.HandleFunc(&v1.AppInstance{}, appdefinition.RequireNamespace(appdefinition.CreateSecrets))
	router.HandleFunc(&v1.AppInstance{}, appdefinition.ReleaseVolume)

	router.Type(&corev1.PersistentVolumeClaim{}).Selector(map[string]string{
		labels.HerdManaged: "true",
	}).HandlerFunc(pvc.MarkAndSave)

	router.Type(&corev1.Namespace{}).Selector(map[string]string{
		labels.HerdManaged: "true",
	}).HandlerFunc(namespace.DeleteOrphaned)
}
