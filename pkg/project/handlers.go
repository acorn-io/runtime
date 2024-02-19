package project

import (
	"fmt"
	"time"

	"github.com/acorn-io/baaah/pkg/router"
	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	internaladminv1 "github.com/acorn-io/runtime/pkg/apis/internal.admin.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/computeclasses"
	"github.com/acorn-io/runtime/pkg/labels"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/strings/slices"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// SetDefaultComputeClass sets the default compute class status field of a [v1.ProjectInstance] to the value of its spec
// field if set.
func SetDefaultComputeClass(req router.Request, _ router.Response) error {
	project := req.Object.(*v1.ProjectInstance)
	if cc := project.Spec.DefaultComputeClass; cc != "" && project.Status.DefaultComputeClass != cc {
		// The spec has been changed, update the status field to match.
		project.Status.DefaultComputeClass = cc
	}

	// Check if the given compute class exists
	if computeClassName := project.Status.DefaultComputeClass; computeClassName != "" {
		if _, err := computeclasses.GetAsProjectComputeClassInstance(req.Ctx, req.Client, project.Name, computeClassName); err != nil {
			if !apierrors.IsNotFound(err) {
				return fmt.Errorf("failed to check existence of default compute class [%s] specified by project [%s] status: %w", computeClassName, project.Name, err)
			}

			// The compute class does not exist, clear the status field.
			project.Status.DefaultComputeClass = ""
		}
	}

	// Pick a default from the available compute classes
	if project.Status.DefaultComputeClass == "" {
		computeClassName, err := internaladminv1.GetDefaultComputeClassName(req.Ctx, req.Client, project.Name)
		if kclient.IgnoreNotFound(err) != nil {
			return fmt.Errorf("failed to get default compute class for project [%s]: %w", project.Name, err)
		}

		project.Status.DefaultComputeClass = computeClassName
	}

	return nil
}

func SetSupportedRegions(req router.Request, resp router.Response) error {
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
	apps := new(v1.AppInstanceList)
	if err := req.List(apps, &kclient.ListOptions{
		Namespace: req.Object.GetName(),
	}); err != nil {
		return err
	}

	existingApps := make(map[string]struct{}, len(apps.Items))
	for _, app := range apps.Items {
		existingApps[app.Name] = struct{}{}
	}

	// Note: using index here to avoid the loop variable issue.
	for i := range apps.Items {
		// If the app's parent is gone, then ensure this app is deleted.
		if _, ok := existingApps[apps.Items[i].Labels[labels.AcornParentAcornName]]; !ok && apps.Items[i].DeletionTimestamp.IsZero() {
			if err := req.Client.Delete(req.Ctx, &apps.Items[i]); err != nil && !apierrors.IsNotFound(err) {
				return err
			}
		}
	}

	if len(apps.Items) > 0 {
		resp.RetryAfter(5 * time.Second)
	}

	return nil
}
