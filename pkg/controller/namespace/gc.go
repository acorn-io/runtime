package namespace

import (
	"strings"

	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/labels"
	"github.com/acorn-io/acorn/pkg/namespace"
	"github.com/acorn-io/baaah/pkg/router"
	"golang.org/x/exp/maps"
	corev1 "k8s.io/api/core/v1"
	apierror "k8s.io/apimachinery/pkg/api/errors"
	klabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func DeleteOrphaned(req router.Request, resp router.Response) error {
	ns := req.Object.(*corev1.Namespace)
	if ns.Status.Phase != corev1.NamespaceActive {
		return nil
	}

	appName := req.Object.GetLabels()[labels.AcornAppName]
	appNamespace := req.Object.GetLabels()[labels.AcornAppNamespace]

	err := req.Client.Get(req.Ctx, router.Key(appNamespace, appName), &v1.AppInstance{})
	if apierror.IsNotFound(err) {
		return req.Client.Delete(req.Ctx, ns)
	}
	return err
}

func SetupHierarchy(req router.Request, resp router.Response) error {
	ns := req.Object.(*corev1.Namespace)
	if ns.Status.Phase != corev1.NamespaceActive {
		return nil
	}

	appInstances := &v1.AppInstanceList{}
	err := req.Client.List(req.Ctx, appInstances, &kclient.ListOptions{
		Namespace: ns.Name,
	})
	if err != nil {
		return err
	}

	newChildren := map[string]string{}
	for _, appInstance := range appInstances.Items {
		if appInstance.Status.Namespace == "" {
			continue
		}
		newChildren[appInstance.Name] = appInstance.Status.Namespace

		childNS := &corev1.Namespace{}
		err := req.Client.Get(req.Ctx, router.Key(appInstance.Status.Namespace, ""), childNS)
		if apierror.IsNotFound(err) {
			continue
		} else if err != nil {
			return err
		}

		childData, err := namespace.Children(childNS)
		if err != nil {
			return err
		}

		for k, v := range childData {
			newChildren[appInstance.Name+"."+k] = v
		}
	}

	oldChildren, err := namespace.Children(ns)
	if err != nil {
		return err
	}

	if err := addOrphans(oldChildren, newChildren, req); err != nil {
		return err
	}

	if !maps.Equal(oldChildren, newChildren) {
		if err := namespace.SetChildren(ns, newChildren); err != nil {
			return err
		}
		return req.Client.Update(req.Ctx, ns)
	}

	return nil
}

func addOrphans(oldChildren, newChildren map[string]string, req router.Request) error {
	orphans := map[string]string{}

	for k, v := range oldChildren {
		if _, ok := newChildren[k]; !ok {
			// ignore direct descendents, only 2 levels or more deep
			if strings.Contains(k, ".") {
				orphans[k] = v
			}
		}
	}

	if len(orphans) == 0 {
		return nil
	}

	pvs := &corev1.PersistentVolumeList{}
	nsReq, err := klabels.NewRequirement(labels.AcornAppNamespace, selection.In, maps.Values(orphans))
	if err != nil {
		return err
	}

	err = req.Client.List(req.Ctx, pvs, &kclient.ListOptions{
		LabelSelector: klabels.NewSelector().Add(*nsReq),
	})
	if err != nil {
		return err
	}

	for _, pv := range pvs.Items {
		if pv.Labels[labels.AcornManaged] != "true" {
			continue
		}
		appNS, ok := pv.Labels[labels.AcornAppNamespace]
		if !ok {
			continue
		}
		for k, v := range orphans {
			if v == appNS {
				newChildren[k] = v
			}
		}
	}

	return nil
}
