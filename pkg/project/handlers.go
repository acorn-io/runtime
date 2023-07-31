package project

import (
	"time"

	"github.com/acorn-io/baaah/pkg/router"
	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/labels"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	klabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/utils/strings/slices"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func SetProjectSupportedRegions(req router.Request, resp router.Response) error {
	project := req.Object.(*v1.ProjectInstance)
	project.SetDefaultRegion(apiv1.LocalRegion)
	if slices.Contains(project.Status.SupportedRegions, apiv1.AllRegions) {
		// If the project supports all regions, then ensure the default region and the local region are supported regions.
		project.Status.SupportedRegions = []string{project.Status.DefaultRegion}
		if project.Status.DefaultRegion != apiv1.LocalRegion {
			project.Status.SupportedRegions = append(project.Status.SupportedRegions, apiv1.LocalRegion)
		}
	}

	resp.Objects(req.Object)
	return nil
}

func CreateNamespace(req router.Request, resp router.Response) error {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:        req.Object.GetName(),
			Annotations: make(map[string]string, len(req.Object.GetAnnotations())),
			Labels: map[string]string{
				labels.AcornManaged: "true",
				labels.AcornProject: "true",
			},
		},
	}

	for key, value := range req.Object.GetLabels() {
		ns.Labels[key] = value
	}

	for key, value := range req.Object.GetAnnotations() {
		ns.Annotations[key] = value
	}

	resp.Objects(ns)
	return nil
}

// EnsureAllAppsRemoved ensures that all apps are removed from the project before the namespace is deleted.
func EnsureAllAppsRemoved(req router.Request, resp router.Response) error {
	// A "child" app is one that has a parent label. Therefore, all "parent" apps won't have a parent label.
	parentAppRequirement, err := klabels.NewRequirement(labels.AcornParentAcornName, selection.DoesNotExist, nil)
	if err != nil {
		return err
	}

	// First delete all "parent" apps. If no "parent" apps exist, then ensure that all apps are deleted.
	for _, ls := range []klabels.Selector{klabels.NewSelector().Add(*parentAppRequirement), klabels.Everything()} {
		apps := new(v1.AppInstanceList)
		if err = req.List(apps, &kclient.ListOptions{
			Namespace:     req.Object.GetName(),
			LabelSelector: ls,
		}); err != nil {
			return err
		}

		for _, app := range apps.Items {
			if app.DeletionTimestamp.IsZero() {
				if err = req.Client.Delete(req.Ctx, &app); err != nil && !apierrors.IsNotFound(err) {
					return err
				}
			}
		}

		if len(apps.Items) > 0 {
			resp.RetryAfter(5 * time.Second)
			return nil
		}
	}

	return nil
}
